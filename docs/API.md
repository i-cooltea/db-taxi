# DB-Taxi API 文档

## 概述

DB-Taxi 提供 RESTful API 接口，用于数据库管理和同步功能。所有 API 端点都返回 JSON 格式的响应。

## 基础信息

- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`
- **字符编码**: UTF-8

## 通用响应格式

### 成功响应
```json
{
  "success": true,
  "data": { ... },
  "message": "操作成功"
}
```

### 错误响应
```json
{
  "success": false,
  "error": "错误描述",
  "code": "ERROR_CODE"
}
```

## API 端点

### 1. 健康检查和状态

#### 1.1 健康检查
检查服务器是否正常运行。

**请求**
```
GET /health
```

**响应**
```json
{
  "status": "ok",
  "timestamp": "2024-01-11T10:00:00Z"
}
```

#### 1.2 获取服务器状态
获取服务器和数据库的详细状态信息。

**请求**
```
GET /api/status
```

**响应**
```json
{
  "server": {
    "version": "1.0.0",
    "uptime": 3600,
    "go_version": "go1.21.0"
  },
  "database": {
    "connected": true,
    "version": "8.0.32",
    "max_connections": 100,
    "active_connections": 5
  }
}
```

#### 1.3 测试数据库连接
测试当前配置的数据库连接是否可用。

**请求**
```
GET /api/connection/test
```

**响应**
```json
{
  "success": true,
  "connected": true,
  "latency_ms": 15,
  "database_version": "8.0.32"
}
```

---

### 2. 数据库操作

#### 2.1 获取数据库列表
获取所有可用的数据库列表。

**请求**
```
GET /api/databases
```

**响应**
```json
{
  "databases": [
    {
      "name": "myapp",
      "charset": "utf8mb4",
      "collation": "utf8mb4_unicode_ci",
      "table_count": 15
    },
    {
      "name": "testdb",
      "charset": "utf8mb4",
      "collation": "utf8mb4_unicode_ci",
      "table_count": 8
    }
  ]
}
```

#### 2.2 获取表列表
获取指定数据库中的所有表。

**请求**
```
GET /api/databases/{database}/tables
```

**路径参数**
- `database` (string, required): 数据库名称

**响应**
```json
{
  "tables": [
    {
      "name": "users",
      "engine": "InnoDB",
      "rows": 1250,
      "data_length": 524288,
      "index_length": 131072,
      "collation": "utf8mb4_unicode_ci",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### 2.3 获取表详情
获取表的详细信息，包括列、索引、约束等。

**请求**
```
GET /api/databases/{database}/tables/{table}
```

**路径参数**
- `database` (string, required): 数据库名称
- `table` (string, required): 表名称

**响应**
```json
{
  "name": "users",
  "engine": "InnoDB",
  "charset": "utf8mb4",
  "collation": "utf8mb4_unicode_ci",
  "columns": [
    {
      "name": "id",
      "type": "bigint",
      "nullable": false,
      "key": "PRI",
      "default": null,
      "extra": "auto_increment"
    },
    {
      "name": "email",
      "type": "varchar(255)",
      "nullable": false,
      "key": "UNI",
      "default": null,
      "extra": ""
    }
  ],
  "indexes": [
    {
      "name": "PRIMARY",
      "columns": ["id"],
      "unique": true,
      "type": "BTREE"
    }
  ],
  "constraints": [
    {
      "name": "fk_user_role",
      "type": "FOREIGN KEY",
      "columns": ["role_id"],
      "referenced_table": "roles",
      "referenced_columns": ["id"]
    }
  ]
}
```

#### 2.4 获取表数据
获取表中的数据，支持分页。

**请求**
```
GET /api/databases/{database}/tables/{table}/data?limit=10&offset=0
```

**路径参数**
- `database` (string, required): 数据库名称
- `table` (string, required): 表名称

**查询参数**
- `limit` (integer, optional): 每页记录数，默认 10，最大 1000
- `offset` (integer, optional): 偏移量，默认 0

**响应**
```json
{
  "columns": ["id", "email", "name", "created_at"],
  "rows": [
    [1, "user@example.com", "John Doe", "2024-01-01T00:00:00Z"],
    [2, "admin@example.com", "Admin User", "2024-01-02T00:00:00Z"]
  ],
  "total": 1250,
  "limit": 10,
  "offset": 0
}
```

---

### 3. 同步系统 - 连接管理

#### 3.1 获取所有同步连接
获取所有配置的远程数据库连接。

**请求**
```
GET /api/sync/connections
```

**响应**
```json
{
  "connections": [
    {
      "id": "conn-123",
      "name": "Production DB",
      "host": "prod-mysql.example.com",
      "port": 3306,
      "username": "sync_user",
      "database": "production",
      "local_db_name": "prod_sync",
      "ssl": true,
      "status": {
        "connected": true,
        "last_check": "2024-01-11T10:00:00Z",
        "latency_ms": 25
      },
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-11T09:00:00Z"
    }
  ]
}
```

#### 3.2 创建同步连接
创建新的远程数据库连接配置。

**请求**
```
POST /api/sync/connections
Content-Type: application/json

{
  "name": "Staging DB",
  "host": "staging-mysql.example.com",
  "port": 3306,
  "username": "sync_user",
  "password": "secure_password",
  "database": "staging",
  "local_db_name": "staging_sync",
  "ssl": true
}
```

**响应**
```json
{
  "success": true,
  "connection": {
    "id": "conn-456",
    "name": "Staging DB",
    "host": "staging-mysql.example.com",
    "port": 3306,
    "username": "sync_user",
    "database": "staging",
    "local_db_name": "staging_sync",
    "ssl": true,
    "created_at": "2024-01-11T10:00:00Z"
  }
}
```

#### 3.3 获取连接详情
获取指定连接的详细信息。

**请求**
```
GET /api/sync/connections/{id}
```

**路径参数**
- `id` (string, required): 连接 ID

**响应**
```json
{
  "id": "conn-123",
  "name": "Production DB",
  "host": "prod-mysql.example.com",
  "port": 3306,
  "username": "sync_user",
  "database": "production",
  "local_db_name": "prod_sync",
  "ssl": true,
  "status": {
    "connected": true,
    "last_check": "2024-01-11T10:00:00Z",
    "latency_ms": 25
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-11T09:00:00Z"
}
```

#### 3.4 更新连接配置
更新现有连接的配置信息。

**请求**
```
PUT /api/sync/connections/{id}
Content-Type: application/json

{
  "name": "Production DB Updated",
  "host": "new-prod-mysql.example.com",
  "port": 3306,
  "username": "sync_user",
  "password": "new_password",
  "database": "production",
  "local_db_name": "prod_sync",
  "ssl": true
}
```

**响应**
```json
{
  "success": true,
  "message": "连接配置已更新"
}
```

#### 3.5 删除连接
删除指定的连接配置。

**请求**
```
DELETE /api/sync/connections/{id}
```

**路径参数**
- `id` (string, required): 连接 ID

**响应**
```json
{
  "success": true,
  "message": "连接已删除"
}
```

#### 3.6 测试连接
测试指定连接的可用性。

**请求**
```
POST /api/sync/connections/{id}/test
```

**路径参数**
- `id` (string, required): 连接 ID

**响应**
```json
{
  "success": true,
  "connected": true,
  "latency_ms": 25,
  "database_version": "8.0.32",
  "message": "连接测试成功"
}
```

---

### 4. 同步系统 - 同步配置

#### 4.1 获取同步配置列表
获取所有同步配置。

**请求**
```
GET /api/sync/configs?connection_id=conn-123
```

**查询参数**
- `connection_id` (string, optional): 筛选指定连接的配置

**响应**
```json
{
  "configs": [
    {
      "id": "config-789",
      "connection_id": "conn-123",
      "name": "User Tables Sync",
      "sync_mode": "incremental",
      "schedule": "0 */6 * * *",
      "enabled": true,
      "tables": [
        {
          "source_table": "users",
          "target_table": "users",
          "sync_mode": "incremental",
          "enabled": true
        }
      ],
      "options": {
        "batch_size": 1000,
        "max_concurrency": 5,
        "enable_compression": true,
        "conflict_resolution": "overwrite"
      },
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-11T09:00:00Z"
    }
  ]
}
```

#### 4.2 创建同步配置
创建新的同步配置。

**请求**
```
POST /api/sync/configs
Content-Type: application/json

{
  "connection_id": "conn-123",
  "name": "Orders Sync",
  "sync_mode": "full",
  "schedule": "0 2 * * *",
  "enabled": true,
  "tables": [
    {
      "source_table": "orders",
      "target_table": "orders",
      "sync_mode": "full",
      "enabled": true,
      "where_clause": "created_at > DATE_SUB(NOW(), INTERVAL 30 DAY)"
    }
  ],
  "options": {
    "batch_size": 500,
    "max_concurrency": 3,
    "enable_compression": true,
    "conflict_resolution": "skip"
  }
}
```

**响应**
```json
{
  "success": true,
  "config": {
    "id": "config-101",
    "connection_id": "conn-123",
    "name": "Orders Sync",
    "created_at": "2024-01-11T10:00:00Z"
  }
}
```

#### 4.3 获取配置详情
获取指定同步配置的详细信息。

**请求**
```
GET /api/sync/configs/{id}
```

**路径参数**
- `id` (string, required): 配置 ID

**响应**
```json
{
  "id": "config-789",
  "connection_id": "conn-123",
  "name": "User Tables Sync",
  "sync_mode": "incremental",
  "schedule": "0 */6 * * *",
  "enabled": true,
  "tables": [
    {
      "source_table": "users",
      "target_table": "users",
      "sync_mode": "incremental",
      "enabled": true
    }
  ],
  "options": {
    "batch_size": 1000,
    "max_concurrency": 5,
    "enable_compression": true,
    "conflict_resolution": "overwrite"
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-11T09:00:00Z"
}
```

#### 4.4 更新同步配置
更新现有的同步配置。

**请求**
```
PUT /api/sync/configs/{id}
Content-Type: application/json

{
  "name": "User Tables Sync Updated",
  "sync_mode": "incremental",
  "schedule": "0 */4 * * *",
  "enabled": true,
  "tables": [
    {
      "source_table": "users",
      "target_table": "users",
      "sync_mode": "incremental",
      "enabled": true
    },
    {
      "source_table": "user_profiles",
      "target_table": "user_profiles",
      "sync_mode": "incremental",
      "enabled": true
    }
  ],
  "options": {
    "batch_size": 2000,
    "max_concurrency": 5,
    "enable_compression": true,
    "conflict_resolution": "overwrite"
  }
}
```

**响应**
```json
{
  "success": true,
  "message": "同步配置已更新"
}
```

#### 4.5 删除同步配置
删除指定的同步配置。

**请求**
```
DELETE /api/sync/configs/{id}
```

**路径参数**
- `id` (string, required): 配置 ID

**响应**
```json
{
  "success": true,
  "message": "同步配置已删除"
}
```

---

### 5. 同步系统 - 任务管理

#### 5.1 获取同步任务列表
获取所有同步任务的列表。

**请求**
```
GET /api/sync/jobs?status=running&limit=20&offset=0
```

**查询参数**
- `status` (string, optional): 筛选任务状态 (pending, running, completed, failed, cancelled)
- `config_id` (string, optional): 筛选指定配置的任务
- `limit` (integer, optional): 每页记录数，默认 20
- `offset` (integer, optional): 偏移量，默认 0

**响应**
```json
{
  "jobs": [
    {
      "id": "job-555",
      "config_id": "config-789",
      "config_name": "User Tables Sync",
      "status": "running",
      "progress": {
        "total_tables": 2,
        "completed_tables": 1,
        "total_rows": 10000,
        "processed_rows": 5500,
        "percentage": 55.0
      },
      "start_time": "2024-01-11T10:00:00Z",
      "end_time": null,
      "error": null
    }
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

#### 5.2 启动同步任务
启动新的同步任务。

**请求**
```
POST /api/sync/jobs
Content-Type: application/json

{
  "config_id": "config-789"
}
```

**响应**
```json
{
  "success": true,
  "job": {
    "id": "job-666",
    "config_id": "config-789",
    "status": "pending",
    "start_time": "2024-01-11T10:00:00Z"
  }
}
```

#### 5.3 获取任务详情
获取指定任务的详细信息。

**请求**
```
GET /api/sync/jobs/{id}
```

**路径参数**
- `id` (string, required): 任务 ID

**响应**
```json
{
  "id": "job-555",
  "config_id": "config-789",
  "config_name": "User Tables Sync",
  "status": "running",
  "progress": {
    "total_tables": 2,
    "completed_tables": 1,
    "total_rows": 10000,
    "processed_rows": 5500,
    "percentage": 55.0,
    "current_table": "user_profiles",
    "estimated_time_remaining": 300
  },
  "start_time": "2024-01-11T10:00:00Z",
  "end_time": null,
  "error": null,
  "tables": [
    {
      "name": "users",
      "status": "completed",
      "rows_processed": 5000,
      "duration_seconds": 120
    },
    {
      "name": "user_profiles",
      "status": "running",
      "rows_processed": 500,
      "duration_seconds": 30
    }
  ]
}
```

#### 5.4 停止同步任务
停止正在运行的同步任务。

**请求**
```
POST /api/sync/jobs/{id}/stop
```

**路径参数**
- `id` (string, required): 任务 ID

**响应**
```json
{
  "success": true,
  "message": "任务已停止"
}
```

#### 5.5 获取任务日志
获取任务的执行日志。

**请求**
```
GET /api/sync/jobs/{id}/logs?level=error&limit=100
```

**路径参数**
- `id` (string, required): 任务 ID

**查询参数**
- `level` (string, optional): 日志级别 (info, warn, error)
- `limit` (integer, optional): 返回日志条数，默认 100

**响应**
```json
{
  "logs": [
    {
      "id": 1001,
      "job_id": "job-555",
      "table_name": "users",
      "level": "info",
      "message": "开始同步表 users",
      "created_at": "2024-01-11T10:00:00Z"
    },
    {
      "id": 1002,
      "job_id": "job-555",
      "table_name": "users",
      "level": "info",
      "message": "已处理 1000 行数据",
      "created_at": "2024-01-11T10:00:30Z"
    },
    {
      "id": 1003,
      "job_id": "job-555",
      "table_name": "users",
      "level": "error",
      "message": "数据类型转换错误: column 'age' expected int, got string",
      "created_at": "2024-01-11T10:01:00Z"
    }
  ]
}
```

---

### 6. 同步系统 - 配置管理

#### 6.1 导出同步配置
导出所有同步配置为 JSON 文件。

**请求**
```
GET /api/sync/config/export
```

**响应**
```json
{
  "version": "1.0",
  "export_time": "2024-01-11T10:00:00Z",
  "connections": [
    {
      "id": "conn-123",
      "name": "Production DB",
      "host": "prod-mysql.example.com",
      "port": 3306,
      "username": "sync_user",
      "database": "production",
      "local_db_name": "prod_sync",
      "ssl": true
    }
  ],
  "mappings": [
    {
      "remote_connection_id": "conn-123",
      "local_database_name": "prod_sync"
    }
  ],
  "sync_configs": [
    {
      "id": "config-789",
      "connection_id": "conn-123",
      "name": "User Tables Sync",
      "sync_mode": "incremental",
      "schedule": "0 */6 * * *",
      "enabled": true,
      "tables": [
        {
          "source_table": "users",
          "target_table": "users",
          "sync_mode": "incremental",
          "enabled": true
        }
      ],
      "options": {
        "batch_size": 1000,
        "max_concurrency": 5,
        "enable_compression": true,
        "conflict_resolution": "overwrite"
      }
    }
  ]
}
```

#### 6.2 导入同步配置
从 JSON 文件导入同步配置。

**请求**
```
POST /api/sync/config/import
Content-Type: application/json

{
  "version": "1.0",
  "connections": [...],
  "mappings": [...],
  "sync_configs": [...]
}
```

**响应**
```json
{
  "success": true,
  "imported": {
    "connections": 2,
    "mappings": 2,
    "sync_configs": 5
  },
  "conflicts": [
    {
      "type": "connection",
      "name": "Production DB",
      "action": "skipped",
      "reason": "连接名称已存在"
    }
  ]
}
```

#### 6.3 验证配置文件
验证配置文件的格式和内容。

**请求**
```
POST /api/sync/config/validate
Content-Type: application/json

{
  "version": "1.0",
  "connections": [...],
  "mappings": [...],
  "sync_configs": [...]
}
```

**响应**
```json
{
  "valid": true,
  "errors": [],
  "warnings": [
    {
      "field": "connections[0].password",
      "message": "密码未加密，建议使用加密存储"
    }
  ]
}
```

---

### 7. 同步系统 - 监控和统计

#### 7.1 获取同步系统状态
获取同步系统的整体状态。

**请求**
```
GET /api/sync/status
```

**响应**
```json
{
  "enabled": true,
  "active_jobs": 3,
  "pending_jobs": 5,
  "total_connections": 10,
  "healthy_connections": 9,
  "total_configs": 25,
  "enabled_configs": 20,
  "system_resources": {
    "cpu_usage": 45.5,
    "memory_usage": 62.3,
    "disk_usage": 78.1
  }
}
```

#### 7.2 获取同步统计信息
获取同步系统的统计数据。

**请求**
```
GET /api/sync/stats?period=24h
```

**查询参数**
- `period` (string, optional): 统计周期 (1h, 24h, 7d, 30d)，默认 24h

**响应**
```json
{
  "period": "24h",
  "total_jobs": 150,
  "completed_jobs": 140,
  "failed_jobs": 8,
  "cancelled_jobs": 2,
  "success_rate": 93.3,
  "total_rows_synced": 5000000,
  "total_data_transferred_mb": 2500,
  "average_job_duration_seconds": 180,
  "by_config": [
    {
      "config_id": "config-789",
      "config_name": "User Tables Sync",
      "jobs": 50,
      "success_rate": 98.0,
      "rows_synced": 2000000
    }
  ],
  "by_hour": [
    {
      "hour": "2024-01-11T00:00:00Z",
      "jobs": 6,
      "rows_synced": 200000
    }
  ]
}
```

---

## 错误代码

| 错误代码 | HTTP 状态码 | 描述 |
|---------|-----------|------|
| `INVALID_REQUEST` | 400 | 请求参数无效 |
| `UNAUTHORIZED` | 401 | 未授权访问 |
| `FORBIDDEN` | 403 | 禁止访问 |
| `NOT_FOUND` | 404 | 资源不存在 |
| `CONFLICT` | 409 | 资源冲突 |
| `DATABASE_ERROR` | 500 | 数据库错误 |
| `CONNECTION_ERROR` | 500 | 连接错误 |
| `SYNC_ERROR` | 500 | 同步错误 |
| `INTERNAL_ERROR` | 500 | 内部服务器错误 |

## 使用示例

### 示例 1: 创建连接并启动同步

```bash
# 1. 创建连接
curl -X POST http://localhost:8080/api/sync/connections \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production DB",
    "host": "prod-mysql.example.com",
    "port": 3306,
    "username": "sync_user",
    "password": "secure_password",
    "database": "production",
    "local_db_name": "prod_sync",
    "ssl": true
  }'

# 2. 测试连接
curl -X POST http://localhost:8080/api/sync/connections/conn-123/test

# 3. 创建同步配置
curl -X POST http://localhost:8080/api/sync/configs \
  -H "Content-Type: application/json" \
  -d '{
    "connection_id": "conn-123",
    "name": "User Tables Sync",
    "sync_mode": "incremental",
    "tables": [
      {
        "source_table": "users",
        "target_table": "users",
        "sync_mode": "incremental",
        "enabled": true
      }
    ]
  }'

# 4. 启动同步任务
curl -X POST http://localhost:8080/api/sync/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "config_id": "config-789"
  }'

