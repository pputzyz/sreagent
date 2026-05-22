ALTER TABLE alert_rules ADD COLUMN team_id BIGINT UNSIGNED NULL AFTER name, ADD INDEX idx_alert_rules_team_id (team_id);
