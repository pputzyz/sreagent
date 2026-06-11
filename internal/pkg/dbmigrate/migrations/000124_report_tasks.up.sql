CREATE TABLE IF NOT EXISTS report_tasks (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    cron_expr VARCHAR(64) NOT NULL,
    report_type VARCHAR(32) NOT NULL DEFAULT 'daily',
    scope JSON,
    prompt_template TEXT NOT NULL,
    allowed_tools JSON,
    output_channels JSON NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) DEFAULT NULL,
    INDEX idx_enabled (enabled),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS report_runs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    started_at DATETIME(3) NOT NULL,
    finished_at DATETIME(3) DEFAULT NULL,
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    report_markdown LONGTEXT,
    report_summary VARCHAR(500),
    findings_json JSON,
    error_msg TEXT,
    ai_conversation_id BIGINT UNSIGNED DEFAULT NULL,
    INDEX idx_task_id (task_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
