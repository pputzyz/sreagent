CREATE INDEX `idx_notify_records_dedup` ON `notify_records` (`fingerprint`, `channel_id`, `policy_id`, `status`);
