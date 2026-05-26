ALTER TABLE `recording_rules` DROP KEY `idx_deleted_at`;
ALTER TABLE `recording_rules` DROP COLUMN `deleted_at`;
