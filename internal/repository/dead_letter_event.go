package repository

import (
	"context"

	"github.com/webhook-platform/internal/domain"
)

type DeadLetterEventRepository interface {
	Create(ctx context.Context, event *domain.DeadLetterEvent) error
	ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.DeadLetterEvent, error)
	CountByTenantID(ctx context.Context, tenantID string) (int, error)
	GetByID(ctx context.Context, id string) (*domain.DeadLetterEvent, error)
	Delete(ctx context.Context, id string) error
}
