# Job Engine 问题修复总结

## 问题描述

Job Engine 的 `Start()` 方法虽然被正确调用并启动了，但是之前创建的 pending 状态任务没有被自动执行。

## 根本原因

1. **Job Engine 启动正常** ✅
   - `Start()` 方法被正确调用
   - 5 个 worker 线程成功启动
   - Job dispatcher 正常运行

2. **Pending 任务未被处理** ❌
   - 在 Job Engine 启动之前创建的任务保持在 `pending` 状态
   - 这些任务没有被自动提交到执行队列
   - Job Engine 启动时没有恢复机制

## 解决方案

### 1. 添加详细的调试日志

在以下文件中添加了详细的日志输出：

- `internal/sync/sync.go` - Initialize() 方法
- `internal/sync/job_engine.go` - Start() 方法
- `internal/server/server.go` - initSyncSystem() 方法

### 2. 实现 Pending 任务自动恢复

在 `internal/sync/job_engine.go` 中添加了 `resumePendingJobs()` 方法：

```go
// Job Engine 启动时自动执行
func (je *JobEngineService) resumePendingJobs() {
    // 1. 查询所有 pending 状态的任务
    // 2. 检查任务是否过期（超过 24 小时）
    // 3. 将有效任务重新提交到执行队列
    // 4. 将过期任务标记为 failed
}
```

**功能特性：**
- ✅ 自动检测并恢复 pending 任务
- ✅ 跳过超过 24 小时的旧任务
- ✅ 异步执行，不阻塞 Job Engine 启动
- ✅ 详细的日志记录

### 3. 创建诊断和测试工具

**诊断脚本：**
- `scripts/diagnose-job-engine.sh` - 全面的系统诊断
- `scripts/check-job-engine-status.sh` - 快速状态检查

**测试工具：**
- `cmd/test-job-engine/main.go` - 独立的 Job Engine 测试程序

**Makefile 命令：**
```bash
make test-job-engine    # 测试 Job Engine 启动
make diagnose-engine    # 运行诊断脚本
```

### 4. 完善文档

- `docs/TROUBLESHOOTING_JOB_ENGINE.md` - 详细的排查指南
- `JOB_ENGINE_DEBUG.md` - 快速调试参考

## 验证结果

### 测试输出

```
✓ Job Engine 启动成功
✓ 5 个 worker 线程运行中
✓ Job dispatcher 运行中
✓ 发现 11 个 pending 任务
✓ 成功恢复所有 pending 任务
```

### 关键日志

```
INFO Job engine Start() method called
INFO Creating workers... worker_count=5
INFO Job dispatcher started
INFO Job engine started successfully
INFO Checking for pending jobs to resume...
INFO Found pending jobs, resuming... count=11
INFO Pending job resumed successfully job_id=xxx
INFO Finished resuming pending jobs count=11
```

### API 验证

```bash
# 检查系统状态
curl http://localhost:8080/api/sync/status
# 响应: {"success":true,"data":{"status":"healthy"}}

# 检查统计信息
curl http://localhost:8080/api/sync/stats
# 响应: {"success":true,"data":{"total_jobs":11,"running_jobs":0,...}}
```

## 使用方法

### 1. 启动应用（启用调试日志）

```bash
cd db-taxi
LOG_LEVEL=debug go run main.go -config configs/config.local.yaml
```

### 2. 检查 Job Engine 状态

```bash
# 方法 1: 使用状态检查脚本
./scripts/check-job-engine-status.sh

# 方法 2: 使用 API
curl http://localhost:8080/api/sync/status
curl http://localhost:8080/api/sync/stats

# 方法 3: 查看活跃任务
curl http://localhost:8080/api/sync/jobs/active
```

### 3. 测试 Job Engine

```bash
# 运行独立测试
make test-job-engine CONFIG=configs/config.local.yaml

# 或直接运行
go run cmd/test-job-engine/main.go -config configs/config.local.yaml
```

### 4. 提交新任务

```bash
# 通过 API 提交任务
curl -X POST http://localhost:8080/api/sync/jobs \
  -H "Content-Type: application/json" \
  -d '{"config_id": "your-config-id"}'

# 查看任务状态
curl http://localhost:8080/api/sync/jobs/{job_id}

# 查看任务日志
curl http://localhost:8080/api/sync/jobs/{job_id}/logs
```

## 配置说明

确保 `configs/config.local.yaml` 中的配置正确：

```yaml
sync:
  enabled: true              # 必须为 true
  max_concurrency: 5         # worker 数量
  batch_size: 1000          # 批处理大小
  retry_attempts: 3         # 重试次数
  retry_delay: "30s"        # 重试延迟
  job_timeout: "1h"         # 任务超时
  cleanup_age: "720h"       # 清理周期（30天）
```

## 已知问题

### ~~1. 数据库更新错误~~ ✅ 已修复

~~在恢复 pending 任务时，可能会出现 "could not find name error" 的错误。这是因为数据库表结构可能缺少某些字段。~~

**已修复：** 这是 SQL 语句中的字段名错误。已将 `:error` 修改为 `:error_message` 以匹配结构体的 db tag。

详见：`DATABASE_FIELD_NAME_FIX.md`

### 2. 旧的 Pending 任务

超过 24 小时的 pending 任务会被自动标记为 failed。

**如果需要执行这些任务：**
1. 手动将状态改回 pending
2. 重启应用让 Job Engine 重新恢复它们

或者：
1. 删除旧任务
2. 重新创建新任务

## 监控建议

### 1. 启动时检查

应用启动后，检查日志中是否有：
```
✓ "Job engine started successfully"
✓ "Checking for pending jobs to resume..."
✓ "Finished resuming pending jobs"
```

### 2. 运行时监控

定期检查：
```bash
# 每分钟检查一次
watch -n 60 'curl -s http://localhost:8080/api/sync/stats | jq'

# 或使用脚本
watch -n 60 './scripts/check-job-engine-status.sh'
```

### 3. 告警设置

建议监控以下指标：
- `running_jobs` - 运行中的任务数
- `failed_jobs` - 失败的任务数
- pending 任务停留时间（超过 1 小时应告警）

## 相关文档

- [Job Engine 调试指南](JOB_ENGINE_DEBUG.md)
- [详细排查指南](docs/TROUBLESHOOTING_JOB_ENGINE.md)
- [卡住任务排查](docs/TROUBLESHOOTING_STUCK_JOBS.md)
- [快速修复指南](QUICK_FIX_STUCK_JOBS.md)
- [系统集成文档](docs/SYSTEM_INTEGRATION.md)

## 总结

✅ **问题已解决！**

- Job Engine 正常启动并运行
- Pending 任务会被自动恢复和执行
- 添加了完善的日志和诊断工具
- 提供了详细的文档和使用指南

现在 Job Engine 可以：
1. 正常启动和运行
2. 自动恢复 pending 任务
3. 处理新提交的任务
4. 提供详细的状态信息

如有任何问题，请参考相关文档或使用诊断工具进行排查。
