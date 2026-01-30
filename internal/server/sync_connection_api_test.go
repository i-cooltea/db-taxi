package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"db-taxi/internal/config"
	"db-taxi/internal/sync"
)

// mockConnectionManager is a mock implementation of ConnectionManager for testing
type mockConnectionManager struct {
	connections map[string]*sync.Connection
	testResults map[string]*sync.ConnectionStatus
}

func newMockConnectionManager() *mockConnectionManager {
	return &mockConnectionManager{
		connections: make(map[string]*sync.Connection),
		testResults: make(map[string]*sync.ConnectionStatus),
	}
}

func (m *mockConnectionManager) AddConnection(ctx context.Context, config *sync.ConnectionConfig) (*sync.Connection, error) {
	conn := &sync.Connection{
		Config: config,
		Status: sync.ConnectionStatus{
			Connected: true,
			LastCheck: time.Now(),
			Latency:   10,
		},
	}
	m.connections[config.ID] = conn
	return conn, nil
}

func (m *mockConnectionManager) GetConnections(ctx context.Context) ([]*sync.Connection, error) {
	conns := make([]*sync.Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		conns = append(conns, conn)
	}
	return conns, nil
}

func (m *mockConnectionManager) GetConnection(ctx context.Context, id string) (*sync.Connection, error) {
	conn, exists := m.connections[id]
	if !exists {
		return nil, sync.ErrConnectionNotFound
	}
	return conn, nil
}

func (m *mockConnectionManager) UpdateConnection(ctx context.Context, id string, config *sync.ConnectionConfig) error {
	if _, exists := m.connections[id]; !exists {
		return sync.ErrConnectionNotFound
	}
	m.connections[id].Config = config
	return nil
}

func (m *mockConnectionManager) DeleteConnection(ctx context.Context, id string) error {
	if _, exists := m.connections[id]; !exists {
		return sync.ErrConnectionNotFound
	}
	delete(m.connections, id)
	return nil
}

func (m *mockConnectionManager) TestConnection(ctx context.Context, id string) (*sync.ConnectionStatus, error) {
	// Check if connection exists first
	if _, exists := m.connections[id]; !exists {
		return nil, sync.ErrConnectionNotFound
	}

	if status, exists := m.testResults[id]; exists {
		return status, nil
	}
	return &sync.ConnectionStatus{
		Connected: true,
		LastCheck: time.Now(),
		Latency:   15,
	}, nil
}

func (m *mockConnectionManager) TestConnectionConfig(ctx context.Context, config *sync.ConnectionConfig) (*sync.ConnectionStatus, error) {
	// Test connection without saving it
	return &sync.ConnectionStatus{
		Connected: true,
		LastCheck: time.Now(),
		Latency:   20,
	}, nil
}

func (m *mockConnectionManager) Close() error {
	return nil
}

// mockSyncManager is a minimal mock for SyncManager
type mockSyncManager struct{}

func (m *mockSyncManager) CreateSyncConfig(ctx context.Context, config *sync.SyncConfig) error {
	return nil
}

func (m *mockSyncManager) GetSyncConfigs(ctx context.Context, connectionID string) ([]*sync.SyncConfig, error) {
	return []*sync.SyncConfig{}, nil
}

func (m *mockSyncManager) GetSyncConfig(ctx context.Context, id string) (*sync.SyncConfig, error) {
	return nil, sync.ErrSyncConfigNotFound
}

func (m *mockSyncManager) UpdateSyncConfig(ctx context.Context, id string, config *sync.SyncConfig) error {
	return nil
}

func (m *mockSyncManager) DeleteSyncConfig(ctx context.Context, id string) error {
	return nil
}

func (m *mockSyncManager) StartSync(ctx context.Context, configID string) (*sync.SyncJob, error) {
	return nil, nil
}

func (m *mockSyncManager) StopSync(ctx context.Context, jobID string) error {
	return nil
}

func (m *mockSyncManager) GetSyncStatus(ctx context.Context, jobID string) (*sync.SyncJob, error) {
	return nil, nil
}

