CREATE TABLE IF NOT EXISTS dispatch_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    incident_id BIGINT UNSIGNED NOT NULL,
    dispatch_policy_id BIGINT UNSIGNED DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    attempt INT DEFAULT 1,
    next_attempt_at BIGINT NULL,
    note TEXT,
    INDEX idx_dl_incident_id (incident_id),
    INDEX idx_dl_dispatch_policy_id (dispatch_policy_id),
    INDEX idx_dl_deleted_at (deleted_at)
)