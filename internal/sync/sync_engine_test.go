package sync

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestNewSyncEngine tests the creation of a new sync engine
func TestNewSyncEngine(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create a mock database connection (nil is acceptable for this test)
	var db *sqlx.DB

	engine := NewSyncEngine(db, mockRepo, logger)

	assert.NotNil(t, engine, "SyncEngine should not be nil")
	assert.IsType(t, &DefaultSyncEngine{}, engine, "Should return DefaultSyncEngine type")
}

// TestSyncEngine_SyncTable_UnsupportedMode tests handling of unsupported sync modes
func TestSyncEngine_SyncTable_UnsupportedMode(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	var db *sqlx.DB
	engine := NewSyncEngine(db, mockRepo, logger)

	ctx := context.Background()
	job := &SyncJob{
		ID:       "test-job-1",
		ConfigID: "test-config-1",
		Status:   JobStatusRunning,
	}

	mapping := &TableMapping{
		ID:           "test-mapping-1",
		SyncConfigID: "test-config-1",
		SourceTable:  "source_table",
		TargetTable:  "target_table",
		SyncMode:     SyncMode("invalid_mode"),
		Enabled:      true,
	}

	err := engine.SyncTable(ctx, job, mapping)

	assert.Error(t, err, "Should return error for unsupported sync mode")
	assert.Contains(t, err.Error(), "unsupported sync mode", "Error should mention unsupported sync mode")
}

// TestSyncEngine_SyncIncremental_NoCheckpoint tests that incremental sync falls back to full sync when no checkpoint exists
func TestSyncEngine_SyncIncremental_NoCheckpoint(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	var db *sqlx.DB
	engine := NewSyncEngine(db, mockRepo, logger)

	ctx := context.Background()
	job := &SyncJob{
		ID:       "test-job-1",
		ConfigID: "test-config-1",
		Status:   JobStatusRunning,
	}

	mapping := &TableMapping{
		ID:           "test-mapping-1",
		SyncConfigID: "test-config-1",
		SourceTable:  "source_table",
		TargetTable:  "target_table",
		SyncMode:     SyncModeIncremental,
		Enabled:      true,
	}

	// Mock GetSyncConfig to return a valid config
	syncConfig := &SyncConfig{
		ID:           "test-config-1",
		ConnectionID: "test-conn-1",
		Name:         "Test Sync",
		SyncMode:     SyncModeIncremental,
		Enabled:      true,
	}
	mockRepo.On("GetSyncConfig", ctx, mapping.SyncConfigID).Return(syncConfig, nil)

	// Mock GetConnection to return a valid connection config
	connConfig := &ConnectionConfig{
		ID:          "test-conn-1",
		Name:        "Test Connection",
		Host:        "localhost",
		Port:        3306,
		Username:    "test",
		Password:    "test",
		Database:    "test_db",
		LocalDBName: "local_test_db",
		SSL:         false,
	}
	mockRepo.On("GetConnection", ctx, syncConfig.ConnectionID).Return(connConfig, nil)

	// Mock GetCheckpoint to return nil (no checkpoint exists)
	mockRepo.On("GetCheckpoint", ctx, mapping.ID).Return(nil, nil)

	// Since there's no checkpoint, it should attempt to fall back to full sync
	// which will fail due to inability to connect to remote database, but that's expected
	err := engine.SyncIncremental(ctx, job, mapping)

	assert.Error(t, err, "Should return error when trying to connect to remote database")
	// The error should be about connection, not "not implemented"
	assert.NotContains(t, err.Error(), "not yet implemented", "Should not return not implemented error")
}

// TestSyncEngine_ValidateData_NotImplemented tests that data validation returns not implemented error
func TestSyncEngine_ValidateData_NotImplemented(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	var db *sqlx.DB
	engine := NewSyncEngine(db, mockRepo, logger)

	ctx := context.Background()
	mapping := &TableMapping{
		ID:           "test-mapping-1",
		SyncConfigID: "test-config-1",
		SourceTable:  "source_table",
		TargetTable:  "target_table",
		SyncMode:     SyncModeFull,
		Enabled:      true,
	}

	err := engine.ValidateData(ctx, mapping)

	assert.Error(t, err, "Should return error for not implemented data validation")
	assert.Contains(t, err.Error(), "not yet implemented", "Error should mention not implemented")
}

