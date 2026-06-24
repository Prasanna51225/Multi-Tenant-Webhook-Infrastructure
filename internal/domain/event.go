package domain

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	EndpointID   string          `json:"endpoint_id"`
	EventType    string          `json:"event_type"`
	Payload      json.RawMessage `json:"payload"`
	Signature    string          `json:"-"`
	Status       string          `json:"status"`
	AttemptCount int             `json:"attempt_count"`
	MaxAttempts  int             `json:"max_attempts"`
	NextRetryAt  *time.Time      `json:"next_retry_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CreateEventInput struct {
	EndpointID string          `json:"endpoint_id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
}

func (i CreateEventInput) Validate() []ValidationError {
	var errs []ValidationError
	if i.EndpointID == "" {
		errs = append(errs, ValidationError{Field: "endpoint_id", Message: "is required"})
	}
	if i.EventType == "" {
		errs = append(errs, ValidationError{Field: "event_type", Message: "is required"})
	}
	if len(i.Payload) == 0 {
		errs = append(errs, ValidationError{Field: "payload", Message: "is required"})
	}
	if len(i.EventType) > 255 {
		errs = append(errs, ValidationError{Field: "event_type", Message: "must be 255 characters or less"})
	}
	return errs
}

const (
	EventStatusPending    = "pending"
	EventStatusQueued     = "queued"
	EventStatusDelivering = "delivering"
	EventStatusDelivered  = "delivered"
	EventStatusFailed     = "failed"
	EventStatusRetrying   = "retrying"
)
