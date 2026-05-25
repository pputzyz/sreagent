-- Event Pipeline v2: add processor_configs column and rename execution table
-- The original 000017 migration created event_pipelines with nodes/connections
-- for DAG support. This migration restructures for linear processor pipeline.

-- Add processor_configs column (replaces nodes for linear pipeline use)
ALTER TABLE event_pipelines ADD COLUMN processor_configs JSON NOT NULL AFTER label_filters;

-- Drop the unused connections column (DAG support deferred)
ALTER TABLE event_pipelines DROP COLUMN connections;

-- Rename pipeline_executions to event_pipeline_executions and add missing columns
RENAME TABLE pipeline_executions TO event_pipeline_executions;

ALTER TABLE event_pipeline_executions
    ADD COLUMN pipeline_name VARCHAR(128) NOT NULL DEFAULT '' AFTER pipeline_id,
    ADD COLUMN mode VARCHAR(16) NOT NULL DEFAULT 'event' AFTER event_id,
    ADD COLUMN trigger_by VARCHAR(64) NOT NULL DEFAULT '' AFTER duration_ms,
    ADD COLUMN created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) AFTER trigger_by;
