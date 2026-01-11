package sync

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// Service provides the main sync service implementation
type Service struct {
	repo    Repository
	logger  *logrus.Logger
	localDB *sqlx.DB // Local database connection for creating local databases
}

// NewService creates a new sync service instance
func NewService(repo Repository, logger *logrus.Logger, localDB *sqlx.DB) *Service {
	return &Service{
		repo:    repo,
		logger:  logger,
		localDB: localDB,
	}
}

// ConnectionManagerService implements ConnectionManager interface
type ConnectionManagerService struct {
	*Service
	statusCache    map[string]*ConnectionStatus
	statusMutex    sync.RWMutex
	connectionPool map[string]*sqlx.DB
	poolMutex      sync.RWMutex
	healthChecker  *HealthChecker
}

// HealthChecker manages periodic health checks for connections
type HealthChecker struct {
	manager       *ConnectionManagerService
	checkInterval time.Duration
	stopChan      chan struct{}
	running       bool
	mutex         sync.Mutex
}

// ConnectionPoolConfig defines connection pool configuration
type ConnectionPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewConnectionManager creates a new connection manager service
func NewConnectionManager(repo Repository, logger *logrus.Logger, localDB *sqlx.DB) ConnectionManager {
	cm := &ConnectionManagerService{
		Service:        NewService(repo, logger, localDB),
		statusCache:    make(map[string]*ConnectionStatus),
		connectionPool: make(map[string]*sqlx.DB),
	}

	// Initialize health checker with 30-second interval
	cm.healthChecker = &HealthChecker{
		manager:       cm,
		checkInterval: 30 * time.Second,
		stopChan:      make(chan struct{}),
	}

	// Start periodic health checking
	cm.startHealthChecker()

	return cm
}

func (s *ConnectionManagerService) AddConnection(ctx context.Context, config *ConnectionConfig) (*Connection, error) {
	// Generate ID if not provided
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	// Validate configuration
	if err := s.validateConnectionConfig(config); err != nil {
		return nil, fmt.Errorf("invalid connection config: %w", err)
	}

	// Test the remote connection before saving
	status, err := s.testRemoteConnection(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote database: %w", err)
	}

	// Create local database if it doesn't exist
	if err := s.createLocalDatabase(ctx, config.LocalDBName); err != nil {
		return nil, fmt.Errorf("failed to create local database: %w", err)
	}

	// Create connection in repository
	if err := s.repo.CreateConnection(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": config.ID,
		"name":          config.Name,
		"host":          config.Host,
		"local_db":      config.LocalDBName,
	}).Info("Connection created successfully")

	return &Connection{
		Config: config,
		Status: *status,
	}, nil
}

func (s *ConnectionManagerService) GetConnections(ctx context.Context) ([]*Connection, error) {
	configs, err := s.repo.GetConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}

	connections := make([]*Connection, len(configs))
	for i, config := range configs {
		// Get cached status first, fallback to testing if not available
		status := s.getCachedStatus(config.ID)
		if status == nil {
			// Test connection status if not cached
			status, err = s.testRemoteConnection(ctx, config)
			if err != nil {
				status = &ConnectionStatus{
					Connected: false,
					LastCheck: time.Now(),
					Error:     err.Error(),
				}
			}
			// Cache the status
			s.setCachedStatus(config.ID, status)
		}

		connections[i] = &Connection{
			Config: config,
			Status: *status,
		}
	}

	return connections, nil
}

func (s *ConnectionManagerService) GetConnection(ctx context.Context, id string) (*Connection, error) {
	config, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Get cached status first, fallback to testing if not available
	status := s.getCachedStatus(id)
	if status == nil {
		// Test connection status if not cached
		status, err = s.testRemoteConnection(ctx, config)
		if err != nil {
			status = &ConnectionStatus{
				Connected: false,
				LastCheck: time.Now(),
				Error:     err.Error(),
			}
		}
		// Cache the status
		s.setCachedStatus(id, status)
	}

	return &Connection{
		Config: config,
		Status: *status,
	}, nil
}

func (s *ConnectionManagerService) UpdateConnection(ctx context.Context, id string, config *ConnectionConfig) error {
	// Get existing connection to check if local database name changed
	existingConfig, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get existing connection: %w", err)
	}

	// Validate configuration
	if err := s.validateConnectionConfig(config); err != nil {
		return fmt.Errorf("invalid connection config: %w", err)
	}

	// Test the remote connection before updating
	if _, err := s.testRemoteConnection(ctx, config); err != nil {
		return fmt.Errorf("failed to connect to remote database: %w", err)
	}

	// Create new local database if the name changed
	if config.LocalDBName != existingConfig.LocalDBName {
		if err := s.createLocalDatabase(ctx, config.LocalDBName); err != nil {
			return fmt.Errorf("failed to create local database: %w", err)
		}
	}

	// Update in repository
	if err := s.repo.UpdateConnection(ctx, id, config); err != nil {
		return fmt.Errorf("failed to update connection: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": id,
		"name":          config.Name,
		"local_db":      config.LocalDBName,
	}).Info("Connection updated successfully")

	return nil
}

