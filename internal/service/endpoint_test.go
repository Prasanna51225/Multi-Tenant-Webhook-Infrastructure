package service

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/webhook-platform/internal/domain"
)

type mockEndpointRepo struct {
    endpoints map[string]*domain.Endpoint
    byTenant  map[string][]string
}

func newMockEndpointRepo() *mockEndpointRepo {
    return &mockEndpointRepo{
        endpoints: make(map[string]*domain.Endpoint),
        byTenant:  make(map[string][]string),
    }
}

func (m *mockEndpointRepo) Create(_ context.Context, ep *domain.Endpoint) error {
    if _, exists := m.endpoints[ep.ID]; exists {
        return domain.ErrAlreadyExists
    }
    m.endpoints[ep.ID] = ep
    m.byTenant[ep.TenantID] = append(m.byTenant[ep.TenantID], ep.ID)
    return nil
}

func (m *mockEndpointRepo) GetByID(_ context.Context, id string) (*domain.Endpoint, error) {
    ep, ok := m.endpoints[id]
    if !ok {
        return nil, domain.ErrNotFound
    }
    return ep, nil
}

func (m *mockEndpointRepo) ListByTenantID(_ context.Context, tenantID string, limit, offset int) ([]*domain.Endpoint, error) {
    ids, ok := m.byTenant[tenantID]
    if !ok {
        return []*domain.Endpoint{}, nil
    }

    start := offset
    if start > len(ids) {
        return []*domain.Endpoint{}, nil
    }

    end := start + limit
    if end > len(ids) {
        end = len(ids)
    }

    result := make([]*domain.Endpoint, 0, end-start)
    for i := start; i < end; i++ {
        result = append(result, m.endpoints[ids[i]])
    }
    return result, nil
}

func (m *mockEndpointRepo) CountByTenantID(_ context.Context, tenantID string) (int, error) {
    return len(m.byTenant[tenantID]), nil
}

func (m *mockEndpointRepo) Update(_ context.Context, ep *domain.Endpoint) error {
    if _, ok := m.endpoints[ep.ID]; !ok {
        return domain.ErrNotFound
    }
    m.endpoints[ep.ID] = ep
    return nil
}

func (m *mockEndpointRepo) Delete(_ context.Context, id string) error {
    ep, ok := m.endpoints[id]
    if !ok {
        return domain.ErrNotFound
    }
    delete(m.endpoints, id)

    tenantIDs := m.byTenant[ep.TenantID]
    for i, eid := range tenantIDs {
        if eid == id {
            m.byTenant[ep.TenantID] = append(tenantIDs[:i], tenantIDs[i+1:]...)
            break
        }
    }
    return nil
}

func TestEndpointService_Create(t *testing.T) {
    repo := newMockEndpointRepo()
    svc := NewEndpointService(repo)
    tenantID := "tenant-123"

    t.Run("creates endpoint with valid input", func(t *testing.T) {
        input := domain.CreateEndpointInput{
            URL:        "https://api.example.com/webhooks",
            EventTypes: []string{"payment.*"},
        }

        ep, secret, err := svc.Create(context.Background(), tenantID, input)

        require.NoError(t, err)
        assert.NotEmpty(t, ep.ID)
        assert.Equal(t, tenantID, ep.TenantID)
        assert.Equal(t, "https://api.example.com/webhooks", ep.URL)
        assert.Contains(t, secret, "whsec_")
        assert.True(t, ep.Active)
        assert.Equal(t, []string{"payment.*"}, ep.EventTypes)
    })

    t.Run("creates endpoint with active=false", func(t *testing.T) {
        inactive := false
        input := domain.CreateEndpointInput{
            URL:    "https://api.example.com/webhooks",
            Active: &inactive,
        }

        ep, _, err := svc.Create(context.Background(), tenantID, input)

        require.NoError(t, err)
        assert.False(t, ep.Active)
    })

    t.Run("rejects empty url", func(t *testing.T) {
        input := domain.CreateEndpointInput{
            URL: "",
        }

        _, _, err := svc.Create(context.Background(), tenantID, input)
        assert.ErrorIs(t, err, domain.ErrValidation)
    })

    t.Run("rejects non-http url", func(t *testing.T) {
        input := domain.CreateEndpointInput{
            URL: "ftp://example.com/webhooks",
        }

        _, _, err := svc.Create(context.Background(), tenantID, input)
        assert.ErrorIs(t, err, domain.ErrValidation)
    })

    t.Run("rejects url without host", func(t *testing.T) {
        input := domain.CreateEndpointInput{
            URL: "https://",
        }

        _, _, err := svc.Create(context.Background(), tenantID, input)
        assert.ErrorIs(t, err, domain.ErrValidation)
    })

    t.Run("sets empty event types when nil", func(t *testing.T) {
        input := domain.CreateEndpointInput{
            URL: "https://api.example.com/webhooks",
        }

        ep, _, err := svc.Create(context.Background(), tenantID, input)
        require.NoError(t, err)
        assert.Equal(t, []string{}, ep.EventTypes)
    })
}

