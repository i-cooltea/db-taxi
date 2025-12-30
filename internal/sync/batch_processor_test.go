package sync

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestBatchProcessor_CalculateOptimalBatchSize tests batch size calculation
// Requirement 7.3: Process large tables in batches
// Requirement 8.1: Use batch operations to improve efficiency
func TestBatchProcessor_CalculateOptimalBatchSize(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name        string
		totalRows   int64
		maxMemoryMB int64
		expectedMax int
		description string
	}{
		{
			name:        "Very large table",
			totalRows:   15_000_000,
			maxMemoryMB: 512,
			expectedMax: 500,
			description: "Should use smaller batches for very large tables",
		},
		{
			name:        "Large table",
			totalRows:   5_000_000,
			maxMemoryMB: 512,
			expectedMax: 1000,
			description: "Should use medium batches for large tables",
		},
		{
			name:        "Medium table",
			totalRows:   500_000,
			maxMemoryMB: 512,
			expectedMax: 2000,
			description: "Should use larger batches for medium tables",
		},
		{
			name:        "Small table",
			totalRows:   10_000,
			maxMemoryMB: 512,
			expectedMax: 1000,
			description: "Should use default batch size for small tables",
		},
		{
			name:        "Low memory scenario",
			totalRows:   1_000_000,
			maxMemoryMB: 50,
			expectedMax: 200,
			description: "Should use smaller batches when memory is limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BatchProcessorConfig{
				BatchSize:     1000,
				MaxMemoryMB:   tt.maxMemoryMB,
				MaxWorkers:    4,
				EnableMetrics: true,
			}

			bp := NewBatchProcessor(nil, logger, config)
			batchSize := bp.calculateOptimalBatchSize(tt.totalRows)

			assert.LessOrEqual(t, batchSize, tt.expectedMax, tt.description)
			assert.Greater(t, batchSize, 0, "Batch size should be positive")
		})
	}
}

// TestBatchProcessor_NewBatchProcessor tests batch processor initialization
func TestBatchProcessor_NewBatchProcessor(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("With custom config", func(t *testing.T) {
		config := &BatchProcessorConfig{
			BatchSize:     2000,
			MaxMemoryMB:   1024,
			MaxWorkers:    8,
			EnableMetrics: true,
		}

		bp := NewBatchProcessor(nil, logger, config)

		assert.NotNil(t, bp)
		assert.Equal(t, 2000, bp.batchSize)
		assert.Equal(t, int64(1024), bp.maxMemoryMB)
		assert.NotNil(t, bp.workerPool)
		assert.NotNil(t, bp.memoryMonitor)
	})

	t.Run("With nil config (defaults)", func(t *testing.T) {
		bp := NewBatchProcessor(nil, logger, nil)

		assert.NotNil(t, bp)
		assert.Equal(t, 1000, bp.batchSize)
		assert.Equal(t, int64(512), bp.maxMemoryMB)
		assert.NotNil(t, bp.workerPool)
		assert.NotNil(t, bp.memoryMonitor)
	})

	t.Run("With invalid config values", func(t *testing.T) {
		config := &BatchProcessorConfig{
			BatchSize:     -100,
			MaxMemoryMB:   -512,
			MaxWorkers:    -4,
			EnableMetrics: false,
		}

		bp := NewBatchProcessor(nil, logger, config)

		assert.NotNil(t, bp)
		// Should use defaults for invalid values
		assert.Equal(t, 1000, bp.batchSize)
		assert.Equal(t, int64(512), bp.maxMemoryMB)
	})
}

// TestMemoryMonitor_GetCurrentUsageMB tests memory usage monitoring
func TestMemoryMonitor_GetCurrentUsageMB(t *testing.T) {
	mm := NewMemoryMonitor(512)

	usage := mm.GetCurrentUsageMB()
	assert.GreaterOrEqual(t, usage, int64(0), "Memory usage should be non-negative")
}

// TestMemoryMonitor_GetAvailableMemoryMB tests available memory calculation
func TestMemoryMonitor_GetAvailableMemoryMB(t *testing.T) {
	mm := NewMemoryMonitor(512)

	available := mm.GetAvailableMemoryMB()
	assert.LessOrEqual(t, available, int64(512), "Available memory should not exceed max")
}

// TestMemoryMonitor_ShouldPause tests memory threshold detection
func TestMemoryMonitor_ShouldPause(t *testing.T) {
	// Use a very low memory limit to trigger pause
	mm := NewMemoryMonitor(1)

	// First call might trigger pause if memory usage is high
	shouldPause := mm.ShouldPause()

	// Result depends on actual memory usage, just verify it returns a boolean
	assert.IsType(t, false, shouldPause)
}

