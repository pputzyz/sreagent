-- Migration 000110: Fix remaining tables with non-standard charset.
-- These tables were created before the charset standardization in 000107 and
-- still use the MySQL default utf8mb4_0900_ai_ci. Convert to utf8mb4_unicode_ci
-- for consistency with all other tables.
ALTER TABLE label_registry CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
ALTER TABLE channels CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
ALTER TABLE incidents CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
