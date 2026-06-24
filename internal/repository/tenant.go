package repository

import (
    "context"

    "github.com/webhook-platform/internal/domain"
)

type TenantRepository interface {
    Create(ctx context.Context, tenant *domain.Tenant) error
    GetByID(ctx context.Context, id string) (*domain.Tenant, error)
    GetByAPIKey(ctx context.Context, apiKey string) (*domain.Tenant, error)
    Update(ctx context.Context, tenant *domain.Tenant) error
}