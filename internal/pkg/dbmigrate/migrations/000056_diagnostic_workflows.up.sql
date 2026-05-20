CREATE TABLE IF NOT EXISTS `diagnostic_workflows` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(255) NOT NULL,
    `description` TEXT DEFAULT NULL,
    `trigger_labels` JSON DEFAULT NULL,
    `trigger_severity` VARCHAR(20) DEFAULT NULL,
    `category` VARCHAR(50) NOT NULL DEFAULT 'general',
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `max_steps` INT NOT NULL DEFAULT 10,
    `require_approval` TINYINT(1) NOT NULL DEFAULT 1,
    `created_by` BIGINT UNSIGNED DEFAULT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    INDEX `idx_dw_enabled` (`enabled`),
    INDEX `idx_dw_category` (`category`),
    INDEX `idx_dw_deleted` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `diagnostic_workflow_steps` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `workflow_id` BIGINT UNSIGNED NOT NULL,
    `step_order` INT NOT NULL DEFAULT 0,
    `name` VARCHAR(255) NOT NULL,
    `step_type` VARCHAR(20) NOT NULL DEFAULT 'query',
    `datasource_id` BIGINT UNSIGNED DEFAULT NULL,
    `expression` TEXT DEFAULT NULL,
    `condition_expr` VARCHAR(500) DEFAULT NULL,
    `auto_advance` TINYINT(1) NOT NULL DEFAULT 1,
    `timeout_seconds` INT NOT NULL DEFAULT 30,
    `on_failure` VARCHAR(20) NOT NULL DEFAULT 'continue',
    `created_at` DATETIME(3) NOT NULL,
    INDEX `idx_dws_workflow` (`workflow_id`),
    CONSTRAINT `fk_dws_workflow` FOREIGN KEY (`workflow_id`) REFERENCES `diagnostic_workflows`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `diagnostic_runs` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `workflow_id` BIGINT UNSIGNED NOT NULL,
    `incident_id` BIGINT UNSIGNED DEFAULT NULL,
    `user_id` BIGINT UNSIGNED DEFAULT NULL,
    `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
    `current_step` INT NOT NULL DEFAULT 0,
    `result_summary` TEXT DEFAULT NULL,
    `started_at` DATETIME(3) DEFAULT NULL,
    `completed_at` DATETIME(3) DEFAULT NULL,
    `created_at` DATETIME(3) NOT NULL,
    INDEX `idx_dr_workflow` (`workflow_id`),
    INDEX `idx_dr_incident` (`incident_id`),
    INDEX `idx_dr_status` (`status`),
    CONSTRAINT `fk_dr_workflow` FOREIGN KEY (`workflow_id`) REFERENCES `diagnostic_workflows`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `diagnostic_run_steps` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `run_id` BIGINT UNSIGNED NOT NULL,
    `step_order` INT NOT NULL DEFAULT 0,
    `step_name` VARCHAR(255) NOT NULL,
    `step_type` VARCHAR(20) NOT NULL,
    `expression` TEXT DEFAULT NULL,
    `result` TEXT DEFAULT NULL,
    `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
    `duration_ms` BIGINT DEFAULT 0,
    `error` TEXT DEFAULT NULL,
    `started_at` DATETIME(3) DEFAULT NULL,
    `completed_at` DATETIME(3) DEFAULT NULL,
    `created_at` DATETIME(3) NOT NULL,
    INDEX `idx_drs_run` (`run_id`),
    CONSTRAINT `fk_drs_run` FOREIGN KEY (`run_id`) REFERENCES `diagnostic_runs`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
