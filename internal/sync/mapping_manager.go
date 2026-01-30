package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// MappingManagerImpl implements the MappingManager interface
type MappingManagerImpl struct {
	db     *sqlx.DB
	repo   Repository
	logger *logrus.Logger
}

// NewMappingManager creates a new MappingManager instance
func NewMappingManager(db *sqlx.DB, repo Repository, logger *logrus.Logger) MappingManager {
	return &MappingManagerImpl{
		db:     db,
		repo:   repo,
		logger: logger,
	}
}

// CreateDatabaseMapping creates a new database mapping
func (m *MappingManagerImpl) CreateDatabaseMapping(ctx context.Context, mapping *DatabaseMapping) error {
	m.logger.WithFields(logrus.Fields{
		"remote_connection_id": mapping.RemoteConnectionID,
		"local_database_name":  mapping.LocalDatabaseName,
	}).Info("Creating database mapping")

	// Validate the remote connection exists
	_, err := m.repo.GetConnection(ctx, mapping.RemoteConnectionID)
	if err != nil {
		return fmt.Errorf("remote connection not found: %w", err)
	}

	// Check if local database name is already used by another mapping
	existingMappings, err := m.repo.GetDatabaseMappings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get existing database mappings: %w", err)
	}
	for _, existing := range existingMappings {
		if existing.LocalDatabaseName == mapping.LocalDatabaseName && existing.RemoteConnectionID != mapping.RemoteConnectionID {
			return fmt.Errorf("local database name '%s' is already used by connection '%s'",
				mapping.LocalDatabaseName, existing.RemoteConnectionID)
		}
	}

	// Create local database if it doesn't exist
	err = m.createLocalDatabase(ctx, mapping.LocalDatabaseName)
	if err != nil {
		return fmt.Errorf("failed to create local database: %w", err)
	}

	// Persist mapping to database_mappings table
	if err := m.repo.CreateDatabaseMapping(ctx, mapping); err != nil {
		return fmt.Errorf("failed to save database mapping: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"remote_connection_id": mapping.RemoteConnectionID,
		"local_database_name":  mapping.LocalDatabaseName,
	}).Info("Database mapping created successfully")

	return nil
}

// GetDatabaseMappings returns all database mappings
func (m *MappingManagerImpl) GetDatabaseMappings(ctx context.Context) ([]*DatabaseMapping, error) {
	m.logger.Info("Getting all database mappings")

	mappings, err := m.repo.GetDatabaseMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database mappings: %w", err)
	}

	m.logger.WithField("count", len(mappings)).Info("Retrieved database mappings")
	return mappings, nil
}

// CheckTableConflicts checks for table name conflicts in local database
func (m *MappingManagerImpl) CheckTableConflicts(ctx context.Context, localDB string, tables []string) ([]string, error) {
	m.logger.WithFields(logrus.Fields{
		"local_database": localDB,
		"table_count":    len(tables),
	}).Info("Checking table conflicts")

	if len(tables) == 0 {
		return []string{}, nil
	}

	// Get existing tables in the local database
	existingTables, err := m.getExistingTables(ctx, localDB)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing tables: %w", err)
	}

	// Create a map for faster lookup
	existingTableMap := make(map[string]bool)
	for _, table := range existingTables {
		existingTableMap[strings.ToLower(table)] = true
	}

	// Check for conflicts
	var conflicts []string
	for _, table := range tables {
		if existingTableMap[strings.ToLower(table)] {
			conflicts = append(conflicts, table)
		}
	}

	m.logger.WithFields(logrus.Fields{
		"local_database": localDB,
		"conflicts":      len(conflicts),
	}).Info("Table conflict check completed")

	return conflicts, nil
}

