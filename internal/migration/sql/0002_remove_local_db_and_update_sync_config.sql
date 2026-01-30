-- Version: 2
-- Name: remove_local_db_and_update_sync_config
-- Description: 移除连接中的local_db_name字段，修改sync_configs表支持源连接和目标连接

-- Step 1: Remove local_db_name column and index from connections table
-- First, drop the index if it exists (using a stored procedure approach)
SET @exist := (SELECT COUNT(*) FROM information_schema.statistics 
               WHERE table_schema = DATABASE() 
               AND table_name = 'connections' 
               AND index_name = 'uk_connections_local_db');
SET @sqlstmt := IF(@exist > 0, 'ALTER TABLE `connections` DROP INDEX `uk_connections_local_db`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Then, drop the column if it exists
SET @exist := (SELECT COUNT(*) FROM information_schema.columns 
               WHERE table_schema = DATABASE() 
               AND table_name = 'connections' 
               AND column_name = 'local_db_name');
SET @sqlstmt := IF(@exist > 0, 'ALTER TABLE `connections` DROP COLUMN `local_db_name`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 2: Add source_connection_id and target_connection_id to sync_configs table
-- First, check and add source_connection_id column if it doesn't exist
SET @exist := (SELECT COUNT(*) FROM information_schema.columns 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND column_name = 'source_connection_id');
SET @sqlstmt := IF(@exist = 0, 'ALTER TABLE `sync_configs` ADD COLUMN `source_connection_id` VARCHAR(36) NULL AFTER `id`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Then, check and add target_connection_id column if it doesn't exist
SET @exist := (SELECT COUNT(*) FROM information_schema.columns 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND column_name = 'target_connection_id');
SET @sqlstmt := IF(@exist = 0, 'ALTER TABLE `sync_configs` ADD COLUMN `target_connection_id` VARCHAR(36) NULL AFTER `source_connection_id`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 3: Migrate existing data (if any)
-- Copy connection_id to both source and target for existing records (only if connection_id column exists)
SET @exist := (SELECT COUNT(*) FROM information_schema.columns 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND column_name = 'connection_id');
SET @sqlstmt := IF(@exist > 0, 
    'UPDATE `sync_configs` SET `source_connection_id` = `connection_id`, `target_connection_id` = `connection_id` WHERE `source_connection_id` IS NULL', 
    'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 4: Make the new columns NOT NULL (only if they exist)
-- Check if source_connection_id exists and modify it
SET @exist := (SELECT COUNT(*) FROM information_schema.columns 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND column_name = 'source_connection_id');
SET @sqlstmt := IF(@exist > 0, 'ALTER TABLE `sync_configs` MODIFY COLUMN `source_connection_id` VARCHAR(36) NOT NULL', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Check if target_connection_id exists and modify it
SET @exist := (SELECT COUNT(*) FROM information_schema.columns 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND column_name = 'target_connection_id');
SET @sqlstmt := IF(@exist > 0, 'ALTER TABLE `sync_configs` MODIFY COLUMN `target_connection_id` VARCHAR(36) NOT NULL', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 5: Add foreign key constraints
-- First, drop existing foreign keys if they exist
SET @exist := (SELECT COUNT(*) FROM information_schema.table_constraints 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND constraint_name = 'fk_sync_configs_source_connection');
SET @sqlstmt := IF(@exist > 0, 'ALTER TABLE `sync_configs` DROP FOREIGN KEY `fk_sync_configs_source_connection`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @exist := (SELECT COUNT(*) FROM information_schema.table_constraints 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND constraint_name = 'fk_sync_configs_target_connection');
SET @sqlstmt := IF(@exist > 0, 'ALTER TABLE `sync_configs` DROP FOREIGN KEY `fk_sync_configs_target_connection`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Now add the foreign key constraints
ALTER TABLE `sync_configs`
ADD CONSTRAINT `fk_sync_configs_source_connection` 
    FOREIGN KEY (`source_connection_id`) REFERENCES `connections`(`id`) ON DELETE CASCADE,
ADD CONSTRAINT `fk_sync_configs_target_connection` 
    FOREIGN KEY (`target_connection_id`) REFERENCES `connections`(`id`) ON DELETE CASCADE;

-- Step 6: Update unique constraint
-- If an old FK exists on `connection_id` (auto-named), drop it before dropping the old unique index.
-- Otherwise MySQL can refuse to drop the index if it is needed by the FK.
SET @fk_name := (
    SELECT kcu.constraint_name
    FROM information_schema.key_column_usage kcu
    WHERE kcu.table_schema = DATABASE()
      AND kcu.table_name = 'sync_configs'
      AND kcu.column_name = 'connection_id'
      AND kcu.referenced_table_name = 'connections'
    LIMIT 1
);
SET @sqlstmt := IF(@fk_name IS NOT NULL AND @fk_name <> 'PRIMARY',
    CONCAT('ALTER TABLE `sync_configs` DROP FOREIGN KEY `', @fk_name, '`'),
    'SELECT 1'
);
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Drop old index if exists
SET @exist := (SELECT COUNT(*) FROM information_schema.statistics 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND index_name = 'uk_sync_configs_connection_name');
SET @sqlstmt := IF(@exist > 0, 'ALTER TABLE `sync_configs` DROP INDEX `uk_sync_configs_connection_name`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add new unique constraint (check if it doesn't exist first)
SET @exist := (SELECT COUNT(*) FROM information_schema.statistics 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND index_name = 'uk_sync_configs_source_name');
SET @sqlstmt := IF(@exist = 0, 'ALTER TABLE `sync_configs` ADD UNIQUE KEY `uk_sync_configs_source_name` (`source_connection_id`, `name`)', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 7: Add indexes for better query performance
-- Check and add idx_sync_configs_source_connection if it doesn't exist
SET @exist := (SELECT COUNT(*) FROM information_schema.statistics 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND index_name = 'idx_sync_configs_source_connection');
SET @sqlstmt := IF(@exist = 0, 'ALTER TABLE `sync_configs` ADD INDEX `idx_sync_configs_source_connection` (`source_connection_id`)', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Check and add idx_sync_configs_target_connection if it doesn't exist
SET @exist := (SELECT COUNT(*) FROM information_schema.statistics 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND index_name = 'idx_sync_configs_target_connection');
SET @sqlstmt := IF(@exist = 0, 'ALTER TABLE `sync_configs` ADD INDEX `idx_sync_configs_target_connection` (`target_connection_id`)', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 8: Remove old connection_id column if it exists
SET @exist := (SELECT COUNT(*) FROM information_schema.columns 
               WHERE table_schema = DATABASE() 
               AND table_name = 'sync_configs' 
               AND column_name = 'connection_id');
SET @sqlstmt := IF(@exist > 0, 'ALTER TABLE `sync_configs` DROP COLUMN `connection_id`', 'SELECT 1');
PREPARE stmt FROM @sqlstmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
