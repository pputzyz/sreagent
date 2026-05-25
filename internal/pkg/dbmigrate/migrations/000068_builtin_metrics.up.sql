CREATE TABLE IF NOT EXISTS `builtin_metrics` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `collector` varchar(191) NOT NULL DEFAULT '' COMMENT 'category/collector, e.g. node_exporter',
  `typ` varchar(191) NOT NULL DEFAULT '' COMMENT 'component type, e.g. Linux, MySQL',
  `name` varchar(191) NOT NULL DEFAULT '' COMMENT 'metric display name',
  `unit` varchar(191) NOT NULL DEFAULT '' COMMENT 'unit string',
  `note` varchar(4096) NOT NULL DEFAULT '' COMMENT 'description (supports markdown)',
  `lang` varchar(32) NOT NULL DEFAULT 'zh' COMMENT 'language code',
  `expression` varchar(4096) NOT NULL DEFAULT '' COMMENT 'PromQL expression or metric name',
  `expression_type` varchar(32) NOT NULL DEFAULT 'metric_name' COMMENT 'metric_name or promql',
  `metric_type` varchar(64) NOT NULL DEFAULT '' COMMENT 'gauge, counter, or histogram',
  `extra_fields` text COMMENT 'custom key-value pairs JSON',
  `translation` text COMMENT 'multilingual translations JSON',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_by` varchar(64) NOT NULL DEFAULT '',
  `updated_by` varchar(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_expr_collector_typ` (`expression`(255), `collector`, `typ`),
  KEY `idx_collector` (`collector`),
  KEY `idx_typ` (`typ`),
  KEY `idx_lang` (`lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `metric_filters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(191) NOT NULL DEFAULT '' COMMENT 'filter name',
  `configs` varchar(4096) NOT NULL DEFAULT '[]' COMMENT 'JSON array of label filter conditions',
  `groups_perm` text COMMENT 'team-based permissions JSON',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_by` varchar(64) NOT NULL DEFAULT '',
  `updated_by` varchar(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_created_by` (`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
