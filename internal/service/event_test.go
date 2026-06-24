package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/webhook-platform/internal/domain"
)

type mockEventRepo struct {
	events   map[string]*domain.Event
	byTenant map[string][]string
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{
		events:   make(map[string]*domain.Event),
		byTenant: make(map[string][]string),
	}
}

func (m *mockEventRepo) Create(_ context.Context, event *domain.Event) error {
	m.events[event.ID] = event
	m.byTenant[event.TenantID] = append(m.byTenant[event.TenantID], event.ID)
	return nil
}

func (m *mockEventRepo) GetByID(_ context.Context, id string) (*domain.Event, error) {
	e, ok := m.events[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return e, nil
}

func (m *mockEventRepo) ListByTenantID(_ context.Context, tenantID string, limit, offset int) ([]*domain.Event, error) {
	ids, ok := m.byTenant[tenantID]
	if !ok {
		return []*domain.Event{}, nil
	}
	start := offset
	if start > len(ids) {
		start = len(ids)
	}
	end := start + limit
	if end > len(ids) {
		end = len(ids)
	}
	result := make([]*domain.Event, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, m.events[ids[i]])
	}
	return result, nil
}

func (m *mockEventRepo) CountByTenantID(_ context.Context, tenantID string) (int, error) {
	return len(m.byTenant[tenantID]), nil
}

func (m *mockEventRepo) UpdateStatus(_ context.Context, id string, status string) error {
	e, ok := m.events[id]
	if !ok {
		return domain.ErrNotFound
	}
	e.Status = status
	return nil
}

type mockPublisher struct {
	published []string
	fail      bool
}

