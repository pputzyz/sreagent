ALTER TABLE alert_rules
  ADD COLUMN nodata_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER suppress_enabled,
  ADD COLUMN nodata_duration VARCHAR(32) NOT NULL DEFAULT '5m' AFTER nodata_enabled;
