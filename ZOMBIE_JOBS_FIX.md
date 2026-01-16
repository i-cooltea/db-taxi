# Zombie Jobs 问题修复

## 问题描述

运行中的任务列表只增不减，已完成的任务仍然显示在"运行中的任务"列表中。

## 根本原因

在 `internal/sync/monitoring.go` 的 `FinishJobMonitoring` 函数中，存在以下问题：

1. **数据库更新失败导致任务无法从activeJobs移除**
   - 函数先尝试从数据库获取任务并更新状态
   - 如果数据库操作失败（如任务不存在），函数会返回错误
   - 返回错误后，`delete(m.activeJobs, jobID)` 不会被执行
   - 导致任务永远留在内存中的activeJobs map中

2. **数据库中任务丢失的原因**
   - 可能是并发更新导致的竞态条件
   - 或者是数据库事务问题

## 症状

1. Active jobs API 返回已完成的任务
2. 日志显示：
   ```
   Failed to update sync job in repository" error="sync job not found
   Failed to finish job monitoring" error="failed to update sync job: sync job not found
   ```
3. 任务在history中状态为 `completed`，但仍在active列表中

## 解决方案

### 修复 FinishJobMonitoring 函数

使用 `defer` 确保无论数据库操作是否成功，都会从 activeJobs 中移除任务：

```go
func (m *MonitoringServiceImpl) FinishJobMonitoring(ctx context.Context, jobID string, status JobStatus, errorMsg string) error {
    m.jobsMutex.Lock()
    defer m.jobsMutex.Unlock()

    monitor, exists := m.activeJobs[jobID]
    if !exists {
        return fmt.Errorf("job monitor not found: %s", jobID)
    }

    // Always remove from active jobs, even if database update fails
    // This prevents zombie jobs
    defer func() {
        delete(m.activeJobs, jobID)
        // Invalidate statistics cache
        m.statsMutex.Lock()
        m.statisticsCache = nil
        m.statsMutex.Unlock()
    }()

    // Try to update database, but don't fail if it doesn't work
    job, err := m.repo.GetSyncJob(ctx, jobID)
    if err != nil {
        m.logger.WithError(err).Error("Failed to get sync job")
        return nil // Don't return error - we still removed from activeJobs
    }

    // Update job status...
    // Even if this fails, the defer will clean up activeJobs
}
```

### 关键改进

1. **使用 defer 确保清理**：无论函数如何退出，都会执行清理
2. **不返回错误**：即使数据库操作失败，也返回 nil，因为内存清理已完成
3. **详细日志**：记录每个步骤，便于调试

## 测试验证

使用监控脚本测试：

```bash
# 1. 启动一个同步任务，记录job_id
# 2. 运行监控脚本
./monitor_job_lifecycle.sh <job_id>

# 3. 等待任务完成，脚本会显示：
# ✓ Job correctly removed from active list
```

## 预防措施

1. **数据库操作容错**：关键的内存清理操作不应依赖数据库操作的成功
2. **使用 defer 进行清理**：确保资源总是被释放
3. **详细日志**：记录所有关键步骤，便于问题诊断
4. **监控告警**：定期检查僵尸任务并告警

## 相关文件

- `internal/sync/monitoring.go` - 主要修复文件
- `internal/sync/job_engine.go` - 任务执行引擎
- `monitor_job_lifecycle.sh` - 实时监控脚本
- `cleanup_zombie_jobs.sh` - 清理脚本
