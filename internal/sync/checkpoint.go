package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// CheckpointManager manages sync checkpoints for resume functionality
type CheckpointManager struct {
	repo   Repository
	logger *logrus.Logger
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(repo Repository, logger *logrus.Logger) *CheckpointManager {
	return &CheckpointManager{
		repo:   repo,
		logger: logger,
	}
}

// JobCheckpoint represents a checkpoint for a sync job
type JobCheckpoint struct {
	JobID           string                 `json:"job_id"`
	ConfigID        string                 `json:"config_id"`
	CompletedTables []string               `json:"completed_tables"`
	CurrentTable    string                 `json:"current_table,omitempty"`
	TableCheckpoint *TableCheckpoint       `json:"table_checkpoint,omitempty"`
	Progress        *Progress              `json:"progress"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// TableCheckpoint represents a checkpoint for table synchronization
type TableCheckpoint struct {
	TableName          string      `json:"table_name"`
	LastProcessedID    interface{} `json:"last_processed_id,omitempty"`
	LastProcessedValue string      `json:"last_processed_value,omitempty"`
	ProcessedRows      int64       `json:"processed_rows"`
	TotalRows          int64       `json:"total_rows"`
	BatchNumber        int         `json:"batch_number"`
	Timestamp          time.Time   `json:"timestamp"`
}

// SaveJobCheckpoint saves a checkpoint for a job
func (cm *CheckpointManager) SaveJobCheckpoint(ctx context.Context, checkpoint *JobCheckpoint) error {
	checkpoint.UpdatedAt = time.Now()
	if checkpoint.CreatedAt.IsZero() {
		checkpoint.CreatedAt = checkpoint.UpdatedAt
	}

	// Serialize checkpoint data
	checkpointData, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// Store in sync_checkpoints table using a special table mapping ID
	syncCheckpoint := &SyncCheckpoint{
		ID:             fmt.Sprintf("job_%s", checkpoint.JobID),
		TableMappingID: checkpoint.ConfigID, // Use config ID as mapping ID for job-level checkpoints
		LastSyncTime:   checkpoint.UpdatedAt,
		CheckpointData: string(checkpointData),
		CreatedAt:      checkpoint.CreatedAt,
		UpdatedAt:      checkpoint.UpdatedAt,
	}

	// Try to get existing checkpoint
	existing, err := cm.repo.GetCheckpoint(ctx, syncCheckpoint.ID)
	if err == nil && existing != nil {
		// Update existing checkpoint
		if err := cm.repo.UpdateCheckpoint(ctx, syncCheckpoint.ID, syncCheckpoint); err != nil {
			return fmt.Errorf("failed to update checkpoint: %w", err)
		}
	} else {
		// Create new checkpoint
		if err := cm.repo.CreateCheckpoint(ctx, syncCheckpoint); err != nil {
			return fmt.Errorf("failed to create checkpoint: %w", err)
		}
	}

	cm.logger.WithFields(logrus.Fields{
		"job_id":           checkpoint.JobID,
		"completed_tables": len(checkpoint.CompletedTables),
		"current_table":    checkpoint.CurrentTable,
	}).Debug("Job checkpoint saved")

	return nil
}

// LoadJobCheckpoint loads a checkpoint for a job
func (cm *CheckpointManager) LoadJobCheckpoint(ctx context.Context, jobID string) (*JobCheckpoint, error) {
	checkpointID := fmt.Sprintf("job_%s", jobID)

	syncCheckpoint, err := cm.repo.GetCheckpoint(ctx, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	if syncCheckpoint == nil {
		return nil, nil // No checkpoint found
	}

	// Deserialize checkpoint data
	var checkpoint JobCheckpoint
	if err := json.Unmarshal([]byte(syncCheckpoint.CheckpointData), &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	cm.logger.WithFields(logrus.Fields{
		"job_id":           checkpoint.JobID,
		"completed_tables": len(checkpoint.CompletedTables),
		"current_table":    checkpoint.CurrentTable,
	}).Info("Job checkpoint loaded")

	return &checkpoint, nil
}

// DeleteJobCheckpoint deletes a checkpoint for a job
func (cm *CheckpointManager) DeleteJobCheckpoint(ctx context.Context, jobID string) error {
	checkpointID := fmt.Sprintf("job_%s", jobID)

	// Note: Repository doesn't have DeleteCheckpoint, so we'll update with empty data
	// In a real implementation, you'd add DeleteCheckpoint to the Repository interface
	syncCheckpoint := &SyncCheckpoint{
		ID:             checkpointID,
		TableMappingID: "",
		LastSyncTime:   time.Now(),
		CheckpointData: "{}",
		UpdatedAt:      time.Now(),
	}

	if err := cm.repo.UpdateCheckpoint(ctx, checkpointID, syncCheckpoint); err != nil {
		cm.logger.WithError(err).WithField("job_id", jobID).Warn("Failed to delete checkpoint")
		return err
	}

	cm.logger.WithField("job_id", jobID).Debug("Job checkpoint deleted")
	return nil
}

// SaveTableCheckpoint saves a checkpoint for table synchronization
func (cm *CheckpointManager) SaveTableCheckpoint(ctx context.Context, tableMappingID string, checkpoint *TableCheckpoint) error {
	checkpoint.Timestamp = time.Now()

	// Serialize checkpoint data
	checkpointData, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal table checkpoint: %w", err)
	}

	syncCheckpoint := &SyncCheckpoint{
		ID:             tableMappingID,
		TableMappingID: tableMappingID,
		LastSyncTime:   checkpoint.Timestamp,
		LastSyncValue:  checkpoint.LastProcessedValue,
		CheckpointData: string(checkpointData),
		UpdatedAt:      checkpoint.Timestamp,
	}

	// Try to get existing checkpoint
	existing, err := cm.repo.GetCheckpoint(ctx, tableMappingID)
	if err == nil && existing != nil {
		// Update existing checkpoint
		if err := cm.repo.UpdateCheckpoint(ctx, tableMappingID, syncCheckpoint); err != nil {
			return fmt.Errorf("failed to update table checkpoint: %w", err)
		}
	} else {
		// Create new checkpoint
		syncCheckpoint.CreatedAt = checkpoint.Timestamp
		if err := cm.repo.CreateCheckpoint(ctx, syncCheckpoint); err != nil {
			return fmt.Errorf("failed to create table checkpoint: %w", err)
		}
	}

	cm.logger.WithFields(logrus.Fields{
		"table_mapping_id": tableMappingID,
		"table_name":       checkpoint.TableName,
		"processed_rows":   checkpoint.ProcessedRows,
		"batch_number":     checkpoint.BatchNumber,
	}).Debug("Table checkpoint saved")

	return nil
}

// LoadTableCheckpoint loads a checkpoint for table synchronization
func (cm *CheckpointManager) LoadTableCheckpoint(ctx context.Context, tableMappingID string) (*TableCheckpoint, error) {
	syncCheckpoint, err := cm.repo.GetCheckpoint(ctx, tableMappingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load table checkpoint: %w", err)
	}

	if syncCheckpoint == nil {
		return nil, nil // No checkpoint found
	}

	// Deserialize checkpoint data
	var checkpoint TableCheckpoint
	if err := json.Unmarshal([]byte(syncCheckpoint.CheckpointData), &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal table checkpoint: %w", err)
	}

	cm.logger.WithFields(logrus.Fields{
		"table_mapping_id": tableMappingID,
		"table_name":       checkpoint.TableName,
		"processed_rows":   checkpoint.ProcessedRows,
	}).Debug("Table checkpoint loaded")

	return &checkpoint, nil
}

// CanResumeJob checks if a job can be resumed from a checkpoint
func (cm *CheckpointManager) CanResumeJob(ctx context.Context, jobID string) (bool, error) {
	checkpoint, err := cm.LoadJobCheckpoint(ctx, jobID)
	if err != nil {
		return false, err
	}

	return checkpoint != nil, nil
}

// GetResumePoint returns the resume point for a job
func (cm *CheckpointManager) GetResumePoint(ctx context.Context, jobID string) (*JobCheckpoint, error) {
	checkpoint, err := cm.LoadJobCheckpoint(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if checkpoint == nil {
		return nil, fmt.Errorf("no checkpoint found for job %s", jobID)
	}

	return checkpoint, nil
}

// MarkTableCompleted marks a table as completed in the job checkpoint
func (cm *CheckpointManager) MarkTableCompleted(ctx context.Context, jobID, tableName string) error {
	checkpoint, err := cm.LoadJobCheckpoint(ctx, jobID)
	if err != nil {
		return err
	}

	if checkpoint == nil {
		checkpoint = &JobCheckpoint{
			JobID:           jobID,
			CompletedTables: []string{},
			Progress:        &Progress{},
		}
	}

	// Add table to completed list if not already there
	found := false
	for _, completed := range checkpoint.CompletedTables {
		if completed == tableName {
			found = true
			break
		}
	}

	if !found {
		checkpoint.CompletedTables = append(checkpoint.CompletedTables, tableName)
	}

	// Clear current table if it matches
	if checkpoint.CurrentTable == tableName {
		checkpoint.CurrentTable = ""
		checkpoint.TableCheckpoint = nil
	}

	return cm.SaveJobCheckpoint(ctx, checkpoint)
}

// IsTableCompleted checks if a table has been completed in the checkpoint
func (cm *CheckpointManager) IsTableCompleted(ctx context.Context, jobID, tableName string) (bool, error) {
	checkpoint, err := cm.LoadJobCheckpoint(ctx, jobID)
	if err != nil {
		return false, err
	}

	if checkpoint == nil {
		return false, nil
	}

	for _, completed := range checkpoint.CompletedTables {
		if completed == tableName {
			return true, nil
		}
	}

	return false, nil
}

// UpdateJobProgress updates the progress in the job checkpoint
func (cm *CheckpointManager) UpdateJobProgress(ctx context.Context, jobID string, progress *Progress) error {
	checkpoint, err := cm.LoadJobCheckpoint(ctx, jobID)
	if err != nil {
		return err
	}

	if checkpoint == nil {
		return fmt.Errorf("no checkpoint found for job %s", jobID)
	}

	checkpoint.Progress = progress
	return cm.SaveJobCheckpoint(ctx, checkpoint)
}

// CleanupOldCheckpoints removes checkpoints older than the specified duration
func (cm *CheckpointManager) CleanupOldCheckpoints(ctx context.Context, olderThan time.Duration) error {
	// This would require additional repository methods to list and delete old checkpoints
	// For now, we'll log that cleanup is needed
	cm.logger.WithField("older_than", olderThan).Info("Checkpoint cleanup requested (not implemented)")
	return nil
}
