CREATE TABLE dead_letter_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_id         UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    original_event_id   UUID NOT NULL,
    event_type          VARCHAR(255) NOT NULL,
    payload             JSONB NOT NULL,
    failure_reason      TEXT,
    last_status_code    INTEGER,
    total_attempts      INTEGER NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dle_tenant_id ON dead_letter_events(tenant_id);
CREATE INDEX idx_dle_endpoint_id ON dead_letter_events(endpoint_id);
CREATE INDEX idx_dle_created_at ON dead_letter_events(created_at DESC);