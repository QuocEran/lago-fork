-- Phase 6: Plans and Charges tables.
-- Aligned with Rails schema. Charges embed model-specific config in JSONB properties.

CREATE TABLE IF NOT EXISTS plans (
  id                         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id            UUID        NOT NULL,
  parent_id                  UUID,
  name                       TEXT        NOT NULL,
  code                       TEXT        NOT NULL,
  description                TEXT,
  interval                   INTEGER     NOT NULL,
  amount_cents               BIGINT      NOT NULL,
  amount_currency            TEXT        NOT NULL,
  pay_in_advance             BOOLEAN     NOT NULL DEFAULT FALSE,
  bill_charges_monthly       BOOLEAN,
  bill_fixed_charges_monthly BOOLEAN     NOT NULL DEFAULT FALSE,
  trial_period               FLOAT,
  invoice_display_name       TEXT,
  pending_deletion           BOOLEAN     NOT NULL DEFAULT FALSE,
  deleted_at                 TIMESTAMP WITHOUT TIME ZONE,
  created_at                 TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now(),
  updated_at                 TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_plans_org_code_unique
  ON plans (organization_id, code)
  WHERE deleted_at IS NULL AND parent_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_plans_organization_id ON plans (organization_id);
CREATE INDEX IF NOT EXISTS idx_plans_parent_id        ON plans (parent_id);
CREATE INDEX IF NOT EXISTS idx_plans_deleted_at       ON plans (deleted_at);

CREATE TABLE IF NOT EXISTS charges (
  id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id      UUID        NOT NULL,
  plan_id              UUID        NOT NULL REFERENCES plans(id),
  billable_metric_id   UUID,
  parent_id            UUID,
  charge_model         INTEGER     NOT NULL DEFAULT 0,
  code                 TEXT        NOT NULL,
  properties           JSONB       NOT NULL DEFAULT '{}',
  pay_in_advance       BOOLEAN     NOT NULL DEFAULT FALSE,
  invoiceable          BOOLEAN     NOT NULL DEFAULT TRUE,
  prorated             BOOLEAN     NOT NULL DEFAULT FALSE,
  min_amount_cents     BIGINT      NOT NULL DEFAULT 0,
  invoice_display_name TEXT,
  regroup_paid_fees    INTEGER,
  deleted_at           TIMESTAMP WITHOUT TIME ZONE,
  created_at           TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now(),
  updated_at           TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_charges_plan_code_unique
  ON charges (plan_id, code)
  WHERE deleted_at IS NULL AND parent_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_charges_plan_id             ON charges (plan_id);
CREATE INDEX IF NOT EXISTS idx_charges_organization_id     ON charges (organization_id);
CREATE INDEX IF NOT EXISTS idx_charges_billable_metric_id  ON charges (billable_metric_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_charges_deleted_at          ON charges (deleted_at);
CREATE INDEX IF NOT EXISTS idx_charges_parent_id           ON charges (parent_id);

CREATE TABLE IF NOT EXISTS charge_filters (
  id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  charge_id            UUID        NOT NULL REFERENCES charges(id),
  organization_id      UUID        NOT NULL,
  invoice_display_name TEXT,
  properties           JSONB       NOT NULL DEFAULT '{}',
  deleted_at           TIMESTAMP WITHOUT TIME ZONE,
  created_at           TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now(),
  updated_at           TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_charge_filters_charge_id        ON charge_filters (charge_id);
CREATE INDEX IF NOT EXISTS idx_charge_filters_organization_id  ON charge_filters (organization_id);
CREATE INDEX IF NOT EXISTS idx_active_charge_filters
  ON charge_filters (charge_id) WHERE deleted_at IS NULL;
