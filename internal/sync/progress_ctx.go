package sync

import "context"

type progressContextKey struct{}

// TableProgressReporter is called during table sync to report progress (processed/total rows).
// It is injected via context by the job engine and called by the sync engine and batch processor.
type TableProgressReporter func(tableName string, status TableSyncStatus, processedRows, totalRows int64)

// WithTableProgressReporter returns a context that carries the given reporter.
func WithTableProgressReporter(ctx context.Context, reporter TableProgressReporter) context.Context {
	return context.WithValue(ctx, progressContextKey{}, reporter)
}

// ReportTableProgress calls the reporter from ctx if present; no-op otherwise.
func ReportTableProgress(ctx context.Context, tableName string, status TableSyncStatus, processedRows, totalRows int64) {
	if r, ok := ctx.Value(progressContextKey{}).(TableProgressReporter); ok && r != nil {
		r(tableName, status, processedRows, totalRows)
	}
}
