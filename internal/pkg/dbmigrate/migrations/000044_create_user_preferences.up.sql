CREATE TABLE IF NOT EXISTS user_preferences (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    theme VARCHAR(16) DEFAULT 'auto',
    language VARCHAR(16) DEFAULT 'zh-CN',
    timezone VARCHAR(64) DEFAULT 'Asia/Shanghai',
    default_time_range VARCHAR(16) DEFAULT '24h',
    notification_severities JSON NULL,
    ai_chat_mode VARCHAR(16) DEFAULT 'sidebar',
    UNIQUE INDEX idx_user_preferences_user_id (user_id),
    INDEX idx_user_preferences_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
