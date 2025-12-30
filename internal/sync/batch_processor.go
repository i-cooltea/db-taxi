package sync

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// BatchProcessor handles optimized batch operations for data synchronization
// Requirement 7.3: Process large tables in batches to avoid long locks
// Requirement 8.1: Use batch operations to improve efficiency when syncing large amounts of data
type BatchProcessor struct {
	localDB       *sqlx.DB
	logger        *logrus.Logger
	batchSize     int
	maxMemoryMB   int64
	workerPool    *WorkerPool
	memoryMonitor *MemoryMonitor
}

// BatchProcessorConfig configures the batch processor
type BatchProcessorConfig struct {
	BatchSize     int   // Number of rows per batch
	MaxMemoryMB   int64 // Maximum memory usage in MB
	MaxWorkers    int   // Maximum concurrent workers
	EnableMetrics bool  // Enable performance metrics
}

// NewBatchProcessor creates a new batch processor with optimized settings
func NewBatchProcessor(localDB *sqlx.DB, logger *logrus.Logger, config *BatchProcessorConfig) *BatchProcessor {
	if config == nil {
		config = &BatchProcessorConfig{
			BatchSize:     1000,
			MaxMemoryMB:   512,
			MaxWorkers:    4,
			EnableMetrics: true,
		}
	}

	// Ensure reasonable defaults
	if config.BatchSize <= 0 {
		config.BatchSize = 1000
	}
	if config.MaxMemoryMB <= 0 {
		config.MaxMemoryMB = 512
	}
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU()
	}

	return &BatchProcessor{
		localDB:       localDB,
		logger:        logger,
		batchSize:     config.BatchSize,
		maxMemoryMB:   config.MaxMemoryMB,
		workerPool:    NewWorkerPool(config.MaxWorkers),
		memoryMonitor: NewMemoryMonitor(config.MaxMemoryMB),
	}
}

// BatchInsertResult contains the result of a batch insert operation
type BatchInsertResult struct {
	TotalRows      int64
	ProcessedRows  int64
	FailedRows     int64
	BatchCount     int
	Duration       time.Duration
	AvgBatchTime   time.Duration
	MemoryPeakMB   int64
	ThroughputRows float64 // Rows per second
}

