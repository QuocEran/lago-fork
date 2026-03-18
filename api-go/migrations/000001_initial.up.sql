-- Phase 0: api_keys table required by the API-key auth middleware.
-- The remaining schema is managed by the existing Rails app (db/structure.sql).
-- This migration only creates objects that api-go owns exclusively.
CREATE TABLE
  IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    organization_id UUID NOT NULL,
    value TEXT NOT NULL,
    name TEXT,
    permissions JSONB,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now (),
    CONSTRAINT api_keys_value_unique UNIQUE (value)
  );

CREATE INDEX IF NOT EXISTS idx_api_keys_organization_id ON api_keys (organization_id);
