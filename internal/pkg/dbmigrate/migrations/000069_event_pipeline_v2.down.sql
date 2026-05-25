-- Reverse: restore original schema from 000017

-- Remove added columns from event_pipeline_executions
ALTER TABLE event_pipeline_executions
    DROP COLUMN pipeline_name,
    DROP COLUMN mode,
    DROP COLUMN trigger_by,
    DROP COLUMN created_at;

-- Rename back
RENAME TABLE event_pipeline_executions TO pipeline_executions;

-- Restore connections column and drop processor_configs
ALTER TABLE event_pipelines ADD COLUMN connections JSON AFTER nodes;
ALTER TABLE event_pipelines DROP COLUMN processor_configs;
