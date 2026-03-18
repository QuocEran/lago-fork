-- Phase 1: Identity domain tables shared with the Rails app.
-- These tables are Rails-owned (Rails manages writes); api-go only reads them.
-- We use CREATE TABLE IF NOT EXISTS so this is safe to run even after Rails migrations.

CREATE TABLE IF NOT EXISTS organizations (
  id                               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  name                             TEXT        NOT NULL,
  api_key                          TEXT,
  webhook_url                      TEXT,
  vat_rate                         FLOAT       NOT NULL DEFAULT 0,
  country                          TEXT,
  address_line1                    TEXT,
  address_line2                    TEXT,
  state                            TEXT,
  zipcode                          TEXT,
  email                            TEXT,
  city                             TEXT,
  logo                             TEXT,
  legal_name                       TEXT,
  legal_number                     TEXT,
  invoice_footer                   TEXT,
  invoice_grace_period             INTEGER     NOT NULL DEFAULT 0,
  timezone                         TEXT        NOT NULL DEFAULT 'UTC',
  document_locale                  TEXT        NOT NULL DEFAULT 'en',
  email_settings                   TEXT[]      NOT NULL DEFAULT '{}',
  tax_identification_number        TEXT,
  net_payment_term                 INTEGER     NOT NULL DEFAULT 0,
  default_currency                 TEXT        NOT NULL DEFAULT 'USD',
  document_numbering               INTEGER     NOT NULL DEFAULT 0,
  document_number_prefix           TEXT,
  eu_tax_management                BOOLEAN     NOT NULL DEFAULT FALSE,
  premium_integrations             TEXT[]      NOT NULL DEFAULT '{}',
  custom_aggregation               BOOLEAN     NOT NULL DEFAULT FALSE,
  finalize_zero_amount_invoice     BOOLEAN     NOT NULL DEFAULT TRUE,
  clickhouse_events_store          BOOLEAN     NOT NULL DEFAULT FALSE,
  clickhouse_deduplication_enabled BOOLEAN     NOT NULL DEFAULT FALSE,
  hmac_key                         TEXT        NOT NULL DEFAULT '',
  authentication_methods           TEXT[]      NOT NULL DEFAULT '{"email_password","google_oauth"}',
  audit_logs_period                INTEGER     NOT NULL DEFAULT 30,
  pre_filter_events                BOOLEAN     NOT NULL DEFAULT FALSE,
  feature_flags                    TEXT[]      NOT NULL DEFAULT '{}',
  max_wallets                      INTEGER,
  created_at                       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                       TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT organizations_api_key_unique UNIQUE (api_key),
  CONSTRAINT organizations_hmac_key_unique UNIQUE (hmac_key)
);

CREATE TABLE IF NOT EXISTS billing_entities (
  id                                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id                       UUID        NOT NULL REFERENCES organizations(id),
  code                                  TEXT,
  name                                  TEXT        NOT NULL,
  is_default                            BOOLEAN     NOT NULL DEFAULT FALSE,
  timezone                              TEXT        NOT NULL DEFAULT 'UTC',
  default_currency                      TEXT        NOT NULL DEFAULT 'USD',
  address_line1                         TEXT,
  address_line2                         TEXT,
  city                                  TEXT,
  zipcode                               TEXT,
  state                                 TEXT,
  country                               TEXT,
  email                                 TEXT,
  legal_name                            TEXT,
  legal_number                          TEXT,
  tax_identification_number             TEXT,
  logo                                  TEXT,
  document_locale                       TEXT        NOT NULL DEFAULT 'en',
  document_numbering                    INTEGER     NOT NULL DEFAULT 0,
  document_number_prefix                TEXT,
  net_payment_term                      INTEGER     NOT NULL DEFAULT 0,
  invoice_grace_period                  INTEGER     NOT NULL DEFAULT 0,
  invoice_footer                        TEXT,
  vat_rate                              FLOAT       NOT NULL DEFAULT 0,
  finalize_zero_amount_invoice          BOOLEAN     NOT NULL DEFAULT TRUE,
  email_settings                        TEXT[]      NOT NULL DEFAULT '{}',
  einvoicing_enabled                    BOOLEAN     NOT NULL DEFAULT FALSE,
  last_sequential_invoice_number        INTEGER     NOT NULL DEFAULT 0,
  organization_sequential_id            INTEGER     NOT NULL DEFAULT 0,
  deleted_at                            TIMESTAMPTZ,
  created_at                            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_billing_entities_organization_id ON billing_entities (organization_id);
CREATE INDEX IF NOT EXISTS idx_billing_entities_deleted_at      ON billing_entities (deleted_at);

CREATE TABLE IF NOT EXISTS users (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  email           TEXT        NOT NULL,
  password_digest TEXT        NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS memberships (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID        NOT NULL REFERENCES users(id),
  organization_id UUID        NOT NULL REFERENCES organizations(id),
  status          INTEGER     NOT NULL DEFAULT 0,
  revoked_at      TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_memberships_user_id         ON memberships (user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_organization_id ON memberships (organization_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_memberships_by_id_and_organization
  ON memberships (id, organization_id);

CREATE TABLE IF NOT EXISTS roles (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID        REFERENCES organizations(id),
  code            TEXT        NOT NULL,
  name            TEXT        NOT NULL,
  description     TEXT,
  admin           BOOLEAN     NOT NULL DEFAULT FALSE,
  permissions     TEXT[]      NOT NULL DEFAULT '{}',
  deleted_at      TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX  IF NOT EXISTS idx_roles_organization_id ON roles (organization_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_admin_unique
  ON roles (admin) WHERE (admin = TRUE AND deleted_at IS NULL);
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_code_per_organization
  ON roles (organization_id NULLS FIRST, code) WHERE (deleted_at IS NULL);

CREATE TABLE IF NOT EXISTS membership_roles (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  membership_id   UUID        NOT NULL,
  role_id         UUID        NOT NULL REFERENCES roles(id),
  organization_id UUID        NOT NULL REFERENCES organizations(id),
  deleted_at      TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT membership_role_membership_fk
    FOREIGN KEY (membership_id, organization_id)
    REFERENCES memberships (id, organization_id)
);

CREATE INDEX  IF NOT EXISTS idx_membership_roles_role_id ON membership_roles (role_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_membership_roles_uniqueness
  ON membership_roles (membership_id, role_id) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX IF NOT EXISTS idx_membership_roles_by_membership_and_organization
  ON membership_roles (membership_id, organization_id) WHERE (deleted_at IS NULL);

CREATE TABLE IF NOT EXISTS invites (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID        NOT NULL REFERENCES organizations(id),
  membership_id   UUID        REFERENCES memberships(id),
  email           TEXT        NOT NULL,
  token           TEXT        NOT NULL,
  status          INTEGER     NOT NULL DEFAULT 0,
  roles           TEXT[]      NOT NULL DEFAULT '{}',
  accepted_at     TIMESTAMPTZ,
  revoked_at      TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT invites_token_unique UNIQUE (token)
);

CREATE INDEX IF NOT EXISTS idx_invites_organization_id ON invites (organization_id);
CREATE INDEX IF NOT EXISTS idx_invites_membership_id   ON invites (membership_id);

CREATE TABLE IF NOT EXISTS password_resets (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID        NOT NULL REFERENCES users(id),
  token      TEXT        NOT NULL,
  expire_at  TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT password_resets_token_unique UNIQUE (token)
);

CREATE INDEX IF NOT EXISTS idx_password_resets_user_id ON password_resets (user_id);
