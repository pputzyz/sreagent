-- =============================================================================
-- Migration 000105 (down): Drop all foreign key constraints added in up.sql
-- Idempotent: golang-migrate ignores errno 1091 (can't drop, doesn't exist)
-- =============================================================================

ALTER TABLE `alert_rule_histories`    DROP FOREIGN KEY `fk_alert_rule_histories_rule_id`;
ALTER TABLE `oncall_shifts`           DROP FOREIGN KEY `fk_oncall_shifts_schedule_id`;
ALTER TABLE `schedule_participants`   DROP FOREIGN KEY `fk_schedule_participants_schedule_id`;
ALTER TABLE `dispatch_policies`       DROP FOREIGN KEY `fk_dispatch_policies_channel_id`;
ALTER TABLE `incidents`               DROP FOREIGN KEY `fk_incidents_channel_id`;
ALTER TABLE `escalation_steps`        DROP FOREIGN KEY `fk_escalation_steps_policy_id`;
ALTER TABLE `alert_timelines`         DROP FOREIGN KEY `fk_alert_timelines_event_id`;
ALTER TABLE `alert_events`            DROP FOREIGN KEY `fk_alert_events_rule_id`;
ALTER TABLE `team_members`            DROP FOREIGN KEY `fk_team_members_user_id`;
ALTER TABLE `team_members`            DROP FOREIGN KEY `fk_team_members_team_id`;
