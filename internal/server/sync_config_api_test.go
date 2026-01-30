package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"db-taxi/internal/config"
	"db-taxi/internal/sync"
)

// MockSyncManager is a mock implementation of SyncManagerInterface
type MockSyncManager struct {
	mock.Mock
}

func (m *MockSyncManager) GetConnectionManager() sync.ConnectionManager {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(sync.ConnectionManager)
}

func (m *MockSyncManager) GetSyncManager() sync.SyncManager {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(sync.SyncManager)
}

func (m *MockSyncManager) GetMappingManager() sync.MappingManager {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(sync.MappingManager)
}

func (m *MockSyncManager) GetJobEngine() sync.JobEngine {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(sync.JobEngine)
}

func (m *MockSyncManager) GetSyncEngine() sync.SyncEngine {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(sync.SyncEngine)
}

func (m *MockSyncManager) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSyncManager) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSyncManager) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSyncManager) GetStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockSyncManagerService is a mock implementation of sync.SyncManager
type MockSyncManagerService struct {
	mock.Mock
}

func (m *MockSyncManagerService) CreateSyncConfig(ctx context.Context, config *sync.SyncConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockSyncManagerService) GetSyncConfigs(ctx context.Context, connectionID string) ([]*sync.SyncConfig, error) {
	args := m.Called(ctx, connectionID)
	return args.Get(0).([]*sync.SyncConfig), args.Error(1)
}

func (m *MockSyncManagerService) GetSyncConfig(ctx context.Context, id string) (*sync.SyncConfig, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sync.SyncConfig), args.Error(1)
}

func (m *MockSyncManagerService) UpdateSyncConfig(ctx context.Context, id string, config *sync.SyncConfig) error {
	args := m.Called(ctx, id, config)
	return args.Error(0)
}

func (m *MockSyncManagerService) DeleteSyncConfig(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSyncManagerService) StartSync(ctx context.Context, configID string) (*sync.SyncJob, error) {
	args := m.Called(ctx, configID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sync.SyncJob), args.Error(1)
}

func (m *MockSyncManagerService) StopSync(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockSyncManagerService) GetSyncStatus(ctx context.Context, jobID string) (*sync.SyncJob, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sync.SyncJob), args.Error(1)
}

func (m *MockSyncManagerService) GetRemoteDatabases(ctx context.Context, connectionID string) ([]string, error) {
	args := m.Called(ctx, connectionID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSyncManagerService) GetRemoteTables(ctx context.Context, connectionID, database string) ([]string, error) {
	args := m.Called(ctx, connectionID, database)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSyncManagerService) GetRemoteTableSchema(ctx context.Context, connectionID, database, tableName string) (*sync.TableSchema, error) {
	args := m.Called(ctx, connectionID, database, tableName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sync.TableSchema), args.Error(1)
}

func (m *MockSyncManagerService) AddTableMapping(ctx context.Context, syncConfigID string, mapping *sync.TableMapping) error {
	args := m.Called(ctx, syncConfigID, mapping)
	return args.Error(0)
}

func (m *MockSyncManagerService) UpdateTableMapping(ctx context.Context, mappingID string, mapping *sync.TableMapping) error {
	args := m.Called(ctx, mappingID, mapping)
	return args.Error(0)
}

func (m *MockSyncManagerService) RemoveTableMapping(ctx context.Context, mappingID string) error {
	args := m.Called(ctx, mappingID)
	return args.Error(0)
}

func (m *MockSyncManagerService) GetTableMappings(ctx context.Context, syncConfigID string) ([]*sync.TableMapping, error) {
	args := m.Called(ctx, syncConfigID)
	return args.Get(0).([]*sync.TableMapping), args.Error(1)
}

func (m *MockSyncManagerService) ToggleTableMapping(ctx context.Context, mappingID string, enabled bool) error {
	args := m.Called(ctx, mappingID, enabled)
	return args.Error(0)
}

func (m *MockSyncManagerService) SetTableSyncMode(ctx context.Context, mappingID string, syncMode sync.SyncMode) error {
	args := m.Called(ctx, mappingID, syncMode)
	return args.Error(0)
}

func (m *MockSyncManagerService) GetJobProgress(ctx context.Context, jobID string) (*sync.JobSummary, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sync.JobSummary), args.Error(1)
}

