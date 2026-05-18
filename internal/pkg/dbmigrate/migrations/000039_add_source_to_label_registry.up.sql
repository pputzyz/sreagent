ALTER TABLE label_registry ADD COLUMN source VARCHAR(100) DEFAULT '' AFTER label_value;
ALTER TABLE label_registry ADD INDEX idx_source (source);
