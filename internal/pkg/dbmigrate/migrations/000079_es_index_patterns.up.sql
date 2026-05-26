CREATE TABLE IF NOT EXISTS es_index_patterns (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    datasource_id BIGINT NOT NULL DEFAULT 0,
    name VARCHAR(191) NOT NULL DEFAULT '',
    time_field VARCHAR(128) NOT NULL DEFAULT '@timestamp',
    allow_hide_system_indices TINYINT(1) NOT NULL DEFAULT 0,
    fields_format TEXT,
    cross_cluster_enabled TINYINT(1) NOT NULL DEFAULT 0,
    note VARCHAR(512) DEFAULT '',
    created_by VARCHAR(64) DEFAULT '',
    updated_by VARCHAR(64) DEFAULT '',
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    UNIQUE INDEX idx_ds_name (datasource_id, name),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