func (s *ConnectionManagerService) DeleteConnection(ctx context.Context, id string) error {
	// Get connection config to check for related sync jobs
	config, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	// Check for active sync jobs
	activeJobs, err := s.repo.GetJobsByStatus(ctx, JobStatusRunning)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to check for active jobs")
	} else {
		// Stop any running sync jobs for this connection
		for _, job := range activeJobs {
			syncConfig, err := s.repo.GetSyncConfig(ctx, job.ConfigID)
			if err != nil {
				continue
			}
			if syncConfig.ConnectionID == id {
				s.logger.WithField("job_id", job.ID).Info("Stopping sync job for deleted connection")
				// TODO: Implement job cancellation when JobEngine is available
			}
		}
	}

	// Clean up cached status and connection pool
	s.removeCachedStatus(id)
	s.closePooledConnection(id)

	// Delete connection from repository (cascades to sync configs and jobs)
	if err := s.repo.DeleteConnection(ctx, id); err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": id,
		"name":          config.Name,
	}).Info("Connection deleted successfully")

	return nil
}

func (s *ConnectionManagerService) TestConnection(ctx context.Context, id string) (*ConnectionStatus, error) {
	config, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection config: %w", err)
	}

	// Force a fresh connection test and update cache
	status, err := s.testRemoteConnection(ctx, config)
	if err != nil {
		status = &ConnectionStatus{
			Connected: false,
			LastCheck: time.Now(),
			Error:     err.Error(),
		}
	}

	// Update cached status
	s.setCachedStatus(id, status)

	return status, err
}

// TestConnectionConfig tests a connection configuration without saving it
func (s *ConnectionManagerService) TestConnectionConfig(ctx context.Context, config *ConnectionConfig) (*ConnectionStatus, error) {
	// Test the connection without saving or caching
	status, err := s.testRemoteConnection(ctx, config)
	if err != nil {
		status = &ConnectionStatus{
			Connected: false,
			LastCheck: time.Now(),
			Error:     err.Error(),
		}
	}

	return status, nil // Return nil error so the status is always returned
}

// testRemoteConnection tests connectivity to a remote database
func (s *ConnectionManagerService) testRemoteConnection(ctx context.Context, config *ConnectionConfig) (*ConnectionStatus, error) {
	start := time.Now()

	// Try to get pooled connection first
	db := s.getPooledConnection(config)
	if db == nil {
		// Create new connection if not pooled
		var err error
		db, err = s.createConnection(config)
		if err != nil {
			return &ConnectionStatus{
				Connected: false,
				LastCheck: time.Now(),
				Error:     fmt.Sprintf("failed to create connection: %v", err),
			}, err
		}
		// Add to pool for reuse
		s.addToPool(config.ID, db)
	}

	// Test the connection with context timeout
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(testCtx); err != nil {
		// Remove failed connection from pool
		s.closePooledConnection(config.ID)
		return &ConnectionStatus{
			Connected: false,
			LastCheck: time.Now(),
			Error:     fmt.Sprintf("ping failed: %v", err),
		}, err
	}

	// Test basic query to ensure we can read from the database
	var version string
	if err := db.GetContext(testCtx, &version, "SELECT VERSION()"); err != nil {
		// Remove failed connection from pool
		s.closePooledConnection(config.ID)
		return &ConnectionStatus{
			Connected: false,
			LastCheck: time.Now(),
			Error:     fmt.Sprintf("version query failed: %v", err),
		}, err
	}

	latency := time.Since(start).Milliseconds()

	s.logger.WithFields(logrus.Fields{
		"connection_id": config.ID,
		"host":          config.Host,
		"database":      config.Database,
		"latency_ms":    latency,
		"version":       version,
	}).Debug("Remote connection test successful")

	return &ConnectionStatus{
		Connected: true,
		LastCheck: time.Now(),
		Latency:   latency,
	}, nil
}

// createConnection creates a new database connection with proper configuration
func (s *ConnectionManagerService) createConnection(config *ConnectionConfig) (*sqlx.DB, error) {
	// Build MySQL DSN for remote connection
	dsn, err := s.buildRemoteDSN(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	// Open connection to remote database
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool settings
	poolConfig := s.getConnectionPoolConfig()
	db.SetMaxOpenConns(poolConfig.MaxOpenConns)
	db.SetMaxIdleConns(poolConfig.MaxIdleConns)
	db.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	db.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)

	return db, nil
}

