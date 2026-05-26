-- Remove multi-query support columns from alert_rules table

ALTER TABLE alert_rules
DROP COLUMN queries,
DROP COLUMN trigger_exp,
DROP COLUMN join_type,
DROP COLUMN join_keys;
