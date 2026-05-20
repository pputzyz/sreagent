CREATE TABLE IF NOT EXISTS `ai_conversations` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `title` VARCHAR(255) NOT NULL DEFAULT '',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active',
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    INDEX `idx_ai_conv_user` (`user_id`),
    INDEX `idx_ai_conv_deleted` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ai_tool_calls` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `conversation_id` BIGINT UNSIGNED NOT NULL,
    `step_index` INT NOT NULL DEFAULT 0,
    `tool_name` VARCHAR(100) NOT NULL,
    `parameters` JSON DEFAULT NULL,
    `result` TEXT DEFAULT NULL,
    `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
    `duration_ms` BIGINT DEFAULT 0,
    `error` TEXT DEFAULT NULL,
    `created_at` DATETIME(3) NOT NULL,
    INDEX `idx_ai_call_conv` (`conversation_id`),
    CONSTRAINT `fk_ai_call_conv` FOREIGN KEY (`conversation_id`) REFERENCES `ai_conversations`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
