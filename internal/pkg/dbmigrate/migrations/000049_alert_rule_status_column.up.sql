ALTER TABLE `alert_rules` ADD COLUMN `status` VARCHAR(32) NOT NULL DEFAULT 'enabled' AFTER `enabled`;
UPDATE `alert_rules` SET `status` = 'enabled' WHERE `status` = 'active';
ALTER TABLE `alert_rules` ADD INDEX `idx_alert_rules_status` (`status`);
