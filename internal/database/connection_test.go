package database

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"db-taxi/internal/config"
)

func TestBuildDSN(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         3306,
		Username:     "root",
		Password:     "password",
		Database:     "testdb",
		SSL:          false,
		QueryTimeout: 30 * time.Second,
	}

	dsn, err := buildDSN(cfg)
	if err != nil {
		t.Errorf("buildDSN() error = %v", err)
		return
	}

	if dsn == "" {
		t.Error("buildDSN() returned empty DSN")
	}

	// Check that DSN contains expected components
	expectedComponents := []string{"root", "localhost:3306", "testdb"}
	for _, component := range expectedComponents {
		if len(dsn) == 0 {
			t.Errorf("buildDSN() DSN does not contain expected component: %s", component)
		}
	}
}

func TestNewConnectionPool_InvalidConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise during tests

	// Test with empty host (should fail before attempting connection)
	cfg := &config.DatabaseConfig{
		Host:            "", // Empty host
		Port:            3306,
		Username:        "root",
		Password:        "password",
		Database:        "test",
		SSL:             false,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		QueryTimeout:    30 * time.Second,
	}

	_, err := NewConnectionPool(cfg, logger)
	if err == nil {
		t.Error("NewConnectionPool() should fail with empty host")
	}
}

func TestNewSchemaExplorer(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Test with nil database (should not panic)
	explorer := NewSchemaExplorer(nil, logger)
	if explorer == nil {
		t.Error("NewSchemaExplorer should not return nil")
	}

	if explorer.logger == nil {
		t.Error("Logger should be set")
	}

	// Test with nil logger (should use default)
	explorer2 := NewSchemaExplorer(nil, nil)
	if explorer2.logger == nil {
		t.Error("Default logger should be created when nil is passed")
	}
}

func TestSchemaExplorer_ValidateInputs(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	explorer := NewSchemaExplorer(nil, logger)

	// Test GetTables with empty database
	_, err := explorer.GetTables("")
	if err == nil {
		t.Error("GetTables should return error for empty database name")
	}

	// Test GetTableInfo with empty parameters
	_, err = explorer.GetTableInfo("", "table")
	if err == nil {
		t.Error("GetTableInfo should return error for empty database name")
	}

	_, err = explorer.GetTableInfo("db", "")
	if err == nil {
		t.Error("GetTableInfo should return error for empty table name")
	}

	// Test GetTableData with empty parameters
	_, err = explorer.GetTableData("", "table", 0, 10)
	if err == nil {
		t.Error("GetTableData should return error for empty database name")
	}

	_, err = explorer.GetTableData("db", "", 0, 10)
	if err == nil {
		t.Error("GetTableData should return error for empty table name")
	}
}
