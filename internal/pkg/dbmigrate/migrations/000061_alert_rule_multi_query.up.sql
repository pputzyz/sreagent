-- Add multi-query support columns to alert_rules table
-- These columns enable Nightingale-style multi-query evaluation with joins

ALTER TABLE alert_rules
ADD COLUMN queries JSON COMMENT 'Multiple queries (A, B, C...) for multi-query evaluation' AFTER expression,
ADD COLUMN trigger_exp VARCHAR(512) DEFAULT '' COMMENT 'Trigger expression referencing $A, $B, etc.' AFTER queries,
ADD COLUMN join_type VARCHAR(32) DEFAULT '' COMMENT 'Join type: inner_join, left_join, right_join, none' AFTER trigger_exp,
ADD COLUMN join_keys JSON COMMENT 'Label keys to join on' AFTER join_type;
