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

type mockEndpointService struct {
	endpoints map[string]*domain.Endpoint
	byTenant  map[string][]string
	counter   int
}

func newMockEndpointService() *mockEndpointService {
	return &mockEndpointService{
		endpoints: make(map[string]*domain.Endpoint),
		byTenant:  make(map[string][]string),
	}
}

func (m *mockEndpointService) Create(_ context.Context, tenantID string, input domain.CreateEndpointInput) (*domain.Endpoint, string, error) {
	if errs := input.Validate(); len(errs) > 0 {
		return nil, "", fmt.Errorf("%w: %v", domain.ErrValidation, errs)
	}

	m.counter++
	active := true
	if input.Active != nil {
		active = *input.Active
	}

	eventTypes := input.EventTypes
	if eventTypes == nil {
		eventTypes = []string{}
	}

	now := time.Now().UTC()
	ep := &domain.Endpoint{
		ID:          fmt.Sprintf("ep-%d", m.counter),
		TenantID:    tenantID,
		URL:         input.URL,
		Description: input.Description,
		EventTypes:  eventTypes,
		Secret:      "whsec_test_secret",
		Active:      active,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m.endpoints[ep.ID] = ep
	m.byTenant[tenantID] = append(m.byTenant[tenantID], ep.ID)
	return ep, ep.Secret, nil
}

func (m *mockEndpointService) GetByID(_ context.Context, id string) (*domain.Endpoint, error) {
	ep, ok := m.endpoints[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return ep, nil
}

func (m *mockEndpointService) ListByTenantID(_ context.Context, tenantID string, limit, offset int) ([]*domain.Endpoint, int, error) {
	ids, ok := m.byTenant[tenantID]
	if !ok {
		return []*domain.Endpoint{}, 0, nil
	}

	start := offset
	if start > len(ids) {
		start = len(ids)
	}
	end := start + limit
	if end > len(ids) {
		end = len(ids)
	}

	result := make([]*domain.Endpoint, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, m.endpoints[ids[i]])
	}
	return result, len(ids), nil
}

func (m *mockEndpointService) Update(_ context.Context, id string, tenantID string, input domain.UpdateEndpointInput) (*domain.Endpoint, error) {
	ep, ok := m.endpoints[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if ep.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}
	if input.URL != nil {
		ep.URL = *input.URL
	}
	if input.Description != nil {
		ep.Description = *input.Description
	}
	if input.EventTypes != nil {
		ep.EventTypes = input.EventTypes
	}
	if input.Active != nil {
		ep.Active = *input.Active
	}
	ep.UpdatedAt = time.Now().UTC()
	m.endpoints[id] = ep
	return ep, nil
}

func (m *mockEndpointService) Delete(_ context.Context, id string, tenantID string) error {
	ep, ok := m.endpoints[id]
	if !ok {
		return domain.ErrNotFound
	}
	if ep.TenantID != tenantID {
		return domain.ErrForbidden
	}
	delete(m.endpoints, id)
	return nil
}

type combinedMockService struct {
	*mockTenantService
	*mockEndpointService
}

func setupEndpointRouter() (*combinedMockService, chi.Router) {
	tenantSvc := newMockTenantService()
	endpointSvc := newMockEndpointService()

	th := NewTenantHandler(tenantSvc)
	eh := NewEndpointHandler(endpointSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/tenants", th.Create)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(tenantSvc.GetByAPIKey))
		r.Get("/api/v1/tenants/me", th.GetMe)
		r.Put("/api/v1/tenants/me", th.UpdateMe)
		r.Post("/api/v1/endpoints", eh.Create)
		r.Get("/api/v1/endpoints", eh.List)
		r.Get("/api/v1/endpoints/{id}", eh.Get)
		r.Put("/api/v1/endpoints/{id}", eh.Update)
		r.Delete("/api/v1/endpoints/{id}", eh.Delete)
	})

	tenantSvc.Create(context.Background(), domain.CreateTenantInput{Name: "test_tenant"})

	return &combinedMockService{tenantSvc, endpointSvc}, r
}

func TestEndpointHandler_Create(t *testing.T) {
	_, router := setupEndpointRouter()

	t.Run("creates endpoint successfully", func(t *testing.T) {
		body := createEndpointRequest{
			URL:        "https://api.example.com/webhooks",
			EventTypes: []string{"payment.*"},
		}
		b, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer whpk_live_test_test_tenant")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp createEndpointResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com/webhooks", resp.URL)
		assert.Contains(t, resp.Secret, "whsec_")
	})

	t.Run("rejects without auth", func(t *testing.T) {
		body := createEndpointRequest{
			URL: "https://api.example.com/webhooks",
		}
		b, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestEndpointHandler_List(t *testing.T) {
	_, router := setupEndpointRouter()

	authHeader := "Bearer whpk_live_test_test_tenant"

	for i := 0; i < 3; i++ {
		body := createEndpointRequest{
			URL:        fmt.Sprintf("https://api%d.example.com/webhooks", i),
			EventTypes: []string{"test.event"},
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	t.Run("lists endpoints for tenant", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp endpointListResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
	})
}

func TestEndpointHandler_Delete(t *testing.T) {
	_, router := setupEndpointRouter()

	authHeader := "Bearer whpk_live_test_test_tenant"

	body := createEndpointRequest{
		URL: "https://api.example.com/to-delete",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp createEndpointResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	t.Run("deletes endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/endpoints/"+createResp.ID, nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