// getConnectionPoolConfig returns the connection pool configuration
func (s *ConnectionManagerService) getConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxOpenConns:    10,               // Maximum number of open connections
		MaxIdleConns:    5,                // Maximum number of idle connections
		ConnMaxLifetime: 30 * time.Minute, // Maximum connection lifetime
		ConnMaxIdleTime: 5 * time.Minute,  // Maximum idle time
	}
}

// getPooledConnection retrieves a connection from the pool
func (s *ConnectionManagerService) getPooledConnection(config *ConnectionConfig) *sqlx.DB {
	s.poolMutex.RLock()
	defer s.poolMutex.RUnlock()

	return s.connectionPool[config.ID]
}

// addToPool adds a connection to the pool
func (s *ConnectionManagerService) addToPool(connectionID string, db *sqlx.DB) {
	s.poolMutex.Lock()
	defer s.poolMutex.Unlock()

	// Close existing connection if any
	if existingDB, exists := s.connectionPool[connectionID]; exists {
		existingDB.Close()
	}

	s.connectionPool[connectionID] = db

	s.logger.WithField("connection_id", connectionID).Debug("Connection added to pool")
}

// closePooledConnection closes and removes a connection from the pool
func (s *ConnectionManagerService) closePooledConnection(connectionID string) {
	s.poolMutex.Lock()
	defer s.poolMutex.Unlock()

	if db, exists := s.connectionPool[connectionID]; exists {
		db.Close()
		delete(s.connectionPool, connectionID)
		s.logger.WithField("connection_id", connectionID).Debug("Connection removed from pool")
	}
}

// getCachedStatus retrieves cached connection status
func (s *ConnectionManagerService) getCachedStatus(connectionID string) *ConnectionStatus {
	s.statusMutex.RLock()
	defer s.statusMutex.RUnlock()

	if status, exists := s.statusCache[connectionID]; exists {
		// Check if status is still fresh (within 1 minute)
		if time.Since(status.LastCheck) < time.Minute {
			return status
		}
	}

	return nil
}

// setCachedStatus updates the cached connection status
func (s *ConnectionManagerService) setCachedStatus(connectionID string, status *ConnectionStatus) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

	s.statusCache[connectionID] = status
}

// removeCachedStatus removes cached status for a connection
func (s *ConnectionManagerService) removeCachedStatus(connectionID string) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

	delete(s.statusCache, connectionID)
}

// startHealthChecker starts the periodic health checking
func (s *ConnectionManagerService) startHealthChecker() {
	s.healthChecker.Start()
}

// stopHealthChecker stops the periodic health checking
func (s *ConnectionManagerService) stopHealthChecker() {
	s.healthChecker.Stop()
}

// Start begins the health checking routine
func (hc *HealthChecker) Start() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if hc.running {
		return
	}

	hc.running = true
	go hc.run()
}

// Stop terminates the health checking routine
func (hc *HealthChecker) Stop() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if !hc.running {
		return
	}

	hc.running = false
	close(hc.stopChan)
}

// run executes the periodic health checking
func (hc *HealthChecker) run() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performHealthChecks()
		case <-hc.stopChan:
			return
		}
	}
}

// performHealthChecks checks the health of all connections
func (hc *HealthChecker) performHealthChecks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all connections
	configs, err := hc.manager.repo.GetConnections(ctx)
	if err != nil {
		hc.manager.logger.WithError(err).Error("Failed to get connections for health check")
		return
	}

	// Check each connection
	for _, config := range configs {
		go hc.checkConnection(ctx, config)
	}
}

// checkConnection performs health check for a single connection
func (hc *HealthChecker) checkConnection(ctx context.Context, config *ConnectionConfig) {
	status, err := hc.manager.testRemoteConnection(ctx, config)
	if err != nil {
		status = &ConnectionStatus{
			Connected: false,
			LastCheck: time.Now(),
			Error:     err.Error(),
		}
	}

	// Update cached status
	hc.manager.setCachedStatus(config.ID, status)

	// Log status changes
	if cachedStatus := hc.manager.getCachedStatus(config.ID); cachedStatus != nil {
		if cachedStatus.Connected != status.Connected {
			if status.Connected {
				hc.manager.logger.WithFields(logrus.Fields{
					"connection_id": config.ID,
					"name":          config.Name,
				}).Info("Connection restored")
			} else {
				hc.manager.logger.WithFields(logrus.Fields{
					"connection_id": config.ID,
					"name":          config.Name,
					"error":         status.Error,
				}).Warn("Connection failed")
			}
		}
	}
}

