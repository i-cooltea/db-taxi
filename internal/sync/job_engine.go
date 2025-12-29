package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// JobEngineService implements the JobEngine interface
type JobEngineService struct {
	repo       Repository
	logger     *logrus.Logger
	monitoring MonitoringService
	syncEngine SyncEngine

	// Job queue and worker management
	jobQueue    chan *SyncJob
	workers     []*JobWorker
	workerCount int
	running     bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mutex       sync.RWMutex

	// Active jobs tracking
	activeJobs map[string]*JobExecution
	jobsMutex  sync.RWMutex

	// Error handling and recovery
	errorHandler      *ErrorHandler
	checkpointManager *CheckpointManager
	enableCheckpoints bool
}

// JobWorker represents a worker that processes sync jobs
type JobWorker struct {
	id         int
	engine     *JobEngineService
	jobChan    chan *SyncJob
	stopChan   chan struct{}
	logger     *logrus.Logger
	processing bool
	mutex      sync.Mutex
}

// JobExecution tracks the execution state of a job
type JobExecution struct {
	Job       *SyncJob
	Worker    *JobWorker
	StartTime time.Time
	Context   context.Context
	Cancel    context.CancelFunc
}

// NewJobEngine creates a new job engine instance
func NewJobEngine(repo Repository, logger *logrus.Logger, monitoring MonitoringService, syncEngine SyncEngine) JobEngine {
	// Create error notifier
	notifier := NewLogNotifier(logger)

	// Create error handler
	errorHandler := NewErrorHandler(logger, monitoring, notifier)

	// Create checkpoint manager
	checkpointManager := NewCheckpointManager(repo, logger)

	return &JobEngineService{
		repo:              repo,
		logger:            logger,
		monitoring:        monitoring,
		syncEngine:        syncEngine,
		jobQueue:          make(chan *SyncJob, 100), // Buffer for 100 jobs
		workerCount:       5,                        // Default 5 concurrent workers
		activeJobs:        make(map[string]*JobExecution),
		stopChan:          make(chan struct{}),
		errorHandler:      errorHandler,
		checkpointManager: checkpointManager,
		enableCheckpoints: true, // Enable checkpoints by default
	}
}

// Start initializes and starts the job engine
func (je *JobEngineService) Start() error {
	je.mutex.Lock()
	defer je.mutex.Unlock()

	if je.running {
		return fmt.Errorf("job engine is already running")
	}

	je.running = true
	je.stopChan = make(chan struct{})

	// Create and start workers
	je.workers = make([]*JobWorker, je.workerCount)
	for i := 0; i < je.workerCount; i++ {
		worker := &JobWorker{
			id:       i,
			engine:   je,
			jobChan:  make(chan *SyncJob, 1),
			stopChan: make(chan struct{}),
			logger:   je.logger,
		}
		je.workers[i] = worker

		je.wg.Add(1)
		go worker.run()
	}

	// Start job dispatcher
	je.wg.Add(1)
	go je.dispatcher()

	je.logger.WithField("worker_count", je.workerCount).Info("Job engine started successfully")
	return nil
}

// Stop gracefully shuts down the job engine
func (je *JobEngineService) Stop() error {
	je.mutex.Lock()
	defer je.mutex.Unlock()

	if !je.running {
		return nil
	}

	je.logger.Info("Stopping job engine...")

	// Signal all workers to stop
	close(je.stopChan)

	// Cancel all active jobs
	je.jobsMutex.Lock()
	for jobID, execution := range je.activeJobs {
		je.logger.WithField("job_id", jobID).Info("Cancelling active job")
		execution.Cancel()
	}
	je.jobsMutex.Unlock()

	// Wait for all workers to finish
	je.wg.Wait()

	je.running = false
	je.logger.Info("Job engine stopped successfully")
	return nil
}

