-- Version: 3
-- Name: sync_config_databases
-- Description: 在 sync_configs 中增加源/目标数据库字段，用于在同步配置中选择数据库名称

-- Add source_database column if it doesn't exist
SET @exist := (SELECT COUNT(*) FROM information_schema.columns
               WHERE table_schema = DATABASE()
                 AND table_name = 'sync_configs'
                 AND column_name = 'source_database');
SET @sqlstmt := IF(@exist = 0, 'ALTER TABLE `sync_configs` ADD COLUMN `source_database` VARCHAR(255) NULL AFTER `target_connection_id`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add target_database column if it doesn't exist
SET @exist := (SELECT COUNT(*) FROM information_schema.columns
               WHERE table_schema = DATABASE()
                 AND table_name = 'sync_configs'
                 AND column_name = 'target_database');
SET @sqlstmt := IF(@exist = 0, 'ALTER TABLE `sync_configs` ADD COLUMN `target_database` VARCHAR(255) NULL AFTER `source_database`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