func (m *mockPublisher) PublishEvent(_ context.Context, eventID string, _ string, _ string, _ string, _ []byte, _ string) error {
	if m.fail {
		return fmt.Errorf("kafka unavailable")
	}
	m.published = append(m.published, eventID)
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

func TestEventService_Create(t *testing.T) {
	endpointRepo := newMockEndpointRepo()
	eventRepo := newMockEventRepo()
	publisher := &mockPublisher{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	endpointRepo.Create(context.Background(), &domain.Endpoint{
		ID:         "ep-1",
		TenantID:   "tenant-1",
		URL:        "https://api.example.com/webhooks",
		Secret:     "whsec_testsecret",
		EventTypes: []string{"payment.*", "customer.created"},
		Active:     true,
	})

	svc := NewEventService(eventRepo, endpointRepo, publisher, logger)

	t.Run("creates event with valid input", func(t *testing.T) {
		input := domain.CreateEventInput{
			EndpointID: "ep-1",
			EventType:  "payment.succeeded",
			Payload:    json.RawMessage(`{"amount":9999}`),
		}

		event, err := svc.Create(context.Background(), "tenant-1", input)

		require.NoError(t, err)
		assert.NotEmpty(t, event.ID)
		assert.Equal(t, "tenant-1", event.TenantID)
		assert.Equal(t, "ep-1", event.EndpointID)
		assert.Equal(t, "payment.succeeded", event.EventType)
		assert.Equal(t, domain.EventStatusQueued, event.Status)
		assert.Contains(t, event.Signature, "t=")
		assert.Contains(t, event.Signature, "v1=")
		assert.Len(t, publisher.published, 1)
	})

	t.Run("creates event matching exact event type", func(t *testing.T) {
		input := domain.CreateEventInput{
			EndpointID: "ep-1",
			EventType:  "customer.created",
			Payload:    json.RawMessage(`{"name":"John"}`),
		}

		event, err := svc.Create(context.Background(), "tenant-1", input)
		require.NoError(t, err)
		assert.Equal(t, "customer.created", event.EventType)
	})

	t.Run("creates event when endpoint has empty event types", func(t *testing.T) {
		endpointRepo.Create(context.Background(), &domain.Endpoint{
			ID:         "ep-2",
			TenantID:   "tenant-1",
			URL:        "https://api.example.com/all-events",
			Secret:     "whsec_testsecret2",
			EventTypes: []string{},
			Active:     true,
		})

		input := domain.CreateEventInput{
			EndpointID: "ep-2",
			EventType:  "anything.happened",
			Payload:    json.RawMessage(`{"test":true}`),
		}

		event, err := svc.Create(context.Background(), "tenant-1", input)
		require.NoError(t, err)
		assert.Equal(t, "anything.happened", event.EventType)
	})

	t.Run("rejects event type not matching subscriptions", func(t *testing.T) {
		input := domain.CreateEventInput{
			EndpointID: "ep-1",
			EventType:  "order.placed",
			Payload:    json.RawMessage(`{"order":"123"}`),
		}

		_, err := svc.Create(context.Background(), "tenant-1", input)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("rejects missing endpoint_id", func(t *testing.T) {
		input := domain.CreateEventInput{
			EventType: "payment.succeeded",
			Payload:   json.RawMessage(`{"amount":9999}`),
		}

		_, err := svc.Create(context.Background(), "tenant-1", input)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("rejects missing event_type", func(t *testing.T) {
		input := domain.CreateEventInput{
			EndpointID: "ep-1",
			Payload:    json.RawMessage(`{"amount":9999}`),
		}

		_, err := svc.Create(context.Background(), "tenant-1", input)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("rejects missing payload", func(t *testing.T) {
		input := domain.CreateEventInput{
			EndpointID: "ep-1",
			EventType:  "payment.succeeded",
		}

		_, err := svc.Create(context.Background(), "tenant-1", input)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("rejects endpoint from different tenant", func(t *testing.T) {
		input := domain.CreateEventInput{
			EndpointID: "ep-1",
			EventType:  "payment.succeeded",
			Payload:    json.RawMessage(`{"amount":9999}`),
		}

		_, err := svc.Create(context.Background(), "tenant-2", input)
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("rejects inactive endpoint", func(t *testing.T) {
		endpointRepo.Create(context.Background(), &domain.Endpoint{
			ID:         "ep-inactive",
			TenantID:   "tenant-1",
			URL:        "https://api.example.com/inactive",
			Secret:     "whsec_inactive",
			EventTypes: []string{},
			Active:     false,
		})

		input := domain.CreateEventInput{
			EndpointID: "ep-inactive",
			EventType:  "test.event",
			Payload:    json.RawMessage(`{"test":true}`),
		}

		_, err := svc.Create(context.Background(), "tenant-1", input)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("handles kafka publish failure gracefully", func(t *testing.T) {
		failPublisher := &mockPublisher{fail: true}
		svcWithFail := NewEventService(eventRepo, endpointRepo, failPublisher, logger)

		input := domain.CreateEventInput{
			EndpointID: "ep-1",
			EventType:  "payment.failed_kafka",
			Payload:    json.RawMessage(`{"test":"kafka_fail"}`),
		}

		event, err := svcWithFail.Create(context.Background(), "tenant-1", input)
		require.NoError(t, err)
		assert.Equal(t, domain.EventStatusPending, event.Status)
	})

	t.Run("rejects nonexistent endpoint", func(t *testing.T) {
		input := domain.CreateEventInput{
			EndpointID: "ep-nonexistent",
			EventType:  "payment.succeeded",
			Payload:    json.RawMessage(`{"amount":9999}`),
		}

		_, err := svc.Create(context.Background(), "tenant-1", input)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestEventService_GetByID(t *testing.T) {
	eventRepo := newMockEventRepo()
	publisher := &mockPublisher{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	svc := NewEventService(eventRepo, newMockEndpointRepo(), publisher, logger)

	event := &domain.Event{
		ID:         "evt-1",
		TenantID:   "tenant-1",
		EndpointID: "ep-1",
		EventType:  "test.event",
		Payload:    json.RawMessage(`{"test":true}`),
		Status:     domain.EventStatusQueued,
	}
	eventRepo.Create(context.Background(), event)

	t.Run("returns event by id", func(t *testing.T) {
		e, err := svc.GetByID(context.Background(), "evt-1")
		require.NoError(t, err)
		assert.Equal(t, "evt-1", e.ID)
	})

	t.Run("returns not found for missing event", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "nonexistent")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestEventService_ListByTenantID(t *testing.T) {
	eventRepo := newMockEventRepo()
	publisher := &mockPublisher{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	svc := NewEventService(eventRepo, newMockEndpointRepo(), publisher, logger)

	for i := 0; i < 5; i++ {
		eventRepo.Create(context.Background(), &domain.Event{
			ID:         fmt.Sprintf("evt-%d", i),
			TenantID:   "tenant-1",
			EndpointID: "ep-1",
			EventType:  "test.event",
			Payload:    json.RawMessage(`{"test":true}`),
			Status:     domain.EventStatusPending,
		})
	}

	t.Run("returns events for tenant", func(t *testing.T) {
		events, total, err := svc.ListByTenantID(context.Background(), "tenant-1", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.Len(t, events, 5)
	})

	t.Run("returns empty for unknown tenant", func(t *testing.T) {
		events, total, err := svc.ListByTenantID(context.Background(), "unknown", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Len(t, events, 0)
	})
}

func TestMatchesEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		patterns  []string
		eventType string
		expected  bool
	}{
		{"empty patterns accept all", []string{}, "anything", true},
		{"exact match", []string{"payment.succeeded"}, "payment.succeeded", true},
		{"exact no match", []string{"payment.succeeded"}, "payment.failed", false},
		{"wildcard prefix match", []string{"payment.*"}, "payment.succeeded", true},
		{"wildcard prefix no match", []string{"payment.*"}, "order.created", false},
		{"global wildcard", []string{"*"}, "anything.at.all", true},
		{"multiple patterns match first", []string{"payment.*", "order.*"}, "payment.refunded", true},
		{"multiple patterns match second", []string{"payment.*", "order.*"}, "order.created", true},
		{"multiple patterns no match", []string{"payment.*", "order.*"}, "customer.created", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesEventTypes(tt.patterns, tt.eventType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