func (m *mockSyncManager) GetRemoteDatabases(ctx context.Context, connectionID string) ([]string, error) {
	return []string{}, nil
}

func (m *mockSyncManager) GetRemoteTables(ctx context.Context, connectionID, database string) ([]string, error) {
	return []string{}, nil
}

func (m *mockSyncManager) GetRemoteTableSchema(ctx context.Context, connectionID, database, tableName string) (*sync.TableSchema, error) {
	return nil, nil
}

func (m *mockSyncManager) AddTableMapping(ctx context.Context, syncConfigID string, mapping *sync.TableMapping) error {
	return nil
}

func (m *mockSyncManager) UpdateTableMapping(ctx context.Context, mappingID string, mapping *sync.TableMapping) error {
	return nil
}

func (m *mockSyncManager) RemoveTableMapping(ctx context.Context, mappingID string) error {
	return nil
}

func (m *mockSyncManager) GetTableMappings(ctx context.Context, syncConfigID string) ([]*sync.TableMapping, error) {
	return []*sync.TableMapping{}, nil
}

func (m *mockSyncManager) ToggleTableMapping(ctx context.Context, mappingID string, enabled bool) error {
	return nil
}

func (m *mockSyncManager) SetTableSyncMode(ctx context.Context, mappingID string, syncMode sync.SyncMode) error {
	return nil
}

func (m *mockSyncManager) GetJobProgress(ctx context.Context, jobID string) (*sync.JobSummary, error) {
	return nil, nil
}

func (m *mockSyncManager) GetSyncHistory(ctx context.Context, limit, offset int) ([]*sync.JobHistory, error) {
	return []*sync.JobHistory{}, nil
}

func (m *mockSyncManager) GetSyncStatistics(ctx context.Context) (*sync.SyncStatistics, error) {
	return nil, nil
}

func (m *mockSyncManager) GetActiveJobs(ctx context.Context) ([]*sync.JobSummary, error) {
	return []*sync.JobSummary{}, nil
}

func (m *mockSyncManager) GetJobLogs(ctx context.Context, jobID string) ([]*sync.SyncLog, error) {
	return []*sync.SyncLog{}, nil
}

// mockSyncSystemManager wraps both connection and sync managers
type mockSyncSystemManager struct {
	connMgr *mockConnectionManager
	syncMgr *mockSyncManager
}

func (m *mockSyncSystemManager) GetConnectionManager() sync.ConnectionManager {
	return m.connMgr
}

func (m *mockSyncSystemManager) GetSyncManager() sync.SyncManager {
	return m.syncMgr
}

func (m *mockSyncSystemManager) GetMappingManager() sync.MappingManager {
	return nil
}

func (m *mockSyncSystemManager) GetJobEngine() sync.JobEngine {
	return nil
}

func (m *mockSyncSystemManager) GetSyncEngine() sync.SyncEngine {
	return nil
}

func (m *mockSyncSystemManager) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockSyncSystemManager) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockSyncSystemManager) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockSyncSystemManager) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return gin.H{}, nil
}

// setupTestServer creates a test server with mock sync manager
func setupTestServer() (*Server, *mockConnectionManager) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "json",
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := gin.New()
	engine.Use(gin.Recovery())

	connMgr := newMockConnectionManager()
	syncMgr := &mockSyncManager{}

	server := &Server{
		config:      cfg,
		engine:      engine,
		logger:      logger,
		syncManager: &mockSyncSystemManager{connMgr: connMgr, syncMgr: syncMgr},
	}

	// Register routes
	api := server.engine.Group("/api")
	server.registerSyncRoutes(api)

	return server, connMgr
}

