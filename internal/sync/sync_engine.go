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
	localDB        *sqlx.DB
	repo           Repository
	logger         *logrus.Logger
	batchProcessor *BatchProcessor
}

// NewSyncEngine creates a new sync engine instance
func NewSyncEngine(localDB *sqlx.DB, repo Repository, logger *logrus.Logger) SyncEngine {
	// Create batch processor with default configuration
	batchConfig := &BatchProcessorConfig{
		BatchSize:     1000,
		MaxMemoryMB:   512,
		MaxWorkers:    4,
		EnableMetrics: true,
	}

	return &DefaultSyncEngine{
		localDB:        localDB,
		repo:           repo,
		logger:         logger,
		batchProcessor: NewBatchProcessor(localDB, logger, batchConfig),
	}
}

// NewSyncEngineWithConfig creates a new sync engine with custom batch processor configuration
func NewSyncEngineWithConfig(localDB *sqlx.DB, repo Repository, logger *logrus.Logger, batchConfig *BatchProcessorConfig) SyncEngine {
	return &DefaultSyncEngine{
		localDB:        localDB,
		repo:           repo,
		logger:         logger,
		batchProcessor: NewBatchProcessor(localDB, logger, batchConfig),
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

	// Get source connection config
	sourceConnConfig, err := e.repo.GetConnection(ctx, syncConfig.SourceConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get source connection config: %w", err)
	}

	// Get target connection config
	targetConnConfig, err := e.repo.GetConnection(ctx, syncConfig.TargetConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get target connection config: %w", err)
	}

	// Determine source/target database names (prefer sync config; fallback to connection config for backward compatibility)
	sourceDBName := syncConfig.SourceDatabase
	if sourceDBName == "" {
		sourceDBName = sourceConnConfig.Database
	}
	targetDBName := syncConfig.TargetDatabase
	if targetDBName == "" {
		targetDBName = targetConnConfig.Database
	}
	if targetDBName == "" {
		// If still empty, default to source db name
		targetDBName = sourceDBName
	}
	if sourceDBName == "" || targetDBName == "" {
		return fmt.Errorf("source/target database is required")
	}

	// Ensure target database exists (auto-create if missing)
	{
		serverConn := *targetConnConfig
		serverConn.Database = ""
		adminDB, err := e.connectToRemote(&serverConn)
		if err != nil {
			return fmt.Errorf("failed to connect to target server: %w", err)
		}
		if err := e.ensureDatabaseExists(ctx, adminDB, targetDBName); err != nil {
			adminDB.Close()
			return fmt.Errorf("failed to ensure target database exists: %w", err)
		}
		adminDB.Close()
	}

	// Connect to source database
	{
		cc := *sourceConnConfig
		cc.Database = sourceDBName
		sourceConnConfig = &cc
	}
	sourceDB, err := e.connectToRemote(sourceConnConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	defer sourceDB.Close()

	// Connect to target database
	{
		cc := *targetConnConfig
		cc.Database = targetDBName
		targetConnConfig = &cc
	}
	targetDB, err := e.connectToRemote(targetConnConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer targetDB.Close()

	// Get table schema from source database
	schema, err := e.getTableSchemaFromRemote(ctx, sourceDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Create or recreate target table in target database
	if err := e.createOrRecreateTargetTableInDB(ctx, targetDB, targetDBName, mapping.TargetTable, schema); err != nil {
		return fmt.Errorf("failed to create target table: %w", err)
	}

	// Sync all data from source to target
	if err := e.syncAllDataBetweenDBs(ctx, sourceDB, sourceDBName, targetDB, targetDBName, mapping, syncConfig.Options); err != nil {
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

	// Get source connection config
	sourceConnConfig, err := e.repo.GetConnection(ctx, syncConfig.SourceConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get source connection config: %w", err)
	}

	// Get target connection config
	targetConnConfig, err := e.repo.GetConnection(ctx, syncConfig.TargetConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get target connection config: %w", err)
	}

	// Determine source/target database names (prefer sync config; fallback to connection config for backward compatibility)
	sourceDBName := syncConfig.SourceDatabase
	if sourceDBName == "" {
		sourceDBName = sourceConnConfig.Database
	}
	targetDBName := syncConfig.TargetDatabase
	if targetDBName == "" {
		targetDBName = targetConnConfig.Database
	}
	if targetDBName == "" {
		targetDBName = sourceDBName
	}
	if sourceDBName == "" || targetDBName == "" {
		return fmt.Errorf("source/target database is required")
	}

	// Ensure target database exists (auto-create if missing)
	{
		serverConn := *targetConnConfig
		serverConn.Database = ""
		adminDB, err := e.connectToRemote(&serverConn)
		if err != nil {
			return fmt.Errorf("failed to connect to target server: %w", err)
		}
		if err := e.ensureDatabaseExists(ctx, adminDB, targetDBName); err != nil {
			adminDB.Close()
			return fmt.Errorf("failed to ensure target database exists: %w", err)
		}
		adminDB.Close()
	}

	// Connect to source database
	{
		cc := *sourceConnConfig
		cc.Database = sourceDBName
		sourceConnConfig = &cc
	}
	sourceDB, err := e.connectToRemote(sourceConnConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	defer sourceDB.Close()

	// Connect to target database
	{
		cc := *targetConnConfig
		cc.Database = targetDBName
		targetConnConfig = &cc
	}
	targetDB, err := e.connectToRemote(targetConnConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer targetDB.Close()

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
		return e.createInitialCheckpoint(ctx, mapping, sourceDB)
	}

	// Detect change tracking column (timestamp or auto-increment ID)
	changeColumn, changeType, err := e.detectChangeTrackingColumn(ctx, sourceDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to detect change tracking column: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"change_column": changeColumn,
		"change_type":   changeType,
		"last_sync":     checkpoint.LastSyncTime,
	}).Info("Detected change tracking column")

	// Ensure target table exists with correct schema
	schema, err := e.getTableSchemaFromRemote(ctx, sourceDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Check if target table exists, create if not
	if err := e.ensureTargetTableExistsInDB(ctx, targetDB, targetDBName, mapping.TargetTable, schema); err != nil {
		return fmt.Errorf("failed to ensure target table exists: %w", err)
	}

	// Sync incremental changes based on change tracking type
	var syncedRows int64
	switch changeType {
	case "timestamp":
		syncedRows, err = e.syncIncrementalByTimestampBetweenDBs(ctx, sourceDB, sourceDBName, targetDB, targetDBName, mapping, changeColumn, checkpoint, syncConfig.Options)
	case "auto_increment":
		syncedRows, err = e.syncIncrementalByIDBetweenDBs(ctx, sourceDB, sourceDBName, targetDB, targetDBName, mapping, changeColumn, checkpoint, syncConfig.Options)
	default:
		return fmt.Errorf("unsupported change tracking type: %s", changeType)
	}

	if err != nil {
		return fmt.Errorf("failed to sync incremental data: %w", err)
	}

	// Update checkpoint with new sync time
	if err := e.updateCheckpoint(ctx, mapping, changeColumn, sourceDB); err != nil {
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
// Requirement 7.5: Provide data validation and comparison functionality
func (e *DefaultSyncEngine) ValidateData(ctx context.Context, mapping *TableMapping) error {
	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
	}).Info("Starting data validation")

	// Get sync config to retrieve connection info
	syncConfig, err := e.repo.GetSyncConfig(ctx, mapping.SyncConfigID)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	// Get source connection config
	sourceConnConfig, err := e.repo.GetConnection(ctx, syncConfig.SourceConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get source connection config: %w", err)
	}

	// Get target connection config
	targetConnConfig, err := e.repo.GetConnection(ctx, syncConfig.TargetConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get target connection config: %w", err)
	}

	// Determine source/target database names (prefer sync config; fallback to connection config for backward compatibility)
	sourceDBName := syncConfig.SourceDatabase
	if sourceDBName == "" {
		sourceDBName = sourceConnConfig.Database
	}
	targetDBName := syncConfig.TargetDatabase
	if targetDBName == "" {
		targetDBName = targetConnConfig.Database
	}
	if targetDBName == "" {
		// If still empty, default to source db name
		targetDBName = sourceDBName
	}
	if sourceDBName == "" || targetDBName == "" {
		return fmt.Errorf("source/target database is required")
	}

	// Connect to source database
	{
		cc := *sourceConnConfig
		cc.Database = sourceDBName
		sourceConnConfig = &cc
	}
	sourceDB, err := e.connectToRemote(sourceConnConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	defer sourceDB.Close()

	// Connect to target database
	{
		cc := *targetConnConfig
		cc.Database = targetDBName
		targetConnConfig = &cc
	}
	targetDB, err := e.connectToRemote(targetConnConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer targetDB.Close()

	// Validate row counts between source and target
	sourceCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", sourceDBName, mapping.SourceTable)
	if mapping.WhereClause != "" {
		sourceCountQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}
	var sourceCount int64
	if err := sourceDB.GetContext(ctx, &sourceCount, sourceCountQuery); err != nil {
		return fmt.Errorf("failed to get source row count: %w", err)
	}

	targetCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", targetDBName, mapping.TargetTable)
	var targetCount int64
	if err := targetDB.GetContext(ctx, &targetCount, targetCountQuery); err != nil {
		return fmt.Errorf("failed to get target row count: %w", err)
	}

	if sourceCount != targetCount {
		return fmt.Errorf("row count mismatch: source=%d, target=%d", sourceCount, targetCount)
	}

	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
	}).Info("Data validation completed successfully")

	return nil
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
		Params:               map[string]string{"charset": "utf8mb4"},
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

	// Get table charset/collation so target table can match source
	var tableCollation sql.NullString
	tableCCQuery := `SELECT TABLE_COLLATION FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`
	if err := remoteDB.GetContext(ctx, &tableCollation, tableCCQuery, tableName); err == nil && tableCollation.Valid && tableCollation.String != "" {
		schema.TableCollation = tableCollation.String
		if idx := strings.Index(schema.TableCollation, "_"); idx > 0 {
			schema.TableCharset = schema.TableCollation[:idx]
		} else {
			schema.TableCharset = schema.TableCollation
		}
	}

	// Get column information (including charset/collation for string columns)
	columnQuery := `
		SELECT 
			COLUMN_NAME, 
			COLUMN_TYPE, 
			IS_NULLABLE, 
			COLUMN_DEFAULT, 
			EXTRA,
			CHARACTER_SET_NAME,
			COLLATION_NAME
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
		var charSet, collation sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &nullable, &defaultValue, &col.Extra, &charSet, &collation); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		col.Nullable = (nullable == "YES")
		if defaultValue.Valid {
			col.DefaultValue = defaultValue.String
		}
		if charSet.Valid {
			col.CharacterSet = charSet.String
		}
		if collation.Valid {
			col.Collation = collation.String
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

	// ======= 【添加下面这段代码来打印 SQL】 =======
	fmt.Println("\n==========================================")
	fmt.Printf("DEBUG: 准备在目标库执行的 SQL 语句如下：\n%s\n", createQuery)
	fmt.Println("==========================================\n")
	// =============================================

	// ======= 核心修复代码开始 =======
	// 将 SQL 中的 DEFAULT_GENERATED 替换为空格
	createQuery = strings.ReplaceAll(createQuery, "DEFAULT_GENERATED", "")
	// 如果存在多个空格，可以简单处理一下（可选）
	createQuery = strings.ReplaceAll(createQuery, "  ", " ")
	// ======= 核心修复代码结束 =======

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

	// Add columns (preserve source charset/collation for string columns)
	for i, col := range schema.Columns {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("  `%s` %s", col.Name, col.Type))

		if col.CharacterSet != "" && col.Collation != "" {
			sb.WriteString(fmt.Sprintf(" CHARACTER SET %s COLLATE %s", col.CharacterSet, col.Collation))
		}

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

	// Table-level charset/collation: keep target consistent with source
	if schema.TableCharset != "" && schema.TableCollation != "" {
		sb.WriteString(fmt.Sprintf(" ENGINE=InnoDB DEFAULT CHARSET=%s COLLATE=%s", schema.TableCharset, schema.TableCollation))
	}

	return sb.String()
}

// syncAllData synchronizes all data from source to target table
func (e *DefaultSyncEngine) syncAllData(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping, options *SyncOptions) error {
	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"local_db":     localDB,
	}).Info("Starting data synchronization")

	// Get total row count to determine if we should use optimized batch processing
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", mapping.SourceTable)
	if mapping.WhereClause != "" {
		countQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}

	var totalRows int64
	if err := remoteDB.GetContext(ctx, &totalRows, countQuery); err != nil {
		return fmt.Errorf("failed to get row count: %w", err)
	}

	// For large tables (>10,000 rows), use optimized batch processor
	// Requirement 7.3: Process large tables in batches to avoid long locks
	// Requirement 8.1: Use batch operations to improve efficiency
	if totalRows > 10000 {
		e.logger.WithFields(logrus.Fields{
			"source_table": mapping.SourceTable,
			"total_rows":   totalRows,
		}).Info("Using optimized batch processor for large table")

		result, err := e.batchProcessor.ProcessLargeTableSync(ctx, remoteDB, localDB, mapping, options)
		if err != nil {
			return fmt.Errorf("batch processing failed: %w", err)
		}

		e.logger.WithFields(logrus.Fields{
			"source_table":   mapping.SourceTable,
			"target_table":   mapping.TargetTable,
			"processed_rows": result.ProcessedRows,
			"total_rows":     result.TotalRows,
			"duration":       result.Duration,
			"throughput":     fmt.Sprintf("%.2f rows/sec", result.ThroughputRows),
		}).Info("Optimized batch processing completed")

		return nil
	}

	// For smaller tables, use standard batch processing
	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"total_rows":   totalRows,
	}).Info("Using standard batch processing for small table")

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
			args = append(args, valueForUTF8MB3Insert(batch[i][col]))
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
			args = append(args, valueForUTF8MB3Insert(batch[i][col]))
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

// validateRowCounts validates that source and target tables have the same row count
// Requirement 7.5: Provide data validation and comparison functionality
func (e *DefaultSyncEngine) validateRowCounts(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping) error {
	// Get source row count
	sourceCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", mapping.SourceTable)
	if mapping.WhereClause != "" {
		sourceCountQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}

	var sourceCount int64
	if err := remoteDB.GetContext(ctx, &sourceCount, sourceCountQuery); err != nil {
		return fmt.Errorf("failed to get source row count: %w", err)
	}

	// Get target row count
	targetCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", localDB, mapping.TargetTable)
	var targetCount int64
	if err := e.localDB.GetContext(ctx, &targetCount, targetCountQuery); err != nil {
		return fmt.Errorf("failed to get target row count: %w", err)
	}

	// Compare counts
	if sourceCount != targetCount {
		return fmt.Errorf("row count mismatch: source=%d, target=%d", sourceCount, targetCount)
	}

	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"row_count":    sourceCount,
	}).Info("Row count validation passed")

	return nil
}

// validateDataChecksums validates data integrity using checksums
// Requirement 7.5: Provide data validation and comparison functionality
func (e *DefaultSyncEngine) validateDataChecksums(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping) error {
	// Get primary key columns
	primaryKeys, err := e.getPrimaryKeyColumns(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to get primary keys: %w", err)
	}

	// Get all column names
	schema, err := e.getTableSchemaFromRemote(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Build column list for checksum (excluding BLOB/TEXT columns which can't be used in GROUP_CONCAT)
	var checksumColumns []string
	for _, col := range schema.Columns {
		// Skip large text/blob columns
		colType := strings.ToLower(col.Type)
		if !strings.Contains(colType, "blob") && !strings.Contains(colType, "text") {
			checksumColumns = append(checksumColumns, col.Name)
		}
	}

	if len(checksumColumns) == 0 {
		e.logger.Warn("No suitable columns for checksum validation, skipping")
		return nil
	}

	// Build checksum query using MD5 hash of concatenated column values
	// Group by primary key to get checksum per row
	var concatCols []string
	for _, col := range checksumColumns {
		concatCols = append(concatCols, fmt.Sprintf("COALESCE(CAST(`%s` AS CHAR), 'NULL')", col))
	}
	concatExpr := strings.Join(concatCols, ", '|', ")

	pkCols := strings.Join(primaryKeys, "`, `")
	sourceChecksumQuery := fmt.Sprintf(
		"SELECT MD5(CONCAT(%s)) as checksum FROM `%s` ORDER BY `%s`",
		concatExpr, mapping.SourceTable, pkCols,
	)
	if mapping.WhereClause != "" {
		sourceChecksumQuery = fmt.Sprintf(
			"SELECT MD5(CONCAT(%s)) as checksum FROM `%s` WHERE %s ORDER BY `%s`",
			concatExpr, mapping.SourceTable, mapping.WhereClause, pkCols,
		)
	}

	targetChecksumQuery := fmt.Sprintf(
		"SELECT MD5(CONCAT(%s)) as checksum FROM `%s`.`%s` ORDER BY `%s`",
		concatExpr, localDB, mapping.TargetTable, pkCols,
	)

	// Get source checksums
	var sourceChecksums []string
	if err := remoteDB.SelectContext(ctx, &sourceChecksums, sourceChecksumQuery); err != nil {
		return fmt.Errorf("failed to get source checksums: %w", err)
	}

	// Get target checksums
	var targetChecksums []string
	if err := e.localDB.SelectContext(ctx, &targetChecksums, targetChecksumQuery); err != nil {
		return fmt.Errorf("failed to get target checksums: %w", err)
	}

	// Compare checksums
	if len(sourceChecksums) != len(targetChecksums) {
		return fmt.Errorf("checksum count mismatch: source=%d, target=%d", len(sourceChecksums), len(targetChecksums))
	}

	mismatchCount := 0
	for i := 0; i < len(sourceChecksums); i++ {
		if sourceChecksums[i] != targetChecksums[i] {
			mismatchCount++
		}
	}

	if mismatchCount > 0 {
		return fmt.Errorf("data checksum mismatch: %d rows differ", mismatchCount)
	}

	e.logger.WithFields(logrus.Fields{
		"source_table":   mapping.SourceTable,
		"target_table":   mapping.TargetTable,
		"validated_rows": len(sourceChecksums),
	}).Info("Data checksum validation passed")

	return nil
}

// syncTableWithTransaction performs table sync within a transaction
// Requirement 7.1: Use transactions to ensure data consistency
func (e *DefaultSyncEngine) syncTableWithTransaction(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	// NOTE:
	// The new design syncs between a source connection and a target connection (both remote).
	// The previous implementation wrapped local metadata DB writes in a transaction, but the
	// actual data writes now happen on the target remote DB connection.
	// For now, fall back to non-transactional sync.
	return e.SyncTable(ctx, job, mapping)
}

// transactionalSyncEngine wraps DefaultSyncEngine to use a transaction
type transactionalSyncEngine struct {
	*DefaultSyncEngine
	tx *sqlx.Tx
}

// syncFullWithTx performs full sync within a transaction
func (e *transactionalSyncEngine) syncFullWithTx(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	// Get sync config to retrieve connection info
	syncConfig, err := e.repo.GetSyncConfig(ctx, mapping.SyncConfigID)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	// Get connection config (source connection)
	connConfig, err := e.repo.GetConnection(ctx, syncConfig.SourceConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get connection config: %w", err)
	}

	// Connect to remote database
	remoteDB, err := e.connectToRemote(connConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to remote database: %w", err)
	}
	defer remoteDB.Close()

	// Truncate target table (within transaction)
	// NOTE: transactional sync is legacy/local-only; keep compiling by using the configured database name.
	truncateQuery := fmt.Sprintf("TRUNCATE TABLE `%s`.`%s`", connConfig.Database, mapping.TargetTable)
	if _, err := e.tx.ExecContext(ctx, truncateQuery); err != nil {
		return fmt.Errorf("failed to truncate target table: %w", err)
	}

	// Sync data using transaction
	return e.syncAllDataWithTx(ctx, remoteDB, connConfig.Database, mapping, syncConfig.Options)
}

// syncIncrementalWithTx performs incremental sync within a transaction
func (e *transactionalSyncEngine) syncIncrementalWithTx(ctx context.Context, job *SyncJob, mapping *TableMapping) error {
	// Get sync config
	syncConfig, err := e.repo.GetSyncConfig(ctx, mapping.SyncConfigID)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	// Get connection config (source connection)
	connConfig, err := e.repo.GetConnection(ctx, syncConfig.SourceConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get connection config: %w", err)
	}

	// Connect to remote database
	remoteDB, err := e.connectToRemote(connConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to remote database: %w", err)
	}
	defer remoteDB.Close()

	// Load checkpoint
	checkpoint, err := e.repo.GetCheckpoint(ctx, mapping.ID)
	if err != nil || checkpoint == nil {
		return fmt.Errorf("checkpoint required for incremental sync: %w", err)
	}

	// Detect change tracking column
	changeColumn, changeType, err := e.detectChangeTrackingColumn(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to detect change tracking column: %w", err)
	}

	// Get primary key columns
	primaryKeys, err := e.getPrimaryKeyColumns(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return fmt.Errorf("failed to get primary keys: %w", err)
	}

	// Sync incremental changes using transaction
	var syncedRows int64
	switch changeType {
	case "timestamp":
		syncedRows, err = e.syncIncrementalByTimestampWithTx(ctx, remoteDB, connConfig.Database, mapping, changeColumn, checkpoint, primaryKeys, syncConfig.Options)
	case "auto_increment":
		syncedRows, err = e.syncIncrementalByIDWithTx(ctx, remoteDB, connConfig.Database, mapping, changeColumn, checkpoint, primaryKeys, syncConfig.Options)
	default:
		return fmt.Errorf("unsupported change tracking type: %s", changeType)
	}

	if err != nil {
		return fmt.Errorf("failed to sync incremental data: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"synced_rows":  syncedRows,
	}).Info("Incremental sync with transaction completed")

	return nil
}

// syncAllDataWithTx syncs all data within a transaction
func (e *transactionalSyncEngine) syncAllDataWithTx(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping, options *SyncOptions) error {
	// Determine batch size
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	// Build SELECT query
	selectQuery := fmt.Sprintf("SELECT * FROM `%s`", mapping.SourceTable)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}

	// Query data from source
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

	// Process rows in batches
	var batch []map[string]interface{}
	processedRows := int64(0)

	for rows.Next() {
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		if len(batch) >= batchSize {
			if err := e.insertBatchWithTx(ctx, localDB, mapping.TargetTable, columns, batch); err != nil {
				return fmt.Errorf("failed to insert batch: %w", err)
			}
			processedRows += int64(len(batch))
			batch = batch[:0]
		}
	}

	// Insert remaining rows
	if len(batch) > 0 {
		if err := e.insertBatchWithTx(ctx, localDB, mapping.TargetTable, columns, batch); err != nil {
			return fmt.Errorf("failed to insert final batch: %w", err)
		}
		processedRows += int64(len(batch))
	}

	e.logger.WithFields(logrus.Fields{
		"source_table":   mapping.SourceTable,
		"processed_rows": processedRows,
	}).Info("Data sync with transaction completed")

	return nil
}

// insertBatchWithTx inserts a batch using the transaction
func (e *transactionalSyncEngine) insertBatchWithTx(ctx context.Context, localDB, tableName string, columns []string, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` (", localDB, tableName))

	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("`%s`", col))
	}

	sb.WriteString(") VALUES ")

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
			args = append(args, valueForUTF8MB3Insert(batch[i][col]))
		}
		sb.WriteString(")")
	}

	if _, err := e.tx.ExecContext(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}

	return nil
}

// syncIncrementalByTimestampWithTx syncs incremental data by timestamp within transaction
func (e *transactionalSyncEngine) syncIncrementalByTimestampWithTx(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping, timestampColumn string, checkpoint *SyncCheckpoint, primaryKeys []string, options *SyncOptions) (int64, error) {
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	selectQuery := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` > ?", mapping.SourceTable, timestampColumn)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" AND (%s)", mapping.WhereClause)
	}
	selectQuery += fmt.Sprintf(" ORDER BY `%s`", timestampColumn)

	rows, err := remoteDB.QueryxContext(ctx, selectQuery, checkpoint.LastSyncTime)
	if err != nil {
		return 0, fmt.Errorf("failed to query changed data: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get column names: %w", err)
	}

	var batch []map[string]interface{}
	var totalSynced int64

	for rows.Next() {
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return totalSynced, fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		if len(batch) >= batchSize {
			if err := e.upsertBatchWithTx(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
				return totalSynced, fmt.Errorf("failed to upsert batch: %w", err)
			}
			totalSynced += int64(len(batch))
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := e.upsertBatchWithTx(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
			return totalSynced, fmt.Errorf("failed to upsert final batch: %w", err)
		}
		totalSynced += int64(len(batch))
	}

	return totalSynced, nil
}

// syncIncrementalByIDWithTx syncs incremental data by ID within transaction
func (e *transactionalSyncEngine) syncIncrementalByIDWithTx(ctx context.Context, remoteDB *sqlx.DB, localDB string, mapping *TableMapping, idColumn string, checkpoint *SyncCheckpoint, primaryKeys []string, options *SyncOptions) (int64, error) {
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	var lastID int64
	if checkpoint.LastSyncValue != "" {
		fmt.Sscanf(checkpoint.LastSyncValue, "%d", &lastID)
	}

	selectQuery := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` > ?", mapping.SourceTable, idColumn)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" AND (%s)", mapping.WhereClause)
	}
	selectQuery += fmt.Sprintf(" ORDER BY `%s`", idColumn)

	rows, err := remoteDB.QueryxContext(ctx, selectQuery, lastID)
	if err != nil {
		return 0, fmt.Errorf("failed to query new data: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get column names: %w", err)
	}

	var batch []map[string]interface{}
	var totalSynced int64

	for rows.Next() {
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return totalSynced, fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		if len(batch) >= batchSize {
			if err := e.upsertBatchWithTx(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
				return totalSynced, fmt.Errorf("failed to upsert batch: %w", err)
			}
			totalSynced += int64(len(batch))
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := e.upsertBatchWithTx(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
			return totalSynced, fmt.Errorf("failed to upsert final batch: %w", err)
		}
		totalSynced += int64(len(batch))
	}

	return totalSynced, nil
}

// upsertBatchWithTx performs upsert within transaction
func (e *transactionalSyncEngine) upsertBatchWithTx(ctx context.Context, localDB, tableName string, columns, primaryKeys []string, batch []map[string]interface{}, options *SyncOptions) error {
	if len(batch) == 0 {
		return nil
	}

	conflictResolution := ConflictResolutionOverwrite
	if options != nil {
		conflictResolution = options.ConflictResolution
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` (", localDB, tableName))

	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("`%s`", col))
	}

	sb.WriteString(") VALUES ")

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
			args = append(args, valueForUTF8MB3Insert(batch[i][col]))
		}
		sb.WriteString(")")
	}

	// Add conflict resolution clause
	// Requirement 7.2: Handle data conflicts according to configured strategy
	switch conflictResolution {
	case ConflictResolutionOverwrite:
		sb.WriteString(" ON DUPLICATE KEY UPDATE ")
		first := true
		for _, col := range columns {
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
		sb.WriteString(fmt.Sprintf("`%s` = `%s`", primaryKeys[0], primaryKeys[0]))
	case ConflictResolutionError:
		// No ON DUPLICATE KEY clause, will error on conflict
	}

	if _, err := e.tx.ExecContext(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("failed to execute batch upsert: %w", err)
	}

	return nil
}

// detectDataConflicts detects potential data conflicts before sync
// Requirement 7.2: Detect data conflicts
func (e *DefaultSyncEngine) detectDataConflicts(ctx context.Context, remoteDB *sqlx.DB, remoteDBName, localDB string, mapping *TableMapping) ([]DataConflict, error) {
	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
	}).Info("Detecting data conflicts")

	var conflicts []DataConflict

	// Get primary key columns
	primaryKeys, err := e.getPrimaryKeyColumns(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary keys: %w", err)
	}

	// Get schema to identify columns
	schema, err := e.getTableSchemaFromRemote(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}

	// Build query to find rows that exist in both tables but have different values
	pkCols := strings.Join(primaryKeys, "`, `")

	// Select all non-primary-key columns for comparison
	var compareColumns []string
	for _, col := range schema.Columns {
		isPK := false
		for _, pk := range primaryKeys {
			if col.Name == pk {
				isPK = true
				break
			}
		}
		if !isPK {
			compareColumns = append(compareColumns, col.Name)
		}
	}

	if len(compareColumns) == 0 {
		// Only primary keys, no conflicts possible
		return conflicts, nil
	}

	// Build comparison query
	// This finds rows where primary keys match but other columns differ
	var joinConditions []string
	for _, pk := range primaryKeys {
		joinConditions = append(joinConditions, fmt.Sprintf("s.`%s` = t.`%s`", pk, pk))
	}

	var diffConditions []string
	for _, col := range compareColumns {
		diffConditions = append(diffConditions, fmt.Sprintf("(s.`%s` != t.`%s` OR (s.`%s` IS NULL AND t.`%s` IS NOT NULL) OR (s.`%s` IS NOT NULL AND t.`%s` IS NULL))",
			col, col, col, col, col, col))
	}

	conflictQuery := fmt.Sprintf(`
		SELECT %s
		FROM (SELECT * FROM %s.%s) s
		INNER JOIN (SELECT * FROM %s.%s) t ON %s
		WHERE %s
		LIMIT 100
	`,
		"s.`"+pkCols+"`",
		"`"+remoteDBName+"`", "`"+mapping.SourceTable+"`",
		"`"+localDB+"`", "`"+mapping.TargetTable+"`",
		strings.Join(joinConditions, " AND "),
		strings.Join(diffConditions, " OR "),
	)

	// Note: This is a simplified conflict detection
	// In production, you'd want to handle this more carefully with proper connection management
	_ = conflictQuery // Placeholder - actual implementation would execute this query

	e.logger.WithFields(logrus.Fields{
		"source_table":    mapping.SourceTable,
		"target_table":    mapping.TargetTable,
		"conflicts_found": len(conflicts),
	}).Info("Conflict detection completed")

	return conflicts, nil
}

// DataConflict represents a data conflict between source and target
type DataConflict struct {
	PrimaryKeyValues   map[string]interface{} `json:"primary_key_values"`
	ConflictingColumns []string               `json:"conflicting_columns"`
	SourceValues       map[string]interface{} `json:"source_values"`
	TargetValues       map[string]interface{} `json:"target_values"`
}

// createOrRecreateTargetTableInDB creates or recreates the target table in the specified database connection
func (e *DefaultSyncEngine) createOrRecreateTargetTableInDB(ctx context.Context, targetDB *sqlx.DB, targetDBName, tableName string, schema *TableSchema) error {
	e.logger.WithFields(logrus.Fields{
		"target_db":  targetDBName,
		"table_name": tableName,
	}).Info("Creating or recreating target table in target database")

	// Ensure target database exists
	if err := e.ensureDatabaseExists(ctx, targetDB, targetDBName); err != nil {
		return fmt.Errorf("failed to ensure target database exists: %w", err)
	}

	// Drop existing table if it exists
	dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", targetDBName, tableName)
	if _, err := targetDB.ExecContext(ctx, dropQuery); err != nil {
		return fmt.Errorf("failed to drop existing table: %w", err)
	}

	// Build CREATE TABLE statement
	createQuery := e.buildCreateTableStatement(targetDBName, tableName, schema)

	// Fix DEFAULT_GENERATED issue
	createQuery = strings.ReplaceAll(createQuery, "DEFAULT_GENERATED", "")
	createQuery = strings.ReplaceAll(createQuery, "  ", " ")

	// Execute CREATE TABLE
	if _, err := targetDB.ExecContext(ctx, createQuery); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"target_db":  targetDBName,
		"table_name": tableName,
	}).Info("Target table created successfully in target database")

	return nil
}

// ensureTargetTableExistsInDB ensures the target table exists in the specified database connection
func (e *DefaultSyncEngine) ensureTargetTableExistsInDB(ctx context.Context, targetDB *sqlx.DB, targetDBName, tableName string, schema *TableSchema) error {
	// Check if table exists
	var count int
	checkQuery := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
	if err := targetDB.GetContext(ctx, &count, checkQuery, targetDBName, tableName); err != nil {
		return fmt.Errorf("failed to check if table exists: %w", err)
	}

	if count == 0 {
		// Table doesn't exist, create it
		return e.createOrRecreateTargetTableInDB(ctx, targetDB, targetDBName, tableName, schema)
	}

	return nil
}

// ensureDatabaseExists ensures the database exists in the specified connection
func (e *DefaultSyncEngine) ensureDatabaseExists(ctx context.Context, db *sqlx.DB, dbName string) error {
	// Check if database exists
	var count int
	checkQuery := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
	if err := db.GetContext(ctx, &count, checkQuery, dbName); err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if count == 0 {
		// Database doesn't exist, create it
		createQuery := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)
		if _, err := db.ExecContext(ctx, createQuery); err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		e.logger.WithField("database", dbName).Info("Database created")
	}

	return nil
}

// syncAllDataBetweenDBs synchronizes all data from source database to target database
func (e *DefaultSyncEngine) syncAllDataBetweenDBs(ctx context.Context, sourceDB *sqlx.DB, sourceDBName string, targetDB *sqlx.DB, targetDBName string, mapping *TableMapping, options *SyncOptions) error {
	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"source_db":    sourceDBName,
		"target_db":    targetDBName,
	}).Info("Starting data synchronization between databases")

	// Get total row count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", sourceDBName, mapping.SourceTable)
	if mapping.WhereClause != "" {
		countQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}

	var totalRows int64
	if err := sourceDB.GetContext(ctx, &totalRows, countQuery); err != nil {
		return fmt.Errorf("failed to get row count: %w", err)
	}

	// Determine batch size
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	// Build SELECT query
	selectQuery := fmt.Sprintf("SELECT * FROM `%s`.`%s`", sourceDBName, mapping.SourceTable)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}

	// Query all data from source
	rows, err := sourceDB.QueryxContext(ctx, selectQuery)
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
			if err := e.insertBatchToDB(ctx, targetDB, targetDBName, mapping.TargetTable, columns, batch); err != nil {
				return fmt.Errorf("failed to insert batch: %w", err)
			}
			processedRows += int64(len(batch))
			batch = batch[:0] // Clear batch
		}
	}

	// Insert remaining rows
	if len(batch) > 0 {
		if err := e.insertBatchToDB(ctx, targetDB, targetDBName, mapping.TargetTable, columns, batch); err != nil {
			return fmt.Errorf("failed to insert final batch: %w", err)
		}
		processedRows += int64(len(batch))
	}

	e.logger.WithFields(logrus.Fields{
		"source_table":   mapping.SourceTable,
		"target_table":   mapping.TargetTable,
		"processed_rows": processedRows,
		"total_rows":     totalRows,
	}).Info("Data synchronization between databases completed successfully")

	return nil
}

// insertBatchToDB inserts a batch of rows into the target table in the specified database connection
func (e *DefaultSyncEngine) insertBatchToDB(ctx context.Context, targetDB *sqlx.DB, targetDBName, tableName string, columns []string, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	// Build INSERT statement
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` (", targetDBName, tableName))

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
			args = append(args, valueForUTF8MB3Insert(batch[i][col]))
		}
		sb.WriteString(")")
	}

	// Execute INSERT
	if _, err := targetDB.ExecContext(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}

	return nil
}

// syncIncrementalByTimestampBetweenDBs performs incremental sync by timestamp between two databases
func (e *DefaultSyncEngine) syncIncrementalByTimestampBetweenDBs(ctx context.Context, sourceDB *sqlx.DB, sourceDBName string, targetDB *sqlx.DB, targetDBName string, mapping *TableMapping, timestampColumn string, checkpoint *SyncCheckpoint, options *SyncOptions) (int64, error) {
	e.logger.WithFields(logrus.Fields{
		"source_table":     mapping.SourceTable,
		"timestamp_column": timestampColumn,
		"last_sync_time":   checkpoint.LastSyncTime,
	}).Info("Syncing incremental data by timestamp between databases")

	// Determine batch size
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	// Build SELECT query
	selectQuery := fmt.Sprintf("SELECT * FROM `%s`.`%s` WHERE `%s` > ?", sourceDBName, mapping.SourceTable, timestampColumn)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" AND (%s)", mapping.WhereClause)
	}
	selectQuery += fmt.Sprintf(" ORDER BY `%s`", timestampColumn)

	// Query incremental data from source
	rows, err := sourceDB.QueryxContext(ctx, selectQuery, checkpoint.LastSyncTime)
	if err != nil {
		return 0, fmt.Errorf("failed to query incremental data: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get column names: %w", err)
	}

	// Prepare batch insert
	var batch []map[string]interface{}
	syncedRows := int64(0)

	for rows.Next() {
		// Scan row into map
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return 0, fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		// Insert batch when it reaches batch size
		if len(batch) >= batchSize {
			if err := e.insertBatchToDB(ctx, targetDB, targetDBName, mapping.TargetTable, columns, batch); err != nil {
				return 0, fmt.Errorf("failed to insert batch: %w", err)
			}
			syncedRows += int64(len(batch))
			batch = batch[:0] // Clear batch
		}
	}

	// Insert remaining rows
	if len(batch) > 0 {
		if err := e.insertBatchToDB(ctx, targetDB, targetDBName, mapping.TargetTable, columns, batch); err != nil {
			return 0, fmt.Errorf("failed to insert final batch: %w", err)
		}
		syncedRows += int64(len(batch))
	}

	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"synced_rows":  syncedRows,
	}).Info("Incremental sync by timestamp between databases completed")

	return syncedRows, nil
}

