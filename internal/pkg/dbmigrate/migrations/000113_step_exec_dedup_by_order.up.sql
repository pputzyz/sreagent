-- B6-3: Change escalation step dedup from (event_id, step_id) to (event_id, policy_id, step_order).
-- Step IDs are regenerated on ReplaceEscalationSteps, making step_id-based dedup unreliable.

ALTER TABLE escalation_step_executions ADD COLUMN policy_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER event_id;
ALTER TABLE escalation_step_executions ADD COLUMN step_order INT NOT NULL DEFAULT 0 AFTER policy_id;

-- Populate from escalation_steps (best-effort; 0 for orphaned records).
UPDATE escalation_step_executions e
  JOIN escalation_steps s ON e.step_id = s.id
  SET e.policy_id = s.policy_id, e.step_order = s.step_order;

ALTER TABLE escalation_step_executions DROP INDEX uk_event_step;
ALTER TABLE escalation_step_executions ADD UNIQUE KEY uk_event_policy_order (event_id, policy_id, step_order);
ALTER TABLE escalation_step_executions DROP COLUMN step_id;

-- B6-5: Allow global escalation policies (team_id = 0).
ALTER TABLE escalation_policies MODIFY COLUMN team_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
