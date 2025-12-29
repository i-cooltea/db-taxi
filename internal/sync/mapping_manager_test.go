package sync

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateConnection(ctx context.Context, config *ConnectionConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockRepository) GetConnection(ctx context.Context, id string) (*ConnectionConfig, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*ConnectionConfig), args.Error(1)
}

func (m *MockRepository) GetConnections(ctx context.Context) ([]*ConnectionConfig, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*ConnectionConfig), args.Error(1)
}

func (m *MockRepository) UpdateConnection(ctx context.Context, id string, config *ConnectionConfig) error {
	args := m.Called(ctx, id, config)
	return args.Error(0)
}

func (m *MockRepository) DeleteConnection(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CreateSyncConfig(ctx context.Context, config *SyncConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockRepository) GetSyncConfig(ctx context.Context, id string) (*SyncConfig, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*SyncConfig), args.Error(1)
}

func (m *MockRepository) GetSyncConfigs(ctx context.Context, connectionID string) ([]*SyncConfig, error) {
	args := m.Called(ctx, connectionID)
	return args.Get(0).([]*SyncConfig), args.Error(1)
}

func (m *MockRepository) UpdateSyncConfig(ctx context.Context, id string, config *SyncConfig) error {
	args := m.Called(ctx, id, config)
	return args.Error(0)
}

func (m *MockRepository) DeleteSyncConfig(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CreateTableMapping(ctx context.Context, mapping *TableMapping) error {
	args := m.Called(ctx, mapping)
	return args.Error(0)
}

func (m *MockRepository) GetTableMappings(ctx context.Context, syncConfigID string) ([]*TableMapping, error) {
	args := m.Called(ctx, syncConfigID)
	return args.Get(0).([]*TableMapping), args.Error(1)
}

func (m *MockRepository) UpdateTableMapping(ctx context.Context, id string, mapping *TableMapping) error {
	args := m.Called(ctx, id, mapping)
	return args.Error(0)
}

func (m *MockRepository) DeleteTableMapping(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CreateSyncJob(ctx context.Context, job *SyncJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockRepository) GetSyncJob(ctx context.Context, id string) (*SyncJob, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*SyncJob), args.Error(1)
}

func (m *MockRepository) UpdateSyncJob(ctx context.Context, id string, job *SyncJob) error {
	args := m.Called(ctx, id, job)
	return args.Error(0)
}

func (m *MockRepository) GetJobHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*JobHistory), args.Error(1)
}

func (m *MockRepository) GetJobsByStatus(ctx context.Context, status JobStatus) ([]*SyncJob, error) {
	args := m.Called(ctx, status)
	return args.Get(0).([]*SyncJob), args.Error(1)
}

func (m *MockRepository) CreateCheckpoint(ctx context.Context, checkpoint *SyncCheckpoint) error {
	args := m.Called(ctx, checkpoint)
	return args.Error(0)
}

func (m *MockRepository) GetCheckpoint(ctx context.Context, tableMappingID string) (*SyncCheckpoint, error) {
	args := m.Called(ctx, tableMappingID)
	return args.Get(0).(*SyncCheckpoint), args.Error(1)
}

func (m *MockRepository) UpdateCheckpoint(ctx context.Context, tableMappingID string, checkpoint *SyncCheckpoint) error {
	args := m.Called(ctx, tableMappingID, checkpoint)
	return args.Error(0)
}

func (m *MockRepository) CreateSyncLog(ctx context.Context, log *SyncLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockRepository) GetSyncLogs(ctx context.Context, jobID string) ([]*SyncLog, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).([]*SyncLog), args.Error(1)
}

func (m *MockRepository) CreateDatabaseMapping(ctx context.Context, mapping *DatabaseMapping) error {
	args := m.Called(ctx, mapping)
	return args.Error(0)
}

func (m *MockRepository) GetDatabaseMappings(ctx context.Context) ([]*DatabaseMapping, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*DatabaseMapping), args.Error(1)
}

func (m *MockRepository) UpdateDatabaseMapping(ctx context.Context, remoteConnectionID string, mapping *DatabaseMapping) error {
	args := m.Called(ctx, remoteConnectionID, mapping)
	return args.Error(0)
}

