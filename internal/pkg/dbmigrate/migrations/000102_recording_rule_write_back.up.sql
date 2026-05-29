ALTER TABLE recording_rules ADD COLUMN write_back TINYINT NOT NULL DEFAULT 1 AFTER disabled;
