package sync

import (
	"context"
)

// ConnectionManager manages remote database connections
type ConnectionManager interface {
	// AddConnection adds a new remote database connection
	AddConnection(ctx context.Context, config *ConnectionConfig) (*Connection, error)

	// GetConnections returns all configured connections
	GetConnections(ctx context.Context) ([]*Connection, error)

	// GetConnection returns a specific connection by ID
	GetConnection(ctx context.Context, id string) (*Connection, error)

	// UpdateConnection updates an existing connection configuration
	UpdateConnection(ctx context.Context, id string, config *ConnectionConfig) error

	// DeleteConnection removes a connection and stops related sync jobs
	DeleteConnection(ctx context.Context, id string) error

	// TestConnection tests the connectivity of a connection
	TestConnection(ctx context.Context, id string) (*ConnectionStatus, error)

	// TestConnectionConfig tests a connection configuration without saving it
	TestConnectionConfig(ctx context.Context, config *ConnectionConfig) (*ConnectionStatus, error)

	// Close closes the connection manager and cleans up resources
	Close() error
}

// SyncManager manages synchronization configurations and jobs
type SyncManager interface {
	// CreateSyncConfig creates a new synchronization configuration
	CreateSyncConfig(ctx context.Context, config *SyncConfig) error

	// GetSyncConfigs returns sync configurations for a connection
	GetSyncConfigs(ctx context.Context, connectionID string) ([]*SyncConfig, error)

	// GetSyncConfig returns a specific sync configuration
	GetSyncConfig(ctx context.Context, id string) (*SyncConfig, error)

	// UpdateSyncConfig updates an existing sync configuration
	UpdateSyncConfig(ctx context.Context, id string, config *SyncConfig) error

	// DeleteSyncConfig removes a sync configuration
	DeleteSyncConfig(ctx context.Context, id string) error

	// StartSync starts a synchronization job
	StartSync(ctx context.Context, configID string) (*SyncJob, error)

	// StopSync stops a running synchronization job
	StopSync(ctx context.Context, jobID string) error

	// GetSyncStatus returns the status of a sync job
	GetSyncStatus(ctx context.Context, jobID string) (*SyncJob, error)

	// GetRemoteDatabases retrieves the list of databases from a remote MySQL server (by connection)
	GetRemoteDatabases(ctx context.Context, connectionID string) ([]string, error)

	// GetRemoteTables retrieves the list of tables from a remote database connection
	// Requirement 3.1: Browse remote database and display available tables
	GetRemoteTables(ctx context.Context, connectionID, database string) ([]string, error)

	// GetRemoteTableSchema retrieves the schema information for a specific table
	// Supports requirement 3.1: Browse remote database structure
	GetRemoteTableSchema(ctx context.Context, connectionID, database, tableName string) (*TableSchema, error)

	// AddTableMapping adds a new table mapping to an existing sync configuration
	// Requirement 3.2: Select tables for synchronization and save configuration
	AddTableMapping(ctx context.Context, syncConfigID string, mapping *TableMapping) error

	// UpdateTableMapping updates an existing table mapping
	// Requirements 3.3, 3.4, 3.5: Configure sync rules, table mappings, enable/disable
	UpdateTableMapping(ctx context.Context, mappingID string, mapping *TableMapping) error

	// RemoveTableMapping removes a table mapping from a sync configuration
	// Requirement 3.5: Enable/disable table synchronization (by removal)
	RemoveTableMapping(ctx context.Context, mappingID string) error

	// GetTableMappings retrieves all table mappings for a sync configuration
	GetTableMappings(ctx context.Context, syncConfigID string) ([]*TableMapping, error)

	// ToggleTableMapping enables or disables a specific table mapping
	// Requirement 3.5: Enable/disable table synchronization
	ToggleTableMapping(ctx context.Context, mappingID string, enabled bool) error

	// SetTableSyncMode updates the sync mode for a specific table mapping
	// Requirement 3.3: Configure table sync rules (full/incremental mode)
	SetTableSyncMode(ctx context.Context, mappingID string, syncMode SyncMode) error

	// GetJobProgress returns the current progress of a sync job
	// Requirement 5.1: Real-time display of sync progress and status
	GetJobProgress(ctx context.Context, jobID string) (*JobSummary, error)

	// GetSyncHistory returns historical sync records
	// Requirement 5.2: Display historical sync records and results
	GetSyncHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error)

	// GetSyncStatistics returns overall synchronization statistics
	// Requirement 5.4: Display statistics information including data volume and time consumption
	GetSyncStatistics(ctx context.Context) (*SyncStatistics, error)

	// GetActiveJobs returns currently running sync jobs
	// Requirement 5.1: Real-time display of sync progress and status
	GetActiveJobs(ctx context.Context) ([]*JobSummary, error)

	// GetJobLogs returns logs for a specific job
	// Requirement 5.3: Display detailed error information and suggestions when sync fails
	GetJobLogs(ctx context.Context, jobID string) ([]*SyncLog, error)
}