// createLocalDatabase creates a local database if it doesn't exist
func (s *ConnectionManagerService) createLocalDatabase(ctx context.Context, dbName string) error {
	if s.localDB == nil {
		return fmt.Errorf("local database connection not available")
	}

	// Check if database already exists
	var exists int
	query := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
	if err := s.localDB.GetContext(ctx, &exists, query, dbName); err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if exists > 0 {
		s.logger.WithField("database", dbName).Debug("Local database already exists")
		return nil
	}

	// Create the database
	createQuery := fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
	if _, err := s.localDB.ExecContext(ctx, createQuery); err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}

	s.logger.WithField("database", dbName).Info("Local database created successfully")
	return nil
}

// buildRemoteDSN builds a MySQL DSN for remote connection
func (s *ConnectionManagerService) buildRemoteDSN(config *ConnectionConfig) (string, error) {
	mysqlConfig := mysql.Config{
		User:                 config.Username,
		Passwd:               config.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", config.Host, config.Port),
		DBName:               config.Database,
		Timeout:              10 * time.Second,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		AllowNativePasswords: true,
		ParseTime:            true,
		Loc:                  time.UTC,
	}

	// Configure SSL
	if config.SSL {
		mysqlConfig.TLSConfig = "true"
	} else {
		mysqlConfig.TLSConfig = "false"
	}

	return mysqlConfig.FormatDSN(), nil
}

func (s *ConnectionManagerService) validateConnectionConfig(config *ConnectionConfig) error {
	if config.Name == "" {
		return fmt.Errorf("connection name is required")
	}
	if config.Host == "" {
		return fmt.Errorf("host is required")
	}
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", config.Port)
	}
	if config.Username == "" {
		return fmt.Errorf("username is required")
	}
	if config.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if config.LocalDBName == "" {
		return fmt.Errorf("local database name is required")
	}

	// Validate local database name format (MySQL identifier rules)
	if !isValidMySQLIdentifier(config.LocalDBName) {
		return fmt.Errorf("invalid local database name: %s", config.LocalDBName)
	}

	return nil
}

// isValidMySQLIdentifier checks if a string is a valid MySQL identifier
func isValidMySQLIdentifier(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}

	// Check for valid characters (alphanumeric, underscore, dollar sign)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '$') {
			return false
		}
	}

	// Cannot start with a digit
	if name[0] >= '0' && name[0] <= '9' {
		return false
	}

	return true
}

// Close closes the connection manager and cleans up resources
func (s *ConnectionManagerService) Close() error {
	// Stop health checker
	s.stopHealthChecker()

	// Close all pooled connections
	s.poolMutex.Lock()
	for id, db := range s.connectionPool {
		db.Close()
		s.logger.WithField("connection_id", id).Debug("Closed pooled connection")
	}
	s.connectionPool = make(map[string]*sqlx.DB)
	s.poolMutex.Unlock()

	// Clear status cache
	s.statusMutex.Lock()
	s.statusCache = make(map[string]*ConnectionStatus)
	s.statusMutex.Unlock()

	s.logger.Info("Connection manager closed successfully")
	return nil
}

// SyncManagerService implements SyncManager interface
type SyncManagerService struct {
	*Service
	monitoring MonitoringService
	jobEngine  JobEngine
}

// NewSyncManager creates a new sync manager service
func NewSyncManager(repo Repository, logger *logrus.Logger, localDB *sqlx.DB) SyncManager {
	service := NewService(repo, logger, localDB)
	monitoring := NewMonitoringService(repo, logger)

	return &SyncManagerService{
		Service:    service,
		monitoring: monitoring,
	}
}

func (s *SyncManagerService) CreateSyncConfig(ctx context.Context, config *SyncConfig) error {
	// Generate ID if not provided
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	// Validate configuration
	if err := s.validateSyncConfig(config); err != nil {
		return fmt.Errorf("invalid sync config: %w", err)
	}

	// Create sync config in repository
	if err := s.repo.CreateSyncConfig(ctx, config); err != nil {
		return fmt.Errorf("failed to create sync config: %w", err)
	}

	// Create table mappings
	for _, mapping := range config.Tables {
		if mapping.ID == "" {
			mapping.ID = uuid.New().String()
		}
		mapping.SyncConfigID = config.ID
		mapping.CreatedAt = now
		mapping.UpdatedAt = now

		if err := s.repo.CreateTableMapping(ctx, mapping); err != nil {
			s.logger.WithError(err).Error("Failed to create table mapping")
			return fmt.Errorf("failed to create table mapping: %w", err)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"sync_config_id": config.ID,
		"connection_id":  config.ConnectionID,
		"name":           config.Name,
	}).Info("Sync config created successfully")

	return nil
}

func (s *SyncManagerService) GetSyncConfigs(ctx context.Context, connectionID string) ([]*SyncConfig, error) {
	configs, err := s.repo.GetSyncConfigs(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync configs: %w", err)
	}
	return configs, nil
}

func (s *SyncManagerService) GetSyncConfig(ctx context.Context, id string) (*SyncConfig, error) {
	config, err := s.repo.GetSyncConfig(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync config: %w", err)
	}
	return config, nil
}

