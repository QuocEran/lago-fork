-- Reverse of 000003_events_ingestion.up.sql.
DROP INDEX IF EXISTS idx_events_org_transaction_unique;

DROP INDEX IF EXISTS idx_events_for_distinct_codes;

DROP INDEX IF EXISTS idx_events_billing_lookup;

DROP INDEX IF EXISTS index_events_on_organization_id_and_code;

DROP INDEX IF EXISTS index_events_on_organization_id;

DROP INDEX IF EXISTS index_events_on_created_at;

DROP TABLE IF EXISTS events;
