# Job Engine 调试快速指南

## 问题：jobEngine 的 Start() 方法没有执行

**✅ 已解决！** Job Engine 现在会在启动时自动恢复 pending 状态的任务。

### 新功能

Job Engine 启动时会自动：
1. 检查数据库中所有 `pending` 状态的任务
2. 将这些任务重新提交到执行队列
3. 跳过超过 24 小时的旧任务（标记为 failed）

### 快速诊断

```bash
# 1. 运行诊断脚本
make diagnose-engine

# 2. 测试 Job Engine 启动
make test-job-engine

# 3. 启用调试日志运行应用
LOG_LEVEL=debug make run 2>&1 | tee app.log
```

### 检查清单

- [ ] **配置检查**
  ```bash
  grep SYNC_ENABLED .env
  # 应该输出: SYNC_ENABLED=true
  ```

- [ ] **数据库连接**
  ```bash
  mysql -h localhost -u root -p -e "SELECT 1"
  # 应该成功连接
  ```

- [ ] **Sync 表存在**
  ```bash
  mysql -u root -p myapp -e "SHOW TABLES LIKE 'sync_%'"
  # 应该显示: connections, sync_configs, sync_jobs 等表
  ```

- [ ] **迁移已运行**
  ```bash
  make migrate-status
  # 应该显示当前版本号
  ```

### 关键日志消息

启动应用后，应该看到以下日志（按顺序）：

```
1. "Initializing sync system..."
2. "Creating sync manager..."
3. "Sync system manager initialized successfully"
4. "Initializing sync manager..."
5. "Running sync system database migrations..."
6. "Sync system database migrations completed"
7. "Starting job engine..."
8. "Job engine Start() method called"
9. "Creating workers..."
10. "Worker started" (x5)
11. "Job dispatcher started"
12. "Job engine started successfully"
13. "Sync system initialized successfully"
```

### 如果缺少某些日志

#### 缺少 "Initializing sync system..."
**原因：** `initSyncSystem()` 没有被调用或数据库连接失败

**解决：**
```bash
# 检查数据库配置
cat .env | grep DB_

# 测试数据库连接
mysql -h $DB_HOST -P $DB_PORT -u $DB_USERNAME -p
```

#### 缺少 "Creating sync manager..."
**原因：** Sync 系统被禁用

**解决：**
```bash
echo "SYNC_ENABLED=true" >> .env
```

#### 缺少 "Starting job engine..."
**原因：** 迁移失败或 jobEngine 为 nil

**解决：**
```bash
# 运行迁移
make migrate

# 检查迁移状态
make migrate-status
```

#### 缺少 "Job engine Start() method called"
**原因：** `jobEngine.Start()` 没有被调用

**解决：**
这是一个代码问题，检查 `internal/sync/sync.go` 中的 `Initialize()` 方法

#### 缺少 "Worker started"
**原因：** Worker 创建失败

**解决：**
检查日志中的错误信息，可能是资源不足

### 代码修改说明

我已经在以下文件中添加了详细的调试日志：

1. **internal/sync/sync.go**
   - 在 `Initialize()` 方法中添加了更多日志
   - 检查 jobEngine 是否为 nil

2. **internal/sync/job_engine.go**
   - 在 `Start()` 方法开始时添加日志
   - 为每个 worker 添加启动日志
   - 添加 dispatcher 启动日志

3. **internal/server/server.go**
   - 在 `initSyncSystem()` 中添加详细的步骤日志

### 验证修复

运行测试程序验证 Job Engine 是否正常：

```bash
# 运行测试
make test-job-engine

# 预期输出
========================================
Job Engine 测试工具
========================================

1. 加载配置...
✓ 配置加载成功
  - Sync Enabled: true
  - Database: root@localhost:3306/myapp

2. 连接数据库...
✓ 数据库连接成功

3. 创建 Sync Manager...
✓ Sync Manager 创建成功

4. 初始化 Sync 系统...
✓ Sync 系统初始化成功

5. 检查 Job Engine 状态...
✓ Job Engine 已创建
✓ Job Engine 正在运行
  - 活跃任务数: 0
  - 队列长度: 0

6. 执行健康检查...
✓ 健康检查通过

7. 获取系统统计...
✓ 统计信息:
  - total_jobs: 0
  - running_jobs: 0
  - completed_jobs: 0
  - failed_jobs: 0

8. 保持运行 5 秒以观察日志...

9. 关闭系统...
✓ 系统已关闭

========================================
✓ 所有测试通过！Job Engine 工作正常
========================================
```

### API 验证

通过 API 验证 Job Engine 状态：

```bash
# 1. 检查服务器状态
curl http://localhost:8080/api/status | jq

# 2. 检查 sync 状态
curl http://localhost:8080/api/sync/status | jq

# 3. 检查统计信息
curl http://localhost:8080/api/sync/stats | jq
```

### 常见错误及解决方案

| 错误消息 | 原因 | 解决方案 |
|---------|------|---------|
| "sync system is disabled in configuration" | SYNC_ENABLED=false | 设置 SYNC_ENABLED=true |
| "Database connection required for sync system" | 数据库未连接 | 检查数据库配置和连接 |
| "failed to run migrations" | 迁移失败 | 运行 `make migrate` |
| "Job engine is already running" | 重复启动 | 重启应用 |
| "Job engine is nil" | 初始化失败 | 检查 NewManager 是否成功 |

### 获取更多帮助

- 详细排查指南: `docs/TROUBLESHOOTING_JOB_ENGINE.md`
- 卡住任务排查: `docs/TROUBLESHOOTING_STUCK_JOBS.md`
- 快速修复指南: `QUICK_FIX_STUCK_JOBS.md`
- 系统集成文档: `docs/SYSTEM_INTEGRATION.md`

### 联系支持

如果问题仍然存在，请提供：

1. 完整的启动日志（使用 `LOG_LEVEL=debug`）
2. `make diagnose-engine` 的输出
3. `make test-job-engine` 的输出
4. 数据库配置（隐藏密码）
5. 任何错误消息的完整堆栈跟踪
