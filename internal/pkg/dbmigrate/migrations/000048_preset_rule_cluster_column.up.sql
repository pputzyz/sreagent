ALTER TABLE `preset_rules` ADD COLUMN `cluster` VARCHAR(100) NULL DEFAULT NULL AFTER `component`;
ALTER TABLE `preset_rules` ADD INDEX `idx_preset_rules_cluster` (`cluster`);
