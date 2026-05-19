CREATE TABLE IF NOT EXISTS todo_items (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id      BIGINT UNSIGNED NOT NULL,
    title        VARCHAR(256) NOT NULL,
    description  VARCHAR(1024) DEFAULT '',
    type         VARCHAR(32) NOT NULL DEFAULT 'manual',
    status       VARCHAR(32) NOT NULL DEFAULT 'pending',
    priority     VARCHAR(32) NOT NULL DEFAULT 'medium',
    link         VARCHAR(512) DEFAULT '',
    due_at       DATETIME(3) NULL,
    completed_at DATETIME(3) NULL,
    created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at   DATETIME(3) NULL,
    INDEX idx_todo_items_user_id (user_id),
    INDEX idx_todo_items_status (status),
    INDEX idx_todo_items_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
