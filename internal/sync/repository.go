package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// MySQLRepository implements the Repository interface using MySQL
type MySQLRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewMySQLRepository creates a new MySQL repository instance
func NewMySQLRepository(db *sqlx.DB, logger *logrus.Logger) Repository {
	return &MySQLRepository{
		db:     db,
		logger: logger,
	}
}

// Connection operations

func (r *MySQLRepository) CreateConnection(ctx context.Context, config *ConnectionConfig) error {
	query := `
		INSERT INTO connections (id, name, host, port, username, password, database_name, local_db_name, ssl)
		VALUES (:id, :name, :host, :port, :username, :password, :database, :local_db_name, :ssl)
	`
	_, err := r.db.NamedExecContext(ctx, query, config)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create connection")
		return fmt.Errorf("failed to create connection: %w", err)
	}
	return nil
}

func (r *MySQLRepository) GetConnection(ctx context.Context, id string) (*ConnectionConfig, error) {
	var config ConnectionConfig
	query := `SELECT * FROM connections WHERE id = ?`
	err := r.db.GetContext(ctx, &config, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection not found: %s", id)
		}
		r.logger.WithError(err).WithField("id", id).Error("Failed to get connection")
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	return &config, nil
}

func (r *MySQLRepository) GetConnections(ctx context.Context) ([]*ConnectionConfig, error) {
	var configs []*ConnectionConfig
	query := `SELECT * FROM connections ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &configs, query)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get connections")
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}
	return configs, nil
}

func (r *MySQLRepository) UpdateConnection(ctx context.Context, id string, config *ConnectionConfig) error {
	query := `
		UPDATE connections 
		SET name = :name, host = :host, port = :port, username = :username, 
		    password = :password, database_name = :database, local_db_name = :local_db_name, 
		    ssl = :ssl, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id
	`
	config.ID = id
	result, err := r.db.NamedExecContext(ctx, query, config)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to update connection")
		return fmt.Errorf("failed to update connection: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("connection not found: %s", id)
	}

	return nil
}

func (r *MySQLRepository) DeleteConnection(ctx context.Context, id string) error {
	query := `DELETE FROM connections WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to delete connection")
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("connection not found: %s", id)
	}

	return nil
}

// Sync config operations

