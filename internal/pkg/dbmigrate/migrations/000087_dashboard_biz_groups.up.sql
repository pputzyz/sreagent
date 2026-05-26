CREATE TABLE IF NOT EXISTS dashboard_biz_groups (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    dashboard_id BIGINT UNSIGNED NOT NULL,
    biz_group_id BIGINT UNSIGNED NOT NULL,
    perm_flag VARCHAR(4) NOT NULL DEFAULT 'ro',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE INDEX idx_did_bgid (dashboard_id, biz_group_id),
    INDEX idx_biz_group_id (biz_group_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