func (s *SyncManagerService) UpdateSyncConfig(ctx context.Context, id string, config *SyncConfig) error {
	// Validate configuration
	if err := s.validateSyncConfig(config); err != nil {
		return fmt.Errorf("invalid sync config: %w", err)
	}

	// Get existing config to compare table mappings
	existingConfig, err := s.repo.GetSyncConfig(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get existing sync config: %w", err)
	}

	// Update sync config in repository
	if err := s.repo.UpdateSyncConfig(ctx, id, config); err != nil {
		return fmt.Errorf("failed to update sync config: %w", err)
	}

	// Update table mappings
	if err := s.updateTableMappings(ctx, id, existingConfig.Tables, config.Tables); err != nil {
		return fmt.Errorf("failed to update table mappings: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"sync_config_id": id,
		"name":           config.Name,
	}).Info("Sync config updated successfully")

	return nil
}

func (s *SyncManagerService) DeleteSyncConfig(ctx context.Context, id string) error {
	// TODO: Stop any running sync jobs for this config

	if err := s.repo.DeleteSyncConfig(ctx, id); err != nil {
		return fmt.Errorf("failed to delete sync config: %w", err)
	}

	s.logger.WithField("sync_config_id", id).Info("Sync config deleted successfully")
	return nil
}

func (s *SyncManagerService) StartSync(ctx context.Context, configID string) (*SyncJob, error) {
	// Get sync configuration
	syncConfig, err := s.repo.GetSyncConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync config: %w", err)
	}

	if !syncConfig.Enabled {
		return nil, fmt.Errorf("sync config is disabled")
	}

	// Create sync job
	job := &SyncJob{
		ID:              uuid.New().String(),
		ConfigID:        configID,
		Status:          JobStatusPending,
		StartTime:       time.Now(),
		TotalTables:     len(syncConfig.Tables),
		CompletedTables: 0,
		TotalRows:       0,
		ProcessedRows:   0,
		Progress: &Progress{
			TotalTables: len(syncConfig.Tables),
		},
		CreatedAt: time.Now(),
	}

	// Save job to repository
	if err := s.repo.CreateSyncJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create sync job: %w", err)
	}

	// Start monitoring for this job
	if err := s.monitoring.StartJobMonitoring(ctx, job.ID, len(syncConfig.Tables)); err != nil {
		s.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to start job monitoring")
	}

	// Log job event
	if err := s.monitoring.LogJobEvent(ctx, job.ID, "", "info", "Sync job created"); err != nil {
		s.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to log job event")
	}

	// Submit job to job engine for execution
	if s.jobEngine != nil {
		if err := s.jobEngine.SubmitJob(ctx, job); err != nil {
			s.logger.WithError(err).WithField("job_id", job.ID).Error("Failed to submit job to engine")

			// Update job status to failed
			job.Status = JobStatusFailed
			job.Error = fmt.Sprintf("Failed to submit job: %v", err)
			now := time.Now()
			job.EndTime = &now

			if updateErr := s.repo.UpdateSyncJob(ctx, job.ID, job); updateErr != nil {
				s.logger.WithError(updateErr).Error("Failed to update job status")
			}

			return nil, fmt.Errorf("failed to submit job to engine: %w", err)
		}

		s.logger.WithFields(logrus.Fields{
			"job_id":         job.ID,
			"sync_config_id": configID,
			"total_tables":   len(syncConfig.Tables),
		}).Info("Sync job submitted to engine successfully")
	} else {
		s.logger.WithField("job_id", job.ID).Warn("Job engine not available, job will remain in pending state")
	}

	return job, nil
}

func (s *SyncManagerService) StopSync(ctx context.Context, jobID string) error {
	if s.jobEngine == nil {
		return fmt.Errorf("job engine not available")
	}

	// Cancel the job
	if err := s.jobEngine.CancelJob(ctx, jobID); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	s.logger.WithField("job_id", jobID).Info("Sync job stopped successfully")
	return nil
}

func (s *SyncManagerService) GetSyncStatus(ctx context.Context, jobID string) (*SyncJob, error) {
	job, err := s.repo.GetSyncJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync job: %w", err)
	}
	return job, nil
}

func (s *SyncManagerService) validateSyncConfig(config *SyncConfig) error {
	if config.Name == "" {
		return fmt.Errorf("sync config name is required")
	}
	if config.ConnectionID == "" {
		return fmt.Errorf("connection ID is required")
	}
	if len(config.Tables) == 0 {
		return fmt.Errorf("at least one table mapping is required")
	}

	// Validate table mappings
	for i, mapping := range config.Tables {
		if mapping.SourceTable == "" {
			return fmt.Errorf("source table is required for mapping %d", i)
		}
		if mapping.TargetTable == "" {
			return fmt.Errorf("target table is required for mapping %d", i)
		}
		// Validate sync mode
		if mapping.SyncMode != SyncModeFull && mapping.SyncMode != SyncModeIncremental {
			return fmt.Errorf("invalid sync mode for mapping %d: %s", i, mapping.SyncMode)
		}
	}

	// Validate sync mode
	if config.SyncMode != SyncModeFull && config.SyncMode != SyncModeIncremental {
		return fmt.Errorf("invalid sync mode: %s", config.SyncMode)
	}

	return nil
}

