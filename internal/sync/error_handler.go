package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrorType categorizes different types of errors
type ErrorType string

const (
	ErrorTypeConnection         ErrorType = "connection"
	ErrorTypeAuthentication     ErrorType = "authentication"
	ErrorTypeTimeout            ErrorType = "timeout"
	ErrorTypeDataSync           ErrorType = "data_sync"
	ErrorTypeSchemaConflict     ErrorType = "schema_conflict"
	ErrorTypeDataConversion     ErrorType = "data_conversion"
	ErrorTypePrimaryKeyConflict ErrorType = "primary_key_conflict"
	ErrorTypeSystemResource     ErrorType = "system_resource"
	ErrorTypeDiskSpace          ErrorType = "disk_space"
	ErrorTypeLockTimeout        ErrorType = "lock_timeout"
	ErrorTypeUnknown            ErrorType = "unknown"
)

// ErrorSeverity indicates the severity of an error
type ErrorSeverity string

const (
	ErrorSeverityCritical ErrorSeverity = "critical" // Stop immediately
	ErrorSeverityHigh     ErrorSeverity = "high"     // Retry with caution
	ErrorSeverityMedium   ErrorSeverity = "medium"   // Retry normally
	ErrorSeverityLow      ErrorSeverity = "low"      // Log and continue
)

// SyncError represents a categorized synchronization error
type SyncError struct {
	Type      ErrorType
	Severity  ErrorSeverity
	Message   string
	Original  error
	Retryable bool
	TableName string
	Timestamp time.Time
}

// Error implements the error interface
func (e *SyncError) Error() string {
	if e.TableName != "" {
		return fmt.Sprintf("[%s][%s] %s (table: %s): %v", e.Type, e.Severity, e.Message, e.TableName, e.Original)
	}
	return fmt.Sprintf("[%s][%s] %s: %v", e.Type, e.Severity, e.Message, e.Original)
}

// Unwrap returns the original error
func (e *SyncError) Unwrap() error {
	return e.Original
}

// RetryPolicy defines retry behavior for failed operations
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// DefaultRetryPolicy returns the default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// ErrorHandler handles and categorizes synchronization errors
type ErrorHandler struct {
	logger      *logrus.Logger
	monitoring  MonitoringService
	retryPolicy *RetryPolicy
	notifier    ErrorNotifier
}

// ErrorNotifier defines the interface for error notifications
type ErrorNotifier interface {
	// NotifyError sends a notification about a sync error
	NotifyError(ctx context.Context, jobID string, err *SyncError) error

	// NotifyJobFailure sends a notification about a complete job failure
	NotifyJobFailure(ctx context.Context, jobID string, reason string) error

	// NotifyRecovery sends a notification about successful recovery
	NotifyRecovery(ctx context.Context, jobID string, message string) error
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *logrus.Logger, monitoring MonitoringService, notifier ErrorNotifier) *ErrorHandler {
	return &ErrorHandler{
		logger:      logger,
		monitoring:  monitoring,
		retryPolicy: DefaultRetryPolicy(),
		notifier:    notifier,
	}
}

// SetRetryPolicy updates the retry policy
func (eh *ErrorHandler) SetRetryPolicy(policy *RetryPolicy) {
	eh.retryPolicy = policy
}

