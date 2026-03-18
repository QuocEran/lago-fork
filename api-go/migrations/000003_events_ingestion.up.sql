-- Phase 4: Events ingestion persistence.
-- This creates the events table and core indexes needed by api-go ingestion.
-- Columns intentionally align with Rails schema to reduce parity drift.

CREATE TABLE IF NOT EXISTS events (
  id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id            UUID                      NOT NULL,
  customer_id                UUID,
  transaction_id             TEXT                      NOT NULL,
  code                       TEXT                      NOT NULL,
  properties                 JSONB                     NOT NULL DEFAULT '{}'::jsonb,
  "timestamp"                TIMESTAMP WITHOUT TIME ZONE,
  created_at                 TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now(),
  updated_at                 TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now(),
  metadata                   JSONB                     NOT NULL DEFAULT '{}'::jsonb,
  subscription_id            UUID,
  deleted_at                 TIMESTAMP WITHOUT TIME ZONE,
  external_customer_id       TEXT,
  external_subscription_id   TEXT,
  precise_total_amount_cents NUMERIC(40,15)
);

CREATE INDEX IF NOT EXISTS index_events_on_created_at
  ON events (created_at)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS index_events_on_organization_id
  ON events (organization_id);

CREATE INDEX IF NOT EXISTS index_events_on_organization_id_and_code
  ON events (organization_id, code);

CREATE INDEX IF NOT EXISTS idx_events_billing_lookup
  ON events (external_subscription_id, organization_id, code, "timestamp")
  INCLUDE (properties)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_events_for_distinct_codes
  ON events (external_subscription_id, organization_id, "timestamp")
  INCLUDE (code)
  WHERE deleted_at IS NULL;

-- Keep DB idempotency aligned with current api-go behavior.
CREATE UNIQUE INDEX IF NOT EXISTS idx_events_org_transaction_unique
  ON events (organization_id, transaction_id);
