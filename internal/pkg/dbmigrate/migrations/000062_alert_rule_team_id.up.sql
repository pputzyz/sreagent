ALTER TABLE alert_rules ADD COLUMN team_id BIGINT UNSIGNED NULL AFTER name;
CREATE INDEX idx_alert_rules_team_id ON alert_rules(team_id);