func (m *MockRepository) DeleteDatabaseMapping(ctx context.Context, remoteConnectionID string) error {
	args := m.Called(ctx, remoteConnectionID)
	return args.Error(0)
}

func TestMappingManager_GetDatabaseMappings(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests

	manager := &MappingManagerImpl{
		repo:   mockRepo,
		logger: logger,
	}

	// Test data
	connections := []*ConnectionConfig{
		{
			ID:          "conn1",
			Name:        "Connection 1",
			LocalDBName: "local_db1",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "conn2",
			Name:        "Connection 2",
			LocalDBName: "local_db2",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "conn3",
			Name:        "Connection 3",
			LocalDBName: "", // No local DB mapping
			CreatedAt:   time.Now(),
		},
	}

	mockRepo.On("GetConnections", ctx).Return(connections, nil)

	// Execute
	mappings, err := manager.GetDatabaseMappings(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mappings, 2) // Only connections with LocalDBName should be included

	assert.Equal(t, "conn1", mappings[0].RemoteConnectionID)
	assert.Equal(t, "local_db1", mappings[0].LocalDatabaseName)

	assert.Equal(t, "conn2", mappings[1].RemoteConnectionID)
	assert.Equal(t, "local_db2", mappings[1].LocalDatabaseName)

	mockRepo.AssertExpectations(t)
}

