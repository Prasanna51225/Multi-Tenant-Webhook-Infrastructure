package repository

import (
    "context"

    "github.com/webhook-platform/internal/domain"
)

type EndpointRepository interface {
    Create(ctx context.Context, endpoint *domain.Endpoint) error
    GetByID(ctx context.Context, id string) (*domain.Endpoint, error)
    ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Endpoint, error)
    Update(ctx context.Context, endpoint *domain.Endpoint) error
    Delete(ctx context.Context, id string) error
    CountByTenantID(ctx context.Context, tenantID string) (int, error)
}