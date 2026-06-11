CREATE TABLE IF NOT EXISTS lark_card_entities (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    event_id        BIGINT UNSIGNED DEFAULT NULL,
    card_id         VARCHAR(128) NOT NULL,
    sequence        BIGINT NOT NULL DEFAULT 0,
    card_status     VARCHAR(32) NOT NULL DEFAULT 'active',
    expires_at      DATETIME(3) NOT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_event_card (event_id, card_id),
    INDEX idx_expires (expires_at),
    INDEX idx_event_id (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lark_card_messages (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    card_entity_id  BIGINT UNSIGNED NOT NULL,
    chat_id         VARCHAR(128) NOT NULL,
    message_id      VARCHAR(128) NOT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_entity_id (card_entity_id),
    INDEX idx_chat_id (chat_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
