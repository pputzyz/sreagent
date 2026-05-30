ALTER TABLE notify_rules ADD COLUMN group_aggregate TINYINT(1) NOT NULL DEFAULT 0 AFTER callback_url
