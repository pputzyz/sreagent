ALTER TABLE incidents ADD INDEX idx_incidents_status_channel (status, channel_id);
ALTER TABLE alert_events ADD INDEX idx_alert_events_status_severity_fired (status, severity, fired_at);
