CREATE TABLE IF NOT EXISTS incident_timelines (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    incident_id BIGINT UNSIGNED NOT NULL,
    action VARCHAR(32) NOT NULL,
    actor_id BIGINT UNSIGNED NULL,
    content TEXT,
    extra JSON,
    INDEX idx_it_incident_id (incident_id),
    INDEX idx_it_actor_id (actor_id),
    INDEX idx_it_deleted_at (deleted_at)
)