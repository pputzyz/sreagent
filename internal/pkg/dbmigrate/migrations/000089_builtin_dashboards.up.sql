CREATE TABLE IF NOT EXISTS builtin_dashboards (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(256) NOT NULL DEFAULT '',
    ident VARCHAR(128) NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT '',
    component VARCHAR(64) NOT NULL DEFAULT '',
    tags VARCHAR(512) DEFAULT '',
    config LONGTEXT,
    version INT NOT NULL DEFAULT 1,
    built_in TINYINT(1) NOT NULL DEFAULT 1,
    create_by VARCHAR(64) DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE INDEX idx_ident (ident),
    INDEX idx_category (category),
    INDEX idx_component (component),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
