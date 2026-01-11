-- Version: 2
-- Name: add_initial_data
-- Description: Adds initial configuration data and optimizes indexes for the sync system

-- Add default sync options template (stored as JSON for reference)
-- This can be used by the application to provide default values

-- Create additional composite indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_sync_jobs_config_status ON sync_jobs(config_id, status);
CREATE INDEX IF NOT EXISTS idx_sync_logs_job_level ON sync_logs(job_id, level);
CREATE INDEX IF NOT EXISTS idx_table_mappings_config_enabled ON table_mappings(sync_config_id, enabled);

-- Add performance optimization indexes for common queries
CREATE INDEX IF NOT EXISTS idx_connections_updated_at ON connections(updated_at);
CREATE INDEX IF NOT EXISTS idx_sync_configs_updated_at ON sync_configs(updated_at);
CREATE INDEX IF NOT EXISTS idx_table_mappings_updated_at ON table_mappings(updated_at);

-- Create index for sync job history queries (most recent first)
CREATE INDEX IF NOT EXISTS idx_sync_jobs_created_desc ON sync_jobs(created_at DESC);

-- Create index for active sync jobs monitoring
CREATE INDEX IF NOT EXISTS idx_sync_jobs_active ON sync_jobs(status, start_time) 
    WHERE status IN ('pending', 'running');