// JobEngine manages the execution of synchronization jobs
type JobEngine interface {
	// Start initializes and starts the job engine
	Start() error

	// Stop gracefully shuts down the job engine
	Stop() error

	// SubmitJob submits a sync job for execution
	SubmitJob(ctx context.Context, job *SyncJob) error

	// GetJobStatus returns the current status of a job
	GetJobStatus(ctx context.Context, jobID string) (*SyncJob, error)

	// CancelJob cancels a running job
	CancelJob(ctx context.Context, jobID string) error

	// GetJobHistory returns historical job information
	GetJobHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error)

	// GetJobsByStatus returns jobs filtered by status
	GetJobsByStatus(ctx context.Context, status JobStatus) ([]*SyncJob, error)
}

// MappingManager manages database and table mappings
type MappingManager interface {
	// CreateDatabaseMapping creates a new database mapping
	CreateDatabaseMapping(ctx context.Context, mapping *DatabaseMapping) error

	// GetDatabaseMappings returns all database mappings
	GetDatabaseMappings(ctx context.Context) ([]*DatabaseMapping, error)

	// CheckTableConflicts checks for table name conflicts in local database
	CheckTableConflicts(ctx context.Context, localDB string, tables []string) ([]string, error)

	// ExportConfig exports all sync configurations
	ExportConfig(ctx context.Context) (*ConfigExport, error)

	// ImportConfig imports sync configurations
	ImportConfig(ctx context.Context, config *ConfigExport) error

	// ValidateConfig validates imported configuration
	ValidateConfig(ctx context.Context, config *ConfigExport) error

	// BackupConfig creates a backup of the current configuration
	BackupConfig(ctx context.Context) (*ConfigExport, error)

	// ImportConfigWithConflictResolution imports configuration with conflict resolution
	ImportConfigWithConflictResolution(ctx context.Context, config *ConfigExport, resolveConflicts bool) error

	// ValidateConfigIntegrity performs deep validation of configuration integrity
	ValidateConfigIntegrity(ctx context.Context, config *ConfigExport) error

	// GetConfigurationSummary returns a summary of the current configuration
	GetConfigurationSummary(ctx context.Context) (*ConfigurationSummary, error)
}

// SyncEngine performs the actual data synchronization
type SyncEngine interface {
	// SyncTable synchronizes a single table
	SyncTable(ctx context.Context, job *SyncJob, mapping *TableMapping) error

	// SyncFull performs full table synchronization
	SyncFull(ctx context.Context, job *SyncJob, mapping *TableMapping) error

	// SyncIncremental performs incremental table synchronization
	SyncIncremental(ctx context.Context, job *SyncJob, mapping *TableMapping) error

	// ValidateData validates data consistency between source and target
	ValidateData(ctx context.Context, mapping *TableMapping) error

	// GetTableSchema retrieves table schema from source database
	GetTableSchema(ctx context.Context, connectionID, tableName string) (*TableSchema, error)

	// CreateTargetTable creates target table with source schema
	CreateTargetTable(ctx context.Context, localDB string, schema *TableSchema) error
}