// ProcessLargeTableSync processes a large table sync with chunking and memory optimization
// Requirement 7.3: Process large tables in batches to avoid long locks
// Requirement 8.1: Use batch operations to improve efficiency
func (bp *BatchProcessor) ProcessLargeTableSync(
	ctx context.Context,
	remoteDB *sqlx.DB,
	localDB string,
	mapping *TableMapping,
	options *SyncOptions,
) (*BatchInsertResult, error) {
	startTime := time.Now()
	result := &BatchInsertResult{}

	bp.logger.WithFields(logrus.Fields{
		"source_table": mapping.SourceTable,
		"target_table": mapping.TargetTable,
		"batch_size":   bp.batchSize,
	}).Info("Starting large table sync with batch processing")

	// Get total row count for progress tracking
	totalRows, err := bp.getRowCount(ctx, remoteDB, mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to get row count: %w", err)
	}
	result.TotalRows = totalRows

	// Determine optimal batch size based on table size and available memory
	optimalBatchSize := bp.calculateOptimalBatchSize(totalRows)
	bp.logger.WithFields(logrus.Fields{
		"total_rows":         totalRows,
		"optimal_batch_size": optimalBatchSize,
	}).Info("Calculated optimal batch size")

	// Get column information
	columns, err := bp.getColumnNames(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	// Process data in chunks
	offset := int64(0)
	batchTimes := []time.Duration{}

	for offset < totalRows {
		// Check memory before processing next batch
		if bp.memoryMonitor.ShouldPause() {
			bp.logger.Warn("Memory usage high, pausing to allow GC")
			runtime.GC()
			time.Sleep(time.Second)
			continue
		}

		batchStart := time.Now()

		// Fetch batch from remote
		batch, err := bp.fetchBatch(ctx, remoteDB, mapping, columns, offset, optimalBatchSize)
		if err != nil {
			return result, fmt.Errorf("failed to fetch batch at offset %d: %w", offset, err)
		}

		if len(batch) == 0 {
			break
		}

		// Insert batch into local database
		if err := bp.insertBatchOptimized(ctx, localDB, mapping.TargetTable, columns, batch); err != nil {
			result.FailedRows += int64(len(batch))
			bp.logger.WithError(err).WithField("offset", offset).Error("Failed to insert batch")
			// Continue with next batch instead of failing completely
		} else {
			result.ProcessedRows += int64(len(batch))
		}

		batchDuration := time.Since(batchStart)
		batchTimes = append(batchTimes, batchDuration)
		result.BatchCount++

		// Update memory peak
		currentMemMB := bp.memoryMonitor.GetCurrentUsageMB()
		if currentMemMB > result.MemoryPeakMB {
			result.MemoryPeakMB = currentMemMB
		}

		// Log progress
		progress := float64(offset+int64(len(batch))) / float64(totalRows) * 100
		bp.logger.WithFields(logrus.Fields{
			"processed_rows": result.ProcessedRows,
			"total_rows":     totalRows,
			"progress":       fmt.Sprintf("%.2f%%", progress),
			"batch_time":     batchDuration,
		}).Debug("Batch processed")

		offset += int64(len(batch))

		// Clear batch to free memory
		batch = nil
	}

	result.Duration = time.Since(startTime)
	if len(batchTimes) > 0 {
		var totalBatchTime time.Duration
		for _, bt := range batchTimes {
			totalBatchTime += bt
		}
		result.AvgBatchTime = totalBatchTime / time.Duration(len(batchTimes))
	}

	if result.Duration.Seconds() > 0 {
		result.ThroughputRows = float64(result.ProcessedRows) / result.Duration.Seconds()
	}

	bp.logger.WithFields(logrus.Fields{
		"total_rows":     result.TotalRows,
		"processed_rows": result.ProcessedRows,
		"failed_rows":    result.FailedRows,
		"batch_count":    result.BatchCount,
		"duration":       result.Duration,
		"avg_batch_time": result.AvgBatchTime,
		"throughput":     fmt.Sprintf("%.2f rows/sec", result.ThroughputRows),
		"memory_peak_mb": result.MemoryPeakMB,
	}).Info("Large table sync completed")

	return result, nil
}

// calculateOptimalBatchSize determines the optimal batch size based on table size and memory
func (bp *BatchProcessor) calculateOptimalBatchSize(totalRows int64) int {
	// Start with configured batch size
	batchSize := bp.batchSize

	// For very large tables, use smaller batches to avoid memory issues
	if totalRows > 10_000_000 {
		batchSize = min(batchSize, 500)
	} else if totalRows > 1_000_000 {
		batchSize = min(batchSize, 1000)
	} else if totalRows > 100_000 {
		batchSize = min(batchSize, 2000)
	}

	// Adjust based on available memory
	availableMemMB := bp.memoryMonitor.GetAvailableMemoryMB()
	if availableMemMB < 100 {
		batchSize = min(batchSize, 200)
	} else if availableMemMB < 256 {
		batchSize = min(batchSize, 500)
	}

	return batchSize
}

// getRowCount gets the total row count for a table
func (bp *BatchProcessor) getRowCount(ctx context.Context, remoteDB *sqlx.DB, mapping *TableMapping) (int64, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", mapping.SourceTable)
	if mapping.WhereClause != "" {
		countQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}

	var count int64
	if err := remoteDB.GetContext(ctx, &count, countQuery); err != nil {
		return 0, err
	}

	return count, nil
}

// getColumnNames retrieves column names for a table
func (bp *BatchProcessor) getColumnNames(ctx context.Context, remoteDB *sqlx.DB, tableName string) ([]string, error) {
	query := `
		SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	var columns []string
	if err := remoteDB.SelectContext(ctx, &columns, query, tableName); err != nil {
		return nil, err
	}

	return columns, nil
}

// fetchBatch fetches a batch of rows from the remote database
func (bp *BatchProcessor) fetchBatch(
	ctx context.Context,
	remoteDB *sqlx.DB,
	mapping *TableMapping,
	columns []string,
	offset int64,
	limit int,
) ([]map[string]interface{}, error) {
	selectQuery := fmt.Sprintf("SELECT * FROM `%s`", mapping.SourceTable)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" WHERE %s", mapping.WhereClause)
	}
	selectQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := remoteDB.QueryxContext(ctx, selectQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []map[string]interface{}
	for rows.Next() {
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return nil, err
		}
		batch = append(batch, rowData)
	}

	return batch, nil
}

// insertBatchOptimized performs an optimized batch insert
// Requirement 8.1: Use batch operations to improve efficiency
func (bp *BatchProcessor) insertBatchOptimized(
	ctx context.Context,
	localDB string,
	tableName string,
	columns []string,
	batch []map[string]interface{},
) error {
	if len(batch) == 0 {
		return nil
	}

	// Build optimized INSERT statement with multi-row VALUES
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` (", localDB, tableName))

	// Add column names
	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("`%s`", col))
	}

	sb.WriteString(") VALUES ")

	// Add value placeholders
	args := make([]interface{}, 0, len(batch)*len(columns))
	for i := range batch {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j, col := range columns {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("?")
			args = append(args, batch[i][col])
		}
		sb.WriteString(")")
	}

	// Execute with timeout to prevent long locks
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if _, err := bp.localDB.ExecContext(execCtx, sb.String(), args...); err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}

	return nil
}

