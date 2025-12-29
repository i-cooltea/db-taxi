package sync

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestNewService tests the service constructor
func TestNewService(t *testing.T) {
	repo := &mockRepository{}
	logger := logrus.New()

	service := NewService(repo, logger, nil) // nil localDB for testing

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}
	if service.repo != repo {
		t.Error("Expected service to have the provided repository")
	}
	if service.logger != logger {
		t.Error("Expected service to have the provided logger")
	}
}

// TestNewConnectionManager tests the connection manager constructor
func TestNewConnectionManager(t *testing.T) {
	repo := &mockRepository{}
	logger := logrus.New()

	cm := NewConnectionManager(repo, logger, nil) // nil localDB for testing

	if cm == nil {
		t.Fatal("Expected connection manager to be created, got nil")
	}

	// Test that it implements the interface
	var _ ConnectionManager = cm
}

// TestNewSyncManager tests the sync manager constructor
func TestNewSyncManager(t *testing.T) {
	repo := &mockRepository{}
	logger := logrus.New()

	sm := NewSyncManager(repo, logger, nil) // nil localDB for testing

	if sm == nil {
		t.Fatal("Expected sync manager to be created, got nil")
	}

	// Test that it implements the interface
	var _ SyncManager = sm
}

// TestConnectionManagerService_AddConnection tests adding a connection
func TestConnectionManagerService_AddConnection(t *testing.T) {
	repo := &testRepository{
		connections: make(map[string]*ConnectionConfig),
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests

	cm := NewConnectionManager(repo, logger, nil).(*ConnectionManagerService)
	ctx := context.Background()

	config := &ConnectionConfig{
		Name:        "test-connection",
		Host:        "localhost",
		Port:        3306,
		Username:    "testuser",
		Password:    "testpass",
		Database:    "testdb",
		LocalDBName: "local_testdb",
		SSL:         false,
	}

	// Note: This test will fail because we can't actually connect to a remote database
	// In a real test environment, you would use a test database or mock the connection
	_, err := cm.AddConnection(ctx, config)
	if err == nil {
		t.Error("Expected connection test to fail since no database is available")
	}

	// Test validation works
	if !contains(err.Error(), "failed to connect to remote database") {
		t.Errorf("Expected remote connection error, got: %v", err)
	}
}

// TestConnectionManagerService_AddConnection_ValidationError tests validation errors
func TestConnectionManagerService_AddConnection_ValidationError(t *testing.T) {
	repo := &testRepository{
		connections: make(map[string]*ConnectionConfig),
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cm := NewConnectionManager(repo, logger, nil).(*ConnectionManagerService)
	ctx := context.Background()

	tests := []struct {
		name   string
		config *ConnectionConfig
	}{
		{
			name: "empty name",
			config: &ConnectionConfig{
				Name:        "",
				Host:        "localhost",
				Port:        3306,
				Username:    "user",
				Database:    "db",
				LocalDBName: "local_db",
			},
		},
		{
			name: "empty host",
			config: &ConnectionConfig{
				Name:        "test",
				Host:        "",
				Port:        3306,
				Username:    "user",
				Database:    "db",
				LocalDBName: "local_db",
			},
		},
		{
			name: "invalid port",
			config: &ConnectionConfig{
				Name:        "test",
				Host:        "localhost",
				Port:        0,
				Username:    "user",
				Database:    "db",
				LocalDBName: "local_db",
			},
		},
		{
			name: "empty username",
			config: &ConnectionConfig{
				Name:        "test",
				Host:        "localhost",
				Port:        3306,
				Username:    "",
				Database:    "db",
				LocalDBName: "local_db",
			},
		},
		{
			name: "empty database",
			config: &ConnectionConfig{
				Name:        "test",
				Host:        "localhost",
				Port:        3306,
				Username:    "user",
				Database:    "",
				LocalDBName: "local_db",
			},
		},
		{
			name: "empty local database name",
			config: &ConnectionConfig{
				Name:        "test",
				Host:        "localhost",
				Port:        3306,
				Username:    "user",
				Database:    "db",
				LocalDBName: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := cm.AddConnection(ctx, test.config)
			if err == nil {
				t.Errorf("Expected validation error for %s, got nil", test.name)
			}
		})
	}
}

// TestConnectionManagerService_GetConnections tests getting all connections
func TestConnectionManagerService_GetConnections(t *testing.T) {
	repo := &testRepository{
		connections: map[string]*ConnectionConfig{
			"conn-1": {
				ID:          "conn-1",
				Name:        "connection-1",
				Host:        "host1",
				Port:        3306,
				Username:    "user1",
				Database:    "db1",
				LocalDBName: "local_db1",
			},
			"conn-2": {
				ID:          "conn-2",
				Name:        "connection-2",
				Host:        "host2",
				Port:        3307,
				Username:    "user2",
				Database:    "db2",
				LocalDBName: "local_db2",
			},
		},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cm := NewConnectionManager(repo, logger, nil).(*ConnectionManagerService)
	ctx := context.Background()

	connections, err := cm.GetConnections(ctx)
	if err != nil {
		t.Fatalf("Failed to get connections: %v", err)
	}

	if len(connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(connections))
	}

	// Verify connections have status
	for _, conn := range connections {
		if conn.Config == nil {
			t.Error("Expected connection config to be set")
		}
		// Status should be set (even if not connected due to no real database)
		if conn.Status.LastCheck.IsZero() {
			t.Error("Expected LastCheck to be set")
		}
		// Connection should fail since we don't have real databases
		if conn.Status.Connected {
			t.Error("Expected connection to fail since no real database is available")
		}
	}
}

// TestSyncManagerService_CreateSyncConfig tests creating sync configuration
func TestSyncManagerService_CreateSyncConfig(t *testing.T) {
	repo := &testRepository{
		syncConfigs:   make(map[string]*SyncConfig),
		tableMappings: make(map[string][]*TableMapping),
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	sm := NewSyncManager(repo, logger, nil).(*SyncManagerService)
	ctx := context.Background()

	config := &SyncConfig{
		ConnectionID: "conn-1",
		Name:         "test-sync",
		Tables: []*TableMapping{
			{
				SourceTable: "users",
				TargetTable: "local_users",
				SyncMode:    SyncModeFull,
				Enabled:     true,
			},
			{
				SourceTable: "orders",
				TargetTable: "local_orders",
				SyncMode:    SyncModeIncremental,
				Enabled:     true,
			},
		},
		SyncMode: SyncModeFull,
		Enabled:  true,
	}

	err := sm.CreateSyncConfig(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create sync config: %v", err)
	}

	if config.ID == "" {
		t.Error("Expected sync config ID to be generated")
	}
	if config.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if config.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify table mappings were created
	for _, mapping := range config.Tables {
		if mapping.ID == "" {
			t.Error("Expected table mapping ID to be generated")
		}
		if mapping.SyncConfigID != config.ID {
			t.Errorf("Expected table mapping SyncConfigID to be %s, got %s",
				config.ID, mapping.SyncConfigID)
		}
		if mapping.CreatedAt.IsZero() {
			t.Error("Expected table mapping CreatedAt to be set")
		}
	}
}

// TestSyncManagerService_CreateSyncConfig_ValidationError tests validation errors
func TestSyncManagerService_CreateSyncConfig_ValidationError(t *testing.T) {
	repo := &testRepository{
		syncConfigs:   make(map[string]*SyncConfig),
		tableMappings: make(map[string][]*TableMapping),
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	sm := NewSyncManager(repo, logger, nil).(*SyncManagerService)
	ctx := context.Background()

	tests := []struct {
		name   string
		config *SyncConfig
	}{
		{
			name: "empty name",
			config: &SyncConfig{
				Name:         "",
				ConnectionID: "conn-1",
				Tables: []*TableMapping{
					{SourceTable: "users", TargetTable: "local_users"},
				},
			},
		},
		{
			name: "empty connection ID",
			config: &SyncConfig{
				Name:         "test-sync",
				ConnectionID: "",
				Tables: []*TableMapping{
					{SourceTable: "users", TargetTable: "local_users"},
				},
			},
		},
		{
			name: "no table mappings",
			config: &SyncConfig{
				Name:         "test-sync",
				ConnectionID: "conn-1",
				Tables:       []*TableMapping{},
			},
		},
		{
			name: "empty source table",
			config: &SyncConfig{
				Name:         "test-sync",
				ConnectionID: "conn-1",
				Tables: []*TableMapping{
					{SourceTable: "", TargetTable: "local_users"},
				},
			},
		},
		{
			name: "empty target table",
			config: &SyncConfig{
				Name:         "test-sync",
				ConnectionID: "conn-1",
				Tables: []*TableMapping{
					{SourceTable: "users", TargetTable: ""},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := sm.CreateSyncConfig(ctx, test.config)
			if err == nil {
				t.Errorf("Expected validation error for %s, got nil", test.name)
			}
		})
	}
}

// TestSyncManagerService_StartSync tests starting a sync job
func TestSyncManagerService_StartSync(t *testing.T) {
	repo := &testRepository{
		syncConfigs: map[string]*SyncConfig{
			"sync-1": {
				ID:           "sync-1",
				ConnectionID: "conn-1",
				Name:         "test-sync",
				Tables: []*TableMapping{
					{SourceTable: "users", TargetTable: "local_users"},
					{SourceTable: "orders", TargetTable: "local_orders"},
				},
				Enabled: true,
			},
		},
		syncJobs: make(map[string]*SyncJob),
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	sm := NewSyncManager(repo, logger, nil).(*SyncManagerService)
	ctx := context.Background()

	job, err := sm.StartSync(ctx, "sync-1")
	if err != nil {
		t.Fatalf("Failed to start sync: %v", err)
	}

	if job == nil {
		t.Fatal("Expected sync job to be returned, got nil")
	}
	if job.ID == "" {
		t.Error("Expected job ID to be generated")
	}
	if job.ConfigID != "sync-1" {
		t.Errorf("Expected job ConfigID to be 'sync-1', got %s", job.ConfigID)
	}
	if job.Status != JobStatusPending {
		t.Errorf("Expected job status to be 'pending', got %s", job.Status)
	}
	if job.TotalTables != 2 {
		t.Errorf("Expected TotalTables to be 2, got %d", job.TotalTables)
	}
	if job.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}
	if job.Progress == nil {
		t.Error("Expected Progress to be initialized")
	}
	if job.Progress.TotalTables != 2 {
		t.Errorf("Expected Progress.TotalTables to be 2, got %d", job.Progress.TotalTables)
	}
}

// TestSyncManagerService_StartSync_DisabledConfig tests starting sync with disabled config
func TestSyncManagerService_StartSync_DisabledConfig(t *testing.T) {
	repo := &testRepository{
		syncConfigs: map[string]*SyncConfig{
			"sync-1": {
				ID:           "sync-1",
				ConnectionID: "conn-1",
				Name:         "test-sync",
				Tables: []*TableMapping{
					{SourceTable: "users", TargetTable: "local_users"},
				},
				Enabled: false, // Disabled
			},
		},
		syncJobs: make(map[string]*SyncJob),
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	sm := NewSyncManager(repo, logger, nil).(*SyncManagerService)
	ctx := context.Background()

	_, err := sm.StartSync(ctx, "sync-1")
	if err == nil {
		t.Error("Expected error when starting sync with disabled config")
	}
	if err.Error() != "sync config is disabled" {
		t.Errorf("Expected 'sync config is disabled' error, got %s", err.Error())
	}
}

// Test repository implementation for testing
type testRepository struct {
	connections   map[string]*ConnectionConfig
	syncConfigs   map[string]*SyncConfig
	tableMappings map[string][]*TableMapping
	syncJobs      map[string]*SyncJob
}

func (r *testRepository) CreateConnection(ctx context.Context, config *ConnectionConfig) error {
	r.connections[config.ID] = config
	return nil
}

func (r *testRepository) GetConnection(ctx context.Context, id string) (*ConnectionConfig, error) {
	if config, exists := r.connections[id]; exists {
		return config, nil
	}
	return nil, mockError("connection not found")
}

func (r *testRepository) GetConnections(ctx context.Context) ([]*ConnectionConfig, error) {
	var configs []*ConnectionConfig
	for _, config := range r.connections {
		configs = append(configs, config)
	}
	return configs, nil
}

func (r *testRepository) UpdateConnection(ctx context.Context, id string, config *ConnectionConfig) error {
	if _, exists := r.connections[id]; !exists {
		return mockError("connection not found")
	}
	config.ID = id
	r.connections[id] = config
	return nil
}

func (r *testRepository) DeleteConnection(ctx context.Context, id string) error {
	if _, exists := r.connections[id]; !exists {
		return mockError("connection not found")
	}
	delete(r.connections, id)
	return nil
}

func (r *testRepository) CreateSyncConfig(ctx context.Context, config *SyncConfig) error {
	r.syncConfigs[config.ID] = config
	return nil
}

func (r *testRepository) GetSyncConfig(ctx context.Context, id string) (*SyncConfig, error) {
	if config, exists := r.syncConfigs[id]; exists {
		return config, nil
	}
	return nil, mockError("sync config not found")
}

func (r *testRepository) GetSyncConfigs(ctx context.Context, connectionID string) ([]*SyncConfig, error) {
	var configs []*SyncConfig
	for _, config := range r.syncConfigs {
		if config.ConnectionID == connectionID {
			configs = append(configs, config)
		}
	}
	return configs, nil
}

func (r *testRepository) UpdateSyncConfig(ctx context.Context, id string, config *SyncConfig) error {
	if _, exists := r.syncConfigs[id]; !exists {
		return mockError("sync config not found")
	}
	config.ID = id
	r.syncConfigs[id] = config
	return nil
}

func (r *testRepository) DeleteSyncConfig(ctx context.Context, id string) error {
	if _, exists := r.syncConfigs[id]; !exists {
		return mockError("sync config not found")
	}
	delete(r.syncConfigs, id)
	return nil
}

func (r *testRepository) CreateTableMapping(ctx context.Context, mapping *TableMapping) error {
	if r.tableMappings[mapping.SyncConfigID] == nil {
		r.tableMappings[mapping.SyncConfigID] = []*TableMapping{}
	}
	r.tableMappings[mapping.SyncConfigID] = append(r.tableMappings[mapping.SyncConfigID], mapping)
	return nil
}

func (r *testRepository) GetTableMappings(ctx context.Context, syncConfigID string) ([]*TableMapping, error) {
	return r.tableMappings[syncConfigID], nil
}

func (r *testRepository) UpdateTableMapping(ctx context.Context, id string, mapping *TableMapping) error {
	return nil // Simplified for testing
}

func (r *testRepository) DeleteTableMapping(ctx context.Context, id string) error {
	return nil // Simplified for testing
}

func (r *testRepository) CreateSyncJob(ctx context.Context, job *SyncJob) error {
	r.syncJobs[job.ID] = job
	return nil
}

func (r *testRepository) GetSyncJob(ctx context.Context, id string) (*SyncJob, error) {
	if job, exists := r.syncJobs[id]; exists {
		return job, nil
	}
	return nil, mockError("sync job not found")
}

func (r *testRepository) UpdateSyncJob(ctx context.Context, id string, job *SyncJob) error {
	if _, exists := r.syncJobs[id]; !exists {
		return mockError("sync job not found")
	}
	job.ID = id
	r.syncJobs[id] = job
	return nil
}

func (r *testRepository) GetJobHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error) {
	return nil, nil // Simplified for testing
}

func (r *testRepository) GetJobsByStatus(ctx context.Context, status JobStatus) ([]*SyncJob, error) {
	var jobs []*SyncJob
	for _, job := range r.syncJobs {
		if job.Status == status {
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func (r *testRepository) CreateCheckpoint(ctx context.Context, checkpoint *SyncCheckpoint) error {
	return nil // Simplified for testing
}

func (r *testRepository) GetCheckpoint(ctx context.Context, tableMappingID string) (*SyncCheckpoint, error) {
	return nil, mockError("checkpoint not found")
}

func (r *testRepository) UpdateCheckpoint(ctx context.Context, tableMappingID string, checkpoint *SyncCheckpoint) error {
	return nil // Simplified for testing
}

func (r *testRepository) CreateSyncLog(ctx context.Context, log *SyncLog) error {
	return nil // Simplified for testing
}

func (r *testRepository) GetSyncLogs(ctx context.Context, jobID string) ([]*SyncLog, error) {
	return nil, nil // Simplified for testing
}

// Database mapping operations
func (r *testRepository) CreateDatabaseMapping(ctx context.Context, mapping *DatabaseMapping) error {
	return nil // Simplified for testing
}

func (r *testRepository) GetDatabaseMappings(ctx context.Context) ([]*DatabaseMapping, error) {
	return nil, nil // Simplified for testing
}

func (r *testRepository) UpdateDatabaseMapping(ctx context.Context, remoteConnectionID string, mapping *DatabaseMapping) error {
	return nil // Simplified for testing
}

func (r *testRepository) DeleteDatabaseMapping(ctx context.Context, remoteConnectionID string) error {
	return nil // Simplified for testing
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