// SubmitJob submits a sync job for execution
func (je *JobEngineService) SubmitJob(ctx context.Context, job *SyncJob) error {
	je.mutex.RLock()
	defer je.mutex.RUnlock()

	if !je.running {
		return fmt.Errorf("job engine is not running")
	}

	// Validate job
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	if job.Status == "" {
		job.Status = JobStatusPending
	}

	// Update job status to pending
	job.Status = JobStatusPending
	job.StartTime = time.Now()

	// Save job to repository
	if err := je.repo.CreateSyncJob(ctx, job); err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}

	// Add to job queue
	select {
	case je.jobQueue <- job:
		je.logger.WithFields(logrus.Fields{
			"job_id":    job.ID,
			"config_id": job.ConfigID,
		}).Info("Job submitted successfully")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("job queue is full")
	}
}

// GetJobStatus returns the current status of a job
func (je *JobEngineService) GetJobStatus(ctx context.Context, jobID string) (*SyncJob, error) {
	job, err := je.repo.GetSyncJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job status: %w", err)
	}
	return job, nil
}

// CancelJob cancels a running job
func (je *JobEngineService) CancelJob(ctx context.Context, jobID string) error {
	je.jobsMutex.Lock()
	defer je.jobsMutex.Unlock()

	// Check if job is currently running
	if execution, exists := je.activeJobs[jobID]; exists {
		// Cancel the job context
		execution.Cancel()

		// Update job status
		execution.Job.Status = JobStatusCancelled
		now := time.Now()
		execution.Job.EndTime = &now

		if err := je.repo.UpdateSyncJob(ctx, jobID, execution.Job); err != nil {
			je.logger.WithError(err).WithField("job_id", jobID).Error("Failed to update cancelled job status")
		}

		// Finish monitoring
		if err := je.monitoring.FinishJobMonitoring(ctx, jobID, JobStatusCancelled, "Job cancelled by user"); err != nil {
			je.logger.WithError(err).WithField("job_id", jobID).Warn("Failed to finish job monitoring")
		}

		je.logger.WithField("job_id", jobID).Info("Job cancelled successfully")
		return nil
	}

	// Job is not currently running, check if it's pending
	job, err := je.repo.GetSyncJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.Status == JobStatusPending {
		// Update status to cancelled
		job.Status = JobStatusCancelled
		now := time.Now()
		job.EndTime = &now

		if err := je.repo.UpdateSyncJob(ctx, jobID, job); err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}

		je.logger.WithField("job_id", jobID).Info("Pending job cancelled successfully")
		return nil
	}

	return fmt.Errorf("job cannot be cancelled (status: %s)", job.Status)
}

// GetJobHistory returns historical job information
func (je *JobEngineService) GetJobHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error) {
	history, err := je.repo.GetJobHistory(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get job history: %w", err)
	}
	return history, nil
}

// GetJobsByStatus returns jobs filtered by status
func (je *JobEngineService) GetJobsByStatus(ctx context.Context, status JobStatus) ([]*SyncJob, error) {
	jobs, err := je.repo.GetJobsByStatus(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by status: %w", err)
	}
	return jobs, nil
}

// dispatcher distributes jobs to available workers
func (je *JobEngineService) dispatcher() {
	defer je.wg.Done()

	je.logger.Info("Job dispatcher started")

	for {
		select {
		case job := <-je.jobQueue:
			// Find an available worker
			worker := je.findAvailableWorker()
			if worker != nil {
				select {
				case worker.jobChan <- job:
					je.logger.WithFields(logrus.Fields{
						"job_id":    job.ID,
						"worker_id": worker.id,
					}).Debug("Job dispatched to worker")
				case <-je.stopChan:
					return
				}
			} else {
				// No available workers, put job back in queue
				select {
				case je.jobQueue <- job:
				case <-je.stopChan:
					return
				}
				// Wait a bit before retrying
				time.Sleep(100 * time.Millisecond)
			}
		case <-je.stopChan:
			je.logger.Info("Job dispatcher stopped")
			return
		}
	}
}

// findAvailableWorker finds a worker that is not currently processing a job
func (je *JobEngineService) findAvailableWorker() *JobWorker {
	for _, worker := range je.workers {
		worker.mutex.Lock()
		available := !worker.processing
		worker.mutex.Unlock()

		if available {
			return worker
		}
	}
	return nil
}

