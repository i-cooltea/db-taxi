package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MonitoringServiceImpl provides sync status monitoring and statistics collection
type MonitoringServiceImpl struct {
	repo            Repository
	logger          *logrus.Logger
	activeJobs      map[string]*JobMonitor
	jobsMutex       sync.RWMutex
	statisticsCache *SyncStatistics
	statsMutex      sync.RWMutex
	lastStatsUpdate time.Time
}

// JobMonitor tracks the progress of a single sync job
type JobMonitor struct {
	JobID           string
	ConfigID        string
	StartTime       time.Time
	LastUpdate      time.Time
	CurrentTable    string
	TablesProgress  map[string]*TableProgress
	TotalTables     int
	CompletedTables int
	TotalRows       int64
	ProcessedRows   int64
	ErrorCount      int
	Warnings        []string
	mutex           sync.RWMutex
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(repo Repository, logger *logrus.Logger) MonitoringService {
	return &MonitoringServiceImpl{
		repo:       repo,
		logger:     logger,
		activeJobs: make(map[string]*JobMonitor),
	}
}

// StartJobMonitoring starts monitoring a sync job
// Requirement 5.1: Real-time display of sync progress and status
func (m *MonitoringServiceImpl) StartJobMonitoring(ctx context.Context, jobID string, totalTables int) error {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	// Get job details from repository to fetch ConfigID
	job, err := m.repo.GetSyncJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job details: %w", err)
	}

	monitor := &JobMonitor{
		JobID:          jobID,
		ConfigID:       job.ConfigID,
		StartTime:      time.Now(),
		LastUpdate:     time.Now(),
		TablesProgress: make(map[string]*TableProgress),
		TotalTables:    totalTables,
	}

	m.activeJobs[jobID] = monitor

	m.logger.WithFields(logrus.Fields{
		"job_id":       jobID,
		"total_tables": totalTables,
	}).Info("Started job monitoring")

	return nil
}

// UpdateJobProgress updates the progress of a sync job
// Requirement 5.1: Real-time display of sync progress and status
func (m *MonitoringServiceImpl) UpdateJobProgress(ctx context.Context, jobID string, progress *Progress) error {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	monitor, exists := m.activeJobs[jobID]
	if !exists {
		return fmt.Errorf("job monitor not found: %s", jobID)
	}

	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()

	monitor.LastUpdate = time.Now()
	monitor.CompletedTables = progress.CompletedTables
	monitor.TotalRows = progress.TotalRows
	monitor.ProcessedRows = progress.ProcessedRows

	// Update job in repository
	job, err := m.repo.GetSyncJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get sync job: %w", err)
	}

	job.CompletedTables = progress.CompletedTables
	job.TotalRows = progress.TotalRows
	job.ProcessedRows = progress.ProcessedRows
	job.Progress = progress

	if err := m.repo.UpdateSyncJob(ctx, jobID, job); err != nil {
		return fmt.Errorf("failed to update sync job: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"job_id":           jobID,
		"completed_tables": progress.CompletedTables,
		"total_tables":     progress.TotalTables,
		"processed_rows":   progress.ProcessedRows,
		"total_rows":       progress.TotalRows,
		"percentage":       progress.Percentage,
	}).Debug("Updated job progress")

	return nil
}

// UpdateTableProgress updates the progress of a specific table sync
// Requirement 5.1: Real-time display of sync progress and status
func (m *MonitoringServiceImpl) UpdateTableProgress(ctx context.Context, jobID, tableName string, status TableSyncStatus, processedRows, totalRows int64, errorMsg string) error {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	monitor, exists := m.activeJobs[jobID]
	if !exists {
		return fmt.Errorf("job monitor not found: %s", jobID)
	}

	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()

	tableProgress, exists := monitor.TablesProgress[tableName]
	if !exists {
		tableProgress = &TableProgress{
			TableName: tableName,
			StartTime: time.Now(),
		}
		monitor.TablesProgress[tableName] = tableProgress
	}

	tableProgress.Status = status
	tableProgress.ProcessedRows = processedRows
	tableProgress.TotalRows = totalRows

	if errorMsg != "" {
		tableProgress.ErrorCount++
		tableProgress.LastError = errorMsg
		monitor.ErrorCount++
	}

	if status == TableStatusCompleted || status == TableStatusFailed || status == TableStatusSkipped {
		now := time.Now()
		tableProgress.EndTime = &now
	}

	if status == TableStatusRunning {
		monitor.CurrentTable = tableName
	}

	monitor.LastUpdate = time.Now()

	m.logger.WithFields(logrus.Fields{
		"job_id":         jobID,
		"table_name":     tableName,
		"status":         status,
		"processed_rows": processedRows,
		"total_rows":     totalRows,
	}).Debug("Updated table progress")

	return nil
}

