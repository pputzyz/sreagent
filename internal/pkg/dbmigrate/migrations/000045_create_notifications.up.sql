CREATE TABLE IF NOT EXISTS user_notifications (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id    BIGINT UNSIGNED NOT NULL,
    title      VARCHAR(256) NOT NULL,
    content    VARCHAR(1024) DEFAULT '',
    type       VARCHAR(32) NOT NULL DEFAULT 'system',
    is_read    TINYINT(1) NOT NULL DEFAULT 0,
    link       VARCHAR(512) DEFAULT '',
    metadata   JSON NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    INDEX idx_user_notifications_user_id (user_id),
    INDEX idx_user_notifications_is_read (is_read),
    INDEX idx_user_notifications_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
