ALTER TABLE alert_events DROP INDEX idx_alert_events_datasource_id;
ALTER TABLE alert_events DROP COLUMN datasource_id;
