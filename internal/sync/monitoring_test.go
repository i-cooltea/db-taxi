package sync

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMonitoringService_StartJobMonitoring(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()
	jobID := "test-job-1"
	totalTables := 3

	err := monitoring.StartJobMonitoring(ctx, jobID, totalTables)
	assert.NoError(t, err)

	// Test that we can get active jobs
	activeJobs, err := monitoring.GetActiveJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, activeJobs, 1)
	assert.Equal(t, jobID, activeJobs[0].JobID)
	assert.Equal(t, totalTables, activeJobs[0].TotalTables)
}

func TestMonitoringService_UpdateJobProgress(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()
	jobID := "test-job-1"

	// Start monitoring first
	err := monitoring.StartJobMonitoring(ctx, jobID, 3)
	assert.NoError(t, err)

	// Mock repository calls for updating job
	mockJob := &SyncJob{
		ID:              jobID,
		Status:          JobStatusRunning,
		TotalTables:     3,
		CompletedTables: 1,
		TotalRows:       1000,
		ProcessedRows:   300,
	}

	mockRepo.On("GetSyncJob", ctx, jobID).Return(mockJob, nil)
	mockRepo.On("UpdateSyncJob", ctx, jobID, mock.AnythingOfType("*sync.SyncJob")).Return(nil)

	// Update progress
	progress := &Progress{
		TotalTables:     3,
		CompletedTables: 1,
		TotalRows:       1000,
		ProcessedRows:   300,
		Percentage:      30.0,
	}

	err = monitoring.UpdateJobProgress(ctx, jobID, progress)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_UpdateTableProgress(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()
	jobID := "test-job-1"
	tableName := "users"

	// Start monitoring first
	err := monitoring.StartJobMonitoring(ctx, jobID, 3)
	assert.NoError(t, err)

	// Update table progress
	err = monitoring.UpdateTableProgress(ctx, jobID, tableName, TableStatusRunning, 100, 500, "")
	assert.NoError(t, err)

	// Verify table progress was updated by checking active jobs
	activeJobs, err := monitoring.GetActiveJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, activeJobs, 1)

	tableProgress, exists := activeJobs[0].TableProgress[tableName]
	assert.True(t, exists)
	assert.Equal(t, TableStatusRunning, tableProgress.Status)
	assert.Equal(t, int64(100), tableProgress.ProcessedRows)
	assert.Equal(t, int64(500), tableProgress.TotalRows)
}

