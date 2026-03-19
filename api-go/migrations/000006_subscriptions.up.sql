-- Phase 7: Subscriptions table.
-- Aligned with Rails schema. status and billing_time stored as integers.

CREATE TABLE IF NOT EXISTS subscriptions (
  id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id          UUID        NOT NULL,
  customer_id              UUID        NOT NULL,
  plan_id                  UUID        NOT NULL,
  previous_subscription_id UUID,
  external_id              TEXT        NOT NULL,
  name                     TEXT,
  status                   INTEGER     NOT NULL DEFAULT 0,
  billing_time             INTEGER     NOT NULL DEFAULT 0,
  subscription_at          TIMESTAMP WITHOUT TIME ZONE,
  started_at               TIMESTAMP WITHOUT TIME ZONE,
  ending_at                TIMESTAMP WITHOUT TIME ZONE,
  canceled_at              TIMESTAMP WITHOUT TIME ZONE,
  terminated_at            TIMESTAMP WITHOUT TIME ZONE,
  trial_ended_at           TIMESTAMP WITHOUT TIME ZONE,
  created_at               TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now(),
  updated_at               TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_organization_id          ON subscriptions (organization_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id              ON subscriptions (customer_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_plan_id                  ON subscriptions (plan_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_external_id              ON subscriptions (external_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status                   ON subscriptions (status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_started_at               ON subscriptions (started_at);
CREATE INDEX IF NOT EXISTS idx_subscriptions_previous_subscription_id ON subscriptions (previous_subscription_id, status);