// ProcessIncrementalBatch processes incremental sync with batch optimization
func (bp *BatchProcessor) ProcessIncrementalBatch(
	ctx context.Context,
	remoteDB *sqlx.DB,
	localDB string,
	mapping *TableMapping,
	primaryKeys []string,
	changeColumn string,
	lastSyncValue interface{},
	options *SyncOptions,
) (*BatchInsertResult, error) {
	startTime := time.Now()
	result := &BatchInsertResult{}

	bp.logger.WithFields(logrus.Fields{
		"source_table":  mapping.SourceTable,
		"change_column": changeColumn,
		"last_sync":     lastSyncValue,
	}).Info("Starting incremental batch processing")

	// Get columns
	columns, err := bp.getColumnNames(ctx, remoteDB, mapping.SourceTable)
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	// Build query for changed records
	selectQuery := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` > ?", mapping.SourceTable, changeColumn)
	if mapping.WhereClause != "" {
		selectQuery += fmt.Sprintf(" AND (%s)", mapping.WhereClause)
	}
	selectQuery += fmt.Sprintf(" ORDER BY `%s`", changeColumn)

	rows, err := remoteDB.QueryxContext(ctx, selectQuery, lastSyncValue)
	if err != nil {
		return nil, fmt.Errorf("failed to query changed data: %w", err)
	}
	defer rows.Close()

	// Process in batches
	batch := make([]map[string]interface{}, 0, bp.batchSize)
	batchTimes := []time.Duration{}

	for rows.Next() {
		// Check memory
		if bp.memoryMonitor.ShouldPause() {
			bp.logger.Warn("Memory usage high, pausing")
			runtime.GC()
			time.Sleep(time.Second)
		}

		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			return result, fmt.Errorf("failed to scan row: %w", err)
		}

		batch = append(batch, rowData)

		// Process batch when it reaches batch size
		if len(batch) >= bp.batchSize {
			batchStart := time.Now()

			if err := bp.upsertBatchOptimized(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
				result.FailedRows += int64(len(batch))
				bp.logger.WithError(err).Error("Failed to upsert batch")
			} else {
				result.ProcessedRows += int64(len(batch))
			}

			batchDuration := time.Since(batchStart)
			batchTimes = append(batchTimes, batchDuration)
			result.BatchCount++

			// Clear batch
			batch = batch[:0]

			// Update memory peak
			currentMemMB := bp.memoryMonitor.GetCurrentUsageMB()
			if currentMemMB > result.MemoryPeakMB {
				result.MemoryPeakMB = currentMemMB
			}
		}
	}

	// Process remaining rows
	if len(batch) > 0 {
		batchStart := time.Now()

		if err := bp.upsertBatchOptimized(ctx, localDB, mapping.TargetTable, columns, primaryKeys, batch, options); err != nil {
			result.FailedRows += int64(len(batch))
			bp.logger.WithError(err).Error("Failed to upsert final batch")
		} else {
			result.ProcessedRows += int64(len(batch))
		}

		batchDuration := time.Since(batchStart)
		batchTimes = append(batchTimes, batchDuration)
		result.BatchCount++
	}

	result.Duration = time.Since(startTime)
	if len(batchTimes) > 0 {
		var totalBatchTime time.Duration
		for _, bt := range batchTimes {
			totalBatchTime += bt
		}
		result.AvgBatchTime = totalBatchTime / time.Duration(len(batchTimes))
	}

	if result.Duration.Seconds() > 0 {
		result.ThroughputRows = float64(result.ProcessedRows) / result.Duration.Seconds()
	}

	bp.logger.WithFields(logrus.Fields{
		"processed_rows": result.ProcessedRows,
		"failed_rows":    result.FailedRows,
		"batch_count":    result.BatchCount,
		"duration":       result.Duration,
		"throughput":     fmt.Sprintf("%.2f rows/sec", result.ThroughputRows),
	}).Info("Incremental batch processing completed")

	return result, nil
}

// upsertBatchOptimized performs an optimized batch upsert
func (bp *BatchProcessor) upsertBatchOptimized(
	ctx context.Context,
	localDB string,
	tableName string,
	columns []string,
	primaryKeys []string,
	batch []map[string]interface{},
	options *SyncOptions,
) error {
	if len(batch) == 0 {
		return nil
	}

	conflictResolution := ConflictResolutionOverwrite
	if options != nil {
		conflictResolution = options.ConflictResolution
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` (", localDB, tableName))

	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("`%s`", col))
	}

	sb.WriteString(") VALUES ")

	args := make([]interface{}, 0, len(batch)*len(columns))
	for i := range batch {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j, col := range columns {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("?")
			args = append(args, batch[i][col])
		}
		sb.WriteString(")")
	}

	// Add conflict resolution
	switch conflictResolution {
	case ConflictResolutionOverwrite:
		sb.WriteString(" ON DUPLICATE KEY UPDATE ")
		first := true
		for _, col := range columns {
			isPrimaryKey := false
			for _, pk := range primaryKeys {
				if col == pk {
					isPrimaryKey = true
					break
				}
			}
			if isPrimaryKey {
				continue
			}

			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("`%s` = VALUES(`%s`)", col, col))
			first = false
		}
	case ConflictResolutionSkip:
		sb.WriteString(" ON DUPLICATE KEY UPDATE ")
		sb.WriteString(fmt.Sprintf("`%s` = `%s`", primaryKeys[0], primaryKeys[0]))
	}

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if _, err := bp.localDB.ExecContext(execCtx, sb.String(), args...); err != nil {
		return fmt.Errorf("failed to execute batch upsert: %w", err)
	}

	return nil
}

