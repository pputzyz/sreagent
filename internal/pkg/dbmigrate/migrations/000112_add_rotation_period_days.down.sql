-- Migration 000112 (down): Remove rotation_period_days column from schedules table
ALTER TABLE `schedules`
    DROP COLUMN `rotation_period_days`;
