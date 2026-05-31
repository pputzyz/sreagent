-- Migration 000042: Drop legacy notify_policies table.
-- Replaced by the v2 notification system (notify_rules + notify_medias + message_templates).
-- See migration 000019+ for the new channel-based notification architecture.
DROP TABLE IF EXISTS notify_policies;
