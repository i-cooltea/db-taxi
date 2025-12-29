package sync

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockMonitoringService implements MonitoringService interface for testing
type MockMonitoringService struct {
	mock.Mock
}

func (m *MockMonitoringService) StartJobMonitoring(ctx context.Context, jobID string, totalTables int) error {
	args := m.Called(ctx, jobID, totalTables)
	return args.Error(0)
}

func (m *MockMonitoringService) UpdateJobProgress(ctx context.Context, jobID string, progress *Progress) error {
	args := m.Called(ctx, jobID, progress)
	return args.Error(0)
}

func (m *MockMonitoringService) UpdateTableProgress(ctx context.Context, jobID, tableName string, status TableSyncStatus, processedRows, totalRows int64, errorMsg string) error {
	args := m.Called(ctx, jobID, tableName, status, processedRows, totalRows, errorMsg)
	return args.Error(0)
}

func (m *MockMonitoringService) GetJobProgress(ctx context.Context, jobID string) (*JobSummary, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).(*JobSummary), args.Error(1)
}

func (m *MockMonitoringService) FinishJobMonitoring(ctx context.Context, jobID string, status JobStatus, errorMsg string) error {
	args := m.Called(ctx, jobID, status, errorMsg)
	return args.Error(0)
}

func (m *MockMonitoringService) GetSyncHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*JobHistory), args.Error(1)
}

func (m *MockMonitoringService) GetSyncStatistics(ctx context.Context) (*SyncStatistics, error) {
	args := m.Called(ctx)
	return args.Get(0).(*SyncStatistics), args.Error(1)
}

func (m *MockMonitoringService) GetActiveJobs(ctx context.Context) ([]*JobSummary, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*JobSummary), args.Error(1)
}

func (m *MockMonitoringService) AddJobWarning(ctx context.Context, jobID, warning string) error {
	args := m.Called(ctx, jobID, warning)
	return args.Error(0)
}

func (m *MockMonitoringService) GetJobLogs(ctx context.Context, jobID string) ([]*SyncLog, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).([]*SyncLog), args.Error(1)
}

func (m *MockMonitoringService) LogJobEvent(ctx context.Context, jobID, tableName, level, message string) error {
	args := m.Called(ctx, jobID, tableName, level, message)
	return args.Error(0)
}

// MockSyncEngine implements SyncEngine interface for testing
type MockSyncEngine struct {
	mock.Mock
}

func (m *MockSyncEngine) SyncTable(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	args := m.Called(ctx, job, mapping)
	return args.Error(0)
}

func (m *MockSyncEngine) SyncFull(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	args := m.Called(ctx, job, mapping)
	return args.Error(0)
}

func (m *MockSyncEngine) SyncIncremental(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	args := m.Called(ctx, job, mapping)
	return args.Error(0)
}

func (m *MockSyncEngine) ValidateData(ctx context.Context, mapping *TableMapping) error {
	args := m.Called(ctx, mapping)
	return args.Error(0)
}

func (m *MockSyncEngine) GetTableSchema(ctx context.Context, connectionID, tableName string) (*TableSchema, error) {
	args := m.Called(ctx, connectionID, tableName)
	return args.Get(0).(*TableSchema), args.Error(1)
}

func (m *MockSyncEngine) CreateTargetTable(ctx context.Context, localDB string, schema *TableSchema) error {
	args := m.Called(ctx, localDB, schema)
	return args.Error(0)
}
