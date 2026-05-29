ALTER TABLE `notify_records` ADD COLUMN `fingerprint` varchar(256) DEFAULT '' AFTER `policy_id`;
CREATE INDEX `idx_notify_records_fingerprint` ON `notify_records` (`fingerprint`);
