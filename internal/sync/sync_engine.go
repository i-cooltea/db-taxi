package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// DefaultSyncEngine implements the SyncEngine interface
type DefaultSyncEngine struct {
	localDB *sqlx.DB
	repo    Repository
	logger  *logrus.Logger
}

// NewSyncEngine creates a new sync engine instance
func NewSyncEngine(localDB *sqlx.DB, repo Repository, logger *logrus.Logger) SyncEngine {
	return &DefaultSyncEngine{
		localDB: localDB,
		repo:    repo,
		logger:  logger,
	}
}

// SyncTable synchronizes a single table based on its sync mode
func (e *DefaultSyncEngine) SyncTable(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	e.logger.WithFields(logrus.Fields{
		"job_id":       job.ID,
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"sync_mode":    mapping.SyncMode,
	}).Info("Starting table synchronization")

	switch mapping.SyncMode {
	case SyncModeFull:
		return e.SyncFull(ctx, job, mapping)
	case SyncModeIncremental:
		return e.SyncIncremental(ctx, job, mapping)
	default:
		return fmt.Errorf("unsupported sync mode: %s", mapping.SyncMode)
	}
}

// SyncFull performs full table synchronization
// Requirement 4.2: Execute full sync - copy table structure and sync all data
func (e *DefaultSyncEngine) SyncFull(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	e.logger.WithFields(logrus.Fields{
		"job_id":       job.ID,
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
	}).Info("Starting full table synchronization")

	// Get sync config to retrieve connection info
	syncConfig, err := e.repo.GetSyncConfig(ctx, mapping.SyncConfigID)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	// Get connection config
	connConfig, err := e.repo.GetConnection(ctx, syncConfig.ConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get connection config: %w", err)
	}

	// Connect to remote database
	remoteDB, err := e.connectToRemote(connConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to remote database: %w", err)
	}
	defer remoteDB.Close()

	// Get table schema from remote database
	schema, err := e.getTableSchemaFromRemote(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Create or recreate target table in local database
	if err := e.createOrRecreateTargetTable(ctx, connConfig.LocalDBName, mapping.TargetTable, schema); err != nil {
		return fmt.Errorf("failed to create target table: %w", err)
	}

	// Sync all data from source to target
	if err := e.syncAllData(ctx, remoteDB, connConfig.LocalDBName, mapping, syncConfig.Options); err != nil {
		return fmt.Errorf("failed to sync data: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"job_id":       job.ID,
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
	}).Info("Full table synchronization completed successfully")

	return nil
}

// SyncIncremental performs incremental table synchronization
// Requirement 4.3: Execute incremental sync - sync only changed data
func (e *DefaultSyncEngine) SyncIncremental(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	e.logger.WithFields(logrus.Fields{
		"job_id":       job.ID,
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
	}).Info("Starting incremental table synchronization")

	// Get sync config to retrieve connection info
	syncConfig, err := e.repo.GetSyncConfig(ctx, mapping.SyncConfigID)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	// Get connection config
	connConfig, err := e.repo.GetConnection(ctx, syncConfig.ConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get connection config: %w", err)
	}

	// Connect to remote database
	remoteDB, err := e.connectToRemote(connConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to remote database: %w", err)
	}
	defer remoteDB.Close()

	// Load checkpoint to determine last sync point
	checkpoint, err := e.repo.GetCheckpoint(ctx, mapping.ID)
	if err != nil {
		e.logger.WithError(err).Warn("Failed to load checkpoint, performing full sync instead")
		// If no checkpoint exists, fall back to full sync
		return e.SyncFull(ctx, job, mapping)
	}

	// If no checkpoint exists, perform full sync
	if checkpoint == nil {
		e.logger.Info("No checkpoint found, performing initial full sync")
		if err := e.SyncFull(ctx, job, mapping); err != nil {
			return err
		}
		// Create initial checkpoint after full sync
		return e.createInitialCheckpoint(ctx, mapping, remoteDB)
	}

	// Detect change tracking column (timestamp or auto-increment ID)
	changeColumn, changeType, err := e.detectChangeTrackingColumn(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to detect change tracking column: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"change_column": changeColumn,
		"change_type":   changeType,
		"last_sync":     checkpoint.LastSyncTime,
	}).Info("Detected change tracking column")

	// Ensure target table exists with correct schema
	schema, err := e.getTableSchemaFromRemote(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Check if target table exists, create if not
	if err := e.ensureTargetTableExists(ctx, connConfig.LocalDBName, mapping.TargetTable, schema); err != nil {
		return fmt.Errorf("failed to ensure target table exists: %w", err)
	}

	// Sync incremental changes based on change tracking type
	var syncedRows int64
	switch changeType {
	case "timestamp":
		syncedRows, err = e.syncIncrementalByTimestamp(ctx, remoteDB, connConfig.LocalDBName, mapping, changeColumn, checkpoint, syncConfig.Options)
	case "auto_increment":
		syncedRows, err = e.syncIncrementalByID(ctx, remoteDB, connConfig.LocalDBName, mapping, changeColumn, checkpoint, syncConfig.Options)
	default:
		return fmt.Errorf("unsupported change tracking type: %s", changeType)
	}

	if err != nil {
		return fmt.Errorf("failed to sync incremental data: %w", err)
	}

	// Update checkpoint with new sync time
	if err := e.updateCheckpoint(ctx, mapping, changeColumn, remoteDB); err != nil {
		e.logger.WithError(err).Warn("Failed to update checkpoint")
	}

	e.logger.WithFields(logrus.Fields{
		"job_id":       job.ID,
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"synced_rows":  syncedRows,
	}).Info("Incremental table synchronization completed successfully")

	return nil
}

// ValidateData validates data consistency between source and target
func (e *DefaultSyncEngine) ValidateData(ctx context.Context, mapping *TableMapping) error {
	// TODO: Implement data validation in task 6.5
	return fmt.Errorf("data validation not yet implemented")
}

// GetTableSchema retrieves table schema from source database
func (e *DefaultSyncEngine) GetTableSchema(ctx context.Context, connectionID, tableName string) (*TableSchema, error) {
	// Get connection config
	connConfig, err := e.repo.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection config: %w", err)
	}

	// Connect to remote database
	remoteDB, err := e.connectToRemote(connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote database: %w", err)
	}
	defer remoteDB.Close()

	return e.getTableSchemaFromRemote(ctx, remoteDB, tableName)
}