func TestMonitoringService_FinishJobMonitoring(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()
	jobID := "test-job-1"

	// Start monitoring first
	err := monitoring.StartJobMonitoring(ctx, jobID, 3)
	assert.NoError(t, err)

	// Mock repository calls for finishing job
	mockJob := &SyncJob{
		ID:     jobID,
		Status: JobStatusRunning,
	}

	mockRepo.On("GetSyncJob", ctx, jobID).Return(mockJob, nil)
	mockRepo.On("UpdateSyncJob", ctx, jobID, mock.AnythingOfType("*sync.SyncJob")).Return(nil)

	// Finish monitoring
	err = monitoring.FinishJobMonitoring(ctx, jobID, JobStatusCompleted, "")
	assert.NoError(t, err)

	// Verify job is no longer in active jobs
	activeJobs, err := monitoring.GetActiveJobs(ctx)
	assert.NoError(t, err)
	assert.Empty(t, activeJobs)

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_GetSyncHistory(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()
	limit := 10
	offset := 0

	// Mock repository response
	mockHistory := []*JobHistory{
		{
			SyncJob: &SyncJob{
				ID:     "job-1",
				Status: JobStatusCompleted,
			},
			ConfigName:     "test-config",
			ConnectionName: "test-connection",
		},
	}

	mockRepo.On("GetJobHistory", ctx, limit, offset).Return(mockHistory, nil)

	// Get sync history
	history, err := monitoring.GetSyncHistory(ctx, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "job-1", history[0].SyncJob.ID)
	assert.Equal(t, "test-config", history[0].ConfigName)

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_GetSyncStatistics(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()

	// Mock repository response with job history
	endTime := time.Now()
	mockHistory := []*JobHistory{
		{
			SyncJob: &SyncJob{
				ID:              "job-1",
				Status:          JobStatusCompleted,
				StartTime:       endTime.Add(-1 * time.Hour),
				EndTime:         &endTime,
				ProcessedRows:   1000,
				CompletedTables: 2,
			},
		},
		{
			SyncJob: &SyncJob{
				ID:              "job-2",
				Status:          JobStatusFailed,
				StartTime:       endTime.Add(-2 * time.Hour),
				EndTime:         &endTime,
				ProcessedRows:   500,
				CompletedTables: 1,
			},
		},
	}

	mockRepo.On("GetJobHistory", ctx, 1000, 0).Return(mockHistory, nil)

	// Get sync statistics
	stats, err := monitoring.GetSyncStatistics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalJobs)
	assert.Equal(t, int64(1), stats.CompletedJobs)
	assert.Equal(t, int64(1), stats.FailedJobs)
	assert.Equal(t, int64(1500), stats.TotalRowsSynced)
	assert.Equal(t, int64(3), stats.TotalTablesSynced)
	assert.Equal(t, 50.0, stats.ErrorRate) // 1 failed out of 2 total = 50%

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_LogJobEvent(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()
	jobID := "test-job-1"
	tableName := "users"
	level := "info"
	message := "Table sync started"

	// Mock repository call
	mockRepo.On("CreateSyncLog", ctx, mock.AnythingOfType("*sync.SyncLog")).Return(nil)

	// Log job event
	err := monitoring.LogJobEvent(ctx, jobID, tableName, level, message)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_GetJobLogs(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()
	jobID := "test-job-1"

	// Mock repository response
	mockLogs := []*SyncLog{
		{
			ID:        1,
			JobID:     jobID,
			TableName: "users",
			Level:     "info",
			Message:   "Table sync started",
			CreatedAt: time.Now(),
		},
	}

	mockRepo.On("GetSyncLogs", ctx, jobID).Return(mockLogs, nil)

	// Get job logs
	logs, err := monitoring.GetJobLogs(ctx, jobID)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, jobID, logs[0].JobID)
	assert.Equal(t, "users", logs[0].TableName)
	assert.Equal(t, "info", logs[0].Level)

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_AddJobWarning(t *testing.T) {
	mockRepo := &MockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	monitoring := NewMonitoringService(mockRepo, logger)

	ctx := context.Background()
	jobID := "test-job-1"
	warning := "Table has no primary key"

	// Start monitoring first
	err := monitoring.StartJobMonitoring(ctx, jobID, 3)
	assert.NoError(t, err)

	// Add warning
	err = monitoring.AddJobWarning(ctx, jobID, warning)
	assert.NoError(t, err)

	// Verify warning was added by checking active jobs
	activeJobs, err := monitoring.GetActiveJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, activeJobs, 1)
	assert.Contains(t, activeJobs[0].Warnings, warning)
}

// **Feature: database-sync, Property 10: Status update timeliness**
// **Validates: Requirements 4.5, 5.1**
// Property 10: For any sync job, job status and progress information should be updated promptly when state changes
func TestStatusUpdateTimeliness_Property(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("For any sync job, status and progress updates should be reflected immediately", prop.ForAll(
		func(jobID string, totalTables int, updates []progressUpdate) bool {
			// Skip invalid inputs
			if jobID == "" || totalTables <= 0 || totalTables > 100 {
				return true
			}
			if len(updates) == 0 {
				return true
			}

			mockRepo := &MockRepository{}
			logger := logrus.New()
			logger.SetLevel(logrus.FatalLevel) // Suppress all logs in property tests

			monitoring := NewMonitoringService(mockRepo, logger)
			ctx := context.Background()

			// Start monitoring
			err := monitoring.StartJobMonitoring(ctx, jobID, totalTables)
			if err != nil {
				return false
			}

			// Record time before update
			beforeUpdate := time.Now()

			// Apply each update and verify timeliness
			for _, update := range updates {
				// Ensure update values are valid
				if update.completedTables < 0 || update.completedTables > totalTables {
					continue
				}
				if update.processedRows < 0 || update.totalRows < 0 {
					continue
				}

				// Mock repository calls for updating job
				mockJob := &SyncJob{
					ID:              jobID,
					Status:          JobStatusRunning,
					TotalTables:     totalTables,
					CompletedTables: update.completedTables,
					TotalRows:       update.totalRows,
					ProcessedRows:   update.processedRows,
				}

				mockRepo.On("GetSyncJob", ctx, jobID).Return(mockJob, nil).Once()
				mockRepo.On("UpdateSyncJob", ctx, jobID, mock.AnythingOfType("*sync.SyncJob")).Return(nil).Once()

				// Update progress
				progress := &Progress{
					TotalTables:     totalTables,
					CompletedTables: update.completedTables,
					TotalRows:       update.totalRows,
					ProcessedRows:   update.processedRows,
					Percentage:      float64(update.processedRows) / float64(update.totalRows) * 100,
				}

				err := monitoring.UpdateJobProgress(ctx, jobID, progress)
				if err != nil {
					return false
				}

				// Verify update is reflected immediately (within reasonable time)
				afterUpdate := time.Now()
				updateLatency := afterUpdate.Sub(beforeUpdate)

				// Get active jobs to verify update
				activeJobs, err := monitoring.GetActiveJobs(ctx)
				if err != nil {
					return false
				}

				// Find our job
				var found bool
				for _, job := range activeJobs {
					if job.JobID == jobID {
						found = true
						// Verify the update was applied
						if job.CompletedTables != update.completedTables {
							return false
						}
						if job.ProcessedRows != update.processedRows {
							return false
						}
						if job.TotalRows != update.totalRows {
							return false
						}
						break
					}
				}

				if !found {
					return false
				}

				// Verify timeliness: update should be reflected within 100ms
				// This is a reasonable threshold for "immediate" updates
				if updateLatency > 100*time.Millisecond {
					return false
				}

				beforeUpdate = time.Now()
			}

			return true
		},
		genJobID(),
		genTotalTables(),
		genProgressUpdates(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// progressUpdate represents a progress update event
type progressUpdate struct {
	completedTables int
	totalRows       int64
	processedRows   int64
}

// genJobID generates valid job IDs
func genJobID() gopter.Gen {
	return gen.Identifier()
}

// genTotalTables generates valid total table counts
func genTotalTables() gopter.Gen {
	return gen.IntRange(1, 20)
}

// genProgressUpdates generates a sequence of progress updates
func genProgressUpdates() gopter.Gen {
	return gen.SliceOfN(5, genProgressUpdate()).SuchThat(func(updates []progressUpdate) bool {
		// Ensure updates are monotonically increasing
		for i := 1; i < len(updates); i++ {
			if updates[i].processedRows < updates[i-1].processedRows {
				return false
			}
		}
		return true
	})
}

// genProgressUpdate generates a single progress update
func genProgressUpdate() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(0, 20),       // completedTables
		gen.Int64Range(100, 1000), // totalRows
		gen.Int64Range(0, 1000),   // processedRows
	).Map(func(values []interface{}) progressUpdate {
		completedTables := values[0].(int)
		totalRows := values[1].(int64)
		processedRows := values[2].(int64)

		// Ensure processedRows doesn't exceed totalRows
		if processedRows > totalRows {
			processedRows = totalRows
		}

		return progressUpdate{
			completedTables: completedTables,
			totalRows:       totalRows,
			processedRows:   processedRows,
		}
	})
}
