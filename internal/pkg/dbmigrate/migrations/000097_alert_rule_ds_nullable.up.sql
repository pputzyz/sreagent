ALTER TABLE alert_rules DROP FOREIGN KEY IF EXISTS fk_alert_rules_data_source;
ALTER TABLE alert_rules MODIFY COLUMN data_source_id BIGINT UNSIGNED NULL;
ALTER TABLE alert_rules ADD CONSTRAINT fk_alert_rules_data_source FOREIGN KEY (data_source_id) REFERENCES datasources(id);