func (r *MySQLRepository) CreateSyncConfig(ctx context.Context, config *SyncConfig) error {
	// Serialize options to JSON
	var optionsJSON []byte
	var err error
	if config.Options != nil {
		optionsJSON, err = json.Marshal(config.Options)
		if err != nil {
			return fmt.Errorf("failed to marshal sync options: %w", err)
		}
	}

	query := `
		INSERT INTO sync_configs (id, connection_id, name, sync_mode, schedule, enabled, options)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query, config.ID, config.ConnectionID, config.Name,
		config.SyncMode, config.Schedule, config.Enabled, optionsJSON)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create sync config")
		return fmt.Errorf("failed to create sync config: %w", err)
	}
	return nil
}

func (r *MySQLRepository) GetSyncConfig(ctx context.Context, id string) (*SyncConfig, error) {
	var config SyncConfig
	var optionsJSON sql.NullString

	query := `SELECT id, connection_id, name, sync_mode, schedule, enabled, options, created_at, updated_at 
	          FROM sync_configs WHERE id = ?`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&config.ID, &config.ConnectionID, &config.Name, &config.SyncMode,
		&config.Schedule, &config.Enabled, &optionsJSON, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sync config not found: %s", id)
		}
		r.logger.WithError(err).WithField("id", id).Error("Failed to get sync config")
		return nil, fmt.Errorf("failed to get sync config: %w", err)
	}

	// Deserialize options from JSON
	if optionsJSON.Valid {
		var options SyncOptions
		if err := json.Unmarshal([]byte(optionsJSON.String), &options); err != nil {
			r.logger.WithError(err).Warn("Failed to unmarshal sync options")
		} else {
			config.Options = &options
		}
	}

	// Load table mappings
	mappings, err := r.GetTableMappings(ctx, id)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to load table mappings")
	} else {
		config.Tables = mappings
	}

	return &config, nil
}

func (r *MySQLRepository) GetSyncConfigs(ctx context.Context, connectionID string) ([]*SyncConfig, error) {
	var configs []*SyncConfig
	query := `SELECT id, connection_id, name, sync_mode, schedule, enabled, options, created_at, updated_at 
	          FROM sync_configs WHERE connection_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, connectionID)
	if err != nil {
		r.logger.WithError(err).WithField("connection_id", connectionID).Error("Failed to get sync configs")
		return nil, fmt.Errorf("failed to get sync configs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var config SyncConfig
		var optionsJSON sql.NullString

		err := rows.Scan(&config.ID, &config.ConnectionID, &config.Name, &config.SyncMode,
			&config.Schedule, &config.Enabled, &optionsJSON, &config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan sync config")
			continue
		}

		// Deserialize options from JSON
		if optionsJSON.Valid {
			var options SyncOptions
			if err := json.Unmarshal([]byte(optionsJSON.String), &options); err != nil {
				r.logger.WithError(err).Warn("Failed to unmarshal sync options")
			} else {
				config.Options = &options
			}
		}

		// Load table mappings
		mappings, err := r.GetTableMappings(ctx, config.ID)
		if err != nil {
			r.logger.WithError(err).Warn("Failed to load table mappings")
		} else {
			config.Tables = mappings
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

func (r *MySQLRepository) UpdateSyncConfig(ctx context.Context, id string, config *SyncConfig) error {
	// Serialize options to JSON
	var optionsJSON []byte
	var err error
	if config.Options != nil {
		optionsJSON, err = json.Marshal(config.Options)
		if err != nil {
			return fmt.Errorf("failed to marshal sync options: %w", err)
		}
	}

	query := `
		UPDATE sync_configs 
		SET name = ?, sync_mode = ?, schedule = ?, enabled = ?, options = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query, config.Name, config.SyncMode,
		config.Schedule, config.Enabled, optionsJSON, id)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to update sync config")
		return fmt.Errorf("failed to update sync config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("sync config not found: %s", id)
	}

	return nil
}

func (r *MySQLRepository) DeleteSyncConfig(ctx context.Context, id string) error {
	query := `DELETE FROM sync_configs WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to delete sync config")
		return fmt.Errorf("failed to delete sync config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("sync config not found: %s", id)
	}

	return nil
}

// Table mapping operations

func (r *MySQLRepository) CreateTableMapping(ctx context.Context, mapping *TableMapping) error {
	query := `
		INSERT INTO table_mappings (id, sync_config_id, source_table, target_table, sync_mode, enabled, where_clause)
		VALUES (:id, :sync_config_id, :source_table, :target_table, :sync_mode, :enabled, :where_clause)
	`
	_, err := r.db.NamedExecContext(ctx, query, mapping)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create table mapping")
		return fmt.Errorf("failed to create table mapping: %w", err)
	}
	return nil
}

func (r *MySQLRepository) GetTableMappings(ctx context.Context, syncConfigID string) ([]*TableMapping, error) {
	var mappings []*TableMapping
	query := `SELECT * FROM table_mappings WHERE sync_config_id = ? ORDER BY created_at`
	err := r.db.SelectContext(ctx, &mappings, query, syncConfigID)
	if err != nil {
		r.logger.WithError(err).WithField("sync_config_id", syncConfigID).Error("Failed to get table mappings")
		return nil, fmt.Errorf("failed to get table mappings: %w", err)
	}
	return mappings, nil
}

func (r *MySQLRepository) UpdateTableMapping(ctx context.Context, id string, mapping *TableMapping) error {
	query := `
		UPDATE table_mappings 
		SET source_table = :source_table, target_table = :target_table, sync_mode = :sync_mode, 
		    enabled = :enabled, where_clause = :where_clause, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id
	`
	mapping.ID = id
	result, err := r.db.NamedExecContext(ctx, query, mapping)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to update table mapping")
		return fmt.Errorf("failed to update table mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("table mapping not found: %s", id)
	}

	return nil
}

func (r *MySQLRepository) DeleteTableMapping(ctx context.Context, id string) error {
	query := `DELETE FROM table_mappings WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to delete table mapping")
		return fmt.Errorf("failed to delete table mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("table mapping not found: %s", id)
	}

	return nil
}

// Job operations

func (r *MySQLRepository) CreateSyncJob(ctx context.Context, job *SyncJob) error {
	query := `
		INSERT INTO sync_jobs (id, config_id, status, start_time, total_tables, completed_tables, total_rows, processed_rows, error_message)
		VALUES (:id, :config_id, :status, :start_time, :total_tables, :completed_tables, :total_rows, :processed_rows, :error)
	`
	_, err := r.db.NamedExecContext(ctx, query, job)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create sync job")
		return fmt.Errorf("failed to create sync job: %w", err)
	}
	return nil
}

func (r *MySQLRepository) GetSyncJob(ctx context.Context, id string) (*SyncJob, error) {
	var job SyncJob
	query := `SELECT * FROM sync_jobs WHERE id = ?`
	err := r.db.GetContext(ctx, &job, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sync job not found: %s", id)
		}
		r.logger.WithError(err).WithField("id", id).Error("Failed to get sync job")
		return nil, fmt.Errorf("failed to get sync job: %w", err)
	}

	// Build progress information
	if job.TotalRows > 0 {
		job.Progress = &Progress{
			TotalTables:     job.TotalTables,
			CompletedTables: job.CompletedTables,
			TotalRows:       job.TotalRows,
			ProcessedRows:   job.ProcessedRows,
			Percentage:      float64(job.ProcessedRows) / float64(job.TotalRows) * 100,
		}
	}

	return &job, nil
}

func (r *MySQLRepository) UpdateSyncJob(ctx context.Context, id string, job *SyncJob) error {
	query := `
		UPDATE sync_jobs 
		SET status = :status, end_time = :end_time, total_tables = :total_tables, 
		    completed_tables = :completed_tables, total_rows = :total_rows, 
		    processed_rows = :processed_rows, error_message = :error
		WHERE id = :id
	`
	job.ID = id
	result, err := r.db.NamedExecContext(ctx, query, job)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to update sync job")
		return fmt.Errorf("failed to update sync job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("sync job not found: %s", id)
	}

	return nil
}

func (r *MySQLRepository) GetJobHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error) {
	var history []*JobHistory
	query := `
		SELECT j.*, sc.name as config_name, c.name as connection_name
		FROM sync_jobs j
		JOIN sync_configs sc ON j.config_id = sc.id
		JOIN connections c ON sc.connection_id = c.id
		ORDER BY j.start_time DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get job history")
		return nil, fmt.Errorf("failed to get job history: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var h JobHistory
		var job SyncJob
		err := rows.Scan(&job.ID, &job.ConfigID, &job.Status, &job.StartTime, &job.EndTime,
			&job.TotalTables, &job.CompletedTables, &job.TotalRows, &job.ProcessedRows,
			&job.Error, &job.CreatedAt, &h.ConfigName, &h.ConnectionName)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan job history")
			continue
		}

		h.SyncJob = &job
		history = append(history, &h)
	}

	return history, nil
}

func (r *MySQLRepository) GetJobsByStatus(ctx context.Context, status JobStatus) ([]*SyncJob, error) {
	var jobs []*SyncJob
	query := `SELECT * FROM sync_jobs WHERE status = ? ORDER BY start_time DESC`
	err := r.db.SelectContext(ctx, &jobs, query, status)
	if err != nil {
		r.logger.WithError(err).WithField("status", status).Error("Failed to get jobs by status")
		return nil, fmt.Errorf("failed to get jobs by status: %w", err)
	}
	return jobs, nil
}

// Checkpoint operations

func (r *MySQLRepository) CreateCheckpoint(ctx context.Context, checkpoint *SyncCheckpoint) error {
	query := `
		INSERT INTO sync_checkpoints (id, table_mapping_id, last_sync_time, last_sync_value, checkpoint_data)
		VALUES (:id, :table_mapping_id, :last_sync_time, :last_sync_value, :checkpoint_data)
		ON DUPLICATE KEY UPDATE
		last_sync_time = VALUES(last_sync_time),
		last_sync_value = VALUES(last_sync_value),
		checkpoint_data = VALUES(checkpoint_data),
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.NamedExecContext(ctx, query, checkpoint)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create checkpoint")
		return fmt.Errorf("failed to create checkpoint: %w", err)
	}
	return nil
}

func (r *MySQLRepository) GetCheckpoint(ctx context.Context, tableMappingID string) (*SyncCheckpoint, error) {
	var checkpoint SyncCheckpoint
	query := `SELECT * FROM sync_checkpoints WHERE table_mapping_id = ?`
	err := r.db.GetContext(ctx, &checkpoint, query, tableMappingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("checkpoint not found for table mapping: %s", tableMappingID)
		}
		r.logger.WithError(err).WithField("table_mapping_id", tableMappingID).Error("Failed to get checkpoint")
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}
	return &checkpoint, nil
}

func (r *MySQLRepository) UpdateCheckpoint(ctx context.Context, tableMappingID string, checkpoint *SyncCheckpoint) error {
	query := `
		UPDATE sync_checkpoints 
		SET last_sync_time = :last_sync_time, last_sync_value = :last_sync_value, 
		    checkpoint_data = :checkpoint_data, updated_at = CURRENT_TIMESTAMP
		WHERE table_mapping_id = :table_mapping_id
	`
	checkpoint.TableMappingID = tableMappingID
	result, err := r.db.NamedExecContext(ctx, query, checkpoint)
	if err != nil {
		r.logger.WithError(err).WithField("table_mapping_id", tableMappingID).Error("Failed to update checkpoint")
		return fmt.Errorf("failed to update checkpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("checkpoint not found for table mapping: %s", tableMappingID)
	}

	return nil
}

// Log operations

func (r *MySQLRepository) CreateSyncLog(ctx context.Context, log *SyncLog) error {
	query := `
		INSERT INTO sync_logs (job_id, table_name, level, message)
		VALUES (:job_id, :table_name, :level, :message)
	`
	_, err := r.db.NamedExecContext(ctx, query, log)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create sync log")
		return fmt.Errorf("failed to create sync log: %w", err)
	}
	return nil
}

func (r *MySQLRepository) GetSyncLogs(ctx context.Context, jobID string) ([]*SyncLog, error) {
	var logs []*SyncLog
	query := `SELECT * FROM sync_logs WHERE job_id = ? ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &logs, query, jobID)
	if err != nil {
		r.logger.WithError(err).WithField("job_id", jobID).Error("Failed to get sync logs")
		return nil, fmt.Errorf("failed to get sync logs: %w", err)
	}
	return logs, nil
}

