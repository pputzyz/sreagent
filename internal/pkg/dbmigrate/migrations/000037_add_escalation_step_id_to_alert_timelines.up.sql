ALTER TABLE alert_timelines ADD COLUMN escalation_step_id BIGINT UNSIGNED NULL AFTER extra, ADD INDEX idx_alert_timelines_escalation_step_id (escalation_step_id);
