package sync

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckpointManager_SaveAndLoadJobCheckpoint(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	mockRepo := new(MockRepository)

	cm := NewCheckpointManager(mockRepo, logger)
	ctx := context.Background()

	checkpoint := &JobCheckpoint{
		JobID:           "job-1",
		ConfigID:        "config-1",
		CompletedTables: []string{"table1", "table2"},
		CurrentTable:    "table3",
		Progress: &Progress{
			TotalTables:     5,
			CompletedTables: 2,
			TotalRows:       1000,
			ProcessedRows:   500,
			Percentage:      50.0,
		},
	}

	t.Run("Save new checkpoint", func(t *testing.T) {
		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return((*SyncCheckpoint)(nil), nil).Once()
		mockRepo.On("CreateCheckpoint", ctx, mock.AnythingOfType("*sync.SyncCheckpoint")).Return(nil).Once()

		err := cm.SaveJobCheckpoint(ctx, checkpoint)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Load checkpoint", func(t *testing.T) {
		checkpointData, _ := json.Marshal(checkpoint)
		syncCheckpoint := &SyncCheckpoint{
			ID:             "job_job-1",
			TableMappingID: "config-1",
			CheckpointData: string(checkpointData),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()

		loaded, err := cm.LoadJobCheckpoint(ctx, "job-1")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, checkpoint.JobID, loaded.JobID)
		assert.Equal(t, checkpoint.ConfigID, loaded.ConfigID)
		assert.Equal(t, len(checkpoint.CompletedTables), len(loaded.CompletedTables))
		assert.Equal(t, checkpoint.CurrentTable, loaded.CurrentTable)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Load non-existent checkpoint", func(t *testing.T) {
		mockRepo.On("GetCheckpoint", ctx, "job_job-2").Return((*SyncCheckpoint)(nil), nil).Once()

		loaded, err := cm.LoadJobCheckpoint(ctx, "job-2")
		assert.NoError(t, err)
		assert.Nil(t, loaded)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Update existing checkpoint", func(t *testing.T) {
		existingCheckpoint := &SyncCheckpoint{
			ID:             "job_job-1",
			TableMappingID: "config-1",
			CheckpointData: "{}",
		}

		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(existingCheckpoint, nil).Once()
		mockRepo.On("UpdateCheckpoint", ctx, "job_job-1", mock.AnythingOfType("*sync.SyncCheckpoint")).Return(nil).Once()

		err := cm.SaveJobCheckpoint(ctx, checkpoint)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestCheckpointManager_SaveAndLoadTableCheckpoint(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	mockRepo := new(MockRepository)

	cm := NewCheckpointManager(mockRepo, logger)
	ctx := context.Background()

	tableCheckpoint := &TableCheckpoint{
		TableName:          "test_table",
		LastProcessedID:    12345,
		LastProcessedValue: "2024-01-01",
		ProcessedRows:      1000,
		TotalRows:          5000,
		BatchNumber:        10,
	}

	t.Run("Save table checkpoint", func(t *testing.T) {
		mockRepo.On("GetCheckpoint", ctx, "mapping-1").Return((*SyncCheckpoint)(nil), nil).Once()
		mockRepo.On("CreateCheckpoint", ctx, mock.AnythingOfType("*sync.SyncCheckpoint")).Return(nil).Once()

		err := cm.SaveTableCheckpoint(ctx, "mapping-1", tableCheckpoint)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Load table checkpoint", func(t *testing.T) {
		checkpointData, _ := json.Marshal(tableCheckpoint)
		syncCheckpoint := &SyncCheckpoint{
			ID:             "mapping-1",
			TableMappingID: "mapping-1",
			CheckpointData: string(checkpointData),
			LastSyncValue:  "2024-01-01",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		mockRepo.On("GetCheckpoint", ctx, "mapping-1").Return(syncCheckpoint, nil).Once()

		loaded, err := cm.LoadTableCheckpoint(ctx, "mapping-1")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, tableCheckpoint.TableName, loaded.TableName)
		assert.Equal(t, tableCheckpoint.ProcessedRows, loaded.ProcessedRows)
		assert.Equal(t, tableCheckpoint.BatchNumber, loaded.BatchNumber)

		mockRepo.AssertExpectations(t)
	})
}

func TestCheckpointManager_CanResumeJob(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	mockRepo := new(MockRepository)

	cm := NewCheckpointManager(mockRepo, logger)
	ctx := context.Background()

	t.Run("Can resume when checkpoint exists", func(t *testing.T) {
		checkpoint := &JobCheckpoint{
			JobID:    "job-1",
			ConfigID: "config-1",
		}
		checkpointData, _ := json.Marshal(checkpoint)
		syncCheckpoint := &SyncCheckpoint{
			ID:             "job_job-1",
			CheckpointData: string(checkpointData),
		}

		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()

		canResume, err := cm.CanResumeJob(ctx, "job-1")
		assert.NoError(t, err)
		assert.True(t, canResume)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Cannot resume when checkpoint doesn't exist", func(t *testing.T) {
		mockRepo.On("GetCheckpoint", ctx, "job_job-2").Return((*SyncCheckpoint)(nil), nil).Once()

		canResume, err := cm.CanResumeJob(ctx, "job-2")
		assert.NoError(t, err)
		assert.False(t, canResume)

		mockRepo.AssertExpectations(t)
	})
}

func TestCheckpointManager_MarkTableCompleted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	mockRepo := new(MockRepository)

	cm := NewCheckpointManager(mockRepo, logger)
	ctx := context.Background()

	t.Run("Mark table completed in existing checkpoint", func(t *testing.T) {
		checkpoint := &JobCheckpoint{
			JobID:           "job-1",
			ConfigID:        "config-1",
			CompletedTables: []string{"table1"},
			CurrentTable:    "table2",
		}
		checkpointData, _ := json.Marshal(checkpoint)
		syncCheckpoint := &SyncCheckpoint{
			ID:             "job_job-1",
			CheckpointData: string(checkpointData),
		}

		// First call to load the checkpoint
		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()
		// Second call when saving the updated checkpoint
		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()
		mockRepo.On("UpdateCheckpoint", ctx, "job_job-1", mock.AnythingOfType("*sync.SyncCheckpoint")).Return(nil).Once()

		err := cm.MarkTableCompleted(ctx, "job-1", "table2")
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Mark table completed in new checkpoint", func(t *testing.T) {
		// First call to load (returns nil)
		mockRepo.On("GetCheckpoint", ctx, "job_job-2").Return((*SyncCheckpoint)(nil), nil).Once()
		// Second call when saving the new checkpoint
		mockRepo.On("GetCheckpoint", ctx, "job_job-2").Return((*SyncCheckpoint)(nil), nil).Once()
		mockRepo.On("CreateCheckpoint", ctx, mock.AnythingOfType("*sync.SyncCheckpoint")).Return(nil).Once()

		err := cm.MarkTableCompleted(ctx, "job-2", "table1")
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestCheckpointManager_IsTableCompleted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	mockRepo := new(MockRepository)

	cm := NewCheckpointManager(mockRepo, logger)
	ctx := context.Background()

	t.Run("Table is completed", func(t *testing.T) {
		checkpoint := &JobCheckpoint{
			JobID:           "job-1",
			ConfigID:        "config-1",
			CompletedTables: []string{"table1", "table2"},
		}
		checkpointData, _ := json.Marshal(checkpoint)
		syncCheckpoint := &SyncCheckpoint{
			ID:             "job_job-1",
			CheckpointData: string(checkpointData),
		}

		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()

		completed, err := cm.IsTableCompleted(ctx, "job-1", "table1")
		assert.NoError(t, err)
		assert.True(t, completed)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Table is not completed", func(t *testing.T) {
		checkpoint := &JobCheckpoint{
			JobID:           "job-1",
			ConfigID:        "config-1",
			CompletedTables: []string{"table1"},
		}
		checkpointData, _ := json.Marshal(checkpoint)
		syncCheckpoint := &SyncCheckpoint{
			ID:             "job_job-1",
			CheckpointData: string(checkpointData),
		}

		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()

		completed, err := cm.IsTableCompleted(ctx, "job-1", "table2")
		assert.NoError(t, err)
		assert.False(t, completed)

		mockRepo.AssertExpectations(t)
	})

	t.Run("No checkpoint exists", func(t *testing.T) {
		mockRepo.On("GetCheckpoint", ctx, "job_job-2").Return((*SyncCheckpoint)(nil), nil).Once()

		completed, err := cm.IsTableCompleted(ctx, "job-2", "table1")
		assert.NoError(t, err)
		assert.False(t, completed)

		mockRepo.AssertExpectations(t)
	})
}

func TestCheckpointManager_UpdateJobProgress(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	mockRepo := new(MockRepository)

	cm := NewCheckpointManager(mockRepo, logger)
	ctx := context.Background()

	checkpoint := &JobCheckpoint{
		JobID:    "job-1",
		ConfigID: "config-1",
		Progress: &Progress{
			TotalTables:     5,
			CompletedTables: 2,
		},
	}
	checkpointData, _ := json.Marshal(checkpoint)
	syncCheckpoint := &SyncCheckpoint{
		ID:             "job_job-1",
		CheckpointData: string(checkpointData),
	}

	newProgress := &Progress{
		TotalTables:     5,
		CompletedTables: 3,
		TotalRows:       1000,
		ProcessedRows:   600,
		Percentage:      60.0,
	}

	mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()
	// Second call when saving the updated checkpoint
	mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()
	mockRepo.On("UpdateCheckpoint", ctx, "job_job-1", mock.AnythingOfType("*sync.SyncCheckpoint")).Return(nil).Once()

	err := cm.UpdateJobProgress(ctx, "job-1", newProgress)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestCheckpointManager_DeleteJobCheckpoint(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	mockRepo := new(MockRepository)

	cm := NewCheckpointManager(mockRepo, logger)
	ctx := context.Background()

	mockRepo.On("UpdateCheckpoint", ctx, "job_job-1", mock.AnythingOfType("*sync.SyncCheckpoint")).Return(nil).Once()

	err := cm.DeleteJobCheckpoint(ctx, "job-1")
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestCheckpointManager_GetResumePoint(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	mockRepo := new(MockRepository)

	cm := NewCheckpointManager(mockRepo, logger)
	ctx := context.Background()

	t.Run("Get resume point when checkpoint exists", func(t *testing.T) {
		checkpoint := &JobCheckpoint{
			JobID:           "job-1",
			ConfigID:        "config-1",
			CompletedTables: []string{"table1"},
			CurrentTable:    "table2",
		}
		checkpointData, _ := json.Marshal(checkpoint)
		syncCheckpoint := &SyncCheckpoint{
			ID:             "job_job-1",
			CheckpointData: string(checkpointData),
		}

		mockRepo.On("GetCheckpoint", ctx, "job_job-1").Return(syncCheckpoint, nil).Once()

		resumePoint, err := cm.GetResumePoint(ctx, "job-1")
		assert.NoError(t, err)
		assert.NotNil(t, resumePoint)
		assert.Equal(t, checkpoint.JobID, resumePoint.JobID)
		assert.Equal(t, checkpoint.CurrentTable, resumePoint.CurrentTable)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Get resume point when checkpoint doesn't exist", func(t *testing.T) {
		mockRepo.On("GetCheckpoint", ctx, "job_job-2").Return((*SyncCheckpoint)(nil), nil).Once()

		resumePoint, err := cm.GetResumePoint(ctx, "job-2")
		assert.Error(t, err)
		assert.Nil(t, resumePoint)
		assert.Contains(t, err.Error(), "no checkpoint found")

		mockRepo.AssertExpectations(t)
	})
}
