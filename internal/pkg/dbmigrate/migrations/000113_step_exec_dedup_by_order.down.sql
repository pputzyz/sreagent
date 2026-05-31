-- Rollback B6-3: Restore (event_id, step_id) dedup key.

ALTER TABLE escalation_step_executions ADD COLUMN step_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER policy_id;

ALTER TABLE escalation_step_executions DROP INDEX uk_event_policy_order;
ALTER TABLE escalation_step_executions ADD UNIQUE KEY uk_event_step (event_id, step_id);

ALTER TABLE escalation_step_executions DROP COLUMN step_order;
ALTER TABLE escalation_step_executions DROP COLUMN policy_id;

-- Rollback B6-5: Restore NOT NULL on team_id.
ALTER TABLE escalation_policies MODIFY COLUMN team_id BIGINT UNSIGNED NOT NULL;
