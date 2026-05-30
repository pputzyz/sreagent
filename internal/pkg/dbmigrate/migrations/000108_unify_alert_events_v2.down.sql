-- Rollback migration 000108: Restore alert_events_v2 separation

-- Step 1: Remove migrated v2 data from alert_events
-- Events with alert_id IS NOT NULL were migrated from alert_events_v2.
DELETE FROM `alert_events` WHERE `alert_id` IS NOT NULL;

-- Step 2: Drop foreign key constraint
ALTER TABLE `alert_events` DROP FOREIGN KEY IF EXISTS `fk_alert_events_alert_id`;

-- Step 3: Drop indexes and columns added by the up migration
ALTER TABLE `alert_events`
    DROP INDEX IF EXISTS `idx_alert_events_alert_id`,
    DROP COLUMN IF EXISTS `alert_id`,
    DROP COLUMN IF EXISTS `value`;

-- Step 4: Restore UNIQUE index on fingerprint
-- First drop the regular index, then re-add as UNIQUE.
ALTER TABLE `alert_events`
    DROP INDEX `idx_alert_events_fingerprint`,
    ADD UNIQUE INDEX `idx_alert_events_fingerprint` (`fingerprint`);
