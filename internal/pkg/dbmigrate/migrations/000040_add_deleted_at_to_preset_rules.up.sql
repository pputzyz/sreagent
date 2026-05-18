ALTER TABLE preset_rules ADD COLUMN deleted_at DATETIME(3) NULL;
ALTER TABLE preset_rules ADD INDEX idx_preset_rules_deleted_at (deleted_at);