// MonitoringService provides sync status monitoring and statistics collection
type MonitoringService interface {
	// StartJobMonitoring starts monitoring a sync job
	// Requirement 5.1: Real-time display of sync progress and status
	StartJobMonitoring(ctx context.Context, jobID string, totalTables int) error

	// UpdateJobProgress updates the progress of a sync job
	// Requirement 5.1: Real-time display of sync progress and status
	UpdateJobProgress(ctx context.Context, jobID string, progress *Progress) error

	// UpdateTableProgress updates the progress of a specific table sync
	// Requirement 5.1: Real-time display of sync progress and status
	UpdateTableProgress(ctx context.Context, jobID, tableName string, status TableSyncStatus, processedRows, totalRows int64, errorMsg string) error

	// GetJobProgress returns the current progress of a sync job
	// Requirement 5.1: Real-time display of sync progress and status
	GetJobProgress(ctx context.Context, jobID string) (*JobSummary, error)

	// FinishJobMonitoring completes monitoring for a sync job
	// Requirement 5.2: Display historical sync records and results
	FinishJobMonitoring(ctx context.Context, jobID string, status JobStatus, errorMsg string) error

	// GetSyncHistory returns historical sync records
	// Requirement 5.2: Display historical sync records and results
	GetSyncHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error)

	// GetSyncStatistics returns overall synchronization statistics
	// Requirement 5.4: Display statistics information including data volume and time consumption
	GetSyncStatistics(ctx context.Context) (*SyncStatistics, error)

	// GetActiveJobs returns currently running sync jobs
	// Requirement 5.1: Real-time display of sync progress and status
	GetActiveJobs(ctx context.Context) ([]*JobSummary, error)

	// AddJobWarning adds a warning message to a job
	AddJobWarning(ctx context.Context, jobID, warning string) error

	// GetJobLogs returns logs for a specific job
	// Requirement 5.3: Display detailed error information and suggestions when sync fails
	GetJobLogs(ctx context.Context, jobID string) ([]*SyncLog, error)

	// LogJobEvent logs an event for a sync job
	// Requirement 5.3: Display detailed error information and suggestions when sync fails
	LogJobEvent(ctx context.Context, jobID, tableName, level, message string) error
}

// Repository defines the data access layer for sync operations
type Repository interface {
	// Connection operations
	CreateConnection(ctx context.Context, config *ConnectionConfig) error
	GetConnection(ctx context.Context, id string) (*ConnectionConfig, error)
	GetConnections(ctx context.Context) ([]*ConnectionConfig, error)
	UpdateConnection(ctx context.Context, id string, config *ConnectionConfig) error
	DeleteConnection(ctx context.Context, id string) error

	// Sync config operations
	CreateSyncConfig(ctx context.Context, config *SyncConfig) error
	GetSyncConfig(ctx context.Context, id string) (*SyncConfig, error)
	GetSyncConfigs(ctx context.Context, connectionID string) ([]*SyncConfig, error)
	UpdateSyncConfig(ctx context.Context, id string, config *SyncConfig) error
	DeleteSyncConfig(ctx context.Context, id string) error

	// Table mapping operations
	CreateTableMapping(ctx context.Context, mapping *TableMapping) error
	GetTableMappings(ctx context.Context, syncConfigID string) ([]*TableMapping, error)
	UpdateTableMapping(ctx context.Context, id string, mapping *TableMapping) error
	DeleteTableMapping(ctx context.Context, id string) error

	// Job operations
	CreateSyncJob(ctx context.Context, job *SyncJob) error
	GetSyncJob(ctx context.Context, id string) (*SyncJob, error)
	UpdateSyncJob(ctx context.Context, id string, job *SyncJob) error
	GetJobHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error)
	GetJobsByStatus(ctx context.Context, status JobStatus) ([]*SyncJob, error)

	// Checkpoint operations
	CreateCheckpoint(ctx context.Context, checkpoint *SyncCheckpoint) error
	GetCheckpoint(ctx context.Context, tableMappingID string) (*SyncCheckpoint, error)
	UpdateCheckpoint(ctx context.Context, tableMappingID string, checkpoint *SyncCheckpoint) error

	// Log operations
	CreateSyncLog(ctx context.Context, log *SyncLog) error
	GetSyncLogs(ctx context.Context, jobID string) ([]*SyncLog, error)

	// Database mapping operations
	CreateDatabaseMapping(ctx context.Context, mapping *DatabaseMapping) error
	GetDatabaseMappings(ctx context.Context) ([]*DatabaseMapping, error)
	UpdateDatabaseMapping(ctx context.Context, remoteConnectionID string, mapping *DatabaseMapping) error
	DeleteDatabaseMapping(ctx context.Context, remoteConnectionID string) error
}

// TableSchema represents database table schema information
type TableSchema struct {
	Name    string        `json:"name"`
	Columns []*ColumnInfo `json:"columns"`
	Indexes []*IndexInfo  `json:"indexes"`
	Keys    []*KeyInfo    `json:"keys"`
}

// ColumnInfo represents table column information
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"default_value,omitempty"`
	Extra        string `json:"extra,omitempty"`
}

// IndexInfo represents table index information
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type"`
}

// KeyInfo represents table key information
type KeyInfo struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"` // PRIMARY, FOREIGN, UNIQUE
	Columns []string `json:"columns"`
}
