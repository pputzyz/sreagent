CREATE TABLE IF NOT EXISTS label_registry (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    datasource_id INT UNSIGNED NOT NULL,
    label_key VARCHAR(128) NOT NULL,
    label_value VARCHAR(512) NOT NULL,
    last_seen_at DATETIME(3) NOT NULL,
    hit_count INT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uq_ds_key_val (datasource_id, label_key, label_value(256)),
    KEY idx_label_key (label_key),
    KEY idx_datasource_id (datasource_id)
);
