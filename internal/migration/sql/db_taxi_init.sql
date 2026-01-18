-- Version: 1
-- Name: create_sync_tables
-- Description: 创建同步系统所需的数据库表
-- 适配所有MySQL版本（5.5+/5.6+/5.7+/8.0+），解决BOOLEAN/JSON语法兼容问题
-- Create connections table
CREATE TABLE `connections` (
`id` VARCHAR(36) NOT NULL,
`name` VARCHAR(255) NOT NULL,
`host` VARCHAR(255) NOT NULL,
`port` INT NOT NULL,
`username` VARCHAR(255) NOT NULL,
`password` VARCHAR(255) NOT NULL,
`database_name` VARCHAR(255) NOT NULL,
`local_db_name` VARCHAR(255) NOT NULL,
`ssl` TINYINT(1) DEFAULT 0,
`created_at` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
`updated_at` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
PRIMARY KEY (`id`),
UNIQUE KEY `uk_connections_name` (`name`),
UNIQUE KEY `uk_connections_local_db` (`local_db_name`),
KEY `idx_connections_created_at` (`created_at`),
KEY `idx_connections_host_port` (`host`, `port`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create sync_configs table
CREATE TABLE IF NOT EXISTS `sync_configs` (
`id` VARCHAR(36) PRIMARY KEY,
`connection_id` VARCHAR(36) NOT NULL,
`name` VARCHAR(255) NOT NULL,
`sync_mode` ENUM('full', 'incremental') NOT NULL DEFAULT 'full',
`schedule` VARCHAR(255),
`enabled` TINYINT(1) DEFAULT 1,
`options` TEXT,
`created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
`updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (`connection_id`) REFERENCES `connections`(`id`) ON DELETE CASCADE,
UNIQUE KEY `uk_sync_configs_connection_name` (`connection_id`, `name`),
INDEX `idx_sync_configs_enabled` (`enabled`),
INDEX `idx_sync_configs_created_at` (`created_at`),
INDEX `idx_sync_configs_sync_mode` (`sync_mode`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create table_mappings table
CREATE TABLE IF NOT EXISTS `table_mappings` (
`id` VARCHAR(36) PRIMARY KEY,
`sync_config_id` VARCHAR(36) NOT NULL,
`source_table` VARCHAR(255) NOT NULL,
`target_table` VARCHAR(255) NOT NULL,
`sync_mode` ENUM('full', 'incremental') NOT NULL DEFAULT 'full',
`enabled` TINYINT(1) DEFAULT 1,
`where_clause` TEXT,
`created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
`updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (`sync_config_id`) REFERENCES `sync_configs`(`id`) ON DELETE CASCADE,
UNIQUE KEY `uk_table_mappings_sync_source` (`sync_config_id`, `source_table`),
INDEX `idx_table_mappings_enabled` (`enabled`),
INDEX `idx_table_mappings_created_at` (`created_at`),
INDEX `idx_table_mappings_sync_mode` (`sync_mode`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create sync_jobs table
CREATE TABLE IF NOT EXISTS `sync_jobs` (
`id` VARCHAR(36) PRIMARY KEY,
`config_id` VARCHAR(36) NOT NULL,
`status` ENUM('pending', 'running', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
`start_time` TIMESTAMP NOT NULL,
`end_time` TIMESTAMP NULL,
`total_tables` INT DEFAULT 0,
`completed_tables` INT DEFAULT 0,
`total_rows` BIGINT DEFAULT 0,
`processed_rows` BIGINT DEFAULT 0,
`error_message` TEXT,
`created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (`config_id`) REFERENCES `sync_configs`(`id`) ON DELETE CASCADE,
INDEX `idx_sync_jobs_status` (`status`),
INDEX `idx_sync_jobs_start_time` (`start_time`),
INDEX `idx_sync_jobs_config_id` (`config_id`),
INDEX `idx_sync_jobs_created_at` (`created_at`),
INDEX `idx_sync_jobs_status_start_time` (`status`, `start_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create sync_logs table
CREATE TABLE IF NOT EXISTS `sync_logs` (
`id` BIGINT AUTO_INCREMENT PRIMARY KEY,
`job_id` VARCHAR(36) NOT NULL,
`table_name` VARCHAR(255) NOT NULL,
`level` ENUM('info', 'warn', 'error') NOT NULL DEFAULT 'info',
`message` TEXT NOT NULL,
`created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (`job_id`) REFERENCES `sync_jobs`(`id`) ON DELETE CASCADE,
INDEX `idx_sync_logs_job_id` (`job_id`),
INDEX `idx_sync_logs_level` (`level`),
INDEX `idx_sync_logs_created_at` (`created_at`),
INDEX `idx_sync_logs_table_name` (`table_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create sync_checkpoints table
CREATE TABLE IF NOT EXISTS `sync_checkpoints` (
`id` VARCHAR(36) PRIMARY KEY,
`table_mapping_id` VARCHAR(36) NOT NULL,
`last_sync_time` TIMESTAMP NOT NULL,
`last_sync_value` VARCHAR(255),
`checkpoint_data` TEXT,
`created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
`updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (`table_mapping_id`) REFERENCES `table_mappings`(`id`) ON DELETE CASCADE,
UNIQUE KEY `uk_sync_checkpoints_table_mapping` (`table_mapping_id`),
INDEX `idx_sync_checkpoints_last_sync_time` (`last_sync_time`),
INDEX `idx_sync_checkpoints_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create database_mappings table (FIXED: added missing backticks!)
CREATE TABLE IF NOT EXISTS `database_mappings` (
`remote_connection_id` VARCHAR(36) NOT NULL,
`local_database_name` VARCHAR(255) NOT NULL,
`created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (`remote_connection_id`, `local_database_name`),  -- ✅ both columns quoted
FOREIGN KEY (`remote_connection_id`) REFERENCES `connections`(`id`) ON DELETE CASCADE,
INDEX `idx_database_mappings_local_db` (`local_database_name`),
INDEX `idx_database_mappings_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;