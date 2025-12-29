package sync

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// executeJobWithRecovery executes the actual sync job logic with error handling and recovery
func (w *JobWorker) executeJobWithRecovery(ctx context.Context, job *SyncJob) error {
	// Get sync configuration
	syncConfig, err := w.engine.repo.GetSyncConfig(ctx, job.ConfigID)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	if !syncConfig.Enabled {
		return fmt.Errorf("sync config is disabled")
	}

	// Check if job can be resumed from checkpoint
	var checkpoint *JobCheckpoint
	if w.engine.enableCheckpoints {
		canResume, err := w.engine.checkpointManager.CanResumeJob(ctx, job.ID)
		if err != nil {
			w.engine.logger.WithError(err).Warn("Failed to check resume capability")
		} else if canResume {
			checkpoint, err = w.engine.checkpointManager.GetResumePoint(ctx, job.ID)
			if err != nil {
				w.engine.logger.WithError(err).Warn("Failed to load checkpoint")
			} else {
				w.engine.logger.WithField("job_id", job.ID).Info("Resuming job from checkpoint")

				// Restore progress from checkpoint
				if checkpoint.Progress != nil {
					job.CompletedTables = checkpoint.Progress.CompletedTables
					job.TotalRows = checkpoint.Progress.TotalRows
					job.ProcessedRows = checkpoint.Progress.ProcessedRows
				}

				// Notify recovery
				if w.engine.errorHandler.notifier != nil {
					w.engine.errorHandler.notifier.NotifyRecovery(ctx, job.ID, "Job resumed from checkpoint")
				}
			}
		}
	}

	// Update job with total tables count
	enabledTables := 0
	for _, table := range syncConfig.Tables {
		if table.Enabled {
			enabledTables++
		}
	}

	job.TotalTables = enabledTables
	if checkpoint == nil {
		job.CompletedTables = 0
		job.TotalRows = 0
		job.ProcessedRows = 0
	}

	if err := w.engine.repo.UpdateSyncJob(ctx, job.ID, job); err != nil {
		w.engine.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update job table counts")
	}

	// Initialize job checkpoint
	if w.engine.enableCheckpoints && checkpoint == nil {
		checkpoint = &JobCheckpoint{
			JobID:           job.ID,
			ConfigID:        job.ConfigID,
			CompletedTables: []string{},
			Progress: &Progress{
				TotalTables:     job.TotalTables,
				CompletedTables: job.CompletedTables,
				TotalRows:       job.TotalRows,
				ProcessedRows:   job.ProcessedRows,
			},
		}
	}

	// Process each enabled table
	for _, tableMapping := range syncConfig.Tables {
		if !tableMapping.Enabled {
			w.engine.logger.WithFields(logrus.Fields{
				"job_id":       job.ID,
				"source_table": tableMapping.SourceTable,
			}).Debug("Skipping disabled table")
			continue
		}

		// Check if table was already completed (from checkpoint)
		if checkpoint != nil {
			completed, err := w.engine.checkpointManager.IsTableCompleted(ctx, job.ID, tableMapping.SourceTable)
			if err != nil {
				w.engine.logger.WithError(err).Warn("Failed to check table completion status")
			} else if completed {
				w.engine.logger.WithFields(logrus.Fields{
					"job_id":       job.ID,
					"source_table": tableMapping.SourceTable,
				}).Info("Skipping already completed table")
				continue
			}
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Update checkpoint with current table
		if w.engine.enableCheckpoints && checkpoint != nil {
			checkpoint.CurrentTable = tableMapping.SourceTable
			if err := w.engine.checkpointManager.SaveJobCheckpoint(ctx, checkpoint); err != nil {
				w.engine.logger.WithError(err).Warn("Failed to save checkpoint")
			}
		}

		// Log table sync start
		if err := w.engine.monitoring.LogJobEvent(ctx, job.ID, tableMapping.SourceTable, "info",
			fmt.Sprintf("Starting sync for table %s", tableMapping.SourceTable)); err != nil {
			w.engine.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to log table sync start")
		}

		// Update table progress to running
		if err := w.engine.monitoring.UpdateTableProgress(ctx, job.ID, tableMapping.SourceTable,
			TableStatusRunning, 0, 0, ""); err != nil {
			w.engine.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update table progress")
		}

		// Sync the table with retry logic
		var tableErr error
		retryOperation := func() error {
			return w.engine.syncEngine.SyncTable(ctx, job, tableMapping)
		}

		// Use error handler to retry with backoff
		tableErr = w.engine.errorHandler.RetryOperation(ctx, retryOperation, tableMapping.SourceTable)

		if tableErr != nil {
			// Handle table sync error
			handleErr := w.engine.errorHandler.HandleSyncError(ctx, tableErr, job, tableMapping.SourceTable)

			// Log error with suggestion
			w.engine.errorHandler.LogErrorWithSuggestion(ctx, job.ID, tableErr, tableMapping.SourceTable)

			// Update table progress to failed
			if err := w.engine.monitoring.UpdateTableProgress(ctx, job.ID, tableMapping.SourceTable,
				TableStatusFailed, 0, 0, tableErr.Error()); err != nil {
				w.engine.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update table progress")
			}

			// Check if we should stop the job
			if w.engine.errorHandler.ShouldStopJob(handleErr) {
				// Notify job failure
				if w.engine.errorHandler.notifier != nil {
					w.engine.errorHandler.notifier.NotifyJobFailure(ctx, job.ID, handleErr.Error())
				}
				return handleErr
			}

			// Check error handling strategy from config
			if syncConfig.Options != nil && syncConfig.Options.ConflictResolution == ConflictResolutionError {
				// Stop on first error
				if w.engine.errorHandler.notifier != nil {
					w.engine.errorHandler.notifier.NotifyJobFailure(ctx, job.ID, fmt.Sprintf("Table sync failed: %v", tableErr))
				}
				return fmt.Errorf("table sync failed for %s: %w", tableMapping.SourceTable, tableErr)
			}

			// Continue with other tables (skip or overwrite strategy)
			w.engine.logger.WithError(tableErr).WithFields(logrus.Fields{
				"job_id":       job.ID,
				"source_table": tableMapping.SourceTable,
			}).Warn("Table sync failed, continuing with other tables")
		} else {
			// Table sync successful
			if err := w.engine.monitoring.LogJobEvent(ctx, job.ID, tableMapping.SourceTable, "info",
				fmt.Sprintf("Table sync completed successfully for %s", tableMapping.SourceTable)); err != nil {
				w.engine.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to log table success")
			}

			// Update table progress to completed
			if err := w.engine.monitoring.UpdateTableProgress(ctx, job.ID, tableMapping.SourceTable,
				TableStatusCompleted, 0, 0, ""); err != nil {
				w.engine.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update table progress")
			}

			// Mark table as completed in checkpoint
			if w.engine.enableCheckpoints {
				if err := w.engine.checkpointManager.MarkTableCompleted(ctx, job.ID, tableMapping.SourceTable); err != nil {
					w.engine.logger.WithError(err).Warn("Failed to mark table as completed in checkpoint")
				}
			}
		}

		// Update job progress
		job.CompletedTables++
		if err := w.engine.repo.UpdateSyncJob(ctx, job.ID, job); err != nil {
			w.engine.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update job progress")
		}

		// Update monitoring progress
		progress := &Progress{
			TotalTables:     job.TotalTables,
			CompletedTables: job.CompletedTables,
			TotalRows:       job.TotalRows,
			ProcessedRows:   job.ProcessedRows,
		}
		if job.TotalRows > 0 {
			progress.Percentage = float64(job.ProcessedRows) / float64(job.TotalRows) * 100
		} else {
			progress.Percentage = float64(job.CompletedTables) / float64(job.TotalTables) * 100
		}

		if err := w.engine.monitoring.UpdateJobProgress(ctx, job.ID, progress); err != nil {
			w.engine.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update job progress")
		}

		// Update checkpoint progress
		if w.engine.enableCheckpoints && checkpoint != nil {
			checkpoint.Progress = progress
			if err := w.engine.checkpointManager.SaveJobCheckpoint(ctx, checkpoint); err != nil {
				w.engine.logger.WithError(err).Warn("Failed to update checkpoint progress")
			}
		}
	}

	// Delete checkpoint on successful completion
	if w.engine.enableCheckpoints {
		if err := w.engine.checkpointManager.DeleteJobCheckpoint(ctx, job.ID); err != nil {
			w.engine.logger.WithError(err).Warn("Failed to delete checkpoint after job completion")
		}
	}

	return nil
}

// SetErrorHandler sets a custom error handler
func (je *JobEngineService) SetErrorHandler(handler *ErrorHandler) {
	je.mutex.Lock()
	defer je.mutex.Unlock()
	je.errorHandler = handler
}

// SetCheckpointManager sets a custom checkpoint manager
func (je *JobEngineService) SetCheckpointManager(manager *CheckpointManager) {
	je.mutex.Lock()
	defer je.mutex.Unlock()
	je.checkpointManager = manager
}

// EnableCheckpoints enables or disables checkpoint functionality
func (je *JobEngineService) EnableCheckpoints(enable bool) {
	je.mutex.Lock()
	defer je.mutex.Unlock()
	je.enableCheckpoints = enable
	je.logger.WithField("enabled", enable).Info("Checkpoint functionality updated")
}

// SetRetryPolicy updates the retry policy for error handling
func (je *JobEngineService) SetRetryPolicy(policy *RetryPolicy) {
	je.mutex.Lock()
	defer je.mutex.Unlock()
	if je.errorHandler != nil {
		je.errorHandler.SetRetryPolicy(policy)
		je.logger.WithFields(logrus.Fields{
			"max_retries":    policy.MaxRetries,
			"initial_delay":  policy.InitialDelay,
			"max_delay":      policy.MaxDelay,
			"backoff_factor": policy.BackoffFactor,
		}).Info("Retry policy updated")
	}
}
