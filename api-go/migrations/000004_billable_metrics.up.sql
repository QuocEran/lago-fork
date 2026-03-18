-- Phase 5: Billable Metrics CRUD.
-- Creates billable_metrics and billable_metric_filters tables aligned with Rails schema.
-- aggregation_type stored as integer (Rails integer enum).
-- rounding_function and weighted_interval stored as TEXT to avoid PG enum conflicts.

CREATE TABLE IF NOT EXISTS billable_metrics (
  id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id    UUID        NOT NULL,
  name               TEXT        NOT NULL,
  code               TEXT        NOT NULL,
  description        TEXT,
  aggregation_type   INTEGER     NOT NULL DEFAULT 0,
  field_name         TEXT,
  recurring          BOOLEAN     NOT NULL DEFAULT FALSE,
  expression         TEXT,
  custom_aggregator  TEXT,
  weighted_interval  TEXT,
  rounding_function  TEXT,
  rounding_precision INTEGER,
  properties         JSONB,
  deleted_at         TIMESTAMP WITHOUT TIME ZONE,
  created_at         TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now(),
  updated_at         TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_billable_metrics_org_code_unique
  ON billable_metrics (organization_id, code)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_billable_metrics_organization_id
  ON billable_metrics (organization_id);

CREATE INDEX IF NOT EXISTS idx_billable_metrics_deleted_at
  ON billable_metrics (deleted_at);

CREATE TABLE IF NOT EXISTS billable_metric_filters (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  billable_metric_id  UUID        NOT NULL REFERENCES billable_metrics(id),
  organization_id     UUID        NOT NULL,
  key                 TEXT        NOT NULL,
  values              TEXT[]      NOT NULL DEFAULT '{}',
  deleted_at          TIMESTAMP WITHOUT TIME ZONE,
  created_at          TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now(),
  updated_at          TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_billable_metric_filters_billable_metric_id
  ON billable_metric_filters (billable_metric_id);

CREATE INDEX IF NOT EXISTS idx_billable_metric_filters_organization_id
  ON billable_metric_filters (organization_id);

CREATE INDEX IF NOT EXISTS idx_active_metric_filters
  ON billable_metric_filters (billable_metric_id)
  WHERE deleted_at IS NULL;
