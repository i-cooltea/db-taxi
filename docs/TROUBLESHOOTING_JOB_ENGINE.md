# Job Engine 启动问题排查指南

## 问题描述

Job Engine 的 `Start()` 方法没有被执行，导致同步任务无法正常工作。

## 启动流程

Job Engine 的启动流程如下：

```
main.go
  └─> server.New(cfg)
       └─> server.initSyncSystem()
            └─> sync.NewManager()
                 └─> sync.Manager.Initialize()
                      └─> jobEngine.Start()
```

## 可能的原因

### 1. Sync 系统被禁用

**症状：**
- 日志中没有 "Initializing sync system..." 消息
- 日志中出现 "sync system is disabled in configuration"

**检查方法：**
```bash
# 检查配置文件
grep SYNC_ENABLED .env

# 或者检查配置
cat configs/config.yaml | grep -A 5 "sync:"
```

**解决方案：**
```bash
# 在 .env 文件中设置
echo "SYNC_ENABLED=true" >> .env

# 或在 config.yaml 中设置
# sync:
#   enabled: true
```

### 2. 数据库连接失败

**症状：**
- 日志中出现 "Failed to initialize database connection"
- 日志中出现 "Database connection required for sync system"

**检查方法：**
```bash
# 测试数据库连接
mysql -h localhost -u root -p -e "SELECT 1"

# 检查数据库配置
grep "^DB_" .env
```

**解决方案：**
- 确保数据库服务正在运行
- 检查数据库连接参数（host, port, username, password）
- 确保数据库用户有足够的权限

### 3. 数据库迁移失败

**症状：**
- 日志中出现 "failed to run migrations"
- Sync 系统表不存在

**检查方法：**
```bash
# 检查 sync 表是否存在
mysql -u root -p myapp -e "SHOW TABLES LIKE 'sync_%'"

# 检查迁移状态
mysql -u root -p myapp -e "SELECT * FROM schema_migrations"
```

**解决方案：**
```bash
# 运行数据库迁移
make migrate

# 或手动运行
./scripts/migrate.sh up

# 或使用 migrate 命令
go run cmd/migrate/main.go up
```

### 4. Job Engine 已经在运行

**症状：**
- 日志中出现 "Job engine is already running"

**检查方法：**
```bash
# 检查是否有多个实例在运行
ps aux | grep db-taxi
```

**解决方案：**
- 停止所有运行中的实例
- 重新启动应用

### 5. 初始化过程中出现错误

**症状：**
- 日志中出现 "Failed to initialize sync system"
- 日志中出现 "Failed to start job engine"

**检查方法：**
查看详细的错误日志

**解决方案：**
根据具体错误信息进行处理

## 诊断步骤

### 步骤 1: 运行诊断脚本

```bash
cd db-taxi
./scripts/diagnose-job-engine.sh
```

### 步骤 2: 启用调试日志

在 `.env` 文件中设置：
```bash
LOG_LEVEL=debug
LOG_FORMAT=text
```

### 步骤 3: 重新启动应用并查看日志

```bash
# 启动应用并保存日志
make run 2>&1 | tee db-taxi.log

# 在另一个终端查看日志
tail -f db-taxi.log
```

### 步骤 4: 检查关键日志消息

查找以下日志消息：

1. **Sync 系统初始化：**
   ```
   "Initializing sync system..."
   "Creating sync manager..."
   "Initializing sync manager..."
   ```

2. **Job Engine 启动：**
   ```
   "Starting job engine..."
   "Job engine Start() method called"
   "Creating workers..."
   "Worker started" (应该有 5 条，对应 5 个 worker)
   "Job dispatcher started"
   "Job engine started successfully"
   ```

3. **错误消息：**
   ```
   "Failed to initialize database connection"
   "Failed to create sync manager"
   "Failed to initialize sync system"
   "Failed to start job engine"
   "Job engine is already running"
   ```

### 步骤 5: 验证 Job Engine 状态

通过 API 检查状态：

```bash
# 检查服务器状态
curl http://localhost:8080/api/status

# 检查 sync 系统状态
curl http://localhost:8080/api/sync/status

# 检查 sync 统计信息
curl http://localhost:8080/api/sync/stats
```

预期响应应该包含：
```json
{
  "success": true,
  "data": {
    "sync": {
      "enabled": true,
      "stats": {
        "total_jobs": 0,
        "running_jobs": 0,
        "completed_jobs": 0,
        "failed_jobs": 0
      }
    }
  }
}
```

## 常见问题解决方案

### 问题 1: "sync system is disabled in configuration"

```bash
# 解决方案
echo "SYNC_ENABLED=true" >> .env
# 重启应用
```

### 问题 2: "Database connection required for sync system"

```bash
# 检查数据库配置
cat .env | grep DB_

# 测试连接
mysql -h $DB_HOST -P $DB_PORT -u $DB_USERNAME -p$DB_PASSWORD -e "SELECT 1"

# 如果连接失败，更新配置
vi .env
```

### 问题 3: "failed to run migrations"

```bash
# 运行迁移
make migrate

# 如果失败，检查迁移日志
make migrate 2>&1 | tee migrate.log

# 手动创建表（如果需要）
mysql -u root -p myapp < internal/migration/sql/001_create_sync_tables.sql
mysql -u root -p myapp < internal/migration/sql/002_add_initial_data.sql
```

### 问题 4: Job Engine 启动但没有处理任务

```bash
# 检查是否有 worker 在运行
# 日志中应该有 "Worker started" 消息

# 检查 job queue
# 通过 API 提交测试任务
curl -X POST http://localhost:8080/api/sync/jobs \
  -H "Content-Type: application/json" \
  -d '{"config_id": "your-config-id"}'

# 查看任务状态
curl http://localhost:8080/api/sync/jobs
```

## 验证修复

完成修复后，验证 Job Engine 是否正常工作：

1. **检查日志：**
   ```bash
   grep "Job engine started successfully" db-taxi.log
   ```

2. **检查 API 状态：**
   ```bash
   curl http://localhost:8080/api/sync/status
   ```

3. **提交测试任务：**
   ```bash
   # 首先创建连接和配置
   # 然后提交任务
   curl -X POST http://localhost:8080/api/sync/jobs \
     -H "Content-Type: application/json" \
     -d '{"config_id": "test-config"}'
   ```

4. **监控任务执行：**
   ```bash
   # 查看任务列表
   curl http://localhost:8080/api/sync/jobs
   
   # 查看特定任务
   curl http://localhost:8080/api/sync/jobs/{job_id}
   
   # 查看任务日志
   curl http://localhost:8080/api/sync/jobs/{job_id}/logs
   ```

## 获取帮助

如果问题仍然存在，请收集以下信息：

1. 完整的启动日志（使用 `LOG_LEVEL=debug`）
2. 数据库连接配置（隐藏密码）
3. 数据库表列表（`SHOW TABLES`）
4. API 状态响应
5. 任何错误消息的完整堆栈跟踪

然后查看：
- `docs/TROUBLESHOOTING_STUCK_JOBS.md` - 卡住任务的排查
- `QUICK_FIX_STUCK_JOBS.md` - 快速修复指南
- `docs/SYSTEM_INTEGRATION.md` - 系统集成文档

## 相关文件

- `internal/sync/sync.go` - Sync Manager 实现
- `internal/sync/job_engine.go` - Job Engine 实现
- `internal/server/server.go` - Server 初始化
- `main.go` - 应用入口
- `scripts/diagnose-job-engine.sh` - 诊断脚本