func TestConnectionAPI_CreateConnection(t *testing.T) {
	server, _ := setupTestServer()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "Valid connection creation",
			requestBody: sync.ConnectionConfig{
				ID:          "conn-1",
				Name:        "Test Connection",
				Host:        "localhost",
				Port:        3306,
				Username:    "testuser",
				Password:    "testpass",
				Database:    "testdb",
				LocalDBName: "local_testdb",
				SSL:         false,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.True(t, body["success"].(bool))
				assert.NotNil(t, body["data"])
				data := body["data"].(map[string]interface{})
				config := data["config"].(map[string]interface{})
				assert.Equal(t, "Test Connection", config["name"])
			},
		},
		{
			name:           "Invalid request body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.False(t, body["success"].(bool))
				assert.Contains(t, body["error"].(string), "Invalid request body")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/sync/connections", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			tt.checkResponse(t, response)
		})
	}
}

func TestConnectionAPI_GetConnections(t *testing.T) {
	server, connMgr := setupTestServer()

	// Add test connections
	ctx := context.Background()
	_, err := connMgr.AddConnection(ctx, &sync.ConnectionConfig{
		ID:          "conn-1",
		Name:        "Connection 1",
		Host:        "host1",
		Port:        3306,
		Username:    "user1",
		Password:    "pass1",
		Database:    "db1",
		LocalDBName: "local_db1",
	})
	require.NoError(t, err)

	_, err = connMgr.AddConnection(ctx, &sync.ConnectionConfig{
		ID:          "conn-2",
		Name:        "Connection 2",
		Host:        "host2",
		Port:        3306,
		Username:    "user2",
		Password:    "pass2",
		Database:    "db2",
		LocalDBName: "local_db2",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/connections", nil)
	w := httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].([]interface{})
	assert.Equal(t, 2, len(data))

	meta := response["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["total"])
}

func TestConnectionAPI_GetConnection(t *testing.T) {
	server, connMgr := setupTestServer()

	// Add a test connection
	ctx := context.Background()
	_, err := connMgr.AddConnection(ctx, &sync.ConnectionConfig{
		ID:          "conn-1",
		Name:        "Test Connection",
		Host:        "localhost",
		Port:        3306,
		Username:    "testuser",
		Password:    "testpass",
		Database:    "testdb",
		LocalDBName: "local_testdb",
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		connectionID   string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "Get existing connection",
			connectionID:   "conn-1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.True(t, body["success"].(bool))
				data := body["data"].(map[string]interface{})
				config := data["config"].(map[string]interface{})
				assert.Equal(t, "Test Connection", config["name"])
			},
		},
		{
			name:           "Get non-existent connection",
			connectionID:   "non-existent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.False(t, body["success"].(bool))
				assert.NotNil(t, body["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/sync/connections/"+tt.connectionID, nil)
			w := httptest.NewRecorder()

			server.engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			tt.checkResponse(t, response)
		})
	}
}

func TestConnectionAPI_UpdateConnection(t *testing.T) {
	server, connMgr := setupTestServer()

	// Add a test connection
	ctx := context.Background()
	_, err := connMgr.AddConnection(ctx, &sync.ConnectionConfig{
		ID:          "conn-1",
		Name:        "Original Name",
		Host:        "localhost",
		Port:        3306,
		Username:    "testuser",
		Password:    "testpass",
		Database:    "testdb",
		LocalDBName: "local_testdb",
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		connectionID   string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:         "Update existing connection",
			connectionID: "conn-1",
			requestBody: sync.ConnectionConfig{
				Name:        "Updated Name",
				Host:        "newhost",
				Port:        3307,
				Username:    "newuser",
				Password:    "newpass",
				Database:    "newdb",
				LocalDBName: "new_local_db",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.True(t, body["success"].(bool))
				assert.Contains(t, body["message"].(string), "updated successfully")
			},
		},
		{
			name:           "Update non-existent connection",
			connectionID:   "non-existent",
			requestBody:    sync.ConnectionConfig{Name: "Test"},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.False(t, body["success"].(bool))
			},
		},
		{
			name:           "Invalid request body",
			connectionID:   "conn-1",
			requestBody:    "invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.False(t, body["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/sync/connections/"+tt.connectionID, bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			tt.checkResponse(t, response)
		})
	}
}

