CREATE TABLE IF NOT EXISTS annotations (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    dashboard_id BIGINT UNSIGNED NOT NULL,
    time DATETIME(3) NOT NULL,
    end_time DATETIME(3) DEFAULT NULL,
    text VARCHAR(1024) NOT NULL DEFAULT '',
    tags JSON DEFAULT NULL,
    source VARCHAR(64) NOT NULL DEFAULT 'user',
    created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) DEFAULT NULL,
    INDEX idx_annotations_dashboard_id (dashboard_id),
    INDEX idx_annotations_time (time),
    INDEX idx_annotations_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
