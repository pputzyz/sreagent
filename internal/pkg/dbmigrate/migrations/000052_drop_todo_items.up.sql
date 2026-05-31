-- Migration 000052: Drop unused todo_items table.
-- The todo feature was removed in favor of external task management integrations.
DROP TABLE IF EXISTS todo_items;