func TestConnectionAPI_DeleteConnection(t *testing.T) {
	server, connMgr := setupTestServer()

	// Add a test connection
	ctx := context.Background()
	_, err := connMgr.AddConnection(ctx, &sync.ConnectionConfig{
		ID:          "conn-1",
		Name:        "Test Connection",
		Host:        "localhost",
		Port:        3306,
		Username:    "testuser",
		Password:    "testpass",
		Database:    "testdb",
		LocalDBName: "local_testdb",
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		connectionID   string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "Delete existing connection",
			connectionID:   "conn-1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.True(t, body["success"].(bool))
				assert.Contains(t, body["message"].(string), "deleted successfully")
			},
		},
		{
			name:           "Delete non-existent connection",
			connectionID:   "non-existent",
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.False(t, body["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/sync/connections/"+tt.connectionID, nil)
			w := httptest.NewRecorder()

			server.engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			tt.checkResponse(t, response)
		})
	}
}

func TestConnectionAPI_TestConnection(t *testing.T) {
	server, connMgr := setupTestServer()

	// Add a test connection
	ctx := context.Background()
	_, err := connMgr.AddConnection(ctx, &sync.ConnectionConfig{
		ID:          "conn-1",
		Name:        "Test Connection",
		Host:        "localhost",
		Port:        3306,
		Username:    "testuser",
		Password:    "testpass",
		Database:    "testdb",
		LocalDBName: "local_testdb",
	})
	require.NoError(t, err)

	// Set up test result for successful connection
	connMgr.testResults["conn-1"] = &sync.ConnectionStatus{
		Connected: true,
		LastCheck: time.Now(),
		Latency:   20,
	}

	// Set up test result for failed connection
	connMgr.testResults["conn-failed"] = &sync.ConnectionStatus{
		Connected: false,
		LastCheck: time.Now(),
		Error:     "Connection refused",
	}

	_, err = connMgr.AddConnection(ctx, &sync.ConnectionConfig{
		ID:          "conn-failed",
		Name:        "Failed Connection",
		Host:        "invalid-host",
		Port:        3306,
		Username:    "testuser",
		Password:    "testpass",
		Database:    "testdb",
		LocalDBName: "local_testdb2",
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		connectionID   string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "Test successful connection",
			connectionID:   "conn-1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.True(t, body["success"].(bool))
				data := body["data"].(map[string]interface{})
				assert.True(t, data["connected"].(bool))
				assert.NotNil(t, data["last_check"])
			},
		},
		{
			name:           "Test non-existent connection",
			connectionID:   "non-existent",
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.False(t, body["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/sync/connections/"+tt.connectionID+"/test", nil)
			w := httptest.NewRecorder()

			server.engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			tt.checkResponse(t, response)
		})
	}
}

func TestConnectionAPI_GetConnectionStatus(t *testing.T) {
	server, connMgr := setupTestServer()

	// Add connections with different statuses
	ctx := context.Background()

	// Connected connection
	_, err := connMgr.AddConnection(ctx, &sync.ConnectionConfig{
		ID:          "conn-connected",
		Name:        "Connected",
		Host:        "localhost",
		Port:        3306,
		Username:    "user",
		Password:    "pass",
		Database:    "db",
		LocalDBName: "local_db",
	})
	require.NoError(t, err)

	connMgr.testResults["conn-connected"] = &sync.ConnectionStatus{
		Connected: true,
		LastCheck: time.Now(),
		Latency:   10,
	}

	// Get all connections and verify status is included
	req := httptest.NewRequest(http.MethodGet, "/api/sync/connections", nil)
	w := httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].([]interface{})
	assert.Greater(t, len(data), 0)

	// Verify first connection has status
	firstConn := data[0].(map[string]interface{})
	assert.NotNil(t, firstConn["status"])
	status := firstConn["status"].(map[string]interface{})
	assert.NotNil(t, status["connected"])
	assert.NotNil(t, status["last_check"])
}