// run executes the worker's job processing loop
func (w *JobWorker) run() {
	defer w.engine.wg.Done()

	w.logger.Info("Worker started")

	for {
		select {
		case job := <-w.jobChan:
			w.processJob(job)
		case <-w.stopChan:
			w.logger.Info("Worker stopped")
			return
		case <-w.engine.stopChan:
			w.logger.Info("Worker stopped by engine")
			return
		}
	}
}

// processJob processes a single sync job
func (w *JobWorker) processJob(job *SyncJob) {
	w.mutex.Lock()
	w.processing = true
	w.mutex.Unlock()

	defer func() {
		w.mutex.Lock()
		w.processing = false
		w.mutex.Unlock()
	}()

	// Create job execution context
	ctx, cancel := context.WithCancel(context.Background())
	execution := &JobExecution{
		Job:       job,
		Worker:    w,
		StartTime: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
	}

	// Track active job
	w.engine.jobsMutex.Lock()
	w.engine.activeJobs[job.ID] = execution
	w.engine.jobsMutex.Unlock()

	// Remove from active jobs when done
	defer func() {
		w.engine.jobsMutex.Lock()
		delete(w.engine.activeJobs, job.ID)
		w.engine.jobsMutex.Unlock()
	}()

	w.logger.WithField("job_id", job.ID).Info("Processing job")

	// Update job status to running
	job.Status = JobStatusRunning
	job.StartTime = time.Now()

	if err := w.engine.repo.UpdateSyncJob(ctx, job.ID, job); err != nil {
		w.logger.WithError(err).WithField("job_id", job.ID).Error("Failed to update job status to running")
		return
	}

	// Start job monitoring
	if err := w.engine.monitoring.StartJobMonitoring(ctx, job.ID, job.TotalTables); err != nil {
		w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to start job monitoring")
	}

	// Log job start
	if err := w.engine.monitoring.LogJobEvent(ctx, job.ID, "", "info", "Job execution started"); err != nil {
		w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to log job event")
	}

	// Execute the job
	err := w.executeJobWithRecovery(ctx, job)

	// Update final job status
	now := time.Now()
	job.EndTime = &now

	if err != nil {
		if ctx.Err() == context.Canceled {
			job.Status = JobStatusCancelled
			job.Error = "Job was cancelled"
		} else {
			job.Status = JobStatusFailed
			job.Error = err.Error()
		}
		w.logger.WithError(err).WithField("job_id", job.ID).Error("Job execution failed")
	} else {
		job.Status = JobStatusCompleted
		job.Error = ""
		w.logger.WithField("job_id", job.ID).Info("Job execution completed successfully")
	}

	// Update job in repository
	if updateErr := w.engine.repo.UpdateSyncJob(ctx, job.ID, job); updateErr != nil {
		w.logger.WithError(updateErr).WithField("job_id", job.ID).Error("Failed to update final job status")
	}

	// Finish monitoring
	if monitorErr := w.engine.monitoring.FinishJobMonitoring(ctx, job.ID, job.Status, job.Error); monitorErr != nil {
		w.logger.WithError(monitorErr).WithField("job_id", job.ID).Warn("Failed to finish job monitoring")
	}

	// Log job completion
	logMessage := fmt.Sprintf("Job execution %s", job.Status)
	if err := w.engine.monitoring.LogJobEvent(ctx, job.ID, "", "info", logMessage); err != nil {
		w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to log job completion event")
	}
}

