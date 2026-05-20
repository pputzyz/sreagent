ALTER TABLE `alert_rules` ADD COLUMN `status` VARCHAR(32) NOT NULL DEFAULT 'active' AFTER `enabled`, ADD INDEX `idx_alert_rules_status` (`status`);
