package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/webhook-platform/internal/api/middleware"
	"github.com/webhook-platform/internal/domain"
)

type mockTenantService struct {
	tenants   map[string]*domain.Tenant
	apiKeyMap map[string]string
}

func newMockTenantService() *mockTenantService {
	return &mockTenantService{
		tenants:   make(map[string]*domain.Tenant),
		apiKeyMap: make(map[string]string),
	}
}

func (m *mockTenantService) Create(_ context.Context, input domain.CreateTenantInput) (*domain.Tenant, string, error) {
	if errs := input.Validate(); len(errs) > 0 {
		return nil, "", fmt.Errorf("%w: %v", domain.ErrValidation, errs)
	}

	rateLimit := input.RateLimitPerMinute
	if rateLimit == 0 {
		rateLimit = 1000
	}
	maxRetries := input.MaxRetries
	if maxRetries == 0 {
		maxRetries = 5
	}
	retryBaseMs := input.RetryBaseMs
	if retryBaseMs == 0 {
		retryBaseMs = 1000
	}

	now := time.Now().UTC()
	tenant := &domain.Tenant{
		ID:                 "tenant-" + input.Name,
		Name:               input.Name,
		APIKey:             "whpk_live_test_" + input.Name,
		RateLimitPerMinute: rateLimit,
		MaxRetries:         maxRetries,
		RetryBaseMs:        retryBaseMs,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	m.tenants[tenant.ID] = tenant
	m.apiKeyMap[tenant.APIKey] = tenant.ID
	return tenant, tenant.APIKey, nil
}

func (m *mockTenantService) GetByID(_ context.Context, id string) (*domain.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (m *mockTenantService) GetByAPIKey(_ context.Context, apiKey string) (*domain.Tenant, error) {
	id, ok := m.apiKeyMap[apiKey]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return m.tenants[id], nil
}

func (m *mockTenantService) Update(_ context.Context, id string, input domain.UpdateTenantInput) (*domain.Tenant, error) {
	tenant, ok := m.tenants[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if input.Name != nil {
		tenant.Name = *input.Name
	}
	if input.RateLimitPerMinute != nil {
		tenant.RateLimitPerMinute = *input.RateLimitPerMinute
	}
	if input.MaxRetries != nil {
		tenant.MaxRetries = *input.MaxRetries
	}
	if input.RetryBaseMs != nil {
		tenant.RetryBaseMs = *input.RetryBaseMs
	}
	tenant.UpdatedAt = time.Now().UTC()
	m.tenants[id] = tenant
	return tenant, nil
}

func setupTenantRouter() (*mockTenantService, chi.Router) {
	svc := newMockTenantService()
	h := NewTenantHandler(svc)

	r := chi.NewRouter()
	r.Post("/api/v1/tenants", h.Create)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(svc.GetByAPIKey))
		r.Get("/api/v1/tenants/me", h.GetMe)
		r.Put("/api/v1/tenants/me", h.UpdateMe)
	})

	return svc, r
}

func TestTenantHandler_Create(t *testing.T) {
	_, router := setupTenantRouter()

	t.Run("creates tenant successfully", func(t *testing.T) {
		body := createTenantRequest{
			Name:               "Test Corp",
			RateLimitPerMinute: 500,
		}
		b, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp createTenantResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "Test Corp", resp.Name)
		assert.Contains(t, resp.APIKey, "whpk_live_")
		assert.Equal(t, 500, resp.RateLimitPerMinute)
	})

	t.Run("rejects invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		body := createTenantRequest{Name: ""}
		b, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTenantHandler_GetMe(t *testing.T) {
	svc, router := setupTenantRouter()

	svc.Create(context.Background(), domain.CreateTenantInput{Name: "AuthCorp"})

	t.Run("returns tenant with valid api key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/me", nil)
		req.Header.Set("Authorization", "Bearer whpk_live_test_AuthCorp")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp tenantResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "AuthCorp", resp.Name)
	})

	t.Run("rejects missing auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects invalid api key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/me", nil)
		req.Header.Set("Authorization", "Bearer invalid_key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestTenantHandler_UpdateMe(t *testing.T) {
	svc, router := setupTenantRouter()

	svc.Create(context.Background(), domain.CreateTenantInput{Name: "UpdateCorp"})

	t.Run("updates tenant name", func(t *testing.T) {
		newName := "Updated Corp"
		body := updateTenantRequest{Name: &newName}
		b, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/me", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer whpk_live_test_UpdateCorp")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp tenantResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "Updated Corp", resp.Name)
	})
}