// GetRemoteTables retrieves the list of tables from a remote database connection
// Requirement 3.1: Browse remote database and display available tables
func (s *SyncManagerService) GetRemoteTables(ctx context.Context, connectionID string) ([]string, error) {
	// Get connection configuration
	connectionConfig, err := s.repo.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection config: %w", err)
	}

	// Create connection to remote database
	db, err := s.createRemoteConnection(connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote database: %w", err)
	}
	defer db.Close()

	// Query for table names
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE' ORDER BY table_name"
	var tables []string
	err = db.SelectContext(ctx, &tables, query, connectionConfig.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to query remote tables: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"table_count":   len(tables),
	}).Debug("Retrieved remote tables")

	return tables, nil
}

// GetRemoteTableSchema retrieves the schema information for a specific table
// Supports requirement 3.1: Browse remote database structure
func (s *SyncManagerService) GetRemoteTableSchema(ctx context.Context, connectionID, tableName string) (*TableSchema, error) {
	// Get connection configuration
	connectionConfig, err := s.repo.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection config: %w", err)
	}

	// Create connection to remote database
	db, err := s.createRemoteConnection(connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote database: %w", err)
	}
	defer db.Close()

	schema := &TableSchema{
		Name:    tableName,
		Columns: []*ColumnInfo{},
		Indexes: []*IndexInfo{},
		Keys:    []*KeyInfo{},
	}

	// Get column information
	columnQuery := `
		SELECT column_name, data_type, is_nullable, column_default, extra
		FROM information_schema.columns 
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`

	rows, err := db.QueryContext(ctx, columnQuery, connectionConfig.Database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query table columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var nullable, defaultValue, extra sql.NullString

		err := rows.Scan(&col.Name, &col.Type, &nullable, &defaultValue, &extra)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		col.Nullable = nullable.String == "YES"
		if defaultValue.Valid {
			col.DefaultValue = defaultValue.String
		}
		if extra.Valid {
			col.Extra = extra.String
		}

		schema.Columns = append(schema.Columns, &col)
	}

	// Get index information
	indexQuery := `
		SELECT index_name, column_name, non_unique, index_type
		FROM information_schema.statistics 
		WHERE table_schema = ? AND table_name = ?
		ORDER BY index_name, seq_in_index
	`

	indexRows, err := db.QueryContext(ctx, indexQuery, connectionConfig.Database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query table indexes: %w", err)
	}
	defer indexRows.Close()

	indexMap := make(map[string]*IndexInfo)
	for indexRows.Next() {
		var indexName, columnName, indexType string
		var nonUnique int

		err := indexRows.Scan(&indexName, &columnName, &nonUnique, &indexType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index info: %w", err)
		}

		if index, exists := indexMap[indexName]; exists {
			index.Columns = append(index.Columns, columnName)
		} else {
			indexMap[indexName] = &IndexInfo{
				Name:    indexName,
				Columns: []string{columnName},
				Unique:  nonUnique == 0,
				Type:    indexType,
			}
		}
	}

	for _, index := range indexMap {
		schema.Indexes = append(schema.Indexes, index)
	}

	// Get key information (primary and foreign keys)
	keyQuery := `
		SELECT constraint_name, constraint_type, column_name
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.table_constraints tc ON kcu.constraint_name = tc.constraint_name
		WHERE kcu.table_schema = ? AND kcu.table_name = ?
		ORDER BY constraint_name, ordinal_position
	`

	keyRows, err := db.QueryContext(ctx, keyQuery, connectionConfig.Database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query table keys: %w", err)
	}
	defer keyRows.Close()

	keyMap := make(map[string]*KeyInfo)
	for keyRows.Next() {
		var keyName, keyType, columnName string

		err := keyRows.Scan(&keyName, &keyType, &columnName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key info: %w", err)
		}

		if key, exists := keyMap[keyName]; exists {
			key.Columns = append(key.Columns, columnName)
		} else {
			keyMap[keyName] = &KeyInfo{
				Name:    keyName,
				Type:    keyType,
				Columns: []string{columnName},
			}
		}
	}

	for _, key := range keyMap {
		schema.Keys = append(schema.Keys, key)
	}

	return schema, nil
}

