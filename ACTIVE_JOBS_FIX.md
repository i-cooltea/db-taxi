# 运行中任务进度显示修复

## 问题描述

运行中的任务无法正常显示进度信息，前端无法获取配置名称和详细信息。

## 根本原因

1. **后端问题**：`JobMonitor` 结构体缺少 `ConfigID` 字段
   - `GetActiveJobs` API 返回的数据中没有 `config_id`
   - 前端无法通过 `config_id` 查找配置名称

2. **前端问题**：`getConfigName` 函数依赖 `syncStore.configs`
   - 该 store 可能未被正确加载
   - 导致配置名称显示为 ID 而不是名称

## 修复内容

### 后端修复 (db-taxi/internal/sync/monitoring.go)

1. **添加 ConfigID 字段到 JobMonitor**
```go
type JobMonitor struct {
    JobID           string
    ConfigID        string  // 新增字段
    StartTime       time.Time
    // ... 其他字段
}
```

2. **在 StartJobMonitoring 中获取 ConfigID**
```go
func (m *MonitoringServiceImpl) StartJobMonitoring(ctx context.Context, jobID string, totalTables int) error {
    // 从数据库获取任务详情
    job, err := m.repo.GetSyncJob(ctx, jobID)
    if err != nil {
        return fmt.Errorf("failed to get job details: %w", err)
    }

    monitor := &JobMonitor{
        JobID:    jobID,
        ConfigID: job.ConfigID,  // 设置 ConfigID
        // ... 其他字段
    }
    // ...
}
```

3. **在 GetActiveJobs 中返回 ConfigID**
```go
summary := &JobSummary{
    JobID:    monitor.JobID,
    ConfigID: monitor.ConfigID,  // 包含 ConfigID
    // ... 其他字段
}
```

### 前端修复 (db-taxi/frontend/src/views/Monitoring.vue)

**改进 getConfigName 函数**
```javascript
function getConfigName(configId) {
  // 优先从组件本地加载的配置中查找
  const config = availableConfigs.value.find(c => c.id === configId)
  if (config) {
    return config.name
  }
  // 回退到 syncStore
  const storeConfig = syncStore.configs.find(c => c.id === configId)
  return storeConfig?.name || configId
}
```

## 测试方法

1. **启动服务器**
```bash
cd db-taxi
make run
```

2. **运行测试脚本**
```bash
./test_active_jobs.sh
```

3. **手动测试**
   - 在前端启动一个同步任务
   - 查看"运行中的任务"部分
   - 验证配置名称正确显示
   - 验证进度条正常更新
   - 验证表数量和行数正确显示

## API 响应示例

修复后的 `/api/sync/jobs/active` 响应：

```json
{
  "success": true,
  "data": [
    {
      "job_id": "job-123",
      "config_id": "config-456",  // 现在包含此字段
      "start_time": "2026-01-16T10:00:00Z",
      "total_tables": 10,
      "completed_tables": 3,
      "total_rows": 100000,
      "processed_rows": 30000,
      "progress_percent": 30.0,
      "error_count": 0,
      "table_progress": {
        "users": {
          "table_name": "users",
          "status": "completed",
          "total_rows": 10000,
          "processed_rows": 10000
        }
      }
    }
  ],
  "meta": {
    "total": 1
  }
}
```

## 影响范围

- ✅ 运行中任务的配置名称显示
- ✅ 进度条显示
- ✅ 表同步进度显示
- ✅ 实时数据更新（每5秒自动刷新）

## 相关文件

- `db-taxi/internal/sync/monitoring.go` - 后端监控服务
- `db-taxi/internal/sync/types.go` - 数据结构定义
- `db-taxi/frontend/src/views/Monitoring.vue` - 前端监控页面
- `db-taxi/frontend/src/stores/syncStore.js` - 前端状态管理
