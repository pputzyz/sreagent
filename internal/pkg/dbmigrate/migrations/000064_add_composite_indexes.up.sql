CREATE INDEX IF NOT EXISTS idx_incidents_status_channel ON incidents(status, channel_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_status_severity_fired ON alert_events(status, severity, fired_at);
