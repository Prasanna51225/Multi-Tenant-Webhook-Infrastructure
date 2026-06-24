package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webhook-platform/internal/domain"
)

type TenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepo(pool *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{pool: pool}
}

const tenantColumns = `id, name, api_key, rate_limit_per_minute, max_retries, retry_base_ms, created_at, updated_at`

func (r *TenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
        INSERT INTO tenants (id, name, api_key, rate_limit_per_minute, max_retries, retry_base_ms, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	_, err := r.pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.APIKey,
		tenant.RateLimitPerMinute,
		tenant.MaxRetries,
		tenant.RetryBaseMs,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert tenant: %w", err)
	}

	return nil
}

func (r *TenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	query := fmt.Sprintf(`
        SELECT %s FROM tenants WHERE id = $1
    `, tenantColumns)

	return r.scanTenant(ctx, r.pool.QueryRow(ctx, query, id))
}

func (r *TenantRepo) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Tenant, error) {
	query := fmt.Sprintf(`
        SELECT %s FROM tenants WHERE api_key = $1
    `, tenantColumns)

	return r.scanTenant(ctx, r.pool.QueryRow(ctx, query, apiKey))
}

func (r *TenantRepo) Update(ctx context.Context, tenant *domain.Tenant) error {
	query := `
        UPDATE tenants
        SET name = $2, rate_limit_per_minute = $3, max_retries = $4, retry_base_ms = $5, updated_at = $6
        WHERE id = $1
    `

	tag, err := r.pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.RateLimitPerMinute,
		tenant.MaxRetries,
		tenant.RetryBaseMs,
		tenant.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *TenantRepo) scanTenant(ctx context.Context, row pgx.Row) (*domain.Tenant, error) {
	var t domain.Tenant
	err := row.Scan(
		&t.ID,
		&t.Name,
		&t.APIKey,
		&t.RateLimitPerMinute,
		&t.MaxRetries,
		&t.RetryBaseMs,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scan tenant: %w", err)
	}
	return &t, nil
}
