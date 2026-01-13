-- Version: 2
-- Name: add_initial_data
-- Description: Adds initial configuration data and optimizes indexes for the sync system

-- 极简兼容版：仅创建核心索引，无复杂语法，适配所有极低版本MySQL
# ALTER TABLE `sync_jobs` ADD INDEX idx_sync_jobs_config_status (config_id, status);
# ALTER TABLE `sync_logs` ADD INDEX idx_sync_logs_job_level (job_id, `level`);
# ALTER TABLE `table_mappings` ADD INDEX idx_table_mappings_config_enabled (sync_config_id, enabled);
# ALTER TABLE `sync_jobs` ADD INDEX idx_sync_jobs_created_desc (created_at DESC);
# ALTER TABLE `sync_jobs` ADD INDEX idx_sync_jobs_active (status, start_time);