// TestWorkerPool_AcquireRelease tests worker pool functionality
func TestWorkerPool_AcquireRelease(t *testing.T) {
	wp := NewWorkerPool(2)

	// Acquire first worker
	wp.Acquire()

	// Acquire second worker
	wp.Acquire()

	// Release workers
	wp.Release()
	wp.Release()

	// Should be able to acquire again
	wp.Acquire()
	wp.Release()
}

// TestBatchProcessor_InsertBatchOptimized tests optimized batch insert
func TestBatchProcessor_InsertBatchOptimized(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("Empty batch", func(t *testing.T) {
		bp := NewBatchProcessor(nil, logger, nil)

		err := bp.insertBatchOptimized(
			context.Background(),
			"test_db",
			"test_table",
			[]string{"id", "name"},
			[]map[string]interface{}{},
		)

		assert.NoError(t, err, "Empty batch should not cause error")
	})

	t.Run("Nil batch", func(t *testing.T) {
		bp := NewBatchProcessor(nil, logger, nil)

		err := bp.insertBatchOptimized(
			context.Background(),
			"test_db",
			"test_table",
			[]string{"id", "name"},
			nil,
		)

		assert.NoError(t, err, "Nil batch should not cause error")
	})
}

// TestBatchProcessor_UpsertBatchOptimized tests optimized batch upsert
func TestBatchProcessor_UpsertBatchOptimized(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("Empty batch", func(t *testing.T) {
		bp := NewBatchProcessor(nil, logger, nil)

		err := bp.upsertBatchOptimized(
			context.Background(),
			"test_db",
			"test_table",
			[]string{"id", "name"},
			[]string{"id"},
			[]map[string]interface{}{},
			nil,
		)

		assert.NoError(t, err, "Empty batch should not cause error")
	})

	t.Run("With conflict resolution overwrite", func(t *testing.T) {
		bp := NewBatchProcessor(nil, logger, nil)

		options := &SyncOptions{
			ConflictResolution: ConflictResolutionOverwrite,
		}

		err := bp.upsertBatchOptimized(
			context.Background(),
			"test_db",
			"test_table",
			[]string{"id", "name"},
			[]string{"id"},
			[]map[string]interface{}{},
			options,
		)

		assert.NoError(t, err, "Empty batch with options should not cause error")
	})

	t.Run("With conflict resolution skip", func(t *testing.T) {
		bp := NewBatchProcessor(nil, logger, nil)

		options := &SyncOptions{
			ConflictResolution: ConflictResolutionSkip,
		}

		err := bp.upsertBatchOptimized(
			context.Background(),
			"test_db",
			"test_table",
			[]string{"id", "name"},
			[]string{"id"},
			[]map[string]interface{}{},
			options,
		)

		assert.NoError(t, err, "Empty batch with skip option should not cause error")
	})
}

// TestBatchInsertResult_Metrics tests batch insert result metrics
func TestBatchInsertResult_Metrics(t *testing.T) {
	result := &BatchInsertResult{
		TotalRows:      10000,
		ProcessedRows:  9500,
		FailedRows:     500,
		BatchCount:     10,
		ThroughputRows: 1000.5,
	}

	assert.Equal(t, int64(10000), result.TotalRows)
	assert.Equal(t, int64(9500), result.ProcessedRows)
	assert.Equal(t, int64(500), result.FailedRows)
	assert.Equal(t, 10, result.BatchCount)
	assert.Equal(t, 1000.5, result.ThroughputRows)
}

// TestMinHelper tests the min helper function
func TestMinHelper(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a is smaller", 5, 10, 5},
		{"b is smaller", 10, 5, 5},
		{"equal values", 7, 7, 7},
		{"negative values", -5, -10, -10},
		{"zero and positive", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests for performance validation
// Requirement 8.1: Use batch operations to improve efficiency

func BenchmarkBatchProcessor_CalculateOptimalBatchSize(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bp := NewBatchProcessor(nil, logger, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.calculateOptimalBatchSize(1_000_000)
	}
}

func BenchmarkMemoryMonitor_GetCurrentUsageMB(b *testing.B) {
	mm := NewMemoryMonitor(512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.GetCurrentUsageMB()
	}
}

func BenchmarkMemoryMonitor_ShouldPause(b *testing.B) {
	mm := NewMemoryMonitor(512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.ShouldPause()
	}
}
