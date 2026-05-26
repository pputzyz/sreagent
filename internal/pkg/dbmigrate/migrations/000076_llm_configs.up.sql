CREATE TABLE IF NOT EXISTS llm_configs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(128)    NOT NULL,
    provider    VARCHAR(32)     NOT NULL,
    api_url     VARCHAR(512)    NOT NULL DEFAULT '',
    api_key     VARCHAR(512)    NOT NULL DEFAULT '',
    model       VARCHAR(128)    NOT NULL DEFAULT '',
    extra_config TEXT           NULL,
    enabled     TINYINT(1)      NOT NULL DEFAULT 1,
    is_default  TINYINT(1)      NOT NULL DEFAULT 0,
    description VARCHAR(512)    NOT NULL DEFAULT '',
    created_by  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_by  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at  DATETIME(3)     NULL,
    UNIQUE INDEX idx_llm_configs_name (name),
    INDEX idx_llm_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
