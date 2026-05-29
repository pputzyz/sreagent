ALTER TABLE inspection_tasks ADD COLUMN deleted_at DATETIME(3) NULL;
CREATE INDEX idx_inspection_tasks_deleted_at ON inspection_tasks (deleted_at);
