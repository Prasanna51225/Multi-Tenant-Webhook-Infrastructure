package repository

import (
	"context"

	"github.com/webhook-platform/internal/domain"
)

type DeliveryAttemptRepository interface {
	Create(ctx context.Context, attempt *domain.DeliveryAttempt) error
	ListByEventID(ctx context.Context, eventID string) ([]*domain.DeliveryAttempt, error)
}
