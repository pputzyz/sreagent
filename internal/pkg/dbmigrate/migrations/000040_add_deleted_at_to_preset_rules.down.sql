ALTER TABLE preset_rules DROP INDEX idx_preset_rules_deleted_at;
ALTER TABLE preset_rules DROP COLUMN deleted_at;