// Database mapping operations

func (r *MySQLRepository) CreateDatabaseMapping(ctx context.Context, mapping *DatabaseMapping) error {
	query := `
		INSERT INTO database_mappings (remote_connection_id, local_database_name)
		VALUES (:remote_connection_id, :local_database_name)
		ON DUPLICATE KEY UPDATE
		local_database_name = VALUES(local_database_name),
		created_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.NamedExecContext(ctx, query, mapping)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create database mapping")
		return fmt.Errorf("failed to create database mapping: %w", err)
	}
	return nil
}

func (r *MySQLRepository) GetDatabaseMappings(ctx context.Context) ([]*DatabaseMapping, error) {
	var mappings []*DatabaseMapping
	query := `SELECT remote_connection_id, local_database_name, created_at FROM database_mappings ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &mappings, query)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get database mappings")
		return nil, fmt.Errorf("failed to get database mappings: %w", err)
	}
	return mappings, nil
}

func (r *MySQLRepository) UpdateDatabaseMapping(ctx context.Context, remoteConnectionID string, mapping *DatabaseMapping) error {
	query := `
		UPDATE database_mappings 
		SET local_database_name = :local_database_name
		WHERE remote_connection_id = :remote_connection_id
	`
	mapping.RemoteConnectionID = remoteConnectionID
	result, err := r.db.NamedExecContext(ctx, query, mapping)
	if err != nil {
		r.logger.WithError(err).WithField("remote_connection_id", remoteConnectionID).Error("Failed to update database mapping")
		return fmt.Errorf("failed to update database mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("database mapping not found for connection: %s", remoteConnectionID)
	}

	return nil
}

func (r *MySQLRepository) DeleteDatabaseMapping(ctx context.Context, remoteConnectionID string) error {
	query := `DELETE FROM database_mappings WHERE remote_connection_id = ?`
	result, err := r.db.ExecContext(ctx, query, remoteConnectionID)
	if err != nil {
		r.logger.WithError(err).WithField("remote_connection_id", remoteConnectionID).Error("Failed to delete database mapping")
		return fmt.Errorf("failed to delete database mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("database mapping not found for connection: %s", remoteConnectionID)
	}

	return nil
}