// AddTableMapping adds a new table mapping to an existing sync configuration
// Requirement 3.2: Select tables for synchronization and save configuration
func (s *SyncManagerService) AddTableMapping(ctx context.Context, syncConfigID string, mapping *TableMapping) error {
	// Validate the mapping
	if err := s.validateTableMapping(mapping); err != nil {
		return fmt.Errorf("invalid table mapping: %w", err)
	}

	// Generate ID if not provided
	if mapping.ID == "" {
		mapping.ID = uuid.New().String()
	}

	// Set sync config ID and timestamps
	mapping.SyncConfigID = syncConfigID
	now := time.Now()
	mapping.CreatedAt = now
	mapping.UpdatedAt = now

	// Create table mapping in repository
	if err := s.repo.CreateTableMapping(ctx, mapping); err != nil {
		return fmt.Errorf("failed to create table mapping: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"sync_config_id": syncConfigID,
		"source_table":   mapping.SourceTable,
		"target_table":   mapping.TargetTable,
		"sync_mode":      mapping.SyncMode,
	}).Info("Table mapping added successfully")

	return nil
}

// UpdateTableMapping updates an existing table mapping
// Requirements 3.3, 3.4, 3.5: Configure sync rules, table mappings, enable/disable
func (s *SyncManagerService) UpdateTableMapping(ctx context.Context, mappingID string, mapping *TableMapping) error {
	// Validate the mapping
	if err := s.validateTableMapping(mapping); err != nil {
		return fmt.Errorf("invalid table mapping: %w", err)
	}

	// Update table mapping in repository
	if err := s.repo.UpdateTableMapping(ctx, mappingID, mapping); err != nil {
		return fmt.Errorf("failed to update table mapping: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"mapping_id":   mappingID,
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"sync_mode":    mapping.SyncMode,
		"enabled":      mapping.Enabled,
	}).Info("Table mapping updated successfully")

	return nil
}

// RemoveTableMapping removes a table mapping from a sync configuration
// Requirement 3.5: Enable/disable table synchronization (by removal)
func (s *SyncManagerService) RemoveTableMapping(ctx context.Context, mappingID string) error {
	if err := s.repo.DeleteTableMapping(ctx, mappingID); err != nil {
		return fmt.Errorf("failed to delete table mapping: %w", err)
	}

	s.logger.WithField("mapping_id", mappingID).Info("Table mapping removed successfully")
	return nil
}

// GetTableMappings retrieves all table mappings for a sync configuration
func (s *SyncManagerService) GetTableMappings(ctx context.Context, syncConfigID string) ([]*TableMapping, error) {
	mappings, err := s.repo.GetTableMappings(ctx, syncConfigID)
	if err != nil {
		return nil, fmt.Errorf("failed to get table mappings: %w", err)
	}
	return mappings, nil
}

// ToggleTableMapping enables or disables a specific table mapping
// Requirement 3.5: Enable/disable table synchronization
func (s *SyncManagerService) ToggleTableMapping(ctx context.Context, mappingID string, enabled bool) error {
	// Get existing mapping
	mappings, err := s.repo.GetTableMappings(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get table mappings: %w", err)
	}

	var targetMapping *TableMapping
	for _, mapping := range mappings {
		if mapping.ID == mappingID {
			targetMapping = mapping
			break
		}
	}

	if targetMapping == nil {
		return fmt.Errorf("table mapping not found: %s", mappingID)
	}

	// Update enabled status
	targetMapping.Enabled = enabled

	if err := s.repo.UpdateTableMapping(ctx, mappingID, targetMapping); err != nil {
		return fmt.Errorf("failed to toggle table mapping: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"mapping_id": mappingID,
		"enabled":    enabled,
	}).Info("Table mapping toggled successfully")

	return nil
}

// SetTableSyncMode updates the sync mode for a specific table mapping
// Requirement 3.3: Configure table sync rules (full/incremental mode)
func (s *SyncManagerService) SetTableSyncMode(ctx context.Context, mappingID string, syncMode SyncMode) error {
	// Validate sync mode
	if syncMode != SyncModeFull && syncMode != SyncModeIncremental {
		return fmt.Errorf("invalid sync mode: %s", syncMode)
	}

	// Get existing mapping
	mappings, err := s.repo.GetTableMappings(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get table mappings: %w", err)
	}

	var targetMapping *TableMapping
	for _, mapping := range mappings {
		if mapping.ID == mappingID {
			targetMapping = mapping
			break
		}
	}

	if targetMapping == nil {
		return fmt.Errorf("table mapping not found: %s", mappingID)
	}

	// Update sync mode
	targetMapping.SyncMode = syncMode

	if err := s.repo.UpdateTableMapping(ctx, mappingID, targetMapping); err != nil {
		return fmt.Errorf("failed to update table sync mode: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"mapping_id": mappingID,
		"sync_mode":  syncMode,
	}).Info("Table sync mode updated successfully")

	return nil
}