// ClassifyError categorizes an error and determines handling strategy
func (eh *ErrorHandler) ClassifyError(err error, tableName string) *SyncError {
	if err == nil {
		return nil
	}

	syncErr := &SyncError{
		Original:  err,
		TableName: tableName,
		Timestamp: time.Now(),
	}

	errMsg := strings.ToLower(err.Error())

	// Lock timeout errors (check before general timeout)
	if strings.Contains(errMsg, "lock wait timeout") ||
		strings.Contains(errMsg, "deadlock") {
		syncErr.Type = ErrorTypeLockTimeout
		syncErr.Severity = ErrorSeverityMedium
		syncErr.Message = "Database lock timeout"
		syncErr.Retryable = true
		return syncErr
	}

	// Connection errors
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "network unreachable") {
		syncErr.Type = ErrorTypeConnection
		syncErr.Severity = ErrorSeverityHigh
		syncErr.Message = "Network connection failed"
		syncErr.Retryable = true
		return syncErr
	}

	// Authentication errors
	if strings.Contains(errMsg, "access denied") ||
		strings.Contains(errMsg, "authentication failed") ||
		strings.Contains(errMsg, "invalid credentials") {
		syncErr.Type = ErrorTypeAuthentication
		syncErr.Severity = ErrorSeverityCritical
		syncErr.Message = "Authentication failed"
		syncErr.Retryable = false
		return syncErr
	}

	// Timeout errors
	if strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "context deadline") {
		syncErr.Type = ErrorTypeTimeout
		syncErr.Severity = ErrorSeverityMedium
		syncErr.Message = "Operation timed out"
		syncErr.Retryable = true
		return syncErr
	}

	// Schema conflict errors
	if strings.Contains(errMsg, "table doesn't exist") ||
		strings.Contains(errMsg, "unknown column") ||
		strings.Contains(errMsg, "column count doesn't match") {
		syncErr.Type = ErrorTypeSchemaConflict
		syncErr.Severity = ErrorSeverityHigh
		syncErr.Message = "Table structure mismatch"
		syncErr.Retryable = false
		return syncErr
	}

	// Data conversion errors
	if strings.Contains(errMsg, "data too long") ||
		strings.Contains(errMsg, "incorrect") ||
		strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "truncated") {
		syncErr.Type = ErrorTypeDataConversion
		syncErr.Severity = ErrorSeverityMedium
		syncErr.Message = "Data type conversion error"
		syncErr.Retryable = false
		return syncErr
	}

	// Primary key conflict errors
	if strings.Contains(errMsg, "duplicate entry") ||
		strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "primary key") {
		syncErr.Type = ErrorTypePrimaryKeyConflict
		syncErr.Severity = ErrorSeverityLow
		syncErr.Message = "Primary key or unique constraint violation"
		syncErr.Retryable = false
		return syncErr
	}

	// Disk space errors
	if strings.Contains(errMsg, "no space left") ||
		strings.Contains(errMsg, "disk full") {
		syncErr.Type = ErrorTypeDiskSpace
		syncErr.Severity = ErrorSeverityCritical
		syncErr.Message = "Insufficient disk space"
		syncErr.Retryable = false
		return syncErr
	}

	// Memory errors
	if strings.Contains(errMsg, "out of memory") ||
		strings.Contains(errMsg, "cannot allocate") {
		syncErr.Type = ErrorTypeSystemResource
		syncErr.Severity = ErrorSeverityCritical
		syncErr.Message = "Insufficient memory"
		syncErr.Retryable = false
		return syncErr
	}

	// Unknown error
	syncErr.Type = ErrorTypeUnknown
	syncErr.Severity = ErrorSeverityMedium
	syncErr.Message = "Unknown error occurred"
	syncErr.Retryable = true

	return syncErr
}

// HandleConnectionError handles connection-related errors
func (eh *ErrorHandler) HandleConnectionError(ctx context.Context, err error, config *ConnectionConfig) error {
	syncErr := eh.ClassifyError(err, "")

	eh.logger.WithFields(logrus.Fields{
		"error_type":     syncErr.Type,
		"error_severity": syncErr.Severity,
		"connection_id":  config.ID,
		"connection":     config.Name,
	}).Error("Connection error occurred")

	// For authentication errors, fail immediately
	if syncErr.Type == ErrorTypeAuthentication {
		return fmt.Errorf("authentication failed for connection %s: %w", config.Name, err)
	}

	// For connection errors, apply retry policy
	if syncErr.Type == ErrorTypeConnection || syncErr.Type == ErrorTypeTimeout {
		return eh.retryWithBackoff(ctx, func() error {
			// The actual retry logic should be implemented by the caller
			return err
		})
	}

	return err
}

// HandleSyncError handles data synchronization errors
func (eh *ErrorHandler) HandleSyncError(ctx context.Context, err error, job *SyncJob, tableName string) error {
	syncErr := eh.ClassifyError(err, tableName)

	eh.logger.WithFields(logrus.Fields{
		"error_type":     syncErr.Type,
		"error_severity": syncErr.Severity,
		"job_id":         job.ID,
		"table_name":     tableName,
	}).Error("Sync error occurred")

	// Log the error
	if eh.monitoring != nil {
		logLevel := "error"
		if syncErr.Severity == ErrorSeverityLow {
			logLevel = "warn"
		}

		if logErr := eh.monitoring.LogJobEvent(ctx, job.ID, tableName, logLevel, syncErr.Error()); logErr != nil {
			eh.logger.WithError(logErr).Warn("Failed to log sync error")
		}
	}

	// Send notification for critical errors
	if syncErr.Severity == ErrorSeverityCritical && eh.notifier != nil {
		if notifyErr := eh.notifier.NotifyError(ctx, job.ID, syncErr); notifyErr != nil {
			eh.logger.WithError(notifyErr).Warn("Failed to send error notification")
		}
	}

	// Determine if error should stop the job
	if syncErr.Severity == ErrorSeverityCritical {
		return fmt.Errorf("critical error, stopping job: %w", err)
	}

	// For retryable errors, return the error to trigger retry
	if syncErr.Retryable {
		return err
	}

	// For non-retryable errors, log and continue
	eh.logger.WithFields(logrus.Fields{
		"job_id":     job.ID,
		"table_name": tableName,
	}).Warn("Non-retryable error, skipping table")

	return nil
}

