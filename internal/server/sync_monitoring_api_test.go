package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"db-taxi/internal/sync"
)

// TestGetSyncStatus tests the sync status endpoint
// Requirement 5.1: Real-time display of sync progress and status
func TestGetSyncStatus(t *testing.T) {
	server, mockSyncMgr, _ := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("HealthCheck", mock.Anything).Return(nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/status", nil)
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
	assert.Equal(t, "healthy", data["status"])
	assert.NotNil(t, data["timestamp"])

	mockSyncMgr.AssertExpectations(t)
}

// TestGetSyncStats tests the sync statistics endpoint
// Requirement 5.4: Display statistics information including data volume and time consumption
func TestGetSyncStats(t *testing.T) {
	server, mockSyncMgr, _ := setupSyncConfigTestServer()

	// Setup mock expectations
	stats := map[string]interface{}{
		"total_jobs":                   int64(100),
		"completed_jobs":               int64(85),
		"failed_jobs":                  int64(10),
		"running_jobs":                 int64(5),
		"total_rows_synced":            int64(1000000),
		"total_tables_synced":          int64(250),
		"average_job_duration_minutes": 15.5,
		"error_rate_percentage":        10.0,
		"sync_frequency_per_hour":      4.2,
		"last_sync_time":               time.Now().Format(time.RFC3339),
	}
	mockSyncMgr.On("GetStats", mock.Anything).Return(stats, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/stats", nil)
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
	assert.Equal(t, float64(100), data["total_jobs"])
	assert.Equal(t, float64(85), data["completed_jobs"])
	assert.Equal(t, float64(10), data["failed_jobs"])
	assert.Equal(t, float64(5), data["running_jobs"])
	assert.Equal(t, float64(1000000), data["total_rows_synced"])

	mockSyncMgr.AssertExpectations(t)
}

// TestGetSyncJobHistory tests the sync job history endpoint
// Requirement 5.2: Display historical sync records and results
func TestGetSyncJobHistory(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	now := time.Now()
	endTime := now.Add(10 * time.Minute)
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("GetSyncHistory", mock.Anything, 50, 0).Return([]*sync.JobHistory{
		{
			SyncJob: &sync.SyncJob{
				ID:              "job-1",
				ConfigID:        "config-1",
				Status:          sync.JobStatusCompleted,
				StartTime:       now,
				EndTime:         &endTime,
				TotalTables:     5,
				CompletedTables: 5,
				TotalRows:       10000,
				ProcessedRows:   10000,
			},
			ConfigName:     "Test Config",
			ConnectionName: "Test Connection",
		},
		{
			SyncJob: &sync.SyncJob{
				ID:              "job-2",
				ConfigID:        "config-1",
				Status:          sync.JobStatusFailed,
				StartTime:       now.Add(-1 * time.Hour),
				EndTime:         &now,
				TotalTables:     3,
				CompletedTables: 2,
				TotalRows:       5000,
				ProcessedRows:   3000,
				Error:           "Connection timeout",
			},
			ConfigName:     "Test Config",
			ConnectionName: "Test Connection",
		},
	}, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/jobs/history", nil)
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
	assert.Equal(t, 2, len(data))

	// Check first job
	job1 := data[0].(map[string]interface{})
	assert.Equal(t, "job-1", job1["id"])
	assert.Equal(t, "completed", job1["status"])
	assert.Equal(t, "Test Config", job1["config_name"])

	// Check second job
	job2 := data[1].(map[string]interface{})
	assert.Equal(t, "job-2", job2["id"])
	assert.Equal(t, "failed", job2["status"])
	assert.Equal(t, "Connection timeout", job2["error"])

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}

// TestGetSyncJobHistoryWithPagination tests the sync job history endpoint with pagination
// Requirement 5.2: Display historical sync records and results
func TestGetSyncJobHistoryWithPagination(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("GetSyncHistory", mock.Anything, 10, 20).Return([]*sync.JobHistory{}, nil)

	// Create request with pagination parameters
	req, _ := http.NewRequest("GET", "/api/sync/jobs/history?limit=10&offset=20", nil)
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	meta := response["meta"].(map[string]interface{})
	assert.Equal(t, float64(10), meta["limit"])
	assert.Equal(t, float64(20), meta["offset"])

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}

// TestGetSyncJobLogs tests the sync job logs endpoint
// Requirement 5.3: Display detailed error information and suggestions when sync fails
func TestGetSyncJobLogs(t *testing.T) {
	server, mockSyncMgr, mockSyncMgrService := setupSyncConfigTestServer()

	// Setup mock expectations
	mockSyncMgr.On("GetSyncManager").Return(mockSyncMgrService)
	mockSyncMgrService.On("GetJobLogs", mock.Anything, "job-123").Return([]*sync.SyncLog{
		{
			ID:        1,
			JobID:     "job-123",
			TableName: "users",
			Level:     "info",
			Message:   "Starting table sync",
			CreatedAt: time.Now(),
		},
		{
			ID:        2,
			JobID:     "job-123",
			TableName: "users",
			Level:     "error",
			Message:   "Failed to sync table: connection timeout",
			CreatedAt: time.Now(),
		},
	}, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/jobs/job-123/logs", nil)
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
	assert.Equal(t, 2, len(data))

	// Check first log
	log1 := data[0].(map[string]interface{})
	assert.Equal(t, "users", log1["table_name"])
	assert.Equal(t, "info", log1["level"])
	assert.Equal(t, "Starting table sync", log1["message"])

	// Check second log
	log2 := data[1].(map[string]interface{})
	assert.Equal(t, "error", log2["level"])
	assert.Contains(t, log2["message"], "connection timeout")

	mockSyncMgr.AssertExpectations(t)
	mockSyncMgrService.AssertExpectations(t)
}

// TestGetSyncStatusUnavailable tests the sync status endpoint when sync system is unavailable
func TestGetSyncStatusUnavailable(t *testing.T) {
	server, _, _ := setupSyncConfigTestServer()
	server.syncManager = nil // Simulate unavailable sync system

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/status", nil)
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"], "Sync system not available")
}

// TestGetSyncStatsError tests the sync stats endpoint when an error occurs
func TestGetSyncStatsError(t *testing.T) {
	server, mockSyncMgr, _ := setupSyncConfigTestServer()

	// Setup mock expectations to return error
	mockSyncMgr.On("GetStats", mock.Anything).Return(map[string]interface{}{}, assert.AnError)

	// Create request
	req, _ := http.NewRequest("GET", "/api/sync/stats", nil)
	w := httptest.NewRecorder()

	// Execute request
	server.engine.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockSyncMgr.AssertExpectations(t)
}
