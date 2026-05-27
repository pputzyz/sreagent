-- Migrate alert_channels data into dispatch_policies
-- match_labels/severities converted to match_conditions JSON format
-- throttle_min converted to repeat_interval_seconds (×60)

INSERT INTO dispatch_policies (
    channel_id, name, description, is_enabled, priority,
    match_conditions, datasource_id, active_time_config,
    delay_seconds, escalation_policy_id,
    repeat_interval_seconds, max_repeats,
    notify_mode, unified_media_id, unified_template_id,
    label_enhancement_rules,
    created_at, updated_at
)
SELECT
    0 AS channel_id,
    ac.name,
    CONCAT('Migrated from alert_channel #', ac.id) AS description,
    ac.is_enabled,
    0 AS priority,
    CASE
        WHEN ac.severities IS NOT NULL AND ac.severities != '' THEN
            JSON_ARRAY(
                JSON_OBJECT('field', 'severity', 'operator', 'in', 'value', ac.severities)
            )
        ELSE '[]'
    END AS match_conditions,
    ac.datasource_id,
    '{}' AS active_time_config,
    COALESCE(ac.throttle_min * 60, 0) AS delay_seconds,
    NULL AS escalation_policy_id,
    COALESCE(ac.throttle_min * 60, 0) AS repeat_interval_seconds,
    0 AS max_repeats,
    'unified' AS notify_mode,
    ac.media_id AS unified_media_id,
    ac.template_id AS unified_template_id,
    '[]' AS label_enhancement_rules,
    ac.created_at,
    ac.updated_at
FROM alert_channels ac
WHERE NOT EXISTS (
    SELECT 1 FROM dispatch_policies dp
    WHERE dp.description LIKE CONCAT('Migrated from alert_channel #', ac.id)
);
