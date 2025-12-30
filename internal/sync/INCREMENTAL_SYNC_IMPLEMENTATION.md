# Incremental Sync Implementation

## Overview

Task 6.3 has been successfully completed. The incremental synchronization functionality is fully implemented in the database synchronization system.

## Implementation Details

### 1. Change Data Capture (CDC) - 变更数据捕获

**Location**: `sync_engine.go` - `detectChangeTrackingColumn()` method

The CDC mechanism automatically detects suitable columns for tracking changes:

- **Timestamp-based tracking**: Detects columns like `updated_at`, `modified_at`, `last_modified`, etc.
- **Auto-increment ID tracking**: Detects auto-increment primary key columns
- **Priority order**: Timestamp columns are preferred over auto-increment columns

**Supported column patterns**:
- `updated_at`, `modified_at`, `last_modified`, `update_time`, `modify_time`
- Any `timestamp` or `datetime` column with "update" in the name
- Auto-increment primary key columns

### 2. Checkpoint Mechanism - 检查点机制

**Location**: `checkpoint.go` and `sync_engine.go`

The checkpoint system enables resume functionality and tracks sync progress:

**Key Components**:
- `createInitialCheckpoint()`: Creates checkpoint after first full sync
- `updateCheckpoint()`: Updates checkpoint after each incremental sync
- `SyncCheckpoint` structure stores:
  - Last sync timestamp
  - Last sync value (for ID-based tracking)
  - Checkpoint metadata in JSON format

**Checkpoint Storage**:
- Stored in `sync_checkpoints` table
- Linked to table mapping ID
- Includes both timestamp and value tracking

### 3. Timestamp-based Incremental Sync - 基于时间戳的增量同步

**Location**: `sync_engine.go` - `syncIncrementalByTimestamp()` method

Synchronizes only records modified after the last sync:

**Features**:
- Queries records where timestamp > last_sync_time
- Batch processing for efficiency
- Upsert operations with conflict resolution
- Progress tracking and logging

**SQL Pattern**:
```sql
SELECT * FROM table WHERE updated_at > ? ORDER BY updated_at
```

### 4. ID-based Incremental Sync - 基于ID的增量同步

**Location**: `sync_engine.go` - `syncIncrementalByID()` method

Synchronizes only records with IDs greater than the last synced ID:

**Features**:
- Queries records where id > last_sync_id
- Suitable for append-only tables
- Batch processing with configurable size
- Automatic checkpoint updates

**SQL Pattern**:
```sql
SELECT * FROM table WHERE id > ? ORDER BY id
```

### 5. Upsert Operations - 插入或更新操作

**Location**: `sync_engine.go` - `upsertBatch()` method

Handles data conflicts during incremental sync:

**Conflict Resolution Strategies**:
- `ConflictResolutionOverwrite`: Update existing records with new data
- `ConflictResolutionSkip`: Keep existing records, skip updates
- `ConflictResolutionError`: Fail on conflicts

**SQL Pattern**:
```sql
INSERT INTO table (columns...) VALUES (...)
ON DUPLICATE KEY UPDATE col1 = VALUES(col1), col2 = VALUES(col2)...
```

### 6. Fallback Mechanism - 回退机制

**Location**: `sync_engine.go` - `SyncIncremental()` method

Automatically falls back to full sync when:
- No checkpoint exists (first-time sync)
- Checkpoint load fails
- Change tracking column cannot be detected

This ensures robust operation even in edge cases.

## Testing

### Unit Tests

**Location**: `sync_engine_incremental_test.go`

Comprehensive tests covering:
- Change tracking column detection
- Incremental sync with existing checkpoint
- Checkpoint creation and updates
- Both timestamp and ID-based sync modes
- Fallback to full sync behavior
- Upsert batch operations

### Test Results

All tests pass successfully:
```
=== RUN   TestSyncEngine_IncrementalSync_WithCheckpoint
--- PASS: TestSyncEngine_IncrementalSync_WithCheckpoint
=== RUN   TestSyncEngine_IncrementalSyncModes
--- PASS: TestSyncEngine_IncrementalSyncModes
=== RUN   TestSyncEngine_IncrementalSync_FallbackToFull
--- PASS: TestSyncEngine_IncrementalSync_FallbackToFull
=== RUN   TestSyncEngine_SyncIncremental_NoCheckpoint
--- PASS: TestSyncEngine_SyncIncremental_NoCheckpoint
```

## Usage Example

```go
// Create sync engine
engine := NewSyncEngine(localDB, repo, logger)

// Configure incremental sync
mapping := &TableMapping{
    ID:           "mapping-1",
    SyncConfigID: "config-1",
    SourceTable:  "users",
    TargetTable:  "users",
    SyncMode:     SyncModeIncremental,
    Enabled:      true,
}

// Execute incremental sync
err := engine.SyncIncremental(ctx, job, mapping)
if err != nil {
    log.Fatalf("Incremental sync failed: %v", err)
}
```

## Performance Characteristics

### Advantages of Incremental Sync

1. **Reduced Data Transfer**: Only changed records are transferred
2. **Lower Resource Usage**: Less CPU, memory, and network bandwidth
3. **Faster Sync Times**: Especially beneficial for large tables
4. **Minimal Impact**: Reduced load on source database

### Batch Processing

- Configurable batch size (default: 1000 rows)
- Prevents memory exhaustion on large datasets
- Balances throughput and resource usage

### Checkpoint Overhead

- Minimal storage overhead (one row per table mapping)
- Fast checkpoint updates (single UPDATE query)
- Enables resume functionality for interrupted syncs

## Requirements Validation

✅ **Requirement 4.3**: Execute incremental sync - sync only changed data
- Implemented CDC for change detection
- Checkpoint mechanism for tracking sync progress
- Timestamp-based and ID-based incremental sync
- Automatic fallback to full sync when needed

## Future Enhancements

Potential improvements for future iterations:

1. **Binary Log (Binlog) CDC**: Real-time change capture using MySQL binlog
2. **Multi-column Change Tracking**: Support composite change tracking keys
3. **Soft Delete Handling**: Detect and sync deleted records
4. **Conflict Resolution UI**: User interface for manual conflict resolution
5. **Sync Scheduling**: Automated incremental sync at regular intervals

## Related Files

- `sync_engine.go`: Core incremental sync implementation
- `checkpoint.go`: Checkpoint management system
- `sync_engine_incremental_test.go`: Incremental sync tests
- `types.go`: Data structures and constants
- `interfaces.go`: Interface definitions

## Conclusion

The incremental sync functionality is fully implemented and tested. It provides efficient, reliable synchronization of changed data with automatic change detection, checkpoint management, and conflict resolution. The implementation meets all requirements specified in task 6.3.
