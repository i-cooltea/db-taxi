package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockErrorNotifier is a mock implementation of ErrorNotifier
type MockErrorNotifier struct {
	mock.Mock
}

func (m *MockErrorNotifier) NotifyError(ctx context.Context, jobID string, err *SyncError) error {
	args := m.Called(ctx, jobID, err)
	return args.Error(0)
}

func (m *MockErrorNotifier) NotifyJobFailure(ctx context.Context, jobID string, reason string) error {
	args := m.Called(ctx, jobID, reason)
	return args.Error(0)
}

func (m *MockErrorNotifier) NotifyRecovery(ctx context.Context, jobID string, message string) error {
	args := m.Called(ctx, jobID, message)
	return args.Error(0)
}

func TestErrorHandler_ClassifyError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	notifier := NewNoOpNotifier()
	handler := NewErrorHandler(logger, nil, notifier)

	tests := []struct {
		name              string
		err               error
		tableName         string
		expectedType      ErrorType
		expectedSeverity  ErrorSeverity
		expectedRetryable bool
	}{
		{
			name:              "Connection refused error",
			err:               errors.New("connection refused"),
			tableName:         "test_table",
			expectedType:      ErrorTypeConnection,
			expectedSeverity:  ErrorSeverityHigh,
			expectedRetryable: true,
		},
		{
			name:              "Authentication error",
			err:               errors.New("access denied for user"),
			tableName:         "test_table",
			expectedType:      ErrorTypeAuthentication,
			expectedSeverity:  ErrorSeverityCritical,
			expectedRetryable: false,
		},
		{
			name:              "Timeout error",
			err:               errors.New("context deadline exceeded"),
			tableName:         "test_table",
			expectedType:      ErrorTypeTimeout,
			expectedSeverity:  ErrorSeverityMedium,
			expectedRetryable: true,
		},
		{
			name:              "Schema conflict error",
			err:               errors.New("table doesn't exist"),
			tableName:         "test_table",
			expectedType:      ErrorTypeSchemaConflict,
			expectedSeverity:  ErrorSeverityHigh,
			expectedRetryable: false,
		},
		{
			name:              "Primary key conflict",
			err:               errors.New("duplicate entry for key 'PRIMARY'"),
			tableName:         "test_table",
			expectedType:      ErrorTypePrimaryKeyConflict,
			expectedSeverity:  ErrorSeverityLow,
			expectedRetryable: false,
		},
		{
			name:              "Disk space error",
			err:               errors.New("no space left on device"),
			tableName:         "test_table",
			expectedType:      ErrorTypeDiskSpace,
			expectedSeverity:  ErrorSeverityCritical,
			expectedRetryable: false,
		},
		{
			name:              "Lock timeout error",
			err:               errors.New("lock wait timeout exceeded"),
			tableName:         "test_table",
			expectedType:      ErrorTypeLockTimeout,
			expectedSeverity:  ErrorSeverityMedium,
			expectedRetryable: true,
		},
		{
			name:              "Unknown error",
			err:               errors.New("some random error"),
			tableName:         "test_table",
			expectedType:      ErrorTypeUnknown,
			expectedSeverity:  ErrorSeverityMedium,
			expectedRetryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncErr := handler.ClassifyError(tt.err, tt.tableName)

			assert.NotNil(t, syncErr)
			assert.Equal(t, tt.expectedType, syncErr.Type)
			assert.Equal(t, tt.expectedSeverity, syncErr.Severity)
			assert.Equal(t, tt.expectedRetryable, syncErr.Retryable)
			assert.Equal(t, tt.tableName, syncErr.TableName)
			assert.Equal(t, tt.err, syncErr.Original)
		})
	}
}

func TestErrorHandler_ShouldStopJob(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	notifier := NewNoOpNotifier()
	handler := NewErrorHandler(logger, nil, notifier)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Critical error should stop job",
			err:      errors.New("access denied"),
			expected: true,
		},
		{
			name:     "Non-critical error should not stop job",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "Nil error should not stop job",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ShouldStopJob(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorHandler_GetErrorSuggestion(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	notifier := NewNoOpNotifier()
	handler := NewErrorHandler(logger, nil, notifier)

	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "Connection error suggestion",
			err:      errors.New("connection refused"),
			contains: "network connectivity",
		},
		{
			name:     "Authentication error suggestion",
			err:      errors.New("access denied"),
			contains: "credentials",
		},
		{
			name:     "Timeout error suggestion",
			err:      errors.New("timeout"),
			contains: "timeout settings",
		},
		{
			name:     "Schema conflict suggestion",
			err:      errors.New("table doesn't exist"),
			contains: "table structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := handler.GetErrorSuggestion(tt.err)
			assert.Contains(t, suggestion, tt.contains)
		})
	}
}

