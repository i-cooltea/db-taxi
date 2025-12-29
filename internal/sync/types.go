package sync

import (
	"time"
)

// SyncMode defines the synchronization mode
type SyncMode string

const (
	SyncModeFull        SyncMode = "full"
	SyncModeIncremental SyncMode = "incremental"
)

// ConflictResolution defines how to handle data conflicts
type ConflictResolution string

const (
	ConflictResolutionSkip      ConflictResolution = "skip"
	ConflictResolutionOverwrite ConflictResolution = "overwrite"
	ConflictResolutionError     ConflictResolution = "error"
)

// JobStatus defines the status of a sync job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// ConnectionConfig represents a remote database connection configuration
type ConnectionConfig struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Host        string    `json:"host" db:"host"`
	Port        int       `json:"port" db:"port"`
	Username    string    `json:"username" db:"username"`
	Password    string    `json:"password" db:"password"`
	Database    string    `json:"database" db:"database_name"`
	LocalDBName string    `json:"local_db_name" db:"local_db_name"`
	SSL         bool      `json:"ssl" db:"ssl"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Connection represents a database connection with status
type Connection struct {
	Config *ConnectionConfig `json:"config"`
	Status ConnectionStatus  `json:"status"`
}

// ConnectionStatus represents the status of a database connection
type ConnectionStatus struct {
	Connected bool      `json:"connected"`
	LastCheck time.Time `json:"last_check"`
	Error     string    `json:"error,omitempty"`
	Latency   int64     `json:"latency_ms"`
}

// SyncConfig represents synchronization configuration
type SyncConfig struct {
	ID           string          `json:"id" db:"id"`
	ConnectionID string          `json:"connection_id" db:"connection_id"`
	Name         string          `json:"name" db:"name"`
	Tables       []*TableMapping `json:"tables"`
	SyncMode     SyncMode        `json:"sync_mode" db:"sync_mode"`
	Schedule     string          `json:"schedule" db:"schedule"`
	Enabled      bool            `json:"enabled" db:"enabled"`
	Options      *SyncOptions    `json:"options"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// TableMapping represents the mapping between source and target tables
type TableMapping struct {
	ID           string    `json:"id" db:"id"`
	SyncConfigID string    `json:"sync_config_id" db:"sync_config_id"`
	SourceTable  string    `json:"source_table" db:"source_table"`
	TargetTable  string    `json:"target_table" db:"target_table"`
	SyncMode     SyncMode  `json:"sync_mode" db:"sync_mode"`
	Enabled      bool      `json:"enabled" db:"enabled"`
	WhereClause  string    `json:"where_clause,omitempty" db:"where_clause"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// SyncOptions represents synchronization options
type SyncOptions struct {
	BatchSize          int                `json:"batch_size"`
	MaxConcurrency     int                `json:"max_concurrency"`
	EnableCompression  bool               `json:"enable_compression"`
	ConflictResolution ConflictResolution `json:"conflict_resolution"`
}

// SyncJob represents a synchronization job
type SyncJob struct {
	ID              string     `json:"id" db:"id"`
	ConfigID        string     `json:"config_id" db:"config_id"`
	Status          JobStatus  `json:"status" db:"status"`
	Progress        *Progress  `json:"progress"`
	StartTime       time.Time  `json:"start_time" db:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty" db:"end_time"`
	TotalTables     int        `json:"total_tables" db:"total_tables"`
	CompletedTables int        `json:"completed_tables" db:"completed_tables"`
	TotalRows       int64      `json:"total_rows" db:"total_rows"`
	ProcessedRows   int64      `json:"processed_rows" db:"processed_rows"`
	Error           string     `json:"error,omitempty" db:"error_message"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// Progress represents synchronization progress
type Progress struct {
	TotalTables     int     `json:"total_tables"`
	CompletedTables int     `json:"completed_tables"`
	TotalRows       int64   `json:"total_rows"`
	ProcessedRows   int64   `json:"processed_rows"`
	Percentage      float64 `json:"percentage"`
}

// JobHistory represents historical sync job information
type JobHistory struct {
	*SyncJob
	ConfigName     string `json:"config_name" db:"config_name"`
	ConnectionName string `json:"connection_name" db:"connection_name"`
}

// DatabaseMapping represents the mapping between remote and local databases
type DatabaseMapping struct {
	RemoteConnectionID string    `json:"remote_connection_id" db:"remote_connection_id"`
	LocalDatabaseName  string    `json:"local_database_name" db:"local_database_name"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// ConfigExport represents exported configuration
type ConfigExport struct {
	Version     string              `json:"version"`
	ExportTime  time.Time           `json:"export_time"`
	Connections []*ConnectionConfig `json:"connections"`
	Mappings    []*DatabaseMapping  `json:"mappings"`
	SyncConfigs []*SyncConfig       `json:"sync_configs"`
}

// SyncCheckpoint represents incremental sync checkpoint
type SyncCheckpoint struct {
	ID             string    `json:"id" db:"id"`
	TableMappingID string    `json:"table_mapping_id" db:"table_mapping_id"`
	LastSyncTime   time.Time `json:"last_sync_time" db:"last_sync_time"`
	LastSyncValue  string    `json:"last_sync_value,omitempty" db:"last_sync_value"`
	CheckpointData string    `json:"checkpoint_data,omitempty" db:"checkpoint_data"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// SyncLog represents sync operation log entry
type SyncLog struct {
	ID        int64     `json:"id" db:"id"`
	JobID     string    `json:"job_id" db:"job_id"`
	TableName string    `json:"table_name" db:"table_name"`
	Level     string    `json:"level" db:"level"`
	Message   string    `json:"message" db:"message"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ConfigurationSummary represents a summary of the current configuration
type ConfigurationSummary struct {
	TotalConnections     int       `json:"total_connections"`
	TotalMappings        int       `json:"total_mappings"`
	TotalSyncConfigs     int       `json:"total_sync_configs"`
	EnabledSyncConfigs   int       `json:"enabled_sync_configs"`
	TotalTableMappings   int       `json:"total_table_mappings"`
	EnabledTableMappings int       `json:"enabled_table_mappings"`
	GeneratedAt          time.Time `json:"generated_at"`
}
