package domain

import (
	"encoding/json"
	"time"
)

type DeliveryAttempt struct {
	ID              string          `json:"id"`
	EventID         string          `json:"event_id"`
	EndpointID      string          `json:"endpoint_id"`
	AttemptNumber   int             `json:"attempt_number"`
	StatusCode      int             `json:"status_code"`
	ResponseBody    string          `json:"response_body"`
	ResponseHeaders json.RawMessage `json:"response_headers"`
	ErrorMessage    string          `json:"error_message"`
	DurationMs      int             `json:"duration_ms"`
	CreatedAt       time.Time       `json:"created_at"`
}
