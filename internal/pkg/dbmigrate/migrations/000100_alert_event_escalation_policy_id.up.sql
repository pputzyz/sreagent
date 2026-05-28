ALTER TABLE alert_events ADD COLUMN escalation_policy_id BIGINT UNSIGNED NULL;
CREATE INDEX idx_alert_events_escalation_policy_id ON alert_events(escalation_policy_id);
