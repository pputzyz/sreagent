CREATE TABLE channel_exclusion_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    channel_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512) DEFAULT '',
    conditions JSON,
    is_enabled BOOLEAN DEFAULT TRUE,
    priority INT DEFAULT 0,
    INDEX idx_cer_channel_id (channel_id),
    INDEX idx_cer_priority (priority),
    INDEX idx_cer_deleted_at (deleted_at)
)