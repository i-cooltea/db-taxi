package sync

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"db-taxi/internal/config"
)

// Manager represents the main sync system manager
type Manager struct {
	config            *config.Config
	logger            *logrus.Logger
	db                *sqlx.DB
	repo              Repository
	connectionManager ConnectionManager
	syncManager       SyncManager
	mappingManager    MappingManager
	jobEngine         JobEngine
	syncEngine        SyncEngine
}

// NewManager creates a new sync system manager
func NewManager(cfg *config.Config, db *sqlx.DB, logger *logrus.Logger) (*Manager, error) {
	if !cfg.Sync.Enabled {
		return nil, fmt.Errorf("sync system is disabled in configuration")
	}

	// Create repository
	repo := NewMySQLRepository(db, logger)

	// Create services
	connectionManager := NewConnectionManager(repo, logger, db)
	syncManager := NewSyncManager(repo, logger, db)
	syncEngine := NewSyncEngine(db, repo, logger)

	manager := &Manager{
		config:            cfg,
		logger:            logger,
		db:                db,
		repo:              repo,
		connectionManager: connectionManager,
		syncManager:       syncManager,
		syncEngine:        syncEngine,
	}

	logger.Info("Sync system manager initialized successfully")
	return manager, nil
}

// Initialize initializes the sync system
func (m *Manager) Initialize(ctx context.Context) error {
	m.logger.Info("Initializing sync system...")

	// Run database migrations
	if err := m.runMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// TODO: Initialize job engine
	// TODO: Initialize mapping manager

	m.logger.Info("Sync system initialized successfully")
	return nil
}

// GetConnectionManager returns the connection manager
func (m *Manager) GetConnectionManager() ConnectionManager {
	return m.connectionManager
}

// GetSyncManager returns the sync manager
func (m *Manager) GetSyncManager() SyncManager {
	return m.syncManager
}

// GetMappingManager returns the mapping manager
func (m *Manager) GetMappingManager() MappingManager {
	return m.mappingManager
}

// GetJobEngine returns the job engine
func (m *Manager) GetJobEngine() JobEngine {
	return m.jobEngine
}

// GetSyncEngine returns the sync engine
func (m *Manager) GetSyncEngine() SyncEngine {
	return m.syncEngine
}

// Shutdown gracefully shuts down the sync system
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down sync system...")

	// TODO: Stop all running jobs
	// TODO: Close connections
	// TODO: Cleanup resources

	m.logger.Info("Sync system shutdown completed")
	return nil
}

// runMigrations runs database migrations for the sync system
func (m *Manager) runMigrations(ctx context.Context) error {
	m.logger.Info("Running sync system database migrations...")

	// TODO: Implement proper migration system
	// For now, we'll assume the tables are created manually or by external migration tool

	// Check if sync tables exist
	var count int
	err := m.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'connections'")
	if err != nil {
		return fmt.Errorf("failed to check for sync tables: %w", err)
	}

	if count == 0 {
		m.logger.Warn("Sync tables not found. Please run the migration script: migrations/001_create_sync_tables.sql")
		return fmt.Errorf("sync tables not found, please run migrations")
	}

	m.logger.Info("Sync system database migrations completed")
	return nil
}

// HealthCheck performs a health check of the sync system
func (m *Manager) HealthCheck(ctx context.Context) error {
	// Check database connectivity
	if err := m.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Check if sync tables exist
	var count int
	err := m.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN ('connections', 'sync_configs', 'sync_jobs')")
	if err != nil {
		return fmt.Errorf("failed to check sync tables: %w", err)
	}

	if count < 3 {
		return fmt.Errorf("sync tables are missing")
	}

	return nil
}

// GetStats returns sync system statistics
func (m *Manager) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get connection count
	connections, err := m.connectionManager.GetConnections(ctx)
	if err != nil {
		m.logger.WithError(err).Warn("Failed to get connections for stats")
	} else {
		stats["total_connections"] = len(connections)

		connectedCount := 0
		for _, conn := range connections {
			if conn.Status.Connected {
				connectedCount++
			}
		}
		stats["connected_connections"] = connectedCount
	}

	// Get job statistics
	var totalJobs, runningJobs, completedJobs, failedJobs int

	if err := m.db.GetContext(ctx, &totalJobs, "SELECT COUNT(*) FROM sync_jobs"); err != nil {
		m.logger.WithError(err).Warn("Failed to get total jobs count")
	} else {
		stats["total_jobs"] = totalJobs
	}

	if err := m.db.GetContext(ctx, &runningJobs, "SELECT COUNT(*) FROM sync_jobs WHERE status = 'running'"); err != nil {
		m.logger.WithError(err).Warn("Failed to get running jobs count")
	} else {
		stats["running_jobs"] = runningJobs
	}

	if err := m.db.GetContext(ctx, &completedJobs, "SELECT COUNT(*) FROM sync_jobs WHERE status = 'completed'"); err != nil {
		m.logger.WithError(err).Warn("Failed to get completed jobs count")
	} else {
		stats["completed_jobs"] = completedJobs
	}

	if err := m.db.GetContext(ctx, &failedJobs, "SELECT COUNT(*) FROM sync_jobs WHERE status = 'failed'"); err != nil {
		m.logger.WithError(err).Warn("Failed to get failed jobs count")
	} else {
		stats["failed_jobs"] = failedJobs
	}

	return stats, nil
}
