-- 适配所有MySQL版本（兼容5.5+/5.6+/5.7+/8.0+），解决1064语法错误
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 1. 数据库连接信息表（已移除BOOLEAN，替换为TINYINT(1)）
CREATE TABLE IF NOT EXISTS connections (
id VARCHAR(36) PRIMARY KEY COMMENT '连接唯一标识（UUID）',
name VARCHAR(255) NOT NULL COMMENT '连接名称',
host VARCHAR(255) NOT NULL COMMENT '数据库主机地址/IP',
port INT NOT NULL COMMENT '数据库端口号',
username VARCHAR(255) NOT NULL COMMENT '数据库登录用户名',
password VARCHAR(255) NOT NULL COMMENT '数据库登录密码（建议加密存储）',
database_name VARCHAR(255) NOT NULL COMMENT '远程数据库名称',
local_db_name VARCHAR(255) NOT NULL COMMENT '本地映射数据库名称',
-- 核心修正：移除BOOLEAN，使用TINYINT(1)替代，默认值0（对应FALSE）
ssl TINYINT(1) DEFAULT 0 COMMENT '是否启用SSL连接（0=否，1=是）',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
UNIQUE KEY uk_conn_name (name),
UNIQUE KEY uk_conn_local_db (local_db_name),
INDEX idx_conn_created_at (created_at),
INDEX idx_conn_host_port (host, port)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '数据库连接配置表';

-- 2. 同步配置表（已移除BOOLEAN，替换为TINYINT(1)）
CREATE TABLE IF NOT EXISTS sync_configs (
id VARCHAR(36) PRIMARY KEY COMMENT '同步配置唯一标识（UUID）',
connection_id VARCHAR(36) NOT NULL COMMENT '关联connections表的ID',
name VARCHAR(255) NOT NULL COMMENT '同步配置名称',
sync_mode ENUM('full', 'incremental') NOT NULL DEFAULT 'full' COMMENT '同步模式：全量/增量',
schedule VARCHAR(255) DEFAULT NULL COMMENT '同步调度规则（可兼容crontab表达式）',
-- 核心修正：移除BOOLEAN，使用TINYINT(1)替代，默认值1（对应TRUE）
enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用该同步配置（0=否，1=是）',
options TEXT DEFAULT NULL COMMENT '扩展配置参数（JSON格式，低版本兼容）',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE,
UNIQUE KEY uk_sync_conn_name (connection_id, name),
INDEX idx_sync_enabled (enabled),
INDEX idx_sync_created_at (created_at),
INDEX idx_sync_mode (sync_mode)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '数据同步配置表';

-- 3. 表映射配置表（已移除BOOLEAN，替换为TINYINT(1)）
CREATE TABLE IF NOT EXISTS table_mappings (
id VARCHAR(36) PRIMARY KEY COMMENT '表映射唯一标识（UUID）',
sync_config_id VARCHAR(36) NOT NULL COMMENT '关联sync_configs表的ID',
source_table VARCHAR(255) NOT NULL COMMENT '源端数据表名',
target_table VARCHAR(255) NOT NULL COMMENT '目标端数据表名',
sync_mode ENUM('full', 'incremental') NOT NULL DEFAULT 'full' COMMENT '表级同步模式：全量/增量',
-- 核心修正：移除BOOLEAN，使用TINYINT(1)替代，默认值1（对应TRUE）
enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用该表映射（0=否，1=是）',
where_clause TEXT DEFAULT NULL COMMENT '同步过滤条件（SQL WHERE子句）',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
FOREIGN KEY (sync_config_id) REFERENCES sync_configs(id) ON DELETE CASCADE,
UNIQUE KEY uk_table_map_sync_source (sync_config_id, source_table),
INDEX idx_table_map_enabled (enabled),
INDEX idx_table_map_created_at (created_at),
INDEX idx_table_map_sync_mode (sync_mode)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '数据表映射配置表';

-- 4. 同步任务记录表
CREATE TABLE IF NOT EXISTS sync_jobs (
id VARCHAR(36) PRIMARY KEY COMMENT '同步任务唯一标识（UUID）',
config_id VARCHAR(36) NOT NULL COMMENT '关联sync_configs表的ID',
status ENUM('pending', 'running', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'pending' COMMENT '任务状态：待执行/运行中/已完成/失败/已取消',
start_time TIMESTAMP NOT NULL COMMENT '任务开始时间',
end_time TIMESTAMP NULL DEFAULT NULL COMMENT '任务结束时间（未完成则为NULL）',
total_tables INT DEFAULT 0 COMMENT '本次任务需同步的表总数',
completed_tables INT DEFAULT 0 COMMENT '本次任务已完成同步的表数',
total_rows BIGINT DEFAULT 0 COMMENT '本次任务需同步的总行数',
processed_rows BIGINT DEFAULT 0 COMMENT '本次任务已处理的行数',
error_message TEXT DEFAULT NULL COMMENT '任务失败错误信息',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '任务创建时间',
FOREIGN KEY (config_id) REFERENCES sync_configs(id) ON DELETE CASCADE,
INDEX idx_job_status (status),
INDEX idx_job_start_time (start_time),
INDEX idx_job_config_id (config_id),
INDEX idx_job_created_at (created_at),
INDEX idx_job_status_start_time (status, start_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '数据同步任务执行记录表';

-- 5. 同步操作日志表
CREATE TABLE IF NOT EXISTS sync_logs (
id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '日志自增唯一标识',
job_id VARCHAR(36) NOT NULL COMMENT '关联sync_jobs表的ID',
table_name VARCHAR(255) NOT NULL COMMENT '关联的数据表名',
level ENUM('info', 'warn', 'error') NOT NULL DEFAULT 'info' COMMENT '日志级别：信息/警告/错误',
message TEXT NOT NULL COMMENT '日志详细内容',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '日志创建时间',
FOREIGN KEY (job_id) REFERENCES sync_jobs(id) ON DELETE CASCADE,
INDEX idx_log_job_id (job_id),
INDEX idx_log_level (level),
INDEX idx_log_created_at (created_at),
INDEX idx_log_table_name (table_name)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '数据同步操作明细日志表';

-- 6. 增量同步检查点表
CREATE TABLE IF NOT EXISTS sync_checkpoints (
id VARCHAR(36) PRIMARY KEY COMMENT '检查点唯一标识（UUID）',
table_mapping_id VARCHAR(36) NOT NULL COMMENT '关联table_mappings表的ID',
last_sync_time TIMESTAMP NOT NULL COMMENT '最后一次同步完成时间',
last_sync_value VARCHAR(255) DEFAULT NULL COMMENT '最后一次同步的增量字段值',
checkpoint_data TEXT DEFAULT NULL COMMENT '扩展检查点数据（JSON格式，低版本兼容）',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '检查点创建时间',
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '检查点更新时间',
FOREIGN KEY (table_mapping_id) REFERENCES table_mappings(id) ON DELETE CASCADE,
UNIQUE KEY uk_checkpoint_table_mapping (table_mapping_id),
INDEX idx_checkpoint_last_sync_time (last_sync_time),
INDEX idx_checkpoint_updated_at (updated_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '增量同步检查点状态表';

-- 7. 数据库映射关系表
CREATE TABLE IF NOT EXISTS database_mappings (
 remote_connection_id VARCHAR(36) NOT NULL COMMENT '关联connections表的远程数据库连接ID',
 local_database_name VARCHAR(255) NOT NULL COMMENT '本地数据库名称',
 created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
 PRIMARY KEY (remote_connection_id, local_database_name),
 FOREIGN KEY (remote_connection_id) REFERENCES connections(id) ON DELETE CASCADE,
 INDEX idx_db_map_local_db (local_database_name),
 INDEX idx_db_map_created_at (created_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '远程数据库→本地数据库映射关系表';

SET FOREIGN_KEY_CHECKS = 1;