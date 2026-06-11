package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// DiffTaskStatus represents the current status of a diff task.
type DiffTaskStatus string

const (
	DiffTaskRunning DiffTaskStatus = "running"
	DiffTaskDone    DiffTaskStatus = "done"
	DiffTaskFailed  DiffTaskStatus = "failed"
)

// DiffTask represents an asynchronous table diff task.
type DiffTask struct {
	ID        string         `json:"task_id"`
	Status    DiffTaskStatus `json:"status"`
	Progress  int            `json:"progress"`
	Result    *DiffResult    `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// DiffCompareRequest is the request body for POST /api/sync/diff/compare.
type DiffCompareRequest struct {
	Source  DiffTableRef `json:"source" binding:"required"`
	Target  DiffTableRef `json:"target" binding:"required"`
	MaxRows int64        `json:"max_rows,omitempty"`
}

// DiffTableRef identifies a table by connection + database + table name.
type DiffTableRef struct {
	ConnectionID string `json:"connection_id" binding:"required"`
	Database     string `json:"database" binding:"required"`
	Table        string `json:"table" binding:"required"`
}

// DiffTaskManager manages diff tasks in memory.
type DiffTaskManager struct {
	tasks map[string]*DiffTask
	mu    sync.RWMutex

	diffEngine *DiffEngine
	repo       Repository
	logger     *logrus.Logger
}

// NewDiffTaskManager creates a new DiffTaskManager.
func NewDiffTaskManager(repo Repository, logger *logrus.Logger) *DiffTaskManager {
	return &DiffTaskManager{
		tasks:      make(map[string]*DiffTask),
		diffEngine: NewDiffEngine(repo, logger),
		repo:       repo,
		logger:     logger,
	}
}

// StartTask creates and starts an async diff task, returning its ID.
func (m *DiffTaskManager) StartTask(ctx context.Context, req *DiffCompareRequest) (string, error) {
	taskID := uuid.New().String()
	task := &DiffTask{
		ID:        taskID,
		Status:    DiffTaskRunning,
		Progress:  0,
		CreatedAt: time.Now(),
	}

	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()

	go m.runDiff(task, req)

	return taskID, nil
}

// GetTask returns the current state of a diff task.
func (m *DiffTaskManager) GetTask(taskID string) (*DiffTask, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, false
	}
	return t, true
}

func (m *DiffTaskManager) runDiff(task *DiffTask, req *DiffCompareRequest) {
	bgCtx := context.Background()

	maxRows := req.MaxRows
	if maxRows <= 0 {
		maxRows = 500
	}

	// Open source DB connection
	sourceDB, err := m.openConnection(bgCtx, req.Source)
	if err != nil {
		m.failTask(task, fmt.Sprintf("连接源数据库失败: %v", err))
		return
	}
	defer sourceDB.Close()

	// Open target DB connection
	targetDB, err := m.openConnection(bgCtx, req.Target)
	if err != nil {
		m.failTask(task, fmt.Sprintf("连接目标数据库失败: %v", err))
		return
	}
	defer targetDB.Close()

	// Validate tables exist
	m.updateProgress(task, 10)
	if code, err := m.diffEngine.ValidateDiffTableRequest(bgCtx, sourceDB, req.Source.Database, req.Source.Table); err != nil {
		m.failTask(task, fmt.Sprintf("源表校验失败(code=%d): %v", code, err))
		return
	}
	if code, err := m.diffEngine.ValidateDiffTableRequest(bgCtx, targetDB, req.Target.Database, req.Target.Table); err != nil {
		m.failTask(task, fmt.Sprintf("目标表校验失败(code=%d): %v", code, err))
		return
	}

	m.updateProgress(task, 30)

	// Run diff
	result, err := m.diffEngine.RunDiff(bgCtx, sourceDB, targetDB, req.Source.Database, req.Target.Database, req.Source.Table, req.Target.Table, maxRows)
	if err != nil {
		m.failTask(task, fmt.Sprintf("对比执行失败: %v", err))
		return
	}

	m.updateProgress(task, 100)

	m.mu.Lock()
	task.Status = DiffTaskDone
	task.Result = result
	m.mu.Unlock()
}

func (m *DiffTaskManager) openConnection(ctx context.Context, ref DiffTableRef) (*sqlx.DB, error) {
	connConfig, err := m.repo.GetConnection(ctx, ref.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("获取连接配置失败: %w", err)
	}

	// Build DSN using the same logic as ConnectionManagerService
	svc := &ConnectionManagerService{}
	dsn, err := svc.buildRemoteDSNWithDB(connConfig, ref.Database)
	if err != nil {
		return nil, fmt.Errorf("构建DSN失败: %w", err)
	}

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开连接失败: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接测试失败: %w", err)
	}

	return db, nil
}

func (m *DiffTaskManager) failTask(task *DiffTask, errMsg string) {
	m.mu.Lock()
	task.Status = DiffTaskFailed
	task.Error = errMsg
	m.mu.Unlock()
	m.logger.WithField("task_id", task.ID).Error(errMsg)
}

func (m *DiffTaskManager) updateProgress(task *DiffTask, pct int) {
	m.mu.Lock()
	task.Progress = pct
	m.mu.Unlock()
}
