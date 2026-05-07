CREATE TABLE post_mortems (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    incident_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(256) NOT NULL,
    content LONGTEXT,
    status VARCHAR(32) DEFAULT 'draft',
    author_id BIGINT UNSIGNED NULL,
    published_at DATETIME(3) NULL,
    UNIQUE INDEX idx_pm_incident_id (incident_id),
    INDEX idx_pm_author_id (author_id),
    INDEX idx_pm_deleted_at (deleted_at)
)