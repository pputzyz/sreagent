ALTER TABLE `alert_channels` DROP INDEX `idx_alert_channels_datasource_id`;
ALTER TABLE `alert_channels` DROP COLUMN `datasource_id`;

ALTER TABLE `notify_rules` DROP INDEX `idx_notify_rules_datasource_id`;
ALTER TABLE `notify_rules` DROP COLUMN `datasource_id`;

ALTER TABLE `dispatch_policies` DROP INDEX `idx_dispatch_policies_datasource_id`;
ALTER TABLE `dispatch_policies` DROP COLUMN `datasource_id`;
