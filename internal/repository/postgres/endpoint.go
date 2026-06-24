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

type EndpointRepo struct {
    pool *pgxpool.Pool
}

func NewEndpointRepo(pool *pgxpool.Pool) *EndpointRepo {
    return &EndpointRepo{pool: pool}
}

const endpointColumns = `id, tenant_id, url, description, event_types, secret, active, created_at, updated_at`

func (r *EndpointRepo) Create(ctx context.Context, endpoint *domain.Endpoint) error {
    query := `
        INSERT INTO endpoints (id, tenant_id, url, description, event_types, secret, active, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `

    _, err := r.pool.Exec(ctx, query,
        endpoint.ID,
        endpoint.TenantID,
        endpoint.URL,
        endpoint.Description,
        endpoint.EventTypes,
        endpoint.Secret,
        endpoint.Active,
        endpoint.CreatedAt,
        endpoint.UpdatedAt,
    )

    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return domain.ErrAlreadyExists
        }
        return fmt.Errorf("insert endpoint: %w", err)
    }

    return nil
}

func (r *EndpointRepo) GetByID(ctx context.Context, id string) (*domain.Endpoint, error) {
    query := fmt.Sprintf(`
        SELECT %s FROM endpoints WHERE id = $1
    `, endpointColumns)

    return r.scanEndpoint(ctx, r.pool.QueryRow(ctx, query, id))
}

func (r *EndpointRepo) ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Endpoint, error) {
    query := fmt.Sprintf(`
        SELECT %s FROM endpoints
        WHERE tenant_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `, endpointColumns)

    rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("query endpoints: %w", err)
    }
    defer rows.Close()

    var endpoints []*domain.Endpoint
    for rows.Next() {
        ep, err := r.scanEndpointRow(rows)
        if err != nil {
            return nil, err
        }
        endpoints = append(endpoints, ep)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate endpoints: %w", err)
    }

    return endpoints, nil
}

func (r *EndpointRepo) CountByTenantID(ctx context.Context, tenantID string) (int, error) {
    var count int
    err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM endpoints WHERE tenant_id = $1`, tenantID).Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("count endpoints: %w", err)
    }
    return count, nil
}

func (r *EndpointRepo) Update(ctx context.Context, endpoint *domain.Endpoint) error {
    query := `
        UPDATE endpoints
        SET url = $2, description = $3, event_types = $4, active = $5, updated_at = $6
        WHERE id = $1
    `

    tag, err := r.pool.Exec(ctx, query,
        endpoint.ID,
        endpoint.URL,
        endpoint.Description,
        endpoint.EventTypes,
        endpoint.Active,
        endpoint.UpdatedAt,
    )
    if err != nil {
        return fmt.Errorf("update endpoint: %w", err)
    }

    if tag.RowsAffected() == 0 {
        return domain.ErrNotFound
    }

    return nil
}

func (r *EndpointRepo) Delete(ctx context.Context, id string) error {
    tag, err := r.pool.Exec(ctx, `DELETE FROM endpoints WHERE id = $1`, id)
    if err != nil {
        return fmt.Errorf("delete endpoint: %w", err)
    }

    if tag.RowsAffected() == 0 {
        return domain.ErrNotFound
    }

    return nil
}

func (r *EndpointRepo) scanEndpoint(ctx context.Context, row pgx.Row) (*domain.Endpoint, error) {
    var ep domain.Endpoint
    err := row.Scan(
        &ep.ID,
        &ep.TenantID,
        &ep.URL,
        &ep.Description,
        &ep.EventTypes,
        &ep.Secret,
        &ep.Active,
        &ep.CreatedAt,
        &ep.UpdatedAt,
    )
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrNotFound
        }
        return nil, fmt.Errorf("scan endpoint: %w", err)
    }
    return &ep, nil
}

func (r *EndpointRepo) scanEndpointRow(rows pgx.Rows) (*domain.Endpoint, error) {
    var ep domain.Endpoint
    err := rows.Scan(
        &ep.ID,
        &ep.TenantID,
        &ep.URL,
        &ep.Description,
        &ep.EventTypes,
        &ep.Secret,
        &ep.Active,
        &ep.CreatedAt,
        &ep.UpdatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("scan endpoint row: %w", err)
    }
    return &ep, nil
}