// ExportConfig exports all sync configurations
func (m *MappingManagerImpl) ExportConfig(ctx context.Context) (*ConfigExport, error) {
	m.logger.Info("Exporting sync configuration")

	// Get all connections
	connections, err := m.repo.GetConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}

	// Get all database mappings
	mappings, err := m.GetDatabaseMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database mappings: %w", err)
	}

	// Get all sync configs
	var allSyncConfigs []*SyncConfig
	seenSyncConfigIDs := make(map[string]bool)
	for _, conn := range connections {
		configs, err := m.repo.GetSyncConfigs(ctx, conn.ID)
		if err != nil {
			m.logger.WithError(err).WithField("connection_id", conn.ID).Warn("Failed to get sync configs for connection")
			continue
		}
		for _, sc := range configs {
			if sc == nil || sc.ID == "" {
				continue
			}
			if seenSyncConfigIDs[sc.ID] {
				continue
			}
			seenSyncConfigIDs[sc.ID] = true
			allSyncConfigs = append(allSyncConfigs, sc)
		}
	}

	export := &ConfigExport{
		Version:     "1.0",
		ExportTime:  time.Now(),
		Connections: connections,
		Mappings:    mappings,
		SyncConfigs: allSyncConfigs,
	}

	m.logger.WithFields(logrus.Fields{
		"connections":  len(connections),
		"mappings":     len(mappings),
		"sync_configs": len(allSyncConfigs),
	}).Info("Configuration exported successfully")

	return export, nil
}

