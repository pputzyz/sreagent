CREATE TABLE IF NOT EXISTS notify_policies (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512),
    match_labels JSON NOT NULL,
    severities VARCHAR(128),
    channel_id BIGINT UNSIGNED NOT NULL,
    throttle_minutes INT DEFAULT 5,
    template_name VARCHAR(64) DEFAULT 'default',
    is_enabled TINYINT(1) DEFAULT 1,
    priority INT DEFAULT 0,
    INDEX idx_notify_policies_channel_id (channel_id),
    INDEX idx_notify_policies_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