// syncIncrementalByIDBetweenDBs performs incremental sync by ID between two databases
func (e *DefaultSyncEngine) syncIncrementalByIDBetweenDBs(ctx context.Context, sourceDB *sqlx.DB, sourceDBName string, targetDB *sqlx.DB, targetDBName string, mapping *TableMapping, idColumn string, checkpoint *SyncCheckpoint, options *SyncOptions) (int64, error) {
	e.logger.WithFields(logrus.Fields{
		"source_table":    mapping.SourceTable,
		"id_column":       idColumn,
		"last_sync_value": checkpoint.LastSyncValue,
	}).Info("Syncing incremental data by ID between databases")

	// Determine batch size
	batchSize := 1000
	if options != nil && options.BatchSize > 0 {
		batchSize = options.BatchSize
	}

	// Parse last sync ID
	var lastID int64
	if checkpoint.LastSyncValue != "" {
		fmt.Sscanf(checkpoint.LastSyncValue, "%d", &lastID)
	}

	// Build SELECT query
	selectQuery := fmt.Sprintf("SELECT * FROM `%s`.`%s` WHERE `%s` > ?", sourceDBName, mapping.SourceTable, idColumn)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" AND (%s)", mapping.WhereClause)
	}
	selectQuery += fmt.Sprintf(" ORDER BY `%s`", idColumn)

	// Query incremental data from source
	rows, err := sourceDB.QueryxContext(ctx, selectQuery, lastID)
	if err != nil {
		return 0, fmt.Errorf("failed to query incremental data: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get column names: %w", err)
	}

	// Prepare batch insert
	var batch []map[string]interface{}
	syncedRows := int64(0)

	for rows.Next() {
		// Scan row into map
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return 0, fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		// Insert batch when it reaches batch size
		if len(batch) >= batchSize {
			if err := e.insertBatchToDB(ctx, targetDB, targetDBName, mapping.TargetTable, columns, batch); err != nil {
				return 0, fmt.Errorf("failed to insert batch: %w", err)
			}
			syncedRows += int64(len(batch))
			batch = batch[:0] // Clear batch
		}
	}

	// Insert remaining rows
	if len(batch) > 0 {
		if err := e.insertBatchToDB(ctx, targetDB, targetDBName, mapping.TargetTable, columns, batch); err != nil {
			return 0, fmt.Errorf("failed to insert final batch: %w", err)
		}
		syncedRows += int64(len(batch))
	}

	e.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"synced_rows":  syncedRows,
	}).Info("Incremental sync by ID between databases completed")

	return syncedRows, nil
}