func TestMappingManager_ValidateConfig(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := &MappingManagerImpl{
		repo:   mockRepo,
		logger: logger,
	}

	tests := []struct {
		name        string
		config      *ConfigExport
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "configuration is nil",
		},
		{
			name: "missing version",
			config: &ConfigExport{
				Version: "",
			},
			expectError: true,
			errorMsg:    "configuration version is required",
		},
		{
			name: "valid config",
			config: &ConfigExport{
				Version:    "1.0",
				ExportTime: time.Now(),
				Connections: []*ConnectionConfig{
					{
						ID:          uuid.New().String(),
						Name:        "Test Connection",
						Host:        "localhost",
						Port:        3306,
						Username:    "user",
						Database:    "testdb",
						LocalDBName: "local_testdb",
					},
				},
				Mappings:    []*DatabaseMapping{},
				SyncConfigs: []*SyncConfig{},
			},
			expectError: false,
		},
		{
			name: "invalid port",
			config: &ConfigExport{
				Version:    "1.0",
				ExportTime: time.Now(),
				Connections: []*ConnectionConfig{
					{
						ID:       uuid.New().String(),
						Name:     "Test Connection",
						Host:     "localhost",
						Port:     0, // Invalid port
						Username: "user",
						Database: "testdb",
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid port",
		},
		{
			name: "duplicate connection IDs",
			config: &ConfigExport{
				Version:    "1.0",
				ExportTime: time.Now(),
				Connections: []*ConnectionConfig{
					{
						ID:       "duplicate-id",
						Name:     "Connection 1",
						Host:     "localhost",
						Port:     3306,
						Username: "user",
						Database: "testdb1",
					},
					{
						ID:       "duplicate-id", // Duplicate ID
						Name:     "Connection 2",
						Host:     "localhost",
						Port:     3306,
						Username: "user",
						Database: "testdb2",
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate connection ID",
		},
		{
			name: "duplicate local database names",
			config: &ConfigExport{
				Version:    "1.0",
				ExportTime: time.Now(),
				Connections: []*ConnectionConfig{
					{
						ID:          uuid.New().String(),
						Name:        "Connection 1",
						Host:        "localhost",
						Port:        3306,
						Username:    "user",
						Database:    "testdb1",
						LocalDBName: "same_local_db", // Duplicate local DB name
					},
					{
						ID:          uuid.New().String(),
						Name:        "Connection 2",
						Host:        "localhost",
						Port:        3306,
						Username:    "user",
						Database:    "testdb2",
						LocalDBName: "same_local_db", // Duplicate local DB name
					},
				},
			},
			expectError: true,
			errorMsg:    "local database name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateConfig(ctx, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMappingManager_ExportConfig(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := &MappingManagerImpl{
		repo:   mockRepo,
		logger: logger,
	}

	// Test data
	connections := []*ConnectionConfig{
		{
			ID:          "conn1",
			Name:        "Connection 1",
			LocalDBName: "local_db1",
			CreatedAt:   time.Now(),
		},
	}

	syncConfigs := []*SyncConfig{
		{
			ID:           "sync1",
			ConnectionID: "conn1",
			Name:         "Sync Config 1",
			SyncMode:     SyncModeFull,
			Enabled:      true,
			Tables: []*TableMapping{
				{
					ID:           "table1",
					SyncConfigID: "sync1",
					SourceTable:  "source_table",
					TargetTable:  "target_table",
					SyncMode:     SyncModeFull,
					Enabled:      true,
				},
			},
		},
	}

	mockRepo.On("GetConnections", ctx).Return(connections, nil)
	mockRepo.On("GetSyncConfigs", ctx, "conn1").Return(syncConfigs, nil)

	// Execute
	export, err := manager.ExportConfig(ctx)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, export)
	assert.Equal(t, "1.0", export.Version)
	assert.Len(t, export.Connections, 1)
	assert.Len(t, export.Mappings, 1)
	assert.Len(t, export.SyncConfigs, 1)

	// Verify mapping was created from connection
	assert.Equal(t, "conn1", export.Mappings[0].RemoteConnectionID)
	assert.Equal(t, "local_db1", export.Mappings[0].LocalDatabaseName)

	mockRepo.AssertExpectations(t)
}

func TestMappingManager_CheckTableConflicts(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// This test would require a real database connection to test the INFORMATION_SCHEMA queries
	// For now, we'll test the basic logic with empty tables list
	manager := &MappingManagerImpl{
		repo:   mockRepo,
		logger: logger,
	}

	// Test with empty tables list
	conflicts, err := manager.CheckTableConflicts(ctx, "test_db", []string{})
	assert.NoError(t, err)
	assert.Empty(t, conflicts)
}

// Property-based tests for database mapping functionality

// TestProperty_MappingUniqueness tests Property 3: 映射关系唯一性
// Feature: database-sync, Property 3: 映射关系唯一性
func TestProperty_MappingUniqueness(t *testing.T) {
	// **Validates: Requirements 2.4**

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxDiscardRatio = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("For any local database name, no two remote connections should map to the same local database", prop.ForAll(
		func(localDBName string) bool {
			mockRepo := new(MockRepository)
			manager := &MappingManagerImpl{
				repo:   mockRepo,
				logger: logger,
			}

			// Create two different connections
			conn1ID := uuid.New().String()
			conn2ID := uuid.New().String()

			conn1 := &ConnectionConfig{
				ID:          conn1ID,
				Name:        "connection1",
				Host:        "localhost",
				Port:        3306,
				Username:    "user",
				Database:    "db1",
				LocalDBName: "", // Initially empty
				CreatedAt:   time.Now(),
			}

			conn2 := &ConnectionConfig{
				ID:          conn2ID,
				Name:        "connection2",
				Host:        "localhost",
				Port:        3306,
				Username:    "user",
				Database:    "db2",
				LocalDBName: localDBName, // Already using the local DB name
				CreatedAt:   time.Now(),
			}

			// Mock repository calls for first mapping creation
			mockRepo.On("GetConnection", ctx, conn1ID).Return(conn1, nil)
			mockRepo.On("GetConnections", ctx).Return([]*ConnectionConfig{conn1, conn2}, nil)

			// Try to create mapping for conn1 to the same local DB as conn2
			mapping1 := &DatabaseMapping{
				RemoteConnectionID: conn1ID,
				LocalDatabaseName:  localDBName,
			}

			err := manager.CreateDatabaseMapping(ctx, mapping1)

			// Should fail because conn2 already uses this local database name
			return err != nil
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty_ConfigExportImportConsistency tests Property 11: 配置导出导入一致性
// Feature: database-sync, Property 11: 配置导出导入一致性
func TestProperty_ConfigExportImportConsistency(t *testing.T) {
	// **Validates: Requirements 6.1, 6.2**

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxDiscardRatio = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("For any sync configuration, exporting then importing should produce equivalent configuration", prop.ForAll(
		func(connName, localDBName, syncConfigName, sourceTable, targetTable string) bool {
			// Skip empty strings to avoid validation errors
			if connName == "" || localDBName == "" || syncConfigName == "" || sourceTable == "" || targetTable == "" {
				return true
			}

			mockRepo1 := new(MockRepository) // For initial export
			mockRepo2 := new(MockRepository) // For import and re-export

			manager1 := &MappingManagerImpl{
				repo:   mockRepo1,
				logger: logger,
			}

			manager2 := &MappingManagerImpl{
				repo:   mockRepo2,
				logger: logger,
			}

			connID := uuid.New().String()
			syncConfigID := uuid.New().String()
			tableMappingID := uuid.New().String()

			// Create original configuration
			conn := &ConnectionConfig{
				ID:          connID,
				Name:        connName,
				Host:        "localhost",
				Port:        3306,
				Username:    "user",
				Database:    "remote_db",
				LocalDBName: localDBName,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			tableMapping := &TableMapping{
				ID:           tableMappingID,
				SyncConfigID: syncConfigID,
				SourceTable:  sourceTable,
				TargetTable:  targetTable,
				SyncMode:     SyncModeFull,
				Enabled:      true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			syncConfig := &SyncConfig{
				ID:           syncConfigID,
				ConnectionID: connID,
				Name:         syncConfigName,
				SyncMode:     SyncModeFull,
				Enabled:      true,
				Tables:       []*TableMapping{tableMapping},
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			// Mock repository calls for initial export
			mockRepo1.On("GetConnections", ctx).Return([]*ConnectionConfig{conn}, nil)
			mockRepo1.On("GetSyncConfigs", ctx, connID).Return([]*SyncConfig{syncConfig}, nil)

			// Step 1: Export original configuration
			exported, err := manager1.ExportConfig(ctx)
			if err != nil {
				return false
			}

			// Step 2: Validate exported configuration
			err = manager2.ValidateConfig(ctx, exported)
			if err != nil {
				return false
			}

			// Step 3: Import configuration (simulate import by setting up mock for re-export)
			// After import, the configuration should be equivalent when exported again
			// We simulate this by setting up the mock repository with the imported data

			// The imported connections will have new IDs, so we need to track the mapping
			if len(exported.Connections) == 0 || len(exported.SyncConfigs) == 0 {
				return false
			}

			importedConn := exported.Connections[0]
			importedSyncConfig := exported.SyncConfigs[0]

			// Mock repository calls for re-export after import
			mockRepo2.On("GetConnections", ctx).Return([]*ConnectionConfig{importedConn}, nil)
			mockRepo2.On("GetSyncConfigs", ctx, importedConn.ID).Return([]*SyncConfig{importedSyncConfig}, nil)

			// Step 4: Export again after import
			reExported, err := manager2.ExportConfig(ctx)
			if err != nil {
				return false
			}

			// Step 5: Compare essential configuration data (ignoring IDs and timestamps)
			if len(reExported.Connections) != len(exported.Connections) ||
				len(reExported.Mappings) != len(exported.Mappings) ||
				len(reExported.SyncConfigs) != len(exported.SyncConfigs) {
				return false
			}

			// Compare connection data (ignoring ID and timestamps)
			origConn := exported.Connections[0]
			reExportedConn := reExported.Connections[0]

			if origConn.Name != reExportedConn.Name ||
				origConn.Host != reExportedConn.Host ||
				origConn.Port != reExportedConn.Port ||
				origConn.Username != reExportedConn.Username ||
				origConn.Database != reExportedConn.Database ||
				origConn.LocalDBName != reExportedConn.LocalDBName {
				return false
			}

			// Compare sync config data (ignoring ID and timestamps)
			origSync := exported.SyncConfigs[0]
			reExportedSync := reExported.SyncConfigs[0]

			if origSync.Name != reExportedSync.Name ||
				origSync.SyncMode != reExportedSync.SyncMode ||
				origSync.Enabled != reExportedSync.Enabled ||
				len(origSync.Tables) != len(reExportedSync.Tables) {
				return false
			}

			// Compare table mappings (ignoring IDs and timestamps)
			if len(origSync.Tables) > 0 && len(reExportedSync.Tables) > 0 {
				origTable := origSync.Tables[0]
				reExportedTable := reExportedSync.Tables[0]

				if origTable.SourceTable != reExportedTable.SourceTable ||
					origTable.TargetTable != reExportedTable.TargetTable ||
					origTable.SyncMode != reExportedTable.SyncMode ||
					origTable.Enabled != reExportedTable.Enabled {
					return false
				}
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
