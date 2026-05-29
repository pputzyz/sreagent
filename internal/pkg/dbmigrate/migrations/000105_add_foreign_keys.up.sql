-- =============================================================================
-- Migration 000105: Add foreign key constraints for critical relationships
-- Uses stored procedure to check constraint existence before adding (idempotent)
-- All FKs use BIGINT UNSIGNED to match existing column types
--
-- ON DELETE policy:
--   CASCADE  — child rows that only make sense with a parent (timelines, steps,
--              shifts, participants, rule histories)
--   SET NULL — nullable FK where the child should survive parent deletion
--              (alert_events.rule_id is DEFAULT NULL)
--   RESTRICT — NOT NULL FK where we refuse to delete a parent with live children
--              (team_members PK cols, incidents.channel_id, dispatch_policies.channel_id)
-- =============================================================================

DELIMITER //

CREATE PROCEDURE _migrate_000105_add_fk()
BEGIN
    DECLARE CONTINUE HANDLER FOR SQLEXCEPTION BEGIN END;

    -- 1. team_members.team_id -> teams.id (RESTRICT — part of composite PK, NOT NULL)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'team_members'
          AND CONSTRAINT_NAME = 'fk_team_members_team_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `team_members`
            ADD CONSTRAINT `fk_team_members_team_id`
            FOREIGN KEY (`team_id`) REFERENCES `teams`(`id`)
            ON DELETE RESTRICT;
    END IF;

    -- 2. team_members.user_id -> users.id (RESTRICT — part of composite PK, NOT NULL)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'team_members'
          AND CONSTRAINT_NAME = 'fk_team_members_user_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `team_members`
            ADD CONSTRAINT `fk_team_members_user_id`
            FOREIGN KEY (`user_id`) REFERENCES `users`(`id`)
            ON DELETE RESTRICT;
    END IF;

    -- 3. alert_events.rule_id -> alert_rules.id (SET NULL — column is DEFAULT NULL)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'alert_events'
          AND CONSTRAINT_NAME = 'fk_alert_events_rule_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `alert_events`
            ADD CONSTRAINT `fk_alert_events_rule_id`
            FOREIGN KEY (`rule_id`) REFERENCES `alert_rules`(`id`)
            ON DELETE SET NULL;
    END IF;

    -- 4. alert_timelines.event_id -> alert_events.id (CASCADE — child records)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'alert_timelines'
          AND CONSTRAINT_NAME = 'fk_alert_timelines_event_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `alert_timelines`
            ADD CONSTRAINT `fk_alert_timelines_event_id`
            FOREIGN KEY (`event_id`) REFERENCES `alert_events`(`id`)
            ON DELETE CASCADE;
    END IF;

    -- 5. escalation_steps.policy_id -> escalation_policies.id (CASCADE — child records)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'escalation_steps'
          AND CONSTRAINT_NAME = 'fk_escalation_steps_policy_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `escalation_steps`
            ADD CONSTRAINT `fk_escalation_steps_policy_id`
            FOREIGN KEY (`policy_id`) REFERENCES `escalation_policies`(`id`)
            ON DELETE CASCADE;
    END IF;

    -- 6. incidents.channel_id -> channels.id (RESTRICT — NOT NULL column)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'incidents'
          AND CONSTRAINT_NAME = 'fk_incidents_channel_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `incidents`
            ADD CONSTRAINT `fk_incidents_channel_id`
            FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`)
            ON DELETE RESTRICT;
    END IF;

    -- 7. dispatch_policies.channel_id -> channels.id (RESTRICT — NOT NULL column)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'dispatch_policies'
          AND CONSTRAINT_NAME = 'fk_dispatch_policies_channel_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `dispatch_policies`
            ADD CONSTRAINT `fk_dispatch_policies_channel_id`
            FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`)
            ON DELETE RESTRICT;
    END IF;

    -- 8. schedule_participants.schedule_id -> schedules.id (CASCADE — child records)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'schedule_participants'
          AND CONSTRAINT_NAME = 'fk_schedule_participants_schedule_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `schedule_participants`
            ADD CONSTRAINT `fk_schedule_participants_schedule_id`
            FOREIGN KEY (`schedule_id`) REFERENCES `schedules`(`id`)
            ON DELETE CASCADE;
    END IF;

    -- 9. oncall_shifts.schedule_id -> schedules.id (CASCADE — child records)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'oncall_shifts'
          AND CONSTRAINT_NAME = 'fk_oncall_shifts_schedule_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `oncall_shifts`
            ADD CONSTRAINT `fk_oncall_shifts_schedule_id`
            FOREIGN KEY (`schedule_id`) REFERENCES `schedules`(`id`)
            ON DELETE CASCADE;
    END IF;

    -- 10. alert_rule_histories.rule_id -> alert_rules.id (CASCADE — child records)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = DATABASE()
          AND TABLE_NAME = 'alert_rule_histories'
          AND CONSTRAINT_NAME = 'fk_alert_rule_histories_rule_id'
          AND CONSTRAINT_TYPE = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE `alert_rule_histories`
            ADD CONSTRAINT `fk_alert_rule_histories_rule_id`
            FOREIGN KEY (`rule_id`) REFERENCES `alert_rules`(`id`)
            ON DELETE CASCADE;
    END IF;
END //

DELIMITER ;

CALL _migrate_000105_add_fk();
DROP PROCEDURE IF EXISTS _migrate_000105_add_fk;
