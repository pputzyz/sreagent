ALTER TABLE `recording_rules` ADD COLUMN `deleted_at` datetime DEFAULT NULL AFTER `updated_by`;
ALTER TABLE `recording_rules` ADD KEY `idx_deleted_at` (`deleted_at`);
