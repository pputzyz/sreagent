ALTER TABLE `alert_rules` ADD COLUMN `status` VARCHAR(32) NOT NULL DEFAULT 'active' AFTER `enabled`;
ALTER TABLE `alert_rules` ADD INDEX `idx_alert_rules_status` (`status`);
