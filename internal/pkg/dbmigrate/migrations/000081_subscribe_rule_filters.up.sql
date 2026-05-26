ALTER TABLE subscribe_rules ADD COLUMN tag_filters JSON DEFAULT NULL AFTER severities;
ALTER TABLE subscribe_rules ADD COLUMN datasource_ids JSON DEFAULT NULL AFTER tag_filters;
ALTER TABLE subscribe_rules ADD COLUMN rule_ids JSON DEFAULT NULL AFTER datasource_ids;
ALTER TABLE subscribe_rules ADD COLUMN for_duration INT NOT NULL DEFAULT 0 AFTER rule_ids;
