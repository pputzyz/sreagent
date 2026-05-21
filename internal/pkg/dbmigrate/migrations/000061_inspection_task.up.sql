CREATE TABLE IF NOT EXISTS inspection_tasks (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name            VARCHAR(128) NOT NULL,
  description     TEXT NOT NULL COMMENT '自然语言任务描述，喂给 AI',
  cron_expr       VARCHAR(64) NOT NULL,
  target_type     VARCHAR(32) NOT NULL DEFAULT 'global' COMMENT 'global/biz_group',
  target_ids      JSON NULL COMMENT '[1,2,3]',
  allowed_tools   JSON NULL COMMENT '工具白名单，nil=全只读工具',
  output_channels JSON NOT NULL COMMENT '[{"type":"lark_bot","bot_id":"xxx"},{"type":"email","to":[]}]',
  enabled         TINYINT NOT NULL DEFAULT 1,
  created_by      BIGINT UNSIGNED NOT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS inspection_runs (
  id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  task_id             BIGINT UNSIGNED NOT NULL,
  status              VARCHAR(20) NOT NULL DEFAULT 'running' COMMENT 'running/success/failed',
  started_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  finished_at         DATETIME(3) NULL,
  report_markdown     LONGTEXT NULL,
  report_summary      VARCHAR(500) NULL,
  findings_json       JSON NULL COMMENT '结构化发现项',
  error_msg           TEXT NULL,
  ai_conversation_id  BIGINT UNSIGNED NULL COMMENT '关联 ai_conversations.id 便于追溯',
  INDEX idx_task_started (task_id, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
