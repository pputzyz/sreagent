ALTER TABLE escalation_step_executions ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending' AFTER step_id;
UPDATE escalation_step_executions SET status = 'success';