// executeJob executes the actual sync job logic
func (w *JobWorker) executeJob(ctx context.Context, job *SyncJob) error {
	// Get sync configuration
	syncConfig, err := w.engine.repo.GetSyncConfig(ctx, job.ConfigID)
	if err != nil {
		return fmt.Errorf("failed to get sync config: %w", err)
	}

	if !syncConfig.Enabled {
		return fmt.Errorf("sync config is disabled")
	}

	// Update job with total tables count
	enabledTables := 0
	for _, table := range syncConfig.Tables {
		if table.Enabled {
			enabledTables++
		}
	}

	job.TotalTables = enabledTables
	job.CompletedTables = 0
	job.TotalRows = 0
	job.ProcessedRows = 0

	if err := w.engine.repo.UpdateSyncJob(ctx, job.ID, job); err != nil {
		w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update job table counts")
	}

	// Process each enabled table
	for _, tableMapping := range syncConfig.Tables {
		if !tableMapping.Enabled {
			w.logger.WithFields(logrus.Fields{
				"job_id":       job.ID,
				"source_table": tableMapping.SourceTable,
			}).Debug("Skipping disabled table")
			continue
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Log table sync start
		if err := w.engine.monitoring.LogJobEvent(ctx, job.ID, tableMapping.SourceTable, "info",
			fmt.Sprintf("Starting sync for table %s", tableMapping.SourceTable)); err != nil {
			w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to log table sync start")
		}

		// Update table progress to running
		if err := w.engine.monitoring.UpdateTableProgress(ctx, job.ID, tableMapping.SourceTable,
			TableStatusRunning, 0, 0, ""); err != nil {
			w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update table progress")
		}

		// Sync the table using sync engine
		tableErr := w.engine.syncEngine.SyncTable(ctx, job, tableMapping)

		if tableErr != nil {
			// Handle table sync error based on sync options
			errorMsg := fmt.Sprintf("Table sync failed: %v", tableErr)

			// Log table error
			if err := w.engine.monitoring.LogJobEvent(ctx, job.ID, tableMapping.SourceTable, "error", errorMsg); err != nil {
				w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to log table error")
			}

			// Update table progress to failed
			if err := w.engine.monitoring.UpdateTableProgress(ctx, job.ID, tableMapping.SourceTable,
				TableStatusFailed, 0, 0, errorMsg); err != nil {
				w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update table progress")
			}

			// Check error handling strategy
			if syncConfig.Options != nil && syncConfig.Options.ConflictResolution == ConflictResolutionError {
				// Stop on first error
				return fmt.Errorf("table sync failed for %s: %w", tableMapping.SourceTable, tableErr)
			}

			// Continue with other tables (skip or overwrite strategy)
			w.logger.WithError(tableErr).WithFields(logrus.Fields{
				"job_id":       job.ID,
				"source_table": tableMapping.SourceTable,
			}).Warn("Table sync failed, continuing with other tables")
		} else {
			// Table sync successful
			if err := w.engine.monitoring.LogJobEvent(ctx, job.ID, tableMapping.SourceTable, "info",
				fmt.Sprintf("Table sync completed successfully for %s", tableMapping.SourceTable)); err != nil {
				w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to log table success")
			}

			// Update table progress to completed
			if err := w.engine.monitoring.UpdateTableProgress(ctx, job.ID, tableMapping.SourceTable,
				TableStatusCompleted, 0, 0, ""); err != nil {
				w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update table progress")
			}
		}

		// Update job progress
		job.CompletedTables++
		if err := w.engine.repo.UpdateSyncJob(ctx, job.ID, job); err != nil {
			w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update job progress")
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
			w.logger.WithError(err).WithField("job_id", job.ID).Warn("Failed to update job progress")
		}
	}

	return nil
}

// SetWorkerCount updates the number of concurrent workers
func (je *JobEngineService) SetWorkerCount(count int) error {
	je.mutex.Lock()
	defer je.mutex.Unlock()

	if je.running {
		return fmt.Errorf("cannot change worker count while engine is running")
	}

	if count <= 0 {
		return fmt.Errorf("worker count must be positive")
	}

	je.workerCount = count
	je.logger.WithField("worker_count", count).Info("Worker count updated")
	return nil
}

// GetActiveJobCount returns the number of currently active jobs
func (je *JobEngineService) GetActiveJobCount() int {
	je.jobsMutex.RLock()
	defer je.jobsMutex.RUnlock()
	return len(je.activeJobs)
}

// GetQueueLength returns the current length of the job queue
func (je *JobEngineService) GetQueueLength() int {
	return len(je.jobQueue)
}

// IsRunning returns whether the job engine is currently running
func (je *JobEngineService) IsRunning() bool {
	je.mutex.RLock()
	defer je.mutex.RUnlock()
	return je.running
}