// GetJobProgress returns the current progress of a sync job
// Requirement 5.1: Real-time display of sync progress and status
func (m *MonitoringServiceImpl) GetJobProgress(ctx context.Context, jobID string) (*JobSummary, error) {
	m.jobsMutex.RLock()
	defer m.jobsMutex.RUnlock()

	monitor, exists := m.activeJobs[jobID]
	if !exists {
		// Try to get from repository for completed jobs
		job, err := m.repo.GetSyncJob(ctx, jobID)
		if err != nil {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}

		summary := &JobSummary{
			JobID:           job.ID,
			ConfigID:        job.ConfigID,
			Status:          job.Status,
			StartTime:       job.StartTime,
			EndTime:         job.EndTime,
			TotalTables:     job.TotalTables,
			CompletedTables: job.CompletedTables,
			TotalRows:       job.TotalRows,
			ProcessedRows:   job.ProcessedRows,
		}

		if job.EndTime != nil {
			duration := job.EndTime.Sub(job.StartTime)
			summary.Duration = &duration
		}

		if job.TotalRows > 0 {
			summary.ProgressPercent = float64(job.ProcessedRows) / float64(job.TotalRows) * 100
		}

		return summary, nil
	}

	monitor.mutex.RLock()
	defer monitor.mutex.RUnlock()

	summary := &JobSummary{
		JobID:           monitor.JobID,
		StartTime:       monitor.StartTime,
		TotalTables:     monitor.TotalTables,
		CompletedTables: monitor.CompletedTables,
		TotalRows:       monitor.TotalRows,
		ProcessedRows:   monitor.ProcessedRows,
		ErrorCount:      monitor.ErrorCount,
		Warnings:        monitor.Warnings,
		TableProgress:   make(map[string]*TableProgress),
	}

	// Copy table progress
	for name, progress := range monitor.TablesProgress {
		summary.TableProgress[name] = &TableProgress{
			TableName:     progress.TableName,
			Status:        progress.Status,
			StartTime:     progress.StartTime,
			EndTime:       progress.EndTime,
			TotalRows:     progress.TotalRows,
			ProcessedRows: progress.ProcessedRows,
			ErrorCount:    progress.ErrorCount,
			LastError:     progress.LastError,
		}
	}

	// Calculate progress percentage
	if summary.TotalRows > 0 {
		summary.ProgressPercent = float64(summary.ProcessedRows) / float64(summary.TotalRows) * 100
	}

	// Get job status from repository
	job, err := m.repo.GetSyncJob(ctx, jobID)
	if err == nil {
		summary.ConfigID = job.ConfigID
		summary.Status = job.Status
		summary.EndTime = job.EndTime

		if job.EndTime != nil {
			duration := job.EndTime.Sub(job.StartTime)
			summary.Duration = &duration
		}
	}

	return summary, nil
}

