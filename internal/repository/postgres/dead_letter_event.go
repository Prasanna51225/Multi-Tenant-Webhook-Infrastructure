package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webhook-platform/internal/domain"
)

type DeadLetterEventRepo struct {
	pool *pgxpool.Pool
}

func NewDeadLetterEventRepo(pool *pgxpool.Pool) *DeadLetterEventRepo {
	return &DeadLetterEventRepo{pool: pool}
}

func (r *DeadLetterEventRepo) Create(ctx context.Context, event *domain.DeadLetterEvent) error {
	query := `
        INSERT INTO dead_letter_events (id, tenant_id, endpoint_id, original_event_id, event_type, payload, failure_reason, last_status_code, total_attempts, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `

	_, err := r.pool.Exec(ctx, query,
		event.ID,
		event.TenantID,
		event.EndpointID,
		event.OriginalEventID,
		event.EventType,
		event.Payload,
		event.FailureReason,
		event.LastStatusCode,
		event.TotalAttempts,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert dead letter event: %w", err)
	}

	return nil
}

func (r *DeadLetterEventRepo) ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.DeadLetterEvent, error) {
	query := `
        SELECT id, tenant_id, endpoint_id, original_event_id, event_type, payload, failure_reason, last_status_code, total_attempts, created_at
        FROM dead_letter_events
        WHERE tenant_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query dead letter events: %w", err)
	}
	defer rows.Close()

	var events []*domain.DeadLetterEvent
	for rows.Next() {
		var e domain.DeadLetterEvent
		err := rows.Scan(
			&e.ID,
			&e.TenantID,
			&e.EndpointID,
			&e.OriginalEventID,
			&e.EventType,
			&e.Payload,
			&e.FailureReason,
			&e.LastStatusCode,
			&e.TotalAttempts,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan dead letter event: %w", err)
		}
		events = append(events, &e)
	}

	return events, nil
}

func (r *DeadLetterEventRepo) CountByTenantID(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM dead_letter_events WHERE tenant_id = $1`, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count dead letter events: %w", err)
	}
	return count, nil
}

func (r *DeadLetterEventRepo) GetByID(ctx context.Context, id string) (*domain.DeadLetterEvent, error) {
	query := `
        SELECT id, tenant_id, endpoint_id, original_event_id, event_type, payload, failure_reason, last_status_code, total_attempts, created_at
        FROM dead_letter_events
        WHERE id = $1
    `

	var e domain.DeadLetterEvent
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID,
		&e.TenantID,
		&e.EndpointID,
		&e.OriginalEventID,
		&e.EventType,
		&e.Payload,
		&e.FailureReason,
		&e.LastStatusCode,
		&e.TotalAttempts,
		&e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get dead letter event: %w", err)
	}
	return &e, nil
}

func (r *DeadLetterEventRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM dead_letter_events WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete dead letter event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
