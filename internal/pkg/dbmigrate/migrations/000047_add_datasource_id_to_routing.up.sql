ALTER TABLE `alert_channels` ADD COLUMN `datasource_id` BIGINT UNSIGNED NULL AFTER `match_labels`;
ALTER TABLE `alert_channels` ADD INDEX `idx_alert_channels_datasource_id` (`datasource_id`);

ALTER TABLE `notify_rules` ADD COLUMN `datasource_id` BIGINT UNSIGNED NULL AFTER `match_labels`;
ALTER TABLE `notify_rules` ADD INDEX `idx_notify_rules_datasource_id` (`datasource_id`);

ALTER TABLE `dispatch_policies` ADD COLUMN `datasource_id` BIGINT UNSIGNED NULL AFTER `match_conditions`;
ALTER TABLE `dispatch_policies` ADD INDEX `idx_dispatch_policies_datasource_id` (`datasource_id`);
