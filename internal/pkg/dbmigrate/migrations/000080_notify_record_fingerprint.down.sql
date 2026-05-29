DROP INDEX `idx_notify_records_fingerprint` ON `notify_records`;
ALTER TABLE `notify_records` DROP COLUMN `fingerprint`;
