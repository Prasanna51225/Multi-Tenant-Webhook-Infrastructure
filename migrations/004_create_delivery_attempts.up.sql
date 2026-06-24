CREATE TABLE delivery_attempts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id          UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    endpoint_id       UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    attempt_number    INTEGER NOT NULL,
    status_code       INTEGER,
    response_body     TEXT,
    response_headers  JSONB,
    error_message     TEXT,
    duration_ms       INTEGER,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_attempts_event_id ON delivery_attempts(event_id);
CREATE INDEX idx_delivery_attempts_endpoint_id ON delivery_attempts(endpoint_id);
CREATE INDEX idx_delivery_attempts_created_at ON delivery_attempts(created_at DESC);