func TestEndpointService_ListByTenantID(t *testing.T) {
    repo := newMockEndpointRepo()
    svc := NewEndpointService(repo)
    tenantID := "tenant-list"

    for i := 0; i < 5; i++ {
        input := domain.CreateEndpointInput{
            URL:        "https://api.example.com/webhooks",
            EventTypes: []string{"test.event"},
        }
        _, _, _ = svc.Create(context.Background(), tenantID, input)
    }

    t.Run("returns all endpoints for tenant", func(t *testing.T) {
        endpoints, total, err := svc.ListByTenantID(context.Background(), tenantID, 10, 0)

        require.NoError(t, err)
        assert.Equal(t, 5, total)
        assert.Len(t, endpoints, 5)
    })

    t.Run("respects limit and offset", func(t *testing.T) {
        endpoints, total, err := svc.ListByTenantID(context.Background(), tenantID, 2, 0)

        require.NoError(t, err)
        assert.Equal(t, 5, total)
        assert.Len(t, endpoints, 2)
    })

    t.Run("returns empty for unknown tenant", func(t *testing.T) {
        endpoints, total, err := svc.ListByTenantID(context.Background(), "unknown", 10, 0)

        require.NoError(t, err)
        assert.Equal(t, 0, total)
        assert.Len(t, endpoints, 0)
    })
}

func TestEndpointService_Update(t *testing.T) {
    repo := newMockEndpointRepo()
    svc := NewEndpointService(repo)
    tenantID := "tenant-update"

    input := domain.CreateEndpointInput{
        URL:        "https://api.example.com/webhooks",
        EventTypes: []string{"order.created"},
    }
    created, _, _ := svc.Create(context.Background(), tenantID, input)

    t.Run("updates endpoint url", func(t *testing.T) {
        newURL := "https://api2.example.com/webhooks"
        updated, err := svc.Update(context.Background(), created.ID, tenantID, domain.UpdateEndpointInput{
            URL: &newURL,
        })

        require.NoError(t, err)
        assert.Equal(t, newURL, updated.URL)
    })

    t.Run("rejects update from different tenant", func(t *testing.T) {
        newURL := "https://evil.example.com"
        _, err := svc.Update(context.Background(), created.ID, "different-tenant", domain.UpdateEndpointInput{
            URL: &newURL,
        })
        assert.ErrorIs(t, err, domain.ErrForbidden)
    })

    t.Run("rejects invalid url update", func(t *testing.T) {
        badURL := "not-a-url"
        _, err := svc.Update(context.Background(), created.ID, tenantID, domain.UpdateEndpointInput{
            URL: &badURL,
        })
        assert.ErrorIs(t, err, domain.ErrValidation)
    })
}

func TestEndpointService_Delete(t *testing.T) {
    repo := newMockEndpointRepo()
    svc := NewEndpointService(repo)
    tenantID := "tenant-delete"

    input := domain.CreateEndpointInput{
        URL: "https://api.example.com/webhooks",
    }
    created, _, _ := svc.Create(context.Background(), tenantID, input)

    t.Run("deletes endpoint", func(t *testing.T) {
        err := svc.Delete(context.Background(), created.ID, tenantID)
        require.NoError(t, err)

        _, err = svc.GetByID(context.Background(), created.ID)
        assert.ErrorIs(t, err, domain.ErrNotFound)
    })

    t.Run("rejects delete from different tenant", func(t *testing.T) {
        input2 := domain.CreateEndpointInput{
            URL: "https://api.example.com/other",
        }
        ep2, _, _ := svc.Create(context.Background(), tenantID, input2)

        err := svc.Delete(context.Background(), ep2.ID, "different-tenant")
        assert.ErrorIs(t, err, domain.ErrForbidden)
    })

    t.Run("returns not found for missing endpoint", func(t *testing.T) {
        err := svc.Delete(context.Background(), "nonexistent", tenantID)
        assert.ErrorIs(t, err, domain.ErrNotFound)
    })
}