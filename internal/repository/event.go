package repository

import (
	"context"

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
