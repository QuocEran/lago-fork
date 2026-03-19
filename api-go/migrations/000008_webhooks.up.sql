CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    webhook_url     VARCHAR NOT NULL,
    signature_algo  INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_org ON webhook_endpoints(organization_id);

CREATE TABLE IF NOT EXISTS webhooks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id     UUID NOT NULL,
    webhook_endpoint_id UUID,
    object_id           UUID,
    object_type         VARCHAR,
    webhook_type        VARCHAR,
    status              INTEGER NOT NULL DEFAULT 0,
    retries             INTEGER NOT NULL DEFAULT 0,
    http_status         INTEGER,
    endpoint            VARCHAR,
    payload             JSONB,
    response            JSONB,
    last_retried_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_org ON webhooks(organization_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_endpoint ON webhooks(webhook_endpoint_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_status ON webhooks(status);
