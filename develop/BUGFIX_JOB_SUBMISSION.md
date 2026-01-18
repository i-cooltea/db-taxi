# Bug Fix: 任务启动后一直处于运行中状态

## 问题描述

启动同步任务后，任务一直处于 "pending" 或 "running" 状态，无法正常执行和完成。

## 根本原因

1. **JobEngine 未被初始化和启动**
   - 在 `sync.go` 的 `NewManager` 函数中，JobEngine 被创建但从未被初始化
   - 在 `Initialize` 方法中，有 TODO 注释但没有实际启动 JobEngine

2. **任务未被提交到 JobEngine 执行**
   - 在 `service.go` 的 `StartSync` 方法中，任务被创建并保存到数据库
   - 但有 TODO 注释说明需要提交到 job engine，实际上没有执行这一步
   - 导致任务永远停留在 pending 状态

3. **JobEngine 接口缺少 Start/Stop 方法**
   - JobEngine 接口没有定义 Start 和 Stop 方法
   - 导致无法在系统初始化时启动 JobEngine

## 修复内容

### 1. 更新 JobEngine 接口 (`internal/sync/interfaces.go`)

添加了 Start 和 Stop 方法到 JobEngine 接口：

```go
type JobEngine interface {
	// Start initializes and starts the job engine
	Start() error

	// Stop gracefully shuts down the job engine
	Stop() error
	
	// ... 其他方法
}
```

### 2. 修复 Manager 初始化 (`internal/sync/sync.go`)


**在 NewManager 中创建并注入 JobEngine：**

```go
func NewManager(cfg *config.Config, db *sqlx.DB, logger *logrus.Logger) (*Manager, error) {
	// ... 创建其他组件
	
	// Create monitoring service
	monitoring := NewMonitoringService(repo, logger)
	
	// Create sync engine
	syncEngine := NewSyncEngine(db, repo, logger)
	
	// Create job engine
	jobEngine := NewJobEngine(repo, logger, monitoring, syncEngine)
	
	// ... 创建 sync manager
	
	// Set job engine reference in sync manager
	if syncMgrService, ok := syncManager.(*SyncManagerService); ok {
		syncMgrService.jobEngine = jobEngine
	}
	
	manager := &Manager{
		// ... 其他字段
		jobEngine: jobEngine,
	}
	
	return manager, nil
}
```

**在 Initialize 中启动 JobEngine：**

```go
func (m *Manager) Initialize(ctx context.Context) error {
	// ... 运行迁移
	
	// Start job engine
	if m.jobEngine != nil {
		if err := m.jobEngine.Start(); err != nil {
			return fmt.Errorf("failed to start job engine: %w", err)
		}
		m.logger.Info("Job engine started successfully")
	}
	
	return nil
}
```

**在 Shutdown 中停止 JobEngine：**

```go
func (m *Manager) Shutdown(ctx context.Context) error {
	// Stop job engine
	if m.jobEngine != nil {
		if err := m.jobEngine.Stop(); err != nil {
			m.logger.WithError(err).Error("Failed to stop job engine")
		}
	}
	
	// Close connection manager
	if cm, ok := m.connectionManager.(*ConnectionManagerService); ok {
		if err := cm.Close(); err != nil {
			m.logger.WithError(err).Error("Failed to close connection manager")
		}
	}
	
	return nil
}
```

### 3. 修复 SyncManagerService (`internal/sync/service.go`)

**添加 jobEngine 字段：**

```go
type SyncManagerService struct {
	*Service
	monitoring MonitoringService
	jobEngine  JobEngine  // 新增字段
}
```

**修复 StartSync 方法，提交任务到 JobEngine：**

```go
func (s *SyncManagerService) StartSync(ctx context.Context, configID string) (*SyncJob, error) {
	// ... 创建任务
	
	// Submit job to job engine for execution
	if s.jobEngine != nil {
		if err := s.jobEngine.SubmitJob(ctx, job); err != nil {
			s.logger.WithError(err).WithField("job_id", job.ID).Error("Failed to submit job to engine")
			
			// Update job status to failed
			job.Status = JobStatusFailed
			job.Error = fmt.Sprintf("Failed to submit job: %v", err)
			now := time.Now()
			job.EndTime = &now
			
			if updateErr := s.repo.UpdateSyncJob(ctx, job.ID, job); updateErr != nil {
				s.logger.WithError(updateErr).Error("Failed to update job status")
			}
			
			return nil, fmt.Errorf("failed to submit job to engine: %w", err)
		}
		
		s.logger.WithFields(logrus.Fields{
			"job_id":         job.ID,
			"sync_config_id": configID,
			"total_tables":   len(syncConfig.Tables),
		}).Info("Sync job submitted to engine successfully")
	}
	
	return job, nil
}
```

**实现 StopSync 方法：**

```go
func (s *SyncManagerService) StopSync(ctx context.Context, jobID string) error {
	if s.jobEngine == nil {
		return fmt.Errorf("job engine not available")
	}
	
	// Cancel the job
	if err := s.jobEngine.CancelJob(ctx, jobID); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}
	
	s.logger.WithField("job_id", jobID).Info("Sync job stopped successfully")
	return nil
}
```

### 4. 更新 Mock 实现 (`internal/sync/interfaces_test.go`)

为 mockJobEngine 添加 Start 和 Stop 方法：

```go
func (m *mockJobEngine) Start() error {
	return mockError("Start")
}

func (m *mockJobEngine) Stop() error {
	return mockError("Stop")
}
```

## 验证

1. **编译验证**：
   ```bash
   cd db-taxi
   go build -o db-taxi .
   ```

2. **测试验证**：
   ```bash
   go test ./internal/sync/ -v -run TestJobEngine
   ```

所有测试通过，确认修复有效。

## 影响

- ✅ 任务现在可以正常提交到 JobEngine 执行
- ✅ JobEngine 在系统启动时自动启动
- ✅ JobEngine 在系统关闭时优雅停止
- ✅ 任务状态会正确更新（pending → running → completed/failed）
- ✅ StopSync 功能现在可以正常工作

## 测试建议

启动服务后，创建一个同步配置并启动任务，观察：
1. 任务状态应该从 pending 快速变为 running
2. 任务执行过程中可以看到进度更新
3. 任务完成后状态变为 completed 或 failed
4. 可以通过 StopSync API 取消正在运行的任务
