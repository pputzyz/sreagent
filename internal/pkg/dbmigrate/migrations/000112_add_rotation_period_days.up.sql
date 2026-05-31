-- Migration 000112: Add rotation_period_days column to schedules table
-- B11-11: Supports custom rotation period for RotationCustom type.
ALTER TABLE `schedules`
    ADD COLUMN `rotation_period_days` INT NOT NULL DEFAULT 1 AFTER `handoff_day`;
