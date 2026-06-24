package domain

import (
	"encoding/json"
	"time"
)

type DeadLetterEvent struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	EndpointID      string          `json:"endpoint_id"`
	OriginalEventID string          `json:"original_event_id"`
	EventType       string          `json:"event_type"`
	Payload         json.RawMessage `json:"payload"`
	FailureReason   string          `json:"failure_reason"`
	LastStatusCode  int             `json:"last_status_code"`
	TotalAttempts   int             `json:"total_attempts"`
	CreatedAt       time.Time       `json:"created_at"`
}
