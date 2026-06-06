ALTER TABLE alert_rules
  DROP COLUMN IF EXISTS nodata_duration,
  DROP COLUMN IF EXISTS nodata_enabled,
  DROP COLUMN IF EXISTS suppress_enabled;
