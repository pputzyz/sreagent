ALTER TABLE alert_rules ADD COLUMN var_config JSON DEFAULT NULL AFTER biz_group_id;