// CreateTargetTable creates target table with source schema
func (e *DefaultSyncEngine) CreateTargetTable(ctx context.Context, localDB string, schema *TableSchema) error {
	return e.createOrRecreateTargetTable(ctx, localDB, schema.Name, schema)
}

// connectToRemote establishes a connection to the remote database
func (e *DefaultSyncEngine) connectToRemote(config *ConnectionConfig) (*sqlx.DB, error) {
	mysqlConfig := mysql.Config{
		User:                 config.Username,
		Passwd:               config.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", config.Host, config.Port),
		DBName:               config.Database,
		Timeout:              30 * time.Second,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		AllowNativePasswords: true,
		ParseTime:            true,
		Loc:                  time.UTC,
	}

	if config.SSL {
		mysqlConfig.TLSConfig = "true"
	} else {
		mysqlConfig.TLSConfig = "false"
	}

	dsn := mysqlConfig.FormatDSN()
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping remote database: %w", err)
	}

	return db, nil
}

// getTableSchemaFromRemote retrieves table schema from remote database
func (e *DefaultSyncEngine) getTableSchemaFromRemote(ctx context.Context, remoteDB *sqlx.DB, tableName string) (*TableSchema, error) {
	schema := &TableSchema{
		Name:    tableName,
		Columns: []*ColumnInfo{},
		Indexes: []*IndexInfo{},
		Keys:    []*KeyInfo{},
	}

	// Get column information
	columnQuery := `
		SELECT 
			COLUMN_NAME, 
			COLUMN_TYPE, 
			IS_NULLABLE, 
			COLUMN_DEFAULT, 
			EXTRA
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := remoteDB.QueryContext(ctx, columnQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query column information: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var nullable string
		var defaultValue sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &nullable, &defaultValue, &col.Extra); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		col.Nullable = (nullable == "YES")
		if defaultValue.Valid {
			col.DefaultValue = defaultValue.String
		}

		schema.Columns = append(schema.Columns, &col)
	}

	if len(schema.Columns) == 0 {
		return nil, fmt.Errorf("table not found or has no columns: %s", tableName)
	}

	// Get index information
	indexQuery := `
		SELECT 
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			INDEX_TYPE
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`

	indexRows, err := remoteDB.QueryContext(ctx, indexQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query index information: %w", err)
	}
	defer indexRows.Close()

	indexMap := make(map[string]*IndexInfo)
	for indexRows.Next() {
		var indexName, columnName, indexType string
		var nonUnique int

		if err := indexRows.Scan(&indexName, &columnName, &nonUnique, &indexType); err != nil {
			return nil, fmt.Errorf("failed to scan index info: %w", err)
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[indexName] = &IndexInfo{
				Name:    indexName,
				Columns: []string{columnName},
				Unique:  (nonUnique == 0),
				Type:    indexType,
			}
		}
	}

	for _, idx := range indexMap {
		schema.Indexes = append(schema.Indexes, idx)
	}

	// Get key information (PRIMARY, FOREIGN, UNIQUE)
	keyQuery := `
		SELECT 
			CONSTRAINT_NAME,
			CONSTRAINT_TYPE
		FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
	`

	keyRows, err := remoteDB.QueryContext(ctx, keyQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query key information: %w", err)
	}
	defer keyRows.Close()

	for keyRows.Next() {
		var keyName, keyType string
		if err := keyRows.Scan(&keyName, &keyType); err != nil {
			return nil, fmt.Errorf("failed to scan key info: %w", err)
		}

		// Get columns for this key
		var columns []string
		if idx, exists := indexMap[keyName]; exists {
			columns = idx.Columns
		}

		schema.Keys = append(schema.Keys, &KeyInfo{
			Name:    keyName,
			Type:    keyType,
			Columns: columns,
		})
	}

	return schema, nil
}

// createOrRecreateTargetTable creates or recreates the target table in local database
func (e *DefaultSyncEngine) createOrRecreateTargetTable(ctx context.Context, localDB, tableName string, schema *TableSchema) error {
	e.logger.WithFields(logrus.Fields{
		"local_db":   localDB,
		"table_name": tableName,
	}).Info("Creating or recreating target table")

	// Ensure local database exists
	if err := e.ensureLocalDatabaseExists(ctx, localDB); err != nil {
		return fmt.Errorf("failed to ensure local database exists: %w", err)
	}

	// Drop existing table if it exists
	dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", localDB, tableName)
	if _, err := e.localDB.ExecContext(ctx, dropQuery); err != nil {
		return fmt.Errorf("failed to drop existing table: %w", err)
	}

	// Build CREATE TABLE statement
	createQuery := e.buildCreateTableStatement(localDB, tableName, schema)

	// Execute CREATE TABLE
	if _, err := e.localDB.ExecContext(ctx, createQuery); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"local_db":   localDB,
		"table_name": tableName,
	}).Info("Target table created successfully")

	return nil
}

// buildCreateTableStatement builds a CREATE TABLE statement from schema
func (e *DefaultSyncEngine) buildCreateTableStatement(localDB, tableName string, schema *TableSchema) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TABLE `%s`.`%s` (\n", localDB, tableName))

	// Add columns
	for i, col := range schema.Columns {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("  `%s` %s", col.Name, col.Type))

		if !col.Nullable {
			sb.WriteString(" NOT NULL")
		}

		if col.DefaultValue != "" {
			sb.WriteString(fmt.Sprintf(" DEFAULT %s", col.DefaultValue))
		}

		if col.Extra != "" {
			sb.WriteString(fmt.Sprintf(" %s", col.Extra))
		}
	}

	// Add primary key
	for _, key := range schema.Keys {
		if key.Type == "PRIMARY KEY" {
			sb.WriteString(",\n")
			sb.WriteString(fmt.Sprintf("  PRIMARY KEY (`%s`)", strings.Join(key.Columns, "`, `")))
			break
		}
	}

	// Add unique keys
	for _, key := range schema.Keys {
		if key.Type == "UNIQUE" {
			sb.WriteString(",\n")
			sb.WriteString(fmt.Sprintf("  UNIQUE KEY `%s` (`%s`)", key.Name, strings.Join(key.Columns, "`, `")))
		}
	}

	// Add indexes (excluding primary and unique which are already added)
	for _, idx := range schema.Indexes {
		if idx.Name != "PRIMARY" && !idx.Unique {
			sb.WriteString(",\n")
			sb.WriteString(fmt.Sprintf("  KEY `%s` (`%s`)", idx.Name, strings.Join(idx.Columns, "`, `")))
		}
	}

	sb.WriteString("\n)")

	return sb.String()
}

// syncAllData synchronizes all data from source to target table
func (e *DefaultSyncEngine) syncAllData(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping, options *SyncOptions) error {
	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"local_db":     localDB,
	}).Info("Starting data synchronization")

	// Determine batch size
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	// Build SELECT query with optional WHERE clause
	selectQuery := fmt.Sprintf("SELECT * FROM `%s`", mapping.SourceTable)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}

	// Get total row count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", mapping.SourceTable)
	if mapping.WhereClause != "" {
		countQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}

	var totalRows int64
	if err := remoteDB.GetContext(ctx, &totalRows, countQuery); err != nil {
		return fmt.Errorf("failed to get row count: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"total_rows":   totalRows,
		"batch_size":   batchSize,
	}).Info("Starting batch data transfer")

	// Query all data from source
	rows, err := remoteDB.QueryxContext(ctx, selectQuery)
	if err != nil {
		return fmt.Errorf("failed to query source data: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get column names: %w", err)
	}

	// Prepare batch insert
	var batch []map[string]interface{}
	processedRows := int64(0)

	for rows.Next() {
		// Scan row into map
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		// Insert batch when it reaches batch size
		if len(batch) >= batchSize {
			if err := e.insertBatch(ctx, localDB, mapping.TargetTable, columns, batch); err != nil {
				return fmt.Errorf("failed to insert batch: %w", err)
			}
			processedRows += int64(len(batch))
			batch = batch[:0] // Clear batch

			e.logger.WithFields(logrus.Fields{
				"source_table":   mapping.SourceTable,
				"processed_rows": processedRows,
				"total_rows":     totalRows,
				"progress":       fmt.Sprintf("%.2f%%", float64(processedRows)/float64(totalRows)*100),
			}).Debug("Batch inserted")
		}
	}

	// Insert remaining rows
	if len(batch) > 0 {
		if err := e.insertBatch(ctx, localDB, mapping.TargetTable, columns, batch); err != nil {
			return fmt.Errorf("failed to insert final batch: %w", err)
		}
		processedRows += int64(len(batch))
	}

	e.logger.WithFields(logrus.Fields{
		"source_table":   mapping.SourceTable,
		"target_table":   mapping.TargetTable,
		"processed_rows": processedRows,
		"total_rows":     totalRows,
	}).Info("Data synchronization completed successfully")

	return nil
}

// insertBatch inserts a batch of rows into the target table
func (e *DefaultSyncEngine) insertBatch(ctx context.Context, localDB, tableName string, columns []string, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	// Build INSERT statement
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` (", localDB, tableName))

	// Add column names
	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("`%s`", col))
	}

	sb.WriteString(") VALUES ")

	// Add value placeholders
	args := make([]interface{}, 0, len(batch)*len(columns))
	for i := range batch {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j, col := range columns {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("?")
			args = append(args, batch[i][col])
		}
		sb.WriteString(")")
	}

	// Execute INSERT
	if _, err := e.localDB.ExecContext(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}

	return nil
}

