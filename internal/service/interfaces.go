package service

import (
	"context"

	"github.com/segmentio/kafka-go"

	"github.com/webhook-platform/internal/domain"
)

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	GetByID(ctx context.Context, id string) (*domain.Event, error)
	ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, error)
	CountByTenantID(ctx context.Context, tenantID string) (int, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateForRetry(ctx context.Context, id string, status string, attemptCount int, nextRetryAt interface{}) error
	FindRetryable(ctx context.Context, limit int) ([]*domain.Event, error)
}

type EndpointLookup interface {
	GetByID(ctx context.Context, id string) (*domain.Endpoint, error)
}

type TenantLookup interface {
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
}

type DeliveryAttemptRepository interface {
	Create(ctx context.Context, attempt *domain.DeliveryAttempt) error
}

type DeadLetterEventRepository interface {
	Create(ctx context.Context, event *domain.DeadLetterEvent) error
}

type KafkaConsumer interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type KafkaPublisher interface {
	PublishEvent(ctx context.Context, eventID string, tenantID string, endpointID string, eventType string, payload []byte, signature string) error
}
