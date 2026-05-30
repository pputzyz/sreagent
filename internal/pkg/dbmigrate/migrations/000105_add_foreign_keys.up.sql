-- =============================================================================
-- Migration 000105: Add foreign key constraints for critical relationships
-- Idempotent: golang-migrate ignores errno 1061 (duplicate key name)
-- All FKs use BIGINT UNSIGNED to match existing column types
--
-- ON DELETE policy:
--   CASCADE  — child rows that only make sense with a parent
--   SET NULL — nullable FK where the child should survive parent deletion
--   RESTRICT — NOT NULL FK where we refuse to delete a parent with live children
-- =============================================================================

-- 1. team_members.team_id -> teams.id (RESTRICT — part of composite PK, NOT NULL)
ALTER TABLE `team_members`
    ADD CONSTRAINT `fk_team_members_team_id`
    FOREIGN KEY (`team_id`) REFERENCES `teams`(`id`)
    ON DELETE RESTRICT;

-- 2. team_members.user_id -> users.id (RESTRICT — part of composite PK, NOT NULL)
ALTER TABLE `team_members`
    ADD CONSTRAINT `fk_team_members_user_id`
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`)
    ON DELETE RESTRICT;

-- 3. alert_events.rule_id -> alert_rules.id (SET NULL — column is DEFAULT NULL)
ALTER TABLE `alert_events`
    ADD CONSTRAINT `fk_alert_events_rule_id`
    FOREIGN KEY (`rule_id`) REFERENCES `alert_rules`(`id`)
    ON DELETE SET NULL;

-- 4. alert_timelines.event_id -> alert_events.id (CASCADE — child records)
ALTER TABLE `alert_timelines`
    ADD CONSTRAINT `fk_alert_timelines_event_id`
    FOREIGN KEY (`event_id`) REFERENCES `alert_events`(`id`)
    ON DELETE CASCADE;

-- 5. escalation_steps.policy_id -> escalation_policies.id (CASCADE — child records)
ALTER TABLE `escalation_steps`
    ADD CONSTRAINT `fk_escalation_steps_policy_id`
    FOREIGN KEY (`policy_id`) REFERENCES `escalation_policies`(`id`)
    ON DELETE CASCADE;

-- 6. incidents.channel_id -> channels.id (RESTRICT — NOT NULL column)
ALTER TABLE `incidents`
    ADD CONSTRAINT `fk_incidents_channel_id`
    FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`)
    ON DELETE RESTRICT;

-- 7. dispatch_policies.channel_id -> channels.id (RESTRICT — NOT NULL column)
ALTER TABLE `dispatch_policies`
    ADD CONSTRAINT `fk_dispatch_policies_channel_id`
    FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`)
    ON DELETE RESTRICT;

-- 8. schedule_participants.schedule_id -> schedules.id (CASCADE — child records)
ALTER TABLE `schedule_participants`
    ADD CONSTRAINT `fk_schedule_participants_schedule_id`
    FOREIGN KEY (`schedule_id`) REFERENCES `schedules`(`id`)
    ON DELETE CASCADE;

-- 9. oncall_shifts.schedule_id -> schedules.id (CASCADE — child records)
ALTER TABLE `oncall_shifts`
    ADD CONSTRAINT `fk_oncall_shifts_schedule_id`
    FOREIGN KEY (`schedule_id`) REFERENCES `schedules`(`id`)
    ON DELETE CASCADE;

-- 10. alert_rule_histories.rule_id -> alert_rules.id (CASCADE — child records)
ALTER TABLE `alert_rule_histories`
    ADD CONSTRAINT `fk_alert_rule_histories_rule_id`
    FOREIGN KEY (`rule_id`) REFERENCES `alert_rules`(`id`)
    ON DELETE CASCADE;
