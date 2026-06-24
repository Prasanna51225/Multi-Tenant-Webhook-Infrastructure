package service

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/webhook-platform/internal/domain"
)

type mockTenantRepo struct {
    tenants   map[string]*domain.Tenant
    apiKeyMap map[string]string
}

func newMockTenantRepo() *mockTenantRepo {
    return &mockTenantRepo{
        tenants:   make(map[string]*domain.Tenant),
        apiKeyMap: make(map[string]string),
    }
}

func (m *mockTenantRepo) Create(_ context.Context, tenant *domain.Tenant) error {
    if _, exists := m.tenants[tenant.ID]; exists {
        return domain.ErrAlreadyExists
    }
    m.tenants[tenant.ID] = tenant
    m.apiKeyMap[tenant.APIKey] = tenant.ID
    return nil
}

func (m *mockTenantRepo) GetByID(_ context.Context, id string) (*domain.Tenant, error) {
    t, ok := m.tenants[id]
    if !ok {
        return nil, domain.ErrNotFound
    }
    return t, nil
}

func (m *mockTenantRepo) GetByAPIKey(_ context.Context, apiKey string) (*domain.Tenant, error) {
    id, ok := m.apiKeyMap[apiKey]
    if !ok {
        return nil, domain.ErrNotFound
    }
    return m.tenants[id], nil
}

func (m *mockTenantRepo) Update(_ context.Context, tenant *domain.Tenant) error {
    if _, ok := m.tenants[tenant.ID]; !ok {
        return domain.ErrNotFound
    }
    m.tenants[tenant.ID] = tenant
    return nil
}

func TestTenantService_Create(t *testing.T) {
    repo := newMockTenantRepo()
    svc := NewTenantService(repo)

    t.Run("creates tenant with valid input", func(t *testing.T) {
        input := domain.CreateTenantInput{
            Name:               "Acme Corp",
            RateLimitPerMinute: 500,
            MaxRetries:         3,
            RetryBaseMs:        2000,
        }

        tenant, apiKey, err := svc.Create(context.Background(), input)

        require.NoError(t, err)
        assert.NotEmpty(t, tenant.ID)
        assert.Equal(t, "Acme Corp", tenant.Name)
        assert.NotEmpty(t, apiKey)
        assert.Contains(t, apiKey, "whpk_live_")
        assert.Equal(t, 500, tenant.RateLimitPerMinute)
        assert.Equal(t, 3, tenant.MaxRetries)
        assert.Equal(t, 2000, tenant.RetryBaseMs)
    })

    t.Run("creates tenant with defaults when optional fields are zero", func(t *testing.T) {
        input := domain.CreateTenantInput{
            Name: "Minimal Corp",
        }

        tenant, apiKey, err := svc.Create(context.Background(), input)

        require.NoError(t, err)
        assert.Equal(t, 1000, tenant.RateLimitPerMinute)
        assert.Equal(t, 5, tenant.MaxRetries)
        assert.Equal(t, 1000, tenant.RetryBaseMs)
        assert.NotEmpty(t, apiKey)
    })

    t.Run("rejects empty name", func(t *testing.T) {
        input := domain.CreateTenantInput{
            Name: "",
        }

        _, _, err := svc.Create(context.Background(), input)
        assert.ErrorIs(t, err, domain.ErrValidation)
    })

    t.Run("rejects name too long", func(t *testing.T) {
        input := domain.CreateTenantInput{
            Name: string(make([]byte, 256)),
        }

        _, _, err := svc.Create(context.Background(), input)
        assert.ErrorIs(t, err, domain.ErrValidation)
    })

    t.Run("rejects negative rate limit", func(t *testing.T) {
        input := domain.CreateTenantInput{
            Name:               "Test",
            RateLimitPerMinute: -1,
        }

        _, _, err := svc.Create(context.Background(), input)
        assert.ErrorIs(t, err, domain.ErrValidation)
    })

    t.Run("rejects invalid max retries", func(t *testing.T) {
        input := domain.CreateTenantInput{
            Name:       "Test",
            MaxRetries: 15,
        }

        _, _, err := svc.Create(context.Background(), input)
        assert.ErrorIs(t, err, domain.ErrValidation)
    })

    t.Run("rejects retry base ms too low", func(t *testing.T) {
        input := domain.CreateTenantInput{
            Name:        "Test",
            RetryBaseMs: 50,
        }

        _, _, err := svc.Create(context.Background(), input)
        assert.ErrorIs(t, err, domain.ErrValidation)
    })
}

func TestTenantService_GetByID(t *testing.T) {
    repo := newMockTenantRepo()
    svc := NewTenantService(repo)

    input := domain.CreateTenantInput{Name: "Test Corp"}
    created, _, _ := svc.Create(context.Background(), input)

    t.Run("returns tenant by id", func(t *testing.T) {
        tenant, err := svc.GetByID(context.Background(), created.ID)
        require.NoError(t, err)
        assert.Equal(t, created.ID, tenant.ID)
    })

    t.Run("returns not found for missing id", func(t *testing.T) {
        _, err := svc.GetByID(context.Background(), "nonexistent")
        assert.ErrorIs(t, err, domain.ErrNotFound)
    })
}

func TestTenantService_GetByAPIKey(t *testing.T) {
    repo := newMockTenantRepo()
    svc := NewTenantService(repo)

    input := domain.CreateTenantInput{Name: "Test Corp"}
    _, apiKey, _ := svc.Create(context.Background(), input)

    t.Run("returns tenant by api key", func(t *testing.T) {
        tenant, err := svc.GetByAPIKey(context.Background(), apiKey)
        require.NoError(t, err)
        assert.Equal(t, "Test Corp", tenant.Name)
    })

    t.Run("returns not found for invalid key", func(t *testing.T) {
        _, err := svc.GetByAPIKey(context.Background(), "whpk_live_invalid")
        assert.ErrorIs(t, err, domain.ErrNotFound)
    })
}

func TestTenantService_Update(t *testing.T) {
    repo := newMockTenantRepo()
    svc := NewTenantService(repo)

    input := domain.CreateTenantInput{Name: "Original Corp"}
    created, _, _ := svc.Create(context.Background(), input)

    t.Run("updates tenant name", func(t *testing.T) {
        newName := "Updated Corp"
        updated, err := svc.Update(context.Background(), created.ID, domain.UpdateTenantInput{
            Name: &newName,
        })
        require.NoError(t, err)
        assert.Equal(t, "Updated Corp", updated.Name)
    })

    t.Run("updates multiple fields", func(t *testing.T) {
        newName := "Mega Corp"
        newRateLimit := 2000
        newRetries := 8
        updated, err := svc.Update(context.Background(), created.ID, domain.UpdateTenantInput{
            Name:               &newName,
            RateLimitPerMinute: &newRateLimit,
            MaxRetries:         &newRetries,
        })
        require.NoError(t, err)
        assert.Equal(t, "Mega Corp", updated.Name)
        assert.Equal(t, 2000, updated.RateLimitPerMinute)
        assert.Equal(t, 8, updated.MaxRetries)
    })

    t.Run("rejects empty name update", func(t *testing.T) {
        emptyName := ""
        _, err := svc.Update(context.Background(), created.ID, domain.UpdateTenantInput{
            Name: &emptyName,
        })
        assert.ErrorIs(t, err, domain.ErrValidation)
    })

    t.Run("returns not found for missing tenant", func(t *testing.T) {
        name := "Test"
        _, err := svc.Update(context.Background(), "nonexistent", domain.UpdateTenantInput{
            Name: &name,
        })
        assert.ErrorIs(t, err, domain.ErrNotFound)
    })
}
