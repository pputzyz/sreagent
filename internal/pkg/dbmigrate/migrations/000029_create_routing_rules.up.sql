CREATE TABLE routing_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    integration_id BIGINT UNSIGNED NOT NULL,
    conditions JSON,
    target_channel_id BIGINT UNSIGNED NOT NULL,
    priority INT DEFAULT 0,
    is_enabled BOOLEAN DEFAULT TRUE,
    INDEX idx_rr_integration_id (integration_id),
    INDEX idx_rr_target_channel_id (target_channel_id),
    INDEX idx_rr_priority (priority),
    INDEX idx_rr_deleted_at (deleted_at)
)