// ensureLocalDatabaseExists ensures the local database exists, creating it if necessary
func (e *DefaultSyncEngine) ensureLocalDatabaseExists(ctx context.Context, localDB string) error {
	// Check if database exists
	var count int
	checkQuery := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
	if err := e.localDB.GetContext(ctx, &count, checkQuery, localDB); err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if count == 0 {
		// Create database
		createQuery := fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", localDB)
		if _, err := e.localDB.ExecContext(ctx, createQuery); err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}

		e.logger.WithField("database", localDB).Info("Local database created")
	}

	return nil
}

// detectChangeTrackingColumn detects the column to use for change tracking
// Returns column name, type (timestamp/auto_increment), and error
func (e *DefaultSyncEngine) detectChangeTrackingColumn(ctx context.Context, remoteDB *sqlx.DB, tableName string) (string, string, error) {
	// First, try to find a timestamp column (updated_at, modified_at, etc.)
	timestampQuery := `
		SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = ?
		AND (
			COLUMN_NAME IN ('updated_at', 'modified_at', 'last_modified', 'update_time', 'modify_time')
			OR (DATA_TYPE IN ('timestamp', 'datetime') AND COLUMN_NAME LIKE '%update%')
		)
		ORDER BY 
			CASE COLUMN_NAME
				WHEN 'updated_at' THEN 1
				WHEN 'modified_at' THEN 2
				WHEN 'last_modified' THEN 3
				ELSE 4
			END
		LIMIT 1
	`

	var timestampColumn string
	err := remoteDB.GetContext(ctx, &timestampColumn, timestampQuery, tableName)
	if err == nil && timestampColumn != "" {
		return timestampColumn, "timestamp", nil
	}

	// If no timestamp column, try to find an auto-increment primary key
	autoIncrementQuery := `
		SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = ?
		AND EXTRA LIKE '%auto_increment%'
		LIMIT 1
	`

	var autoIncrementColumn string
	err = remoteDB.GetContext(ctx, &autoIncrementColumn, autoIncrementQuery, tableName)
	if err == nil && autoIncrementColumn != "" {
		return autoIncrementColumn, "auto_increment", nil
	}

	// If neither found, return error
	return "", "", fmt.Errorf("no suitable change tracking column found (need timestamp or auto_increment column)")
}

