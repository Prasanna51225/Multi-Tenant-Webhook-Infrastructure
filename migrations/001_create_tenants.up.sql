CREATE TABLE tenants (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    VARCHAR(255) NOT NULL,
    api_key                 VARCHAR(255) NOT NULL UNIQUE,
    rate_limit_per_minute   INTEGER NOT NULL DEFAULT 1000,
    max_retries             INTEGER NOT NULL DEFAULT 5,
    retry_base_ms           INTEGER NOT NULL DEFAULT 1000,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_api_key ON tenants(api_key);