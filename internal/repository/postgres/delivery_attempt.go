package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webhook-platform/internal/domain"
)

type DeliveryAttemptRepo struct {
	pool *pgxpool.Pool
}

func NewDeliveryAttemptRepo(pool *pgxpool.Pool) *DeliveryAttemptRepo {
	return &DeliveryAttemptRepo{pool: pool}
}

func (r *DeliveryAttemptRepo) Create(ctx context.Context, attempt *domain.DeliveryAttempt) error {
	query := `
        INSERT INTO delivery_attempts (id, event_id, endpoint_id, attempt_number, status_code, response_body, response_headers, error_message, duration_ms, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `

	_, err := r.pool.Exec(ctx, query,
		attempt.ID,
		attempt.EventID,
		attempt.EndpointID,
		attempt.AttemptNumber,
		attempt.StatusCode,
		attempt.ResponseBody,
		attempt.ResponseHeaders,
		attempt.ErrorMessage,
		attempt.DurationMs,
		attempt.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert delivery attempt: %w", err)
	}

	return nil
}

func (r *DeliveryAttemptRepo) ListByEventID(ctx context.Context, eventID string) ([]*domain.DeliveryAttempt, error) {
	query := `
        SELECT id, event_id, endpoint_id, attempt_number, status_code, response_body, response_headers, error_message, duration_ms, created_at
        FROM delivery_attempts
        WHERE event_id = $1
        ORDER BY attempt_number ASC
    `

	rows, err := r.pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("query delivery attempts: %w", err)
	}
	defer rows.Close()

	var attempts []*domain.DeliveryAttempt
	for rows.Next() {
		var a domain.DeliveryAttempt
		err := rows.Scan(
			&a.ID,
			&a.EventID,
			&a.EndpointID,
			&a.AttemptNumber,
			&a.StatusCode,
			&a.ResponseBody,
			&a.ResponseHeaders,
			&a.ErrorMessage,
			&a.DurationMs,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan delivery attempt: %w", err)
		}
		attempts = append(attempts, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate delivery attempts: %w", err)
	}

	return attempts, nil
}
