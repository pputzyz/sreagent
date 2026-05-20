ALTER TABLE alert_events ADD COLUMN datasource_id BIGINT UNSIGNED NULL AFTER source;
ALTER TABLE alert_events ADD INDEX idx_alert_events_datasource_id (datasource_id);
