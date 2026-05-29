DROP INDEX idx_inspection_tasks_deleted_at ON inspection_tasks;
ALTER TABLE inspection_tasks DROP COLUMN deleted_at;