// FinishJobMonitoring completes monitoring for a sync job
// Requirement 5.2: Display historical sync records and results
func (m *MonitoringServiceImpl) FinishJobMonitoring(ctx context.Context, jobID string, status JobStatus, errorMsg string) error {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	m.logger.WithFields(logrus.Fields{
		"job_id": jobID,
		"status": status,
	}).Info("FinishJobMonitoring called")

	monitor, exists := m.activeJobs[jobID]
	if !exists {
		m.logger.WithField("job_id", jobID).Warn("Job monitor not found in activeJobs map")
		return fmt.Errorf("job monitor not found: %s", jobID)
	}

	// Always remove from active jobs, even if database update fails
	// This prevents zombie jobs
	defer func() {
		m.logger.WithField("job_id", jobID).Info("Removing job from activeJobs map")
		delete(m.activeJobs, jobID)
		m.logger.WithField("job_id", jobID).Info("Job removed from activeJobs map successfully")

		// Invalidate statistics cache
		m.statsMutex.Lock()
		m.statisticsCache = nil
		m.statsMutex.Unlock()
	}()

	// Update final job status in database
	job, err := m.repo.GetSyncJob(ctx, jobID)
	if err != nil {
		m.logger.WithError(err).WithField("job_id", jobID).Error("Failed to get sync job from repository")
		// Don't return error - we still want to remove from activeJobs
		return nil
	}

	now := time.Now()
	job.Status = status
	job.EndTime = &now
	if errorMsg != "" {
		job.Error = errorMsg
	}

	if err := m.repo.UpdateSyncJob(ctx, jobID, job); err != nil {
		m.logger.WithError(err).WithField("job_id", jobID).Error("Failed to update sync job in repository")
		// Don't return error - we still want to remove from activeJobs
		return nil
	}

	// Log completion
	duration := now.Sub(monitor.StartTime)
	m.logger.WithFields(logrus.Fields{
		"job_id":           jobID,
		"status":           status,
		"duration":         duration,
		"completed_tables": monitor.CompletedTables,
		"total_tables":     monitor.TotalTables,
		"processed_rows":   monitor.ProcessedRows,
		"error_count":      monitor.ErrorCount,
	}).Info("Finished job monitoring")

	return nil
}

// GetSyncHistory returns historical sync records
// Requirement 5.2: Display historical sync records and results
func (m *MonitoringServiceImpl) GetSyncHistory(ctx context.Context, limit, offset int) ([]*JobHistory, error) {
	history, err := m.repo.GetJobHistory(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync history: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"limit":  limit,
		"offset": offset,
		"count":  len(history),
	}).Debug("Retrieved sync history")

	return history, nil
}

// GetSyncStatistics returns overall synchronization statistics
// Requirement 5.4: Display statistics information including data volume and time consumption
func (m *MonitoringServiceImpl) GetSyncStatistics(ctx context.Context) (*SyncStatistics, error) {
	m.statsMutex.RLock()
	if m.statisticsCache != nil && time.Since(m.lastStatsUpdate) < 5*time.Minute {
		defer m.statsMutex.RUnlock()
		return m.statisticsCache, nil
	}
	m.statsMutex.RUnlock()

	// Calculate fresh statistics
	stats, err := m.calculateStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate statistics: %w", err)
	}

	// Cache the results
	m.statsMutex.Lock()
	m.statisticsCache = stats
	m.lastStatsUpdate = time.Now()
	m.statsMutex.Unlock()

	m.logger.WithFields(logrus.Fields{
		"total_jobs":     stats.TotalJobs,
		"completed_jobs": stats.CompletedJobs,
		"failed_jobs":    stats.FailedJobs,
		"running_jobs":   stats.RunningJobs,
		"total_rows":     stats.TotalRowsSynced,
		"avg_duration":   stats.AverageJobDuration,
		"error_rate":     stats.ErrorRate,
	}).Debug("Generated sync statistics")

	return stats, nil
}

// calculateStatistics calculates fresh synchronization statistics
func (m *MonitoringServiceImpl) calculateStatistics(ctx context.Context) (*SyncStatistics, error) {
	stats := &SyncStatistics{
		GeneratedAt: time.Now(),
	}

	// Get all jobs for statistics
	allJobs, err := m.repo.GetJobHistory(ctx, 1000, 0) // Get last 1000 jobs
	if err != nil {
		return nil, fmt.Errorf("failed to get job history: %w", err)
	}

	var totalDuration time.Duration
	var completedCount, failedCount int64
	var totalRows, totalTables int64
	var lastSyncTime time.Time

	for _, jobHistory := range allJobs {
		job := jobHistory.SyncJob
		stats.TotalJobs++

		switch job.Status {
		case JobStatusCompleted:
			completedCount++
			if job.EndTime != nil && job.EndTime.After(lastSyncTime) {
				lastSyncTime = *job.EndTime
			}
		case JobStatusFailed:
			failedCount++
		case JobStatusRunning:
			stats.RunningJobs++
		}

		totalRows += job.ProcessedRows
		totalTables += int64(job.CompletedTables)

		if job.EndTime != nil {
			totalDuration += job.EndTime.Sub(job.StartTime)
		}
	}

	stats.CompletedJobs = completedCount
	stats.FailedJobs = failedCount
	stats.TotalRowsSynced = totalRows
	stats.TotalTablesSynced = totalTables
	stats.LastSyncTime = lastSyncTime

	// Calculate averages
	if completedCount > 0 {
		stats.AverageJobDuration = totalDuration.Minutes() / float64(completedCount)
	}

	if stats.TotalJobs > 0 {
		stats.ErrorRate = float64(failedCount) / float64(stats.TotalJobs) * 100
	}

	// Calculate sync frequency (jobs per hour in last 24 hours)
	if len(allJobs) > 0 {
		cutoff := time.Now().Add(-24 * time.Hour)
		recentJobs := 0
		for _, jobHistory := range allJobs {
			if jobHistory.SyncJob.StartTime.After(cutoff) {
				recentJobs++
			}
		}
		stats.SyncFrequency = float64(recentJobs) / 24.0
	}

	return stats, nil
}

