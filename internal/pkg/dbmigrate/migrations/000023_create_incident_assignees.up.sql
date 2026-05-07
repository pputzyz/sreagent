CREATE TABLE incident_assignees (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    incident_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    is_acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at DATETIME(3) NULL,
    assigned_at DATETIME(3) NOT NULL,
    source VARCHAR(32) DEFAULT 'policy',
    UNIQUE INDEX idx_incident_user (incident_id, user_id),
    INDEX idx_ia_deleted_at (deleted_at)
)