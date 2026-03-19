CREATE TABLE IF NOT EXISTS inbound_webhooks (
    id              UUID         NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    organization_id UUID         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    source          VARCHAR      NOT NULL,
    event_type      VARCHAR      NOT NULL DEFAULT '',
    payload         JSONB        NOT NULL DEFAULT '{}',
    status          VARCHAR      NOT NULL DEFAULT 'pending',
    code            VARCHAR,
    signature       VARCHAR,
    processing_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inbound_webhooks_org_id     ON inbound_webhooks(organization_id);
CREATE INDEX IF NOT EXISTS idx_inbound_webhooks_status     ON inbound_webhooks(status);
CREATE INDEX IF NOT EXISTS idx_inbound_webhooks_source     ON inbound_webhooks(source);