func (m *MockSyncManagerService) GetSyncHistory(ctx context.Context, limit, offset int) ([]*sync.JobHistory, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*sync.JobHistory), args.Error(1)
}

func (m *MockSyncManagerService) GetSyncStatistics(ctx context.Context) (*sync.SyncStatistics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sync.SyncStatistics), args.Error(1)
}

func (m *MockSyncManagerService) GetActiveJobs(ctx context.Context) ([]*sync.JobSummary, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*sync.JobSummary), args.Error(1)
}

func (m *MockSyncManagerService) GetJobLogs(ctx context.Context, jobID string) ([]*sync.SyncLog, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).([]*sync.SyncLog), args.Error(1)
}

func setupSyncConfigTestServer() (*Server, *MockSyncManager, *MockSyncManagerService) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	engine := gin.New()
	engine.Use(gin.Recovery())

	mockSyncMgr := new(MockSyncManager)
	mockSyncMgrService := new(MockSyncManagerService)

	server := &Server{
		config:      cfg,
		engine:      engine,
		logger:      logger,
		syncManager: mockSyncMgr,
	}

	// Register routes
	api := server.engine.Group("/api")
	server.registerSyncRoutes(api)

	return server, mockSyncMgr, mockSyncMgrService
}

func TestGetRemoteTables(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("GetSyncConfig", mock.Anything, "config-123").Return(&sync.SyncConfig{
		ID:                 "config-123",
		SourceConnectionID: "conn-123",
		SourceDatabase:     "test_db",
	}, nil)
	mockSyncMgrService.On("GetRemoteTables", mock.Anything, "conn-123", "test_db").Return([]string{"users", "orders", "products"}, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/configs/config-123/tables", nil)
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Equal(t, 3, len(data))
	assert.Equal(t, "users", data[0].(string))

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}

func TestAddTableMapping(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("AddTableMapping", mock.Anything, "config-123", mock.AnythingOfType("*sync.TableMapping")).Return(nil)

	// Create request body
	mapping := sync.TableMapping{
		SourceTable: "users",
		TargetTable: "users_local",
		SyncMode:    sync.SyncModeFull,
		Enabled:     true,
	}
	body, _ := json.Marshal(mapping)

	// Create request
	req, _ := http.NewRequest("POST", "/api/sync/configs/config-123/mappings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}

func TestToggleTableMapping(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("ToggleTableMapping", mock.Anything, "mapping-123", false).Return(nil)

	// Create request body
	body, _ := json.Marshal(map[string]bool{"enabled": false})

	// Create request
	req, _ := http.NewRequest("POST", "/api/sync/configs/config-123/mappings/mapping-123/toggle", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}

func TestSetTableSyncMode(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("SetTableSyncMode", mock.Anything, "mapping-123", sync.SyncModeIncremental).Return(nil)

	// Create request body
	body, _ := json.Marshal(map[string]string{"sync_mode": "incremental"})

	// Create request
	req, _ := http.NewRequest("POST", "/api/sync/configs/config-123/mappings/mapping-123/sync-mode", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}

func TestGetActiveSyncJobs(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("GetActiveJobs", mock.Anything).Return([]*sync.JobSummary{
		{
			JobID:    "job-123",
			ConfigID: "config-123",
			Status:   sync.JobStatusRunning,
		},
	}, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/jobs/active", nil)
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Equal(t, 1, len(data))

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}

func TestGetSyncJobProgress(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("GetJobProgress", mock.Anything, "job-123").Return(&sync.JobSummary{
		JobID:           "job-123",
		ConfigID:        "config-123",
		Status:          sync.JobStatusRunning,
		ProgressPercent: 45.5,
	}, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/jobs/job-123/progress", nil)
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "job-123", data["job_id"])
	assert.Equal(t, 45.5, data["progress_percent"])

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}
