CREATE TABLE IF NOT EXISTS alert_events_v2 (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    alert_id BIGINT UNSIGNED NOT NULL,
    event_status VARCHAR(32) NOT NULL,
    event_severity VARCHAR(32) NOT NULL,
    labels JSON,
    annotations JSON,
    value DOUBLE DEFAULT 0,
    timestamp DATETIME(3) NOT NULL,
    fingerprint VARCHAR(64) DEFAULT '',
    INDEX idx_aev2_alert_id (alert_id),
    INDEX idx_aev2_event_status (event_status),
    INDEX idx_aev2_timestamp (timestamp),
    INDEX idx_aev2_fingerprint (fingerprint),
    INDEX idx_aev2_deleted_at (deleted_at)
)