// TestSyncEngine_BuildCreateTableStatement tests the CREATE TABLE statement builder
func TestSyncEngine_BuildCreateTableStatement(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	var db *sqlx.DB
	engine := NewSyncEngine(db, mockRepo, logger).(*DefaultSyncEngine)

	schema := &TableSchema{
		Name: "test_table",
		Columns: []*ColumnInfo{
			{
				Name:     "id",
				Type:     "INT",
				Nullable: false,
				Extra:    "AUTO_INCREMENT",
			},
			{
				Name:         "name",
				Type:         "VARCHAR(255)",
				Nullable:     false,
				DefaultValue: "''",
			},
			{
				Name:     "email",
				Type:     "VARCHAR(255)",
				Nullable: true,
			},
		},
		Keys: []*KeyInfo{
			{
				Name:    "PRIMARY",
				Type:    "PRIMARY KEY",
				Columns: []string{"id"},
			},
		},
		Indexes: []*IndexInfo{
			{
				Name:    "idx_name",
				Columns: []string{"name"},
				Unique:  false,
				Type:    "BTREE",
			},
		},
	}

	createStmt := engine.buildCreateTableStatement("test_db", "test_table", schema)

	assert.Contains(t, createStmt, "CREATE TABLE `test_db`.`test_table`", "Should contain CREATE TABLE statement")
	assert.Contains(t, createStmt, "`id` INT NOT NULL AUTO_INCREMENT", "Should contain id column definition")
	assert.Contains(t, createStmt, "`name` VARCHAR(255) NOT NULL DEFAULT ''", "Should contain name column definition")
	assert.Contains(t, createStmt, "`email` VARCHAR(255)", "Should contain email column definition")
	assert.Contains(t, createStmt, "PRIMARY KEY (`id`)", "Should contain primary key definition")
	assert.Contains(t, createStmt, "KEY `idx_name` (`name`)", "Should contain index definition")
}

// TestSyncEngine_BuildCreateTableStatement_MultiColumnPrimaryKey tests CREATE TABLE with multi-column primary key
func TestSyncEngine_BuildCreateTableStatement_MultiColumnPrimaryKey(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	var db *sqlx.DB
	engine := NewSyncEngine(db, mockRepo, logger).(*DefaultSyncEngine)

	schema := &TableSchema{
		Name: "test_table",
		Columns: []*ColumnInfo{
			{
				Name:     "user_id",
				Type:     "INT",
				Nullable: false,
			},
			{
				Name:     "product_id",
				Type:     "INT",
				Nullable: false,
			},
		},
		Keys: []*KeyInfo{
			{
				Name:    "PRIMARY",
				Type:    "PRIMARY KEY",
				Columns: []string{"user_id", "product_id"},
			},
		},
	}

	createStmt := engine.buildCreateTableStatement("test_db", "test_table", schema)

	assert.Contains(t, createStmt, "PRIMARY KEY (`user_id`, `product_id`)", "Should contain multi-column primary key")
}

// TestSyncEngine_BuildCreateTableStatement_UniqueKey tests CREATE TABLE with unique key
func TestSyncEngine_BuildCreateTableStatement_UniqueKey(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	var db *sqlx.DB
	engine := NewSyncEngine(db, mockRepo, logger).(*DefaultSyncEngine)

	schema := &TableSchema{
		Name: "test_table",
		Columns: []*ColumnInfo{
			{
				Name:     "id",
				Type:     "INT",
				Nullable: false,
			},
			{
				Name:     "email",
				Type:     "VARCHAR(255)",
				Nullable: false,
			},
		},
		Keys: []*KeyInfo{
			{
				Name:    "PRIMARY",
				Type:    "PRIMARY KEY",
				Columns: []string{"id"},
			},
			{
				Name:    "uk_email",
				Type:    "UNIQUE",
				Columns: []string{"email"},
			},
		},
	}

	createStmt := engine.buildCreateTableStatement("test_db", "test_table", schema)

	assert.Contains(t, createStmt, "PRIMARY KEY (`id`)", "Should contain primary key")
	assert.Contains(t, createStmt, "UNIQUE KEY `uk_email` (`email`)", "Should contain unique key")
}
