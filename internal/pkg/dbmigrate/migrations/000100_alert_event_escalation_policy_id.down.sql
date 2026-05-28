DROP INDEX idx_alert_events_escalation_policy_id ON alert_events;
ALTER TABLE alert_events DROP COLUMN escalation_policy_id;
