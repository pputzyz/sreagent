CREATE TABLE IF NOT EXISTS `recording_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `group_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'business group id',
  `name` varchar(255) NOT NULL DEFAULT '' COMMENT 'new metric name produced by the rule',
  `prom_ql` text NOT NULL COMMENT 'PromQL expression to evaluate',
  `datasource_ids` varchar(1024) NOT NULL DEFAULT '[]' COMMENT 'JSON array of datasource ids; 0 means all',
  `cron_pattern` varchar(64) NOT NULL DEFAULT '@every 60s' COMMENT 'cron schedule for evaluation',
  `disabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '0=enabled, 1=disabled',
  `append_tags` text COMMENT 'space-separated key=value pairs appended to output metrics',
  `note` varchar(1024) NOT NULL DEFAULT '' COMMENT 'free-text description',
  `query_configs` text COMMENT 'structured query config JSON',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_by` varchar(64) NOT NULL DEFAULT '',
  `updated_by` varchar(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_disabled` (`disabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
