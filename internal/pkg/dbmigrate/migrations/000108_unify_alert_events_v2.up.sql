-- Migration 000108: Unify alert_events_v2 into alert_events
-- Adds alert_id and value columns to alert_events, migrates data from alert_events_v2,
-- and relaxes the unique fingerprint constraint to a regular index (multiple events
-- can share a fingerprint when they are v2 pipeline snapshots).

-- Step 1: Add new columns
ALTER TABLE `alert_events`
    ADD COLUMN `alert_id` BIGINT UNSIGNED NULL DEFAULT NULL AFTER `escalation_policy_id`,
    ADD COLUMN `value`    DOUBLE          DEFAULT 0    AFTER `alert_id`;

-- Step 2: Relax fingerprint unique constraint → regular index
-- The UNIQUE constraint prevented v2 pipeline from creating multiple snapshot events
-- per fingerprint. A regular index still supports lookups without uniqueness enforcement.
ALTER TABLE `alert_events`
    DROP INDEX `idx_alert_events_fingerprint`,
    ADD INDEX `idx_alert_events_fingerprint` (`fingerprint`);

-- Step 3: Add indexes for the new columns
ALTER TABLE `alert_events`
    ADD INDEX `idx_alert_events_alert_id` (`alert_id`);

-- Step 4: Migrate data from alert_events_v2 into alert_events
-- Each v2 event becomes a new row in alert_events with alert_id set.
-- We map v2 fields to v1 equivalents:
--   event_status firing → status 'firing', event_status resolved → status 'resolved'
--   event_severity → severity
--   timestamp → fired_at (and created_at via BaseModel)
--   alert_id → alert_id
--   value → value
--   fingerprint → fingerprint
--   labels → labels
--   annotations → annotations
--   alert.title → alert_name (via join)
INSERT INTO `alert_events` (
    `created_at`, `updated_at`, `fingerprint`, `alert_name`, `severity`, `status`,
    `labels`, `annotations`, `fired_at`, `alert_id`, `value`
)
SELECT
    v2.`created_at`,
    v2.`updated_at`,
    v2.`fingerprint`,
    COALESCE(a.`title`, ''),
    v2.`event_severity`,
    v2.`event_status`,
    v2.`labels`,
    v2.`annotations`,
    v2.`timestamp`,
    v2.`alert_id`,
    v2.`value`
FROM `alert_events_v2` v2
LEFT JOIN `alerts` a ON a.`id` = v2.`alert_id`;

-- Step 5: Add foreign key for alert_id (SET NULL — event survives alert deletion)
-- The migration runner treats errno 1061 (Duplicate key name) as idempotent, so a
-- plain ALTER TABLE is safe to re-run.
ALTER TABLE `alert_events`
    ADD CONSTRAINT `fk_alert_events_alert_id`
    FOREIGN KEY (`alert_id`) REFERENCES `alerts`(`id`)
    ON DELETE SET NULL;