// GetActiveJobs returns currently running sync jobs
// Requirement 5.1: Real-time display of sync progress and status
func (m *MonitoringServiceImpl) GetActiveJobs(ctx context.Context) ([]*JobSummary, error) {
	m.jobsMutex.RLock()
	defer m.jobsMutex.RUnlock()

	var activeJobs []*JobSummary

	for _, monitor := range m.activeJobs {
		monitor.mutex.RLock()

		summary := &JobSummary{
			JobID:           monitor.JobID,
			ConfigID:        monitor.ConfigID,
			StartTime:       monitor.StartTime,
			TotalTables:     monitor.TotalTables,
			CompletedTables: monitor.CompletedTables,
			TotalRows:       monitor.TotalRows,
			ProcessedRows:   monitor.ProcessedRows,
			ErrorCount:      monitor.ErrorCount,
			Warnings:        make([]string, len(monitor.Warnings)),
			TableProgress:   make(map[string]*TableProgress),
		}

		// Copy warnings
		copy(summary.Warnings, monitor.Warnings)

		// Copy table progress
		for name, progress := range monitor.TablesProgress {
			summary.TableProgress[name] = &TableProgress{
				TableName:     progress.TableName,
				Status:        progress.Status,
				StartTime:     progress.StartTime,
				EndTime:       progress.EndTime,
				TotalRows:     progress.TotalRows,
				ProcessedRows: progress.ProcessedRows,
				ErrorCount:    progress.ErrorCount,
				LastError:     progress.LastError,
			}
		}

		// Calculate progress percentage
		if summary.TotalRows > 0 {
			summary.ProgressPercent = float64(summary.ProcessedRows) / float64(summary.TotalRows) * 100
		}

		monitor.mutex.RUnlock()
		activeJobs = append(activeJobs, summary)
	}

	return activeJobs, nil
}

// AddJobWarning adds a warning message to a job
func (m *MonitoringServiceImpl) AddJobWarning(ctx context.Context, jobID, warning string) error {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	monitor, exists := m.activeJobs[jobID]
	if !exists {
		return fmt.Errorf("job monitor not found: %s", jobID)
	}

	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()

	monitor.Warnings = append(monitor.Warnings, warning)
	monitor.LastUpdate = time.Now()

	m.logger.WithFields(logrus.Fields{
		"job_id":  jobID,
		"warning": warning,
	}).Warn("Added job warning")

	return nil
}

// GetJobLogs returns logs for a specific job
// Requirement 5.3: Display detailed error information and suggestions when sync fails
func (m *MonitoringServiceImpl) GetJobLogs(ctx context.Context, jobID string) ([]*SyncLog, error) {
	logs, err := m.repo.GetSyncLogs(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job logs: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"job_id":    jobID,
		"log_count": len(logs),
	}).Debug("Retrieved job logs")

	return logs, nil
}

// LogJobEvent logs an event for a sync job
// Requirement 5.3: Display detailed error information and suggestions when sync fails
func (m *MonitoringServiceImpl) LogJobEvent(ctx context.Context, jobID, tableName, level, message string) error {
	log := &SyncLog{
		JobID:     jobID,
		TableName: tableName,
		Level:     level,
		Message:   message,
		CreatedAt: time.Now(),
	}

	if err := m.repo.CreateSyncLog(ctx, log); err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"job_id":     jobID,
		"table_name": tableName,
		"level":      level,
		"message":    message,
	}).Debug("Logged job event")

	return nil
}