// WorkerPool manages concurrent batch processing workers
type WorkerPool struct {
	maxWorkers int
	semaphore  chan struct{}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
	}
}

// Acquire acquires a worker slot
func (wp *WorkerPool) Acquire() {
	wp.semaphore <- struct{}{}
}

// Release releases a worker slot
func (wp *WorkerPool) Release() {
	<-wp.semaphore
}

// MemoryMonitor monitors memory usage and provides optimization hints
type MemoryMonitor struct {
	maxMemoryMB int64
	mu          sync.RWMutex
	lastGC      time.Time
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(maxMemoryMB int64) *MemoryMonitor {
	return &MemoryMonitor{
		maxMemoryMB: maxMemoryMB,
		lastGC:      time.Now(),
	}
}

// GetCurrentUsageMB returns current memory usage in MB
func (mm *MemoryMonitor) GetCurrentUsageMB() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc / 1024 / 1024)
}

// GetAvailableMemoryMB returns available memory in MB
func (mm *MemoryMonitor) GetAvailableMemoryMB() int64 {
	currentUsage := mm.GetCurrentUsageMB()
	return mm.maxMemoryMB - currentUsage
}

// ShouldPause returns true if memory usage is too high
func (mm *MemoryMonitor) ShouldPause() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	currentUsage := mm.GetCurrentUsageMB()
	threshold := mm.maxMemoryMB * 80 / 100 // 80% threshold

	// Also check if we haven't GC'd recently
	if currentUsage > threshold && time.Since(mm.lastGC) > 5*time.Second {
		mm.mu.RUnlock()
		mm.mu.Lock()
		mm.lastGC = time.Now()
		mm.mu.Unlock()
		mm.mu.RLock()
		return true
	}

	return false
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
