package internal

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"db-taxi/internal/config"
	"db-taxi/internal/server"
	"db-taxi/internal/sync"
)

// TestSystemIntegration tests the complete system integration
func TestSystemIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8081,
			Host:         "localhost",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            3306,
			Username:        "root",
			Password:        "",
			Database:        "test_db_taxi",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			QueryTimeout:    30 * time.Second,
		},
		Sync: config.SyncConfig{
			Enabled:        true,
			MaxConcurrency: 5,
			BatchSize:      1000,
			RetryAttempts:  3,
			RetryDelay:     30 * time.Second,
			JobTimeout:     1 * time.Hour,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	// Create test database
	setupTestDatabase(t, cfg)
	defer cleanupTestDatabase(t, cfg)

	// Create server instance
	srv := server.New(cfg)
	require.NotNil(t, srv, "Server should be created")

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server start error (expected): %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(1 * time.Second)

	// Test server shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err := srv.Stop(shutdownCtx)
	assert.NoError(t, err, "Server should shutdown gracefully")
}

// TestSyncManagerIntegration tests sync manager component integration
func TestSyncManagerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            3306,
			Username:        "root",
			Password:        "",
			Database:        "test_db_taxi_sync",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Sync: config.SyncConfig{
			Enabled:        true,
			MaxConcurrency: 5,
			BatchSize:      1000,
			RetryAttempts:  3,
			RetryDelay:     30 * time.Second,
			JobTimeout:     1 * time.Hour,
		},
	}

	// Create test database
	setupTestDatabase(t, cfg)
	defer cleanupTestDatabase(t, cfg)

	// Create database connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	db, err := sqlx.Connect("mysql", dsn)
	require.NoError(t, err, "Should connect to database")
	defer db.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create sync manager
	manager, err := sync.NewManager(cfg, db, logger)
	require.NoError(t, err, "Should create sync manager")
	require.NotNil(t, manager, "Sync manager should not be nil")

	// Initialize sync system
	ctx := context.Background()
	err = manager.Initialize(ctx)
	require.NoError(t, err, "Should initialize sync system")

	// Test component access
	t.Run("ComponentAccess", func(t *testing.T) {
		assert.NotNil(t, manager.GetConnectionManager(), "Connection manager should be accessible")
		assert.NotNil(t, manager.GetSyncManager(), "Sync manager should be accessible")
		assert.NotNil(t, manager.GetMappingManager(), "Mapping manager should be accessible")
		assert.NotNil(t, manager.GetJobEngine(), "Job engine should be accessible")
		assert.NotNil(t, manager.GetSyncEngine(), "Sync engine should be accessible")
	})

	// Test health check
	t.Run("HealthCheck", func(t *testing.T) {
		err := manager.HealthCheck(ctx)
		assert.NoError(t, err, "Health check should pass")
	})

	// Test stats retrieval
	t.Run("GetStats", func(t *testing.T) {
		stats, err := manager.GetStats(ctx)
		assert.NoError(t, err, "Should get stats")
		assert.NotNil(t, stats, "Stats should not be nil")
		assert.Contains(t, stats, "total_connections", "Stats should contain total_connections")
		assert.Contains(t, stats, "total_jobs", "Stats should contain total_jobs")
	})

	// Shutdown sync system
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = manager.Shutdown(shutdownCtx)
	assert.NoError(t, err, "Should shutdown gracefully")
}

// TestDependencyInjection tests that all components are properly injected
func TestDependencyInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            3306,
			Username:        "root",
			Password:        "",
			Database:        "test_db_taxi_di",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Sync: config.SyncConfig{
			Enabled:        true,
			MaxConcurrency: 5,
			BatchSize:      1000,
		},
	}

	setupTestDatabase(t, cfg)
	defer cleanupTestDatabase(t, cfg)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	db, err := sqlx.Connect("mysql", dsn)
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create sync manager
	manager, err := sync.NewManager(cfg, db, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Initialize(ctx)
	require.NoError(t, err)

	// Verify all components are non-nil and properly injected
	t.Run("AllComponentsInjected", func(t *testing.T) {
		components := map[string]interface{}{
			"ConnectionManager": manager.GetConnectionManager(),
			"SyncManager":       manager.GetSyncManager(),
			"MappingManager":    manager.GetMappingManager(),
			"JobEngine":         manager.GetJobEngine(),
			"SyncEngine":        manager.GetSyncEngine(),
		}

		for name, component := range components {
			assert.NotNil(t, component, "%s should be injected", name)
		}
	})

	// Cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager.Shutdown(shutdownCtx)
}

// Helper functions

func setupTestDatabase(t *testing.T, cfg *config.Config) {
	// Connect without database to create it
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("Cannot connect to MySQL for integration test: %v", err)
		return
	}
	defer db.Close()

	// Create test database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", cfg.Database.Database))
	if err != nil {
		t.Skipf("Cannot create test database: %v", err)
		return
	}

	t.Logf("Test database %s created", cfg.Database.Database)
}

func cleanupTestDatabase(t *testing.T, cfg *config.Config) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Logf("Cannot connect to MySQL for cleanup: %v", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.Database.Database))
	if err != nil {
		t.Logf("Cannot drop test database: %v", err)
		return
	}

	t.Logf("Test database %s dropped", cfg.Database.Database)
}
