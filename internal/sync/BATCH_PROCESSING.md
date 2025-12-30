# Batch Processing Implementation

## Overview

This document describes the batch processing and memory optimization implementation for the database synchronization system.

## Requirements Addressed

- **Requirement 7.3**: Process large tables in batches to avoid long locks
- **Requirement 8.1**: Use batch operations to improve efficiency when syncing large amounts of data

## Components

### BatchProcessor

The `BatchProcessor` is the main component responsible for optimized batch operations during data synchronization.

#### Key Features

1. **Adaptive Batch Sizing**
   - Automatically adjusts batch size based on table size
   - Considers available memory when determining batch size
   - Smaller batches for very large tables (>10M rows)
   - Larger batches for medium tables (100K-1M rows)

2. **Memory Monitoring**
   - Tracks current memory usage
   - Pauses processing when memory usage exceeds 80% threshold
   - Triggers garbage collection when needed
   - Prevents out-of-memory errors

3. **Performance Optimization**
   - Multi-row INSERT statements for better throughput
   - Timeout protection to prevent long locks
   - Batch metrics collection (throughput, duration, etc.)
   - Zero-allocation batch size calculation

4. **Large Table Support**
   - Chunked processing with LIMIT/OFFSET
   - Progress tracking and logging
   - Graceful handling of batch failures
   - Memory-efficient row processing

### Configuration

```go
type BatchProcessorConfig struct {
    BatchSize     int   // Number of rows per batch (default: 1000)
    MaxMemoryMB   int64 // Maximum memory usage in MB (default: 512)
    MaxWorkers    int   // Maximum concurrent workers (default: CPU count)
    EnableMetrics bool  // Enable performance metrics (default: true)
}
```

### Usage

#### Basic Usage

```go
// Create batch processor with default configuration
batchProcessor := NewBatchProcessor(localDB, logger, nil)

// Process large table sync
result, err := batchProcessor.ProcessLargeTableSync(
    ctx,
    remoteDB,
    localDB,
    mapping,
    options,
)
```

#### Custom Configuration

```go
// Create batch processor with custom configuration
config := &BatchProcessorConfig{
    BatchSize:     2000,
    MaxMemoryMB:   1024,
    MaxWorkers:    8,
    EnableMetrics: true,
}

batchProcessor := NewBatchProcessor(localDB, logger, config)
```

#### Integration with SyncEngine

The batch processor is automatically integrated into the sync engine:

```go
// For tables with >10,000 rows, batch processor is used automatically
func (e *DefaultSyncEngine) syncAllData(...) error {
    if totalRows > 10000 {
        // Use optimized batch processor
        result, err := e.batchProcessor.ProcessLargeTableSync(...)
    } else {
        // Use standard batch processing
        // ...
    }
}
```

## Performance Characteristics

### Batch Size Optimization

| Table Size | Default Batch Size | Rationale |
|------------|-------------------|-----------|
| < 100K rows | 1000-2000 | Fast processing, minimal memory |
| 100K-1M rows | 1000 | Balanced performance |
| 1M-10M rows | 500-1000 | Prevent memory issues |
| > 10M rows | 200-500 | Minimize lock duration |

### Memory Management

- **Threshold**: 80% of configured max memory
- **Action**: Pause processing and trigger GC
- **Recovery**: Resume after memory is freed
- **Protection**: Prevents OOM errors

### Throughput

Based on benchmarks:
- Batch size calculation: ~32,000 ns/op (0 allocs)
- Memory monitoring: ~31,000 ns/op (0 allocs)
- Typical throughput: 1,000-10,000 rows/second (depends on network and data size)

## Metrics

The `BatchInsertResult` provides detailed metrics:

```go
type BatchInsertResult struct {
    TotalRows      int64         // Total rows to process
    ProcessedRows  int64         // Successfully processed rows
    FailedRows     int64         // Failed rows
    BatchCount     int           // Number of batches processed
    Duration       time.Duration // Total processing time
    AvgBatchTime   time.Duration // Average time per batch
    MemoryPeakMB   int64         // Peak memory usage
    ThroughputRows float64       // Rows per second
}
```

## Error Handling

### Batch Failure Strategy

- Individual batch failures don't stop the entire sync
- Failed rows are counted and logged
- Processing continues with next batch
- Final result includes success/failure counts

### Memory Pressure

- Automatic pause when memory usage is high
- Garbage collection triggered
- Processing resumes after memory is freed
- Prevents system-wide memory issues

### Timeout Protection

- Each batch has a 30-second timeout
- Prevents long-running locks on target database
- Ensures responsive system behavior

## Best Practices

1. **Configuration**
   - Set `MaxMemoryMB` based on available system memory
   - Use default `BatchSize` unless you have specific requirements
   - Enable metrics for production monitoring

2. **Large Tables**
   - For tables > 10M rows, consider running sync during off-peak hours
   - Monitor memory usage during initial sync
   - Use incremental sync after initial full sync

3. **Performance Tuning**
   - Increase `BatchSize` for faster networks and powerful databases
   - Decrease `BatchSize` for memory-constrained environments
   - Adjust `MaxWorkers` based on CPU cores

4. **Monitoring**
   - Track `ThroughputRows` to identify performance issues
   - Monitor `MemoryPeakMB` to optimize memory configuration
   - Review `FailedRows` to identify data quality issues

## Testing

### Unit Tests

- Batch size calculation for various table sizes
- Memory monitoring and threshold detection
- Worker pool acquire/release
- Empty batch handling
- Configuration validation

### Benchmarks

- Batch size calculation: ~32 µs per operation
- Memory monitoring: ~31 µs per operation
- Zero allocations for core operations

### Integration Tests

Integration tests should verify:
- End-to-end large table sync
- Memory pressure handling
- Batch failure recovery
- Throughput under load

## Future Enhancements

1. **Parallel Batch Processing**
   - Process multiple batches concurrently
   - Requires careful ordering for incremental sync

2. **Compression**
   - Compress data during transfer
   - Reduce network bandwidth usage

3. **Adaptive Throttling**
   - Automatically adjust batch size based on performance
   - Respond to database load

4. **Progress Callbacks**
   - Real-time progress updates
   - Integration with monitoring systems
