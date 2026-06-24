CREATE TABLE endpoints (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url         VARCHAR(2048) NOT NULL,
    description TEXT,
    event_types TEXT[] NOT NULL DEFAULT '{}',
    secret      VARCHAR(255) NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_endpoints_tenant_id ON endpoints(tenant_id);
CREATE INDEX idx_endpoints_active ON endpoints(active) WHERE active = TRUE;