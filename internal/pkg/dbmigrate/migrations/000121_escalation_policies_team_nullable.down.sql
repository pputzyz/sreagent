UPDATE escalation_policies SET team_id = 0 WHERE team_id IS NULL;
ALTER TABLE escalation_policies MODIFY COLUMN team_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
