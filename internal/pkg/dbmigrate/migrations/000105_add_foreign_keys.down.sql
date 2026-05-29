-- =============================================================================
-- Migration 000105 (down): Drop all foreign key constraints added in up.sql
-- Uses stored procedure to check constraint existence before dropping (idempotent)
-- =============================================================================

DELIMITER //

CREATE PROCEDURE _migrate_000105_drop_fk()
BEGIN
    DECLARE CONTINUE HANDLER FOR SQLEXCEPTION BEGIN END;

    -- 10. alert_rule_histories
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'alert_rule_histories'
          AND CONSTRAINT_NAME = 'fk_alert_rule_histories_rule_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `alert_rule_histories`
            DROP FOREIGN KEY `fk_alert_rule_histories_rule_id`;
    END IF;

    -- 9. oncall_shifts
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'oncall_shifts'
          AND CONSTRAINT_NAME = 'fk_oncall_shifts_schedule_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `oncall_shifts`
            DROP FOREIGN KEY `fk_oncall_shifts_schedule_id`;
    END IF;

    -- 8. schedule_participants
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'schedule_participants'
          AND CONSTRAINT_NAME = 'fk_schedule_participants_schedule_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `schedule_participants`
            DROP FOREIGN KEY `fk_schedule_participants_schedule_id`;
    END IF;

    -- 7. dispatch_policies
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'dispatch_policies'
          AND CONSTRAINT_NAME = 'fk_dispatch_policies_channel_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `dispatch_policies`
            DROP FOREIGN KEY `fk_dispatch_policies_channel_id`;
    END IF;

    -- 6. incidents
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'incidents'
          AND CONSTRAINT_NAME = 'fk_incidents_channel_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `incidents`
            DROP FOREIGN KEY `fk_incidents_channel_id`;
    END IF;

    -- 5. escalation_steps
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'escalation_steps'
          AND CONSTRAINT_NAME = 'fk_escalation_steps_policy_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `escalation_steps`
            DROP FOREIGN KEY `fk_escalation_steps_policy_id`;
    END IF;

    -- 4. alert_timelines
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'alert_timelines'
          AND CONSTRAINT_NAME = 'fk_alert_timelines_event_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `alert_timelines`
            DROP FOREIGN KEY `fk_alert_timelines_event_id`;
    END IF;

    -- 3. alert_events
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'alert_events'
          AND CONSTRAINT_NAME = 'fk_alert_events_rule_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `alert_events`
            DROP FOREIGN KEY `fk_alert_events_rule_id`;
    END IF;

    -- 2. team_members.user_id
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'team_members'
          AND CONSTRAINT_NAME = 'fk_team_members_user_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `team_members`
            DROP FOREIGN KEY `fk_team_members_user_id`;
    END IF;

    -- 1. team_members.team_id
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'team_members'
          AND CONSTRAINT_NAME = 'fk_team_members_team_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `team_members`
            DROP FOREIGN KEY `fk_team_members_team_id`;
    END IF;
END //

DELIMITER ;

CALL _migrate_000105_drop_fk();
DROP PROCEDURE IF EXISTS _migrate_000105_drop_fk;
