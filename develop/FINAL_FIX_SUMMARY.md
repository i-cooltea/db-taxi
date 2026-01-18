# 僵尸任务问题 - 最终修复

## 问题根源

系统中存在**两个不同的MonitoringService实例**，导致任务从一个实例的activeJobs中删除，但API查询的是另一个实例的activeJobs。

### 代码问题

1. **sync.go (第37行)**：创建了第一个MonitoringService实例
   ```go
   monitoring := NewMonitoringService(repo, logger)
   jobEngine := NewJobEngine(repo, logger, monitoring, syncEngine)
   ```

2. **service.go (第706行)**：NewSyncManager又创建了第二个MonitoringService实例
   ```go
   func NewSyncManager(...) SyncManager {
       monitoring := NewMonitoringService(repo, logger)  // 新实例！
       return &SyncManagerService{
           monitoring: monitoring,  // 使用新实例
       }
   }
   ```

3. **结果**：
   - JobEngine使用第一个monitoring实例
   - API调用使用第二个monitoring实例
   - 任务完成时从第一个实例删除
   - API查询第二个实例，仍然看到任务

## 修复方案

### 1. 修改NewSyncManager签名

让它接受monitoring参数，而不是创建新实例：

```go
func NewSyncManager(repo Repository, logger *logrus.Logger, localDB *sqlx.DB, 
                    jobEngine JobEngine, monitoring MonitoringService) SyncManager {
    service := NewService(repo, logger, localDB)
    return &SyncManagerService{
        Service:    service,
        monitoring: monitoring,  // 使用传入的实例
        jobEngine:  jobEngine,
    }
}
```

### 2. 更新sync.go中的调用

传入同一个monitoring实例：

```go
monitoring := NewMonitoringService(repo, logger)
jobEngine := NewJobEngine(repo, logger, monitoring, syncEngine)
syncManager := NewSyncManager(repo, logger, db, jobEngine, monitoring)  // 传入monitoring
```

### 3. 修复FinishJobMonitoring

使用defer确保即使数据库操作失败，也会清理activeJobs：

```go
func (m *MonitoringServiceImpl) FinishJobMonitoring(...) error {
    m.jobsMutex.Lock()
    defer m.jobsMutex.Unlock()

    // Always remove from active jobs, even if database update fails
    defer func() {
        delete(m.activeJobs, jobID)
        // Invalidate statistics cache
        m.statsMutex.Lock()
        m.statisticsCache = nil
        m.statsMutex.Unlock()
    }()

    // Try to update database...
}
```

## 验证步骤

1. **重启服务**：
   ```bash
   pkill db-taxi
   ./db-taxi > logs.txt 2>&1 &
   ```

2. **启动同步任务**并记录job_id

3. **监控任务**：
   ```bash
   ./monitor_job_lifecycle.sh <job_id>
   ```

4. **验证结果**：
   - 任务完成后应该从active列表中消失
   - 监控脚本显示：`✓ Job correctly removed from active list`

5. **检查API**：
   ```bash
   curl -s http://localhost:8080/api/sync/jobs/active | jq '.data | length'
   ```
   应该返回0（如果没有其他运行中的任务）

## 相关文件

- `internal/sync/service.go` - 修改NewSyncManager签名
- `internal/sync/sync.go` - 传入monitoring实例
- `internal/sync/monitoring.go` - 使用defer确保清理
- `monitor_job_lifecycle.sh` - 监控脚本
- `cleanup_zombie_jobs.sh` - 清理脚本

## 经验教训

1. **单例模式**：关键的状态管理服务（如MonitoringService）应该是单例
2. **依赖注入**：通过参数传递依赖，而不是在函数内部创建
3. **资源清理**：使用defer确保资源总是被释放
4. **详细日志**：记录关键操作，便于调试
5. **监控工具**：创建监控脚本帮助快速诊断问题
