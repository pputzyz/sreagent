-- KNOWN: This migration shares version 000080 with 000080_step_exec_dedup_by_order.
-- Cannot be renumbered because it is already applied in production (schema_migrations state).
-- Safe: modifies notify_rules table only (no conflict with the other 000080 migration).

ALTER TABLE notify_rules ADD COLUMN max_notifications INT NOT NULL DEFAULT 0;