// validateTableMapping validates a table mapping configuration
func (s *SyncManagerService) validateTableMapping(mapping *TableMapping) error {
	if mapping.SourceTable == "" {
		return fmt.Errorf("source table is required")
	}
	if mapping.TargetTable == "" {
		return fmt.Errorf("target table is required")
	}
	if mapping.SyncMode != SyncModeFull && mapping.SyncMode != SyncModeIncremental {
		return fmt.Errorf("invalid sync mode: %s", mapping.SyncMode)
	}

	// Validate target table name format (MySQL identifier rules)
	if !isValidMySQLIdentifier(mapping.TargetTable) {
		return fmt.Errorf("invalid target table name: %s", mapping.TargetTable)
	}

	return nil
}

// updateTableMappings handles the update of table mappings when sync config is updated
func (s *SyncManagerService) updateTableMappings(ctx context.Context, syncConfigID string, existingMappings, newMappings []*TableMapping) error {
	// Create maps for easier comparison
	existingMap := make(map[string]*TableMapping)
	for _, mapping := range existingMappings {
		existingMap[mapping.ID] = mapping
	}

	newMap := make(map[string]*TableMapping)
	for _, mapping := range newMappings {
		if mapping.ID != "" {
			newMap[mapping.ID] = mapping
		}
	}

	// Delete removed mappings
	for id := range existingMap {
		if _, exists := newMap[id]; !exists {
			if err := s.repo.DeleteTableMapping(ctx, id); err != nil {
				s.logger.WithError(err).WithField("mapping_id", id).Error("Failed to delete table mapping")
			}
		}
	}

	// Update existing and create new mappings
	now := time.Now()
	for _, mapping := range newMappings {
		if mapping.ID == "" {
			// New mapping
			mapping.ID = uuid.New().String()
			mapping.SyncConfigID = syncConfigID
			mapping.CreatedAt = now
			mapping.UpdatedAt = now

			if err := s.repo.CreateTableMapping(ctx, mapping); err != nil {
				return fmt.Errorf("failed to create table mapping: %w", err)
			}
		} else {
			// Update existing mapping
			mapping.SyncConfigID = syncConfigID
			mapping.UpdatedAt = now

			if err := s.repo.UpdateTableMapping(ctx, mapping.ID, mapping); err != nil {
				return fmt.Errorf("failed to update table mapping: %w", err)
			}
		}
	}

	return nil
}

// createRemoteConnection creates a connection to a remote database
func (s *SyncManagerService) createRemoteConnection(config *ConnectionConfig) (*sqlx.DB, error) {
	// Build MySQL DSN for remote connection
	dsn, err := s.buildRemoteDSN(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	// Open connection to remote database
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	return db, nil
}

// buildRemoteDSN builds a MySQL DSN for remote connection
func (s *SyncManagerService) buildRemoteDSN(config *ConnectionConfig) (string, error) {
	mysqlConfig := mysql.Config{
		User:                 config.Username,
		Passwd:               config.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", config.Host, config.Port),
		DBName:               config.Database,
		Timeout:              10 * time.Second,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		AllowNativePasswords: true,
		ParseTime:            true,
		Loc:                  time.UTC,
	}

	// Configure SSL
	if config.SSL {
		mysqlConfig.TLSConfig = "true"
	} else {
		mysqlConfig.TLSConfig = "false"
	}

	return mysqlConfig.FormatDSN(), nil
}

// GetJobProgress returns the current progress of a sync job
// Requirement 5.1: Real-time display of sync progress and status
func (s *SyncManagerService) GetJobProgress(ctx context.Context, jobID string) (*JobSummary, error) {
	return s.monitoring.GetJobProgress(ctx, jobID)
}

// GetSyncHistory returns historical sync records
// Requirement 5.2: Display historical sync records and results
func (s *SyncManagerService) GetSyncHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error) {
	return s.monitoring.GetSyncHistory(ctx, limit, offset)
}

// GetSyncStatistics returns overall synchronization statistics
// Requirement 5.4: Display statistics information including data volume and time consumption
func (s *SyncManagerService) GetSyncStatistics(ctx context.Context) (*SyncStatistics, error) {
	return s.monitoring.GetSyncStatistics(ctx)
}

// GetActiveJobs returns currently running sync jobs
// Requirement 5.1: Real-time display of sync progress and status
func (s *SyncManagerService) GetActiveJobs(ctx context.Context) ([]*JobSummary, error) {
	return s.monitoring.GetActiveJobs(ctx)
}

// GetJobLogs returns logs for a specific job
// Requirement 5.3: Display detailed error information and suggestions when sync fails
func (s *SyncManagerService) GetJobLogs(ctx context.Context, jobID string) ([]*SyncLog, error) {
	return s.monitoring.GetJobLogs(ctx, jobID)
}