// HandleSystemError handles system-level errors
func (eh *ErrorHandler) HandleSystemError(ctx context.Context, err error) error {
	syncErr := eh.ClassifyError(err, "")

	eh.logger.WithFields(logrus.Fields{
		"error_type":     syncErr.Type,
		"error_severity": syncErr.Severity,
	}).Error("System error occurred")

	// For critical system errors, fail immediately
	if syncErr.Severity == ErrorSeverityCritical {
		return fmt.Errorf("critical system error: %w", err)
	}

	return err
}

// retryWithBackoff executes a function with exponential backoff retry
func (eh *ErrorHandler) retryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	delay := eh.retryPolicy.InitialDelay

	for attempt := 0; attempt <= eh.retryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			eh.logger.WithFields(logrus.Fields{
				"attempt": attempt,
				"delay":   delay,
			}).Info("Retrying operation")

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return ctx.Err()
			}

			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * eh.retryPolicy.BackoffFactor)
			if delay > eh.retryPolicy.MaxDelay {
				delay = eh.retryPolicy.MaxDelay
			}
		}

		lastErr = fn()
		if lastErr == nil {
			if attempt > 0 {
				eh.logger.WithField("attempts", attempt+1).Info("Operation succeeded after retry")
			}
			return nil
		}

		// Check if error is retryable
		syncErr := eh.ClassifyError(lastErr, "")
		if !syncErr.Retryable {
			return lastErr
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", eh.retryPolicy.MaxRetries+1, lastErr)
}

// RetryOperation retries an operation with the configured retry policy
func (eh *ErrorHandler) RetryOperation(ctx context.Context, operation func() error, tableName string) error {
	return eh.retryWithBackoff(ctx, operation)
}

// ShouldStopJob determines if a job should be stopped based on error
func (eh *ErrorHandler) ShouldStopJob(err error) bool {
	if err == nil {
		return false
	}

	syncErr := eh.ClassifyError(err, "")
	return syncErr.Severity == ErrorSeverityCritical
}

// GetErrorSuggestion provides a suggestion for resolving an error
func (eh *ErrorHandler) GetErrorSuggestion(err error) string {
	if err == nil {
		return ""
	}

	syncErr := eh.ClassifyError(err, "")

	switch syncErr.Type {
	case ErrorTypeConnection:
		return "Check network connectivity and ensure the remote database is accessible. Verify firewall rules and network configuration."
	case ErrorTypeAuthentication:
		return "Verify database credentials (username and password). Ensure the user has necessary permissions."
	case ErrorTypeTimeout:
		return "Increase timeout settings or check database performance. Consider reducing batch size for large operations."
	case ErrorTypeSchemaConflict:
		return "Verify table structure matches between source and target. Run schema synchronization before data sync."
	case ErrorTypeDataConversion:
		return "Check data types compatibility between source and target tables. Review data for invalid values."
	case ErrorTypePrimaryKeyConflict:
		return "Configure conflict resolution strategy (skip, overwrite, or error). Consider using incremental sync mode."
	case ErrorTypeDiskSpace:
		return "Free up disk space on the target server. Consider archiving old data or expanding storage."
	case ErrorTypeLockTimeout:
		return "Reduce transaction size or increase lock timeout. Consider running sync during off-peak hours."
	case ErrorTypeSystemResource:
		return "Reduce batch size and concurrent operations. Monitor system resources and consider upgrading hardware."
	default:
		return "Review error logs for more details. Contact support if the issue persists."
	}
}

// LogErrorWithSuggestion logs an error with a helpful suggestion
func (eh *ErrorHandler) LogErrorWithSuggestion(ctx context.Context, jobID string, err error, tableName string) {
	syncErr := eh.ClassifyError(err, tableName)
	suggestion := eh.GetErrorSuggestion(err)

	message := fmt.Sprintf("%s. Suggestion: %s", syncErr.Error(), suggestion)

	if eh.monitoring != nil {
		if logErr := eh.monitoring.LogJobEvent(ctx, jobID, tableName, "error", message); logErr != nil {
			eh.logger.WithError(logErr).Warn("Failed to log error with suggestion")
		}
	}

	eh.logger.WithFields(logrus.Fields{
		"job_id":     jobID,
		"table_name": tableName,
		"error_type": syncErr.Type,
		"suggestion": suggestion,
	}).Error(message)
}