# 5. 查看任务状态
curl http://localhost:8080/api/sync/jobs/job-555
```

### 示例 2: 监控同步进度

```bash
# 获取所有运行中的任务
curl http://localhost:8080/api/sync/jobs?status=running

# 获取特定任务的详细信息
curl http://localhost:8080/api/sync/jobs/job-555

# 获取任务日志
curl http://localhost:8080/api/sync/jobs/job-555/logs?level=error

# 获取系统统计信息
curl http://localhost:8080/api/sync/stats?period=24h
```

### 示例 3: 配置管理

```bash
# 导出配置
curl http://localhost:8080/api/sync/config/export > backup.json

# 验证配置
curl -X POST http://localhost:8080/api/sync/config/validate \
  -H "Content-Type: application/json" \
  -d @backup.json

# 导入配置
curl -X POST http://localhost:8080/api/sync/config/import \
  -H "Content-Type: application/json" \
  -d @backup.json
```

## 速率限制

当前版本暂无速率限制，但建议：
- 避免频繁轮询状态接口（建议间隔 >= 1 秒）
- 批量操作时使用适当的分页参数
- 大数据量同步时合理设置 batch_size

## 版本历史

- **v1.0.0** (2024-01-11): 初始版本，包含完整的同步系统 API

## 支持

如有问题或建议，请查看：
- [系统集成文档](SYSTEM_INTEGRATION.md)
- [迁移文档](MIGRATIONS.md)
- [GitHub Issues](https://github.com/your-repo/db-taxi/issues)
