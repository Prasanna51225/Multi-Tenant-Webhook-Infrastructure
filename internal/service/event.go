package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/webhook-platform/internal/delivery"
	"github.com/webhook-platform/internal/domain"
)

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	GetByID(ctx context.Context, id string) (*domain.Event, error)
	ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, error)
	CountByTenantID(ctx context.Context, tenantID string) (int, error)
	UpdateStatus(ctx context.Context, id string, status string) error
}

type EndpointLookup interface {
	GetByID(ctx context.Context, id string) (*domain.Endpoint, error)
}

type EventPublisher interface {
	PublishEvent(ctx context.Context, eventID string, tenantID string, endpointID string, eventType string, payload []byte, signature string) error
	Close() error
}

type EventService interface {
	Create(ctx context.Context, tenantID string, input domain.CreateEventInput) (*domain.Event, error)
	GetByID(ctx context.Context, id string) (*domain.Event, error)
	ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, int, error)
}

type eventService struct {
	eventRepo    EventRepository
	endpointRepo EndpointLookup
	publisher    EventPublisher
	logger       *slog.Logger
}

func NewEventService(eventRepo EventRepository, endpointRepo EndpointLookup, publisher EventPublisher, logger *slog.Logger) EventService {
	return &eventService{
		eventRepo:    eventRepo,
		endpointRepo: endpointRepo,
		publisher:    publisher,
		logger:       logger,
	}
}

func (s *eventService) Create(ctx context.Context, tenantID string, input domain.CreateEventInput) (*domain.Event, error) {
	if errs := input.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, errs)
	}

	endpoint, err := s.endpointRepo.GetByID(ctx, input.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("get endpoint: %w", err)
	}

	if endpoint.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}

	if !endpoint.Active {
		return nil, fmt.Errorf("%w: endpoint is inactive", domain.ErrValidation)
	}

	if !matchesEventTypes(endpoint.EventTypes, input.EventType) {
		return nil, fmt.Errorf("%w: event type %q does not match endpoint subscriptions %v", domain.ErrValidation, input.EventType, endpoint.EventTypes)
	}

	timestamp := time.Now().Unix()
	sig := delivery.Sign(input.Payload, endpoint.Secret, timestamp)

	now := time.Now().UTC()
	event := &domain.Event{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		EndpointID:   input.EndpointID,
		EventType:    input.EventType,
		Payload:      input.Payload,
		Signature:    sig.String(),
		Status:       domain.EventStatusPending,
		AttemptCount: 0,
		MaxAttempts:  5,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	if err := s.publisher.PublishEvent(
		ctx,
		event.ID,
		event.TenantID,
		event.EndpointID,
		event.EventType,
		event.Payload,
		event.Signature,
	); err != nil {
		s.logger.Error("failed to publish event to kafka",
			slog.String("error", err.Error()),
			slog.String("event_id", event.ID),
		)
	} else {
		if err := s.eventRepo.UpdateStatus(ctx, event.ID, domain.EventStatusQueued); err != nil {
			s.logger.Error("failed to update event status to queued",
				slog.String("error", err.Error()),
				slog.String("event_id", event.ID),
			)
		} else {
			event.Status = domain.EventStatusQueued
		}
	}

	return event, nil
}

func (s *eventService) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	return event, nil
}

func (s *eventService) ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	events, err := s.eventRepo.ListByTenantID(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}

	count, err := s.eventRepo.CountByTenantID(ctx, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	return events, count, nil
}

func matchesEventTypes(patterns []string, eventType string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		if matchEventType(pattern, eventType) {
			return true
		}
	}

	return false
}

func matchEventType(pattern, eventType string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(eventType, prefix+".")
	}
	return pattern == eventType
}

// Ensure json.RawMessage is used
var _ json.RawMessage