// syncIncrementalByTimestamp syncs data based on timestamp column
func (e *DefaultSyncEngine) syncIncrementalByTimestamp(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping, timestampColumn string, checkpoint *SyncCheckpoint, options *SyncOptions) (int64, error) {
	e.logger.WithFields(logrus.Fields{
		"source_table":     mapping.SourceTable,
		"timestamp_column": timestampColumn,
		"last_sync_time":   checkpoint.LastSyncTime,
	}).Info("Syncing incremental data by timestamp")

	// Determine batch size
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	// Build SELECT query for changed records
	selectQuery := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` > ?", mapping.SourceTable, timestampColumn)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" AND (%s)", mapping.WhereClause)
	}
	selectQuery += fmt.Sprintf(" ORDER BY `%s`", timestampColumn)

	// Query changed data from source
	rows, err := remoteDB.QueryxContext(ctx, selectQuery, checkpoint.LastSyncTime)
	if err != nil {
		return 0, fmt.Errorf("failed to query changed data: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get column names: %w", err)
	}

	// Get primary key columns for upsert
	primaryKeys, err := e.getPrimaryKeyColumns(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary key columns: %w", err)
	}

	// Process rows in batches
	var batch []map[string]interface{}
	var totalSynced int64

	for rows.Next() {
		// Scan row into map
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return totalSynced, fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		// Upsert batch when it reaches batch size
		if len(batch) >= batchSize {
			if err := e.upsertBatch(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
				return totalSynced, fmt.Errorf("failed to upsert batch: %w", err)
			}
			totalSynced += int64(len(batch))
			batch = batch[:0] // Clear batch

			e.logger.WithFields(logrus.Fields{
				"source_table": mapping.SourceTable,
				"synced_rows":  totalSynced,
			}).Debug("Batch upserted")
		}
	}

	// Upsert remaining rows
	if len(batch) > 0 {
		if err := e.upsertBatch(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
			return totalSynced, fmt.Errorf("failed to upsert final batch: %w", err)
		}
		totalSynced += int64(len(batch))
	}

	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"synced_rows":  totalSynced,
	}).Info("Timestamp-based incremental sync completed")

	return totalSynced, nil
}

// syncIncrementalByID syncs data based on auto-increment ID column
func (e *DefaultSyncEngine) syncIncrementalByID(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping, idColumn string, checkpoint *SyncCheckpoint, options *SyncOptions) (int64, error) {
	e.logger.WithFields(logrus.Fields{
		"source_table":    mapping.SourceTable,
		"id_column":       idColumn,
		"last_sync_value": checkpoint.LastSyncValue,
	}).Info("Syncing incremental data by ID")

	// Determine batch size
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	// Parse last sync value as integer
	var lastID int64
	if checkpoint.LastSyncValue != "" {
		fmt.Sscanf(checkpoint.LastSyncValue, "%d", &lastID)
	}

	// Build SELECT query for new records
	selectQuery := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` > ?", mapping.SourceTable, idColumn)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" AND (%s)", mapping.WhereClause)
	}
	selectQuery += fmt.Sprintf(" ORDER BY `%s`", idColumn)

	// Query new data from source
	rows, err := remoteDB.QueryxContext(ctx, selectQuery, lastID)
	if err != nil {
		return 0, fmt.Errorf("failed to query new data: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get column names: %w", err)
	}

	// Get primary key columns for upsert
	primaryKeys, err := e.getPrimaryKeyColumns(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary key columns: %w", err)
	}

	// Process rows in batches
	var batch []map[string]interface{}
	var totalSynced int64

	for rows.Next() {
		// Scan row into map
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return totalSynced, fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		// Upsert batch when it reaches batch size
		if len(batch) >= batchSize {
			if err := e.upsertBatch(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
				return totalSynced, fmt.Errorf("failed to upsert batch: %w", err)
			}
			totalSynced += int64(len(batch))
			batch = batch[:0] // Clear batch

			e.logger.WithFields(logrus.Fields{
				"source_table": mapping.SourceTable,
				"synced_rows":  totalSynced,
			}).Debug("Batch upserted")
		}
	}

	// Upsert remaining rows
	if len(batch) > 0 {
		if err := e.upsertBatch(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
			return totalSynced, fmt.Errorf("failed to upsert final batch: %w", err)
		}
		totalSynced += int64(len(batch))
	}

	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"synced_rows":  totalSynced,
	}).Info("ID-based incremental sync completed")

	return totalSynced, nil
}

