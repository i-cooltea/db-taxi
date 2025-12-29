package sync

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestJobEngine_NewJobEngine tests the creation of a new job engine
func TestJobEngine_NewJobEngine(t *testing.T) {
	mockRepo := &MockRepository{}
	mockMonitoring := &MockMonitoringService{}
	mockSyncEngine := &MockSyncEngine{}
	logger := logrus.New()

	engine := NewJobEngine(mockRepo, logger, mockMonitoring, mockSyncEngine)

	assert.NotNil(t, engine)

	// Type assertion to access internal fields
	engineImpl := engine.(*JobEngineService)
	assert.Equal(t, mockRepo, engineImpl.repo)
	assert.Equal(t, logger, engineImpl.logger)
	assert.Equal(t, mockMonitoring, engineImpl.monitoring)
	assert.Equal(t, mockSyncEngine, engineImpl.syncEngine)
	assert.Equal(t, 5, engineImpl.workerCount)
	assert.False(t, engineImpl.running)
}

// TestJobEngine_SubmitJob tests job submission functionality
func TestJobEngine_SubmitJob(t *testing.T) {
	mockRepo := &MockRepository{}
	mockMonitoring := &MockMonitoringService{}
	mockSyncEngine := &MockSyncEngine{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests

	engine := NewJobEngine(mockRepo, logger, mockMonitoring, mockSyncEngine).(*JobEngineService)

	// Test submitting job when engine is not running
	job := &SyncJob{
		ID:       "test-job-1",
		ConfigID: "test-config-1",
	}

	ctx := context.Background()
	err := engine.SubmitJob(ctx, job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job engine is not running")

	// Start the engine
	err = engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Mock repository calls for job processing
	mockRepo.On("CreateSyncJob", ctx, mock.AnythingOfType("*sync.SyncJob")).Return(nil)
	mockRepo.On("GetSyncConfig", mock.Anything, "test-config-1").Return(&SyncConfig{
		ID:           "test-config-1",
		ConnectionID: "test-conn-1",
		Enabled:      false, // Disabled to prevent actual sync processing
		Tables:       []*TableMapping{},
	}, nil)
	mockRepo.On("UpdateSyncJob", mock.Anything, "test-job-1", mock.AnythingOfType("*sync.SyncJob")).Return(nil)

	// Mock monitoring calls
	mockMonitoring.On("StartJobMonitoring", mock.Anything, "test-job-1", mock.AnythingOfType("int")).Return(nil)
	mockMonitoring.On("LogJobEvent", mock.Anything, "test-job-1", "", "info", mock.AnythingOfType("string")).Return(nil)
	mockMonitoring.On("FinishJobMonitoring", mock.Anything, "test-job-1", mock.AnythingOfType("sync.JobStatus"), mock.AnythingOfType("string")).Return(nil)

	// Test successful job submission
	err = engine.SubmitJob(ctx, job)
	assert.NoError(t, err)
	assert.Equal(t, JobStatusPending, job.Status)

	// Wait a bit for job processing to complete
	require.Eventually(t, func() bool {
		return engine.GetActiveJobCount() == 0
	}, 1000*time.Millisecond, 10*time.Millisecond)

	mockRepo.AssertExpectations(t)
}

// TestJobEngine_GetJobStatus tests getting job status
func TestJobEngine_GetJobStatus(t *testing.T) {
	mockRepo := &MockRepository{}
	mockMonitoring := &MockMonitoringService{}
	mockSyncEngine := &MockSyncEngine{}
	logger := logrus.New()

	engine := NewJobEngine(mockRepo, logger, mockMonitoring, mockSyncEngine)

	ctx := context.Background()
	jobID := "test-job-1"

	expectedJob := &SyncJob{
		ID:       jobID,
		ConfigID: "test-config-1",
		Status:   JobStatusRunning,
	}

	mockRepo.On("GetSyncJob", ctx, jobID).Return(expectedJob, nil)

	job, err := engine.GetJobStatus(ctx, jobID)
	assert.NoError(t, err)
	assert.Equal(t, expectedJob, job)

	mockRepo.AssertExpectations(t)
}

// TestJobEngine_CancelJob tests job cancellation
func TestJobEngine_CancelJob(t *testing.T) {
	mockRepo := &MockRepository{}
	mockMonitoring := &MockMonitoringService{}
	mockSyncEngine := &MockSyncEngine{}
	logger := logrus.New()

	engine := NewJobEngine(mockRepo, logger, mockMonitoring, mockSyncEngine).(*JobEngineService)

	ctx := context.Background()
	jobID := "test-job-1"

	// Test cancelling a pending job
	pendingJob := &SyncJob{
		ID:       jobID,
		ConfigID: "test-config-1",
		Status:   JobStatusPending,
	}

	mockRepo.On("GetSyncJob", ctx, jobID).Return(pendingJob, nil)
	mockRepo.On("UpdateSyncJob", ctx, jobID, mock.AnythingOfType("*sync.SyncJob")).Return(nil)

	err := engine.CancelJob(ctx, jobID)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// TestJobEngine_StartStop tests engine start and stop functionality
func TestJobEngine_StartStop(t *testing.T) {
	mockRepo := &MockRepository{}
	mockMonitoring := &MockMonitoringService{}
	mockSyncEngine := &MockSyncEngine{}
	logger := logrus.New()

	engine := NewJobEngine(mockRepo, logger, mockMonitoring, mockSyncEngine).(*JobEngineService)

	// Test initial state
	assert.False(t, engine.IsRunning())
	assert.Equal(t, 0, engine.GetActiveJobCount())

	// Test start
	err := engine.Start()
	assert.NoError(t, err)
	assert.True(t, engine.IsRunning())

	// Test start when already running
	err = engine.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stop
	err = engine.Stop()
	assert.NoError(t, err)
	assert.False(t, engine.IsRunning())

	// Test stop when not running
	err = engine.Stop()
	assert.NoError(t, err) // Should not error
}

// TestJobEngine_SetWorkerCount tests worker count configuration
func TestJobEngine_SetWorkerCount(t *testing.T) {
	mockRepo := &MockRepository{}
	mockMonitoring := &MockMonitoringService{}
	mockSyncEngine := &MockSyncEngine{}
	logger := logrus.New()

	engine := NewJobEngine(mockRepo, logger, mockMonitoring, mockSyncEngine).(*JobEngineService)

	// Test setting worker count when not running
	err := engine.SetWorkerCount(10)
	assert.NoError(t, err)
	assert.Equal(t, 10, engine.workerCount)

	// Test invalid worker count
	err = engine.SetWorkerCount(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker count must be positive")

	err = engine.SetWorkerCount(-1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker count must be positive")

	// Test setting worker count when running
	err = engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	err = engine.SetWorkerCount(15)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot change worker count while engine is running")
}
