-- If any rules have NULL data_source_id, set them to 0 before making column NOT NULL
UPDATE alert_rules SET data_source_id = 0 WHERE data_source_id IS NULL;
ALTER TABLE alert_rules DROP FOREIGN KEY fk_alert_rules_data_source;
ALTER TABLE alert_rules MODIFY COLUMN data_source_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE alert_rules ADD CONSTRAINT fk_alert_rules_data_source FOREIGN KEY (data_source_id) REFERENCES datasources(id);