// getPrimaryKeyColumns retrieves the primary key columns for a table
func (e *DefaultSyncEngine) getPrimaryKeyColumns(ctx context.Context, remoteDB *sqlx.DB, tableName string) ([]string, error) {
	query := `
		SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = ?
		AND CONSTRAINT_NAME = 'PRIMARY'
		ORDER BY ORDINAL_POSITION
	`

	var primaryKeys []string
	rows, err := remoteDB.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query primary keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, fmt.Errorf("failed to scan primary key column: %w", err)
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	if len(primaryKeys) == 0 {
		return nil, fmt.Errorf("no primary key found for table %s", tableName)
	}

	return primaryKeys, nil
}

// upsertBatch performs INSERT ... ON DUPLICATE KEY UPDATE for a batch of rows
func (e *DefaultSyncEngine) upsertBatch(ctx context.Context, localDB, tableName string, columns, primaryKeys []string, batch []map[string]interface{}, options *SyncOptions) error {
	if len(batch) == 0 {
		return nil
	}

	// Determine conflict resolution strategy
	conflictResolution := ConflictResolutionOverwrite
	if options != nil {
		conflictResolution = options.ConflictResolution
	}

	// Build INSERT statement
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` (", localDB, tableName))

	// Add column names
	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("`%s`", col))
	}

	sb.WriteString(") VALUES ")

	// Add value placeholders
	args := make([]interface{}, 0, len(batch)*len(columns))
	for i := range batch {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j, col := range columns {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("?")
			args = append(args, batch[i][col])
		}
		sb.WriteString(")")
	}

	// Add conflict resolution clause
	switch conflictResolution {
	case ConflictResolutionOverwrite:
		sb.WriteString(" ON DUPLICATE KEY UPDATE ")
		first := true
		for _, col := range columns {
			// Skip primary key columns in UPDATE clause
			isPrimaryKey := false
			for _, pk := range primaryKeys {
				if col == pk {
					isPrimaryKey = true
					break
				}
			}
			if isPrimaryKey {
				continue
			}

			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("`%s` = VALUES(`%s`)", col, col))
			first = false
		}
	case ConflictResolutionSkip:
		sb.WriteString(" ON DUPLICATE KEY UPDATE ")
		// Update a dummy column to effectively skip the update
		sb.WriteString(fmt.Sprintf("`%s` = `%s`", primaryKeys[0], primaryKeys[0]))
	case ConflictResolutionError:
		// No ON DUPLICATE KEY clause, will error on conflict
	}

	// Execute INSERT/UPSERT
	if _, err := e.localDB.ExecContext(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("failed to execute batch upsert: %w", err)
	}

	return nil
}

// createInitialCheckpoint creates an initial checkpoint after full sync
func (e *DefaultSyncEngine) createInitialCheckpoint(ctx context.Context, mapping *TableMapping, remoteDB *sqlx.DB) error {
	// Detect change tracking column
	changeColumn, changeType, err := e.detectChangeTrackingColumn(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		e.logger.WithError(err).Warn("Failed to detect change tracking column, checkpoint not created")
		return nil // Don't fail the sync if checkpoint creation fails
	}

	// Get the maximum value of the change tracking column
	var maxValue interface{}
	var query string

	switch changeType {
	case "timestamp":
		query = fmt.Sprintf("SELECT MAX(`%s`) FROM `%s`", changeColumn, mapping.SourceTable)
	case "auto_increment":
		query = fmt.Sprintf("SELECT MAX(`%s`) FROM `%s`", changeColumn, mapping.SourceTable)
	}

	if err := remoteDB.GetContext(ctx, &maxValue, query); err != nil {
		e.logger.WithError(err).Warn("Failed to get max change tracking value")
		return nil
	}

	// Create checkpoint
	checkpoint := &SyncCheckpoint{
		ID:             mapping.ID,
		TableMappingID: mapping.ID,
		LastSyncTime:   time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if maxValue != nil {
		checkpoint.LastSyncValue = fmt.Sprintf("%v", maxValue)
	}

	if err := e.repo.CreateCheckpoint(ctx, checkpoint); err != nil {
		e.logger.WithError(err).Warn("Failed to create initial checkpoint")
		return nil
	}

	e.logger.WithFields(logrus.Fields{
		"table_mapping_id": mapping.ID,
		"change_column":    changeColumn,
		"max_value":        maxValue,
	}).Info("Initial checkpoint created")

	return nil
}

// updateCheckpoint updates the checkpoint after incremental sync
func (e *DefaultSyncEngine) updateCheckpoint(ctx context.Context, mapping *TableMapping, changeColumn string, remoteDB *sqlx.DB) error {
	// Get the maximum value of the change tracking column
	query := fmt.Sprintf("SELECT MAX(`%s`) FROM `%s`", changeColumn, mapping.SourceTable)

	var maxValue interface{}
	if err := remoteDB.GetContext(ctx, &maxValue, query); err != nil {
		return fmt.Errorf("failed to get max change tracking value: %w", err)
	}

	// Update checkpoint
	checkpoint := &SyncCheckpoint{
		ID:             mapping.ID,
		TableMappingID: mapping.ID,
		LastSyncTime:   time.Now(),
		UpdatedAt:      time.Now(),
	}

	if maxValue != nil {
		checkpoint.LastSyncValue = fmt.Sprintf("%v", maxValue)
	}

	if err := e.repo.UpdateCheckpoint(ctx, mapping.ID, checkpoint); err != nil {
		return fmt.Errorf("failed to update checkpoint: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"table_mapping_id": mapping.ID,
		"change_column":    changeColumn,
		"max_value":        maxValue,
	}).Debug("Checkpoint updated")

	return nil
}

// ensureTargetTableExists ensures the target table exists, creating it if necessary
func (e *DefaultSyncEngine) ensureTargetTableExists(ctx context.Context, localDB, tableName string, schema *TableSchema) error {
	// Check if table exists
	var count int
	checkQuery := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
	if err := e.localDB.GetContext(ctx, &count, checkQuery, localDB, tableName); err != nil {
		return fmt.Errorf("failed to check if table exists: %w", err)
	}

	if count == 0 {
		// Table doesn't exist, create it
		e.logger.WithFields(logrus.Fields{
			"local_db":   localDB,
			"table_name": tableName,
		}).Info("Target table does not exist, creating it")

		return e.createOrRecreateTargetTable(ctx, localDB, tableName, schema)
	}

	return nil
}
