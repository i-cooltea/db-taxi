package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// LogNotifier implements ErrorNotifier using logging
type LogNotifier struct {
	logger *logrus.Logger
}

// NewLogNotifier creates a new log-based notifier
func NewLogNotifier(logger *logrus.Logger) ErrorNotifier {
	return &LogNotifier{
		logger: logger,
	}
}

// NotifyError sends a notification about a sync error
func (ln *LogNotifier) NotifyError(ctx context.Context, jobID string, err *SyncError) error {
	ln.logger.WithFields(logrus.Fields{
		"job_id":         jobID,
		"error_type":     err.Type,
		"error_severity": err.Severity,
		"table_name":     err.TableName,
		"timestamp":      err.Timestamp,
		"retryable":      err.Retryable,
	}).Error(fmt.Sprintf("NOTIFICATION: Sync error - %s", err.Message))

	return nil
}

// NotifyJobFailure sends a notification about a complete job failure
func (ln *LogNotifier) NotifyJobFailure(ctx context.Context, jobID string, reason string) error {
	ln.logger.WithFields(logrus.Fields{
		"job_id":    jobID,
		"reason":    reason,
		"timestamp": time.Now(),
	}).Error(fmt.Sprintf("NOTIFICATION: Job failed - %s", reason))

	return nil
}

// NotifyRecovery sends a notification about successful recovery
func (ln *LogNotifier) NotifyRecovery(ctx context.Context, jobID string, message string) error {
	ln.logger.WithFields(logrus.Fields{
		"job_id":    jobID,
		"message":   message,
		"timestamp": time.Now(),
	}).Info(fmt.Sprintf("NOTIFICATION: Job recovered - %s", message))

	return nil
}

// CompositeNotifier combines multiple notifiers
type CompositeNotifier struct {
	notifiers []ErrorNotifier
	logger    *logrus.Logger
}

// NewCompositeNotifier creates a new composite notifier
func NewCompositeNotifier(logger *logrus.Logger, notifiers ...ErrorNotifier) ErrorNotifier {
	return &CompositeNotifier{
		notifiers: notifiers,
		logger:    logger,
	}
}

// NotifyError sends notifications through all notifiers
func (cn *CompositeNotifier) NotifyError(ctx context.Context, jobID string, err *SyncError) error {
	var lastErr error
	for _, notifier := range cn.notifiers {
		if notifyErr := notifier.NotifyError(ctx, jobID, err); notifyErr != nil {
			cn.logger.WithError(notifyErr).Warn("Notifier failed to send error notification")
			lastErr = notifyErr
		}
	}
	return lastErr
}

// NotifyJobFailure sends notifications through all notifiers
func (cn *CompositeNotifier) NotifyJobFailure(ctx context.Context, jobID string, reason string) error {
	var lastErr error
	for _, notifier := range cn.notifiers {
		if notifyErr := notifier.NotifyJobFailure(ctx, jobID, reason); notifyErr != nil {
			cn.logger.WithError(notifyErr).Warn("Notifier failed to send job failure notification")
			lastErr = notifyErr
		}
	}
	return lastErr
}

// NotifyRecovery sends notifications through all notifiers
func (cn *CompositeNotifier) NotifyRecovery(ctx context.Context, jobID string, message string) error {
	var lastErr error
	for _, notifier := range cn.notifiers {
		if notifyErr := notifier.NotifyRecovery(ctx, jobID, message); notifyErr != nil {
			cn.logger.WithError(notifyErr).Warn("Notifier failed to send recovery notification")
			lastErr = notifyErr
		}
	}
	return lastErr
}

// NoOpNotifier is a notifier that does nothing (useful for testing)
type NoOpNotifier struct{}

// NewNoOpNotifier creates a new no-op notifier
func NewNoOpNotifier() ErrorNotifier {
	return &NoOpNotifier{}
}

// NotifyError does nothing
func (n *NoOpNotifier) NotifyError(ctx context.Context, jobID string, err *SyncError) error {
	return nil
}

// NotifyJobFailure does nothing
func (n *NoOpNotifier) NotifyJobFailure(ctx context.Context, jobID string, reason string) error {
	return nil
}

// NotifyRecovery does nothing
func (n *NoOpNotifier) NotifyRecovery(ctx context.Context, jobID string, message string) error {
	return nil
}
