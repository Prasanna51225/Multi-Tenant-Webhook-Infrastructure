package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webhook-platform/internal/domain"
)

type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

const eventColumns = `id, tenant_id, endpoint_id, event_type, payload, signature, status, attempt_count, max_attempts, next_retry_at, created_at, updated_at`

func (r *EventRepo) Create(ctx context.Context, event *domain.Event) error {
	query := `
        INSERT INTO events (id, tenant_id, endpoint_id, event_type, payload, signature, status, attempt_count, max_attempts, next_retry_at, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `

	_, err := r.pool.Exec(ctx, query,
		event.ID,
		event.TenantID,
		event.EndpointID,
		event.EventType,
		event.Payload,
		event.Signature,
		event.Status,
		event.AttemptCount,
		event.MaxAttempts,
		event.NextRetryAt,
		event.CreatedAt,
		event.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func (r *EventRepo) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := fmt.Sprintf(`
        SELECT %s FROM events WHERE id = $1
    `, eventColumns)

	return r.scanEvent(ctx, r.pool.QueryRow(ctx, query, id))
}

func (r *EventRepo) ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, error) {
	query := fmt.Sprintf(`
        SELECT %s FROM events
        WHERE tenant_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `, eventColumns)

	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		e, err := r.scanEventRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}

func (r *EventRepo) CountByTenantID(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM events WHERE tenant_id = $1`, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return count, nil
}

func (r *EventRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
        UPDATE events SET status = $2, updated_at = NOW() WHERE id = $1
    `

	tag, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("update event status: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *EventRepo) UpdateForRetry(ctx context.Context, id string, status string, attemptCount int, nextRetryAt interface{}) error {
	query := `
        UPDATE events
        SET status = $2, attempt_count = $3, next_retry_at = $4, updated_at = NOW()
        WHERE id = $1
    `

	tag, err := r.pool.Exec(ctx, query, id, status, attemptCount, nextRetryAt)
	if err != nil {
		return fmt.Errorf("update event for retry: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *EventRepo) FindRetryable(ctx context.Context, limit int) ([]*domain.Event, error) {
	query := fmt.Sprintf(`
        SELECT %s FROM events
        WHERE status = 'retrying' AND next_retry_at <= NOW()
        ORDER BY next_retry_at ASC
        LIMIT $1
    `, eventColumns)

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("find retryable events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		e, err := r.scanEventRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate retryable events: %w", err)
	}

	return events, nil
}

func (r *EventRepo) scanEvent(ctx context.Context, row pgx.Row) (*domain.Event, error) {
	var e domain.Event
	err := row.Scan(
		&e.ID,
		&e.TenantID,
		&e.EndpointID,
		&e.EventType,
		&e.Payload,
		&e.Signature,
		&e.Status,
		&e.AttemptCount,
		&e.MaxAttempts,
		&e.NextRetryAt,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scan event: %w", err)
	}
	return &e, nil
}

func (r *EventRepo) scanEventRow(rows pgx.Rows) (*domain.Event, error) {
	var e domain.Event
	err := rows.Scan(
		&e.ID,
		&e.TenantID,
		&e.EndpointID,
		&e.EventType,
		&e.Payload,
		&e.Signature,
		&e.Status,
		&e.AttemptCount,
		&e.MaxAttempts,
		&e.NextRetryAt,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan event row: %w", err)
	}
	return &e, nil
}