// ImportConfig imports sync configurations
func (m *MappingManagerImpl) ImportConfig(ctx context.Context, config *ConfigExport) error {
	m.logger.WithFields(logrus.Fields{
		"version":      config.Version,
		"connections":  len(config.Connections),
		"mappings":     len(config.Mappings),
		"sync_configs": len(config.SyncConfigs),
	}).Info("Importing sync configuration")

	// Validate configuration first
	err := m.ValidateConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Start transaction for atomic import
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Import connections
	for _, conn := range config.Connections {
		// Generate new ID to avoid conflicts
		originalID := conn.ID
		conn.ID = uuid.New().String()
		conn.CreatedAt = time.Now()
		conn.UpdatedAt = time.Now()

		// Use transaction directly for connection creation
		query := `
			INSERT INTO connections (id, name, host, port, username, password, database_name, ` + "`ssl`" + `)
			VALUES (:id, :name, :host, :port, :username, :password, :database, :ssl)
		`
		_, err = tx.NamedExecContext(ctx, query, conn)
		if err != nil {
			return fmt.Errorf("failed to import connection '%s': %w", conn.Name, err)
		}

		// Update sync configs to use new connection ID
		for _, syncConfig := range config.SyncConfigs {
			if syncConfig.SourceConnectionID == originalID {
				syncConfig.SourceConnectionID = conn.ID
			}
			if syncConfig.TargetConnectionID == originalID {
				syncConfig.TargetConnectionID = conn.ID
			}
		}

		// Update mappings to use new connection ID
		for _, mapping := range config.Mappings {
			if mapping.RemoteConnectionID == originalID {
				mapping.RemoteConnectionID = conn.ID
			}
		}
	}

	// Create local databases for mappings
	for _, mapping := range config.Mappings {
		err = m.createLocalDatabase(ctx, mapping.LocalDatabaseName)
		if err != nil {
			return fmt.Errorf("failed to create local database '%s': %w", mapping.LocalDatabaseName, err)
		}
	}

	// Import sync configurations
	for _, syncConfig := range config.SyncConfigs {
		// Generate new ID to avoid conflicts
		syncConfig.ID = uuid.New().String()
		syncConfig.CreatedAt = time.Now()
		syncConfig.UpdatedAt = time.Now()

		// Use transaction directly for sync config creation
		query := `
			INSERT INTO sync_configs (id, source_connection_id, target_connection_id, name, sync_mode, schedule, enabled, options)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err = tx.ExecContext(ctx, query, syncConfig.ID, syncConfig.SourceConnectionID, syncConfig.TargetConnectionID, syncConfig.Name,
			syncConfig.SyncMode, syncConfig.Schedule, syncConfig.Enabled, nil) // Simplified options for now
		if err != nil {
			return fmt.Errorf("failed to import sync config '%s': %w", syncConfig.Name, err)
		}

		// Import table mappings
		for _, tableMapping := range syncConfig.Tables {
			tableMapping.ID = uuid.New().String()
			tableMapping.SyncConfigID = syncConfig.ID
			tableMapping.CreatedAt = time.Now()
			tableMapping.UpdatedAt = time.Now()

			// Use transaction directly for table mapping creation
			query := `
				INSERT INTO table_mappings (id, sync_config_id, source_table, target_table, sync_mode, enabled, where_clause)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`
			_, err = tx.ExecContext(ctx, query, tableMapping.ID, tableMapping.SyncConfigID,
				tableMapping.SourceTable, tableMapping.TargetTable, tableMapping.SyncMode,
				tableMapping.Enabled, tableMapping.WhereClause)
			if err != nil {
				return fmt.Errorf("failed to import table mapping '%s': %w", tableMapping.SourceTable, err)
			}
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit import transaction: %w", err)
	}

	m.logger.Info("Configuration imported successfully")
	return nil
}

// ValidateConfig validates imported configuration
func (m *MappingManagerImpl) ValidateConfig(ctx context.Context, config *ConfigExport) error {
	m.logger.Info("Validating configuration")

	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	if config.Version == "" {
		return fmt.Errorf("configuration version is required")
	}

	// Validate connections
	connectionIDs := make(map[string]bool)

	for _, conn := range config.Connections {
		if conn.ID == "" {
			return fmt.Errorf("connection ID is required")
		}
		if conn.Name == "" {
			return fmt.Errorf("connection name is required")
		}
		if conn.Host == "" {
			return fmt.Errorf("connection host is required for connection '%s'", conn.Name)
		}
		if conn.Port <= 0 || conn.Port > 65535 {
			return fmt.Errorf("invalid port %d for connection '%s'", conn.Port, conn.Name)
		}
		if conn.Username == "" {
			return fmt.Errorf("connection username is required for connection '%s'", conn.Name)
		}
		if conn.Database == "" {
			return fmt.Errorf("connection database is required for connection '%s'", conn.Name)
		}

		// Check for duplicate connection IDs
		if connectionIDs[conn.ID] {
			return fmt.Errorf("duplicate connection ID: %s", conn.ID)
		}
		connectionIDs[conn.ID] = true
	}

	// Validate mappings reference valid connections
	for _, mapping := range config.Mappings {
		if !connectionIDs[mapping.RemoteConnectionID] {
			return fmt.Errorf("mapping references non-existent connection ID: %s", mapping.RemoteConnectionID)
		}
		if mapping.LocalDatabaseName == "" {
			return fmt.Errorf("local database name is required for mapping")
		}
	}

	// Validate sync configs
	syncConfigIDs := make(map[string]bool)
	for _, syncConfig := range config.SyncConfigs {
		if syncConfig.ID == "" {
			return fmt.Errorf("sync config ID is required")
		}
		if syncConfig.Name == "" {
			return fmt.Errorf("sync config name is required")
		}
		if !connectionIDs[syncConfig.SourceConnectionID] {
			return fmt.Errorf("sync config '%s' references non-existent source connection ID: %s",
				syncConfig.Name, syncConfig.SourceConnectionID)
		}
		if !connectionIDs[syncConfig.TargetConnectionID] {
			return fmt.Errorf("sync config '%s' references non-existent target connection ID: %s",
				syncConfig.Name, syncConfig.TargetConnectionID)
		}

		// Check for duplicate sync config IDs
		if syncConfigIDs[syncConfig.ID] {
			return fmt.Errorf("duplicate sync config ID: %s", syncConfig.ID)
		}
		syncConfigIDs[syncConfig.ID] = true

		// Validate sync mode
		if syncConfig.SyncMode != SyncModeFull && syncConfig.SyncMode != SyncModeIncremental {
			return fmt.Errorf("invalid sync mode '%s' for sync config '%s'", syncConfig.SyncMode, syncConfig.Name)
		}

		// Validate table mappings
		tableNames := make(map[string]bool)
		for _, tableMapping := range syncConfig.Tables {
			if tableMapping.SourceTable == "" {
				return fmt.Errorf("source table name is required in sync config '%s'", syncConfig.Name)
			}
			if tableMapping.TargetTable == "" {
				return fmt.Errorf("target table name is required in sync config '%s'", syncConfig.Name)
			}

			// Check for duplicate source tables within the same sync config
			if tableNames[tableMapping.SourceTable] {
				return fmt.Errorf("duplicate source table '%s' in sync config '%s'",
					tableMapping.SourceTable, syncConfig.Name)
			}
			tableNames[tableMapping.SourceTable] = true

			// Validate table sync mode
			if tableMapping.SyncMode != SyncModeFull && tableMapping.SyncMode != SyncModeIncremental {
				return fmt.Errorf("invalid sync mode '%s' for table '%s' in sync config '%s'",
					tableMapping.SyncMode, tableMapping.SourceTable, syncConfig.Name)
			}
		}
	}

	m.logger.Info("Configuration validation completed successfully")
	return nil
}

// createLocalDatabase creates a local database if it doesn't exist
func (m *MappingManagerImpl) createLocalDatabase(ctx context.Context, dbName string) error {
	m.logger.WithField("database", dbName).Info("Creating local database")

	// Check if database already exists
	var count int
	query := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
	err := m.db.GetContext(ctx, &count, query, dbName)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if count > 0 {
		m.logger.WithField("database", dbName).Info("Database already exists")
		return nil
	}

	// Create the database
	createQuery := fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
	_, err = m.db.ExecContext(ctx, createQuery)
	if err != nil {
		return fmt.Errorf("failed to create database '%s': %w", dbName, err)
	}

	m.logger.WithField("database", dbName).Info("Database created successfully")
	return nil
}

// getExistingTables returns list of existing tables in the specified database
func (m *MappingManagerImpl) getExistingTables(ctx context.Context, dbName string) ([]string, error) {
	query := "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'"

	var tables []string
	err := m.db.SelectContext(ctx, &tables, query, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing tables for database '%s': %w", dbName, err)
	}

	return tables, nil
}

// BackupConfig creates a backup of the current configuration
func (m *MappingManagerImpl) BackupConfig(ctx context.Context) (*ConfigExport, error) {
	m.logger.Info("Creating configuration backup")

	backup, err := m.ExportConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Add backup metadata
	backup.Version = fmt.Sprintf("backup-%s", backup.Version)

	m.logger.WithFields(logrus.Fields{
		"connections":  len(backup.Connections),
		"mappings":     len(backup.Mappings),
		"sync_configs": len(backup.SyncConfigs),
		"backup_time":  backup.ExportTime,
	}).Info("Configuration backup created successfully")

	return backup, nil
}

// ImportConfigWithConflictResolution imports configuration with conflict resolution options
func (m *MappingManagerImpl) ImportConfigWithConflictResolution(ctx context.Context, config *ConfigExport, resolveConflicts bool) error {
	m.logger.WithFields(logrus.Fields{
		"resolve_conflicts": resolveConflicts,
		"connections":       len(config.Connections),
		"mappings":          len(config.Mappings),
		"sync_configs":      len(config.SyncConfigs),
	}).Info("Importing configuration with conflict resolution")

	// Validate configuration first
	err := m.ValidateConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if resolveConflicts {
		err = m.resolveImportConflicts(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to resolve import conflicts: %w", err)
		}
	}

	return m.ImportConfig(ctx, config)
}

// resolveImportConflicts resolves conflicts in imported configuration
func (m *MappingManagerImpl) resolveImportConflicts(ctx context.Context, config *ConfigExport) error {
	m.logger.Info("Resolving import conflicts")

	// Get existing connections to check for conflicts
	existingConnections, err := m.repo.GetConnections(ctx)
	if err != nil {
		return fmt.Errorf("failed to get existing connections: %w", err)
	}

	// Create maps for conflict detection
	existingNames := make(map[string]bool)

	for _, conn := range existingConnections {
		existingNames[conn.Name] = true
	}

	// Resolve connection name conflicts
	for _, conn := range config.Connections {
		originalName := conn.Name
		counter := 1

		// Resolve name conflicts
		for existingNames[conn.Name] {
			conn.Name = fmt.Sprintf("%s_%d", originalName, counter)
			counter++
		}
		existingNames[conn.Name] = true

	}

	// Resolve sync config name conflicts within each connection
	connectionSyncConfigs := make(map[string]map[string]bool)

	for _, syncConfig := range config.SyncConfigs {
		// Use source_connection_id as the uniqueness scope (matches DB unique index)
		scopeID := syncConfig.SourceConnectionID
		if connectionSyncConfigs[scopeID] == nil {
			connectionSyncConfigs[scopeID] = make(map[string]bool)

			// Get existing sync configs for this connection
			existingSyncConfigs, err := m.repo.GetSyncConfigs(ctx, scopeID)
			if err == nil {
				for _, existing := range existingSyncConfigs {
					connectionSyncConfigs[scopeID][existing.Name] = true
				}
			}
		}

		originalName := syncConfig.Name
		counter := 1

		for connectionSyncConfigs[scopeID][syncConfig.Name] {
			syncConfig.Name = fmt.Sprintf("%s_%d", originalName, counter)
			counter++
		}
		connectionSyncConfigs[scopeID][syncConfig.Name] = true
	}

	m.logger.Info("Import conflicts resolved successfully")
	return nil
}

// ValidateConfigIntegrity performs deep validation of configuration integrity
func (m *MappingManagerImpl) ValidateConfigIntegrity(ctx context.Context, config *ConfigExport) error {
	m.logger.Info("Performing deep configuration integrity validation")

	// Basic validation first
	err := m.ValidateConfig(ctx, config)
	if err != nil {
		return err
	}

	// Additional integrity checks

	// Check that all mappings have corresponding connections
	connectionMap := make(map[string]*ConnectionConfig)
	for _, conn := range config.Connections {
		connectionMap[conn.ID] = conn
	}

	for _, mapping := range config.Mappings {
		conn, exists := connectionMap[mapping.RemoteConnectionID]
		if !exists {
			return fmt.Errorf("mapping references non-existent connection: %s", mapping.RemoteConnectionID)
		}
		_ = conn // mapping->connection consistency is no longer derived from ConnectionConfig fields
	}

	// Check sync config and table mapping consistency
	syncConfigMap := make(map[string]*SyncConfig)
	for _, syncConfig := range config.SyncConfigs {
		syncConfigMap[syncConfig.ID] = syncConfig

		// Verify connection exists
		if _, exists := connectionMap[syncConfig.SourceConnectionID]; !exists {
			return fmt.Errorf("sync config '%s' references non-existent source connection: %s",
				syncConfig.Name, syncConfig.SourceConnectionID)
		}
		if _, exists := connectionMap[syncConfig.TargetConnectionID]; !exists {
			return fmt.Errorf("sync config '%s' references non-existent target connection: %s",
				syncConfig.Name, syncConfig.TargetConnectionID)
		}

		// Validate table mappings
		for _, tableMapping := range syncConfig.Tables {
			if tableMapping.SyncConfigID != "" && tableMapping.SyncConfigID != syncConfig.ID {
				return fmt.Errorf("table mapping '%s' has inconsistent sync config ID", tableMapping.SourceTable)
			}
		}
	}

	// Check for circular dependencies (shouldn't exist in current design, but good to verify)
	// This is a placeholder for future complex dependency checks

	m.logger.Info("Configuration integrity validation completed successfully")
	return nil
}

// GetConfigurationSummary returns a summary of the current configuration
func (m *MappingManagerImpl) GetConfigurationSummary(ctx context.Context) (*ConfigurationSummary, error) {
	m.logger.Info("Getting configuration summary")

	connections, err := m.repo.GetConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}

	mappings, err := m.GetDatabaseMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database mappings: %w", err)
	}

	var totalSyncConfigs, totalTableMappings int
	var enabledSyncConfigs, enabledTableMappings int

	seenSyncConfigIDs := make(map[string]bool)
	for _, conn := range connections {
		syncConfigs, err := m.repo.GetSyncConfigs(ctx, conn.ID)
		if err != nil {
			m.logger.WithError(err).WithField("connection_id", conn.ID).Warn("Failed to get sync configs")
			continue
		}

		for _, syncConfig := range syncConfigs {
			if syncConfig == nil || syncConfig.ID == "" || seenSyncConfigIDs[syncConfig.ID] {
				continue
			}
			seenSyncConfigIDs[syncConfig.ID] = true
			totalSyncConfigs++
			if syncConfig.Enabled {
				enabledSyncConfigs++
			}

			totalTableMappings += len(syncConfig.Tables)
			for _, tableMapping := range syncConfig.Tables {
				if tableMapping.Enabled {
					enabledTableMappings++
				}
			}
		}
	}

	summary := &ConfigurationSummary{
		TotalConnections:     len(connections),
		TotalMappings:        len(mappings),
		TotalSyncConfigs:     totalSyncConfigs,
		EnabledSyncConfigs:   enabledSyncConfigs,
		TotalTableMappings:   totalTableMappings,
		EnabledTableMappings: enabledTableMappings,
		GeneratedAt:          time.Now(),
	}

	m.logger.WithFields(logrus.Fields{
		"connections":            summary.TotalConnections,
		"mappings":               summary.TotalMappings,
		"sync_configs":           summary.TotalSyncConfigs,
		"enabled_sync_configs":   summary.EnabledSyncConfigs,
		"table_mappings":         summary.TotalTableMappings,
		"enabled_table_mappings": summary.EnabledTableMappings,
	}).Info("Configuration summary generated")

	return summary, nil
}