func TestErrorHandler_RetryOperation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	notifier := NewNoOpNotifier()
	handler := NewErrorHandler(logger, nil, notifier)

	// Set a fast retry policy for testing
	handler.SetRetryPolicy(&RetryPolicy{
		MaxRetries:    2,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      50 * time.Millisecond,
		BackoffFactor: 2.0,
	})

	t.Run("Successful operation on first try", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		operation := func() error {
			attempts++
			return nil
		}

		err := handler.RetryOperation(ctx, operation, "test_table")
		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("Successful operation after retry", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		operation := func() error {
			attempts++
			if attempts < 2 {
				return errors.New("connection refused")
			}
			return nil
		}

		err := handler.RetryOperation(ctx, operation, "test_table")
		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("Non-retryable error fails immediately", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		operation := func() error {
			attempts++
			return errors.New("access denied")
		}

		err := handler.RetryOperation(ctx, operation, "test_table")
		assert.Error(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("Max retries exceeded", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		operation := func() error {
			attempts++
			return errors.New("connection refused")
		}

		err := handler.RetryOperation(ctx, operation, "test_table")
		assert.Error(t, err)
		assert.Equal(t, 3, attempts) // Initial + 2 retries
	})

	t.Run("Context cancellation stops retry", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		operation := func() error {
			attempts++
			if attempts == 1 {
				cancel()
			}
			return errors.New("connection refused")
		}

		err := handler.RetryOperation(ctx, operation, "test_table")
		assert.Error(t, err)
		assert.True(t, attempts <= 2)
	})
}

func TestErrorHandler_HandleSyncError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockNotifier := new(MockErrorNotifier)
	mockMonitoring := new(MockMonitoringService)

	handler := NewErrorHandler(logger, mockMonitoring, mockNotifier)

	job := &SyncJob{
		ID:       "test-job-1",
		ConfigID: "test-config-1",
		Status:   JobStatusRunning,
	}

	t.Run("Critical error triggers notification", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("access denied")

		mockMonitoring.On("LogJobEvent", ctx, job.ID, "test_table", "error", mock.Anything).Return(nil)
		mockNotifier.On("NotifyError", ctx, job.ID, mock.AnythingOfType("*sync.SyncError")).Return(nil)

		result := handler.HandleSyncError(ctx, err, job, "test_table")
		assert.Error(t, result)

		mockNotifier.AssertCalled(t, "NotifyError", ctx, job.ID, mock.AnythingOfType("*sync.SyncError"))
	})

	t.Run("Non-critical retryable error returns error", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("connection refused")

		mockMonitoring.On("LogJobEvent", ctx, job.ID, "test_table", "error", mock.Anything).Return(nil)

		result := handler.HandleSyncError(ctx, err, job, "test_table")
		assert.Error(t, result)
	})

	t.Run("Non-critical non-retryable error returns nil", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("duplicate entry")

		mockMonitoring.On("LogJobEvent", ctx, job.ID, "test_table", "warn", mock.Anything).Return(nil)

		result := handler.HandleSyncError(ctx, err, job, "test_table")
		assert.NoError(t, result)
	})
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	assert.Equal(t, 3, policy.MaxRetries)
	assert.Equal(t, 1*time.Second, policy.InitialDelay)
	assert.Equal(t, 30*time.Second, policy.MaxDelay)
	assert.Equal(t, 2.0, policy.BackoffFactor)
}

func TestSyncError_Error(t *testing.T) {
	originalErr := errors.New("original error")

	t.Run("Error with table name", func(t *testing.T) {
		syncErr := &SyncError{
			Type:      ErrorTypeConnection,
			Severity:  ErrorSeverityHigh,
			Message:   "Connection failed",
			Original:  originalErr,
			TableName: "test_table",
		}

		errMsg := syncErr.Error()
		assert.Contains(t, errMsg, "connection")
		assert.Contains(t, errMsg, "high")
		assert.Contains(t, errMsg, "Connection failed")
		assert.Contains(t, errMsg, "test_table")
	})

	t.Run("Error without table name", func(t *testing.T) {
		syncErr := &SyncError{
			Type:     ErrorTypeConnection,
			Severity: ErrorSeverityHigh,
			Message:  "Connection failed",
			Original: originalErr,
		}

		errMsg := syncErr.Error()
		assert.Contains(t, errMsg, "connection")
		assert.Contains(t, errMsg, "high")
		assert.Contains(t, errMsg, "Connection failed")
		assert.NotContains(t, errMsg, "table:")
	})
}

func TestLogNotifier(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	notifier := NewLogNotifier(logger)

	ctx := context.Background()
	syncErr := &SyncError{
		Type:      ErrorTypeConnection,
		Severity:  ErrorSeverityHigh,
		Message:   "Test error",
		TableName: "test_table",
	}

	t.Run("NotifyError", func(t *testing.T) {
		err := notifier.NotifyError(ctx, "job-1", syncErr)
		assert.NoError(t, err)
	})

	t.Run("NotifyJobFailure", func(t *testing.T) {
		err := notifier.NotifyJobFailure(ctx, "job-1", "Test failure")
		assert.NoError(t, err)
	})

	t.Run("NotifyRecovery", func(t *testing.T) {
		err := notifier.NotifyRecovery(ctx, "job-1", "Test recovery")
		assert.NoError(t, err)
	})
}

func TestNoOpNotifier(t *testing.T) {
	notifier := NewNoOpNotifier()
	ctx := context.Background()
	syncErr := &SyncError{
		Type:    ErrorTypeConnection,
		Message: "Test error",
	}

	t.Run("NotifyError does nothing", func(t *testing.T) {
		err := notifier.NotifyError(ctx, "job-1", syncErr)
		assert.NoError(t, err)
	})

	t.Run("NotifyJobFailure does nothing", func(t *testing.T) {
		err := notifier.NotifyJobFailure(ctx, "job-1", "Test failure")
		assert.NoError(t, err)
	})

	t.Run("NotifyRecovery does nothing", func(t *testing.T) {
		err := notifier.NotifyRecovery(ctx, "job-1", "Test recovery")
		assert.NoError(t, err)
	})
}
