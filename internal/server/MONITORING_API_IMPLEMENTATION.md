# Monitoring and Statistics API Implementation

## Overview

This document describes the implementation of monitoring and statistics API endpoints for the database synchronization system.

## Implemented API Endpoints

### 1. Sync Status Query API

**Endpoint:** `GET /api/sync/status`

**Description:** Returns the current health status of the sync system.

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2026-01-09T11:22:50Z"
  }
}
```

**Requirements:** 5.1 - Real-time display of sync progress and status

**Implementation:** `server.go:getSyncStatus()`

---

### 2. Sync Statistics API

**Endpoint:** `GET /api/sync/stats`

**Description:** Returns overall synchronization statistics including job counts, data volumes, and performance metrics.

**Response:**
```json
{
  "success": true,
  "data": {
    "total_jobs": 100,
    "completed_jobs": 85,
    "failed_jobs": 10,
    "running_jobs": 5,
    "total_rows_synced": 1000000,
    "total_tables_synced": 250,
    "average_job_duration_minutes": 15.5,
    "error_rate_percentage": 10.0,
    "sync_frequency_per_hour": 4.2,
    "last_sync_time": "2026-01-09T11:00:00Z",
    "generated_at": "2026-01-09T11:22:50Z"
  }
}
```

**Requirements:** 5.4 - Display statistics information including data volume and time consumption

**Implementation:** `server.go:getSyncStats()`

---

### 3. Sync Job History API

**Endpoint:** `GET /api/sync/jobs/history`

**Query Parameters:**
- `limit` (optional, default: 50): Maximum number of records to return
- `offset` (optional, default: 0): Number of records to skip

**Description:** Returns historical sync job records with pagination support.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "job-1",
      "config_id": "config-1",
      "config_name": "Test Config",
      "connection_name": "Test Connection",
      "status": "completed",
      "start_time": "2026-01-09T10:00:00Z",
      "end_time": "2026-01-09T10:10:00Z",
      "total_tables": 5,
      "completed_tables": 5,
      "total_rows": 10000,
      "processed_rows": 10000,
      "created_at": "2026-01-09T10:00:00Z"
    },
    {
      "id": "job-2",
      "config_id": "config-1",
      "config_name": "Test Config",
      "connection_name": "Test Connection",
      "status": "failed",
      "start_time": "2026-01-09T09:00:00Z",
      "end_time": "2026-01-09T10:00:00Z",
      "total_tables": 3,
      "completed_tables": 2,
      "total_rows": 5000,
      "processed_rows": 3000,
      "error": "Connection timeout",
      "created_at": "2026-01-09T09:00:00Z"
    }
  ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "count": 2
  }
}
```

**Requirements:** 5.2 - Display historical sync records and results

**Implementation:** `server.go:getSyncJobHistory()`

---

### 4. Sync Job Logs API

**Endpoint:** `GET /api/sync/jobs/:id/logs`

**Description:** Returns detailed logs for a specific sync job, including error messages and suggestions.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "job_id": "job-123",
      "table_name": "users",
      "level": "info",
      "message": "Starting table sync",
      "created_at": "2026-01-09T10:00:00Z"
    },
    {
      "id": 2,
      "job_id": "job-123",
      "table_name": "users",
      "level": "error",
      "message": "Failed to sync table: connection timeout",
      "created_at": "2026-01-09T10:05:00Z"
    }
  ],
  "meta": {
    "total": 2
  }
}
```

**Requirements:** 5.3 - Display detailed error information and suggestions when sync fails

**Implementation:** `server.go:getSyncJobLogs()`

---

### 5. Active Sync Jobs API

**Endpoint:** `GET /api/sync/jobs/active`

**Description:** Returns currently running sync jobs with real-time progress information.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "job_id": "job-123",
      "config_id": "config-123",
      "status": "running",
      "start_time": "2026-01-09T10:00:00Z",
      "total_tables": 5,
      "completed_tables": 3,
      "total_rows": 10000,
      "processed_rows": 6000,
      "progress_percent": 60.0,
      "error_count": 0,
      "warnings": [],
      "table_progress": {
        "users": {
          "table_name": "users",
          "status": "completed",
          "start_time": "2026-01-09T10:00:00Z",
          "end_time": "2026-01-09T10:02:00Z",
          "total_rows": 2000,
          "processed_rows": 2000,
          "error_count": 0
        },
        "orders": {
          "table_name": "orders",
          "status": "running",
          "start_time": "2026-01-09T10:02:00Z",
          "total_rows": 4000,
          "processed_rows": 2000,
          "error_count": 0
        }
      }
    }
  ],
  "meta": {
    "total": 1
  }
}
```

**Requirements:** 5.1 - Real-time display of sync progress and status

**Implementation:** `server.go:getActiveSyncJobs()`

---

### 6. Sync Job Progress API

**Endpoint:** `GET /api/sync/jobs/:id/progress`

**Description:** Returns detailed progress information for a specific sync job.

**Response:**
```json
{
  "success": true,
  "data": {
    "job_id": "job-123",
    "config_id": "config-123",
    "status": "running",
    "start_time": "2026-01-09T10:00:00Z",
    "total_tables": 5,
    "completed_tables": 3,
    "total_rows": 10000,
    "processed_rows": 6000,
    "progress_percent": 60.0,
    "error_count": 0,
    "warnings": [],
    "table_progress": {
      "users": {
        "table_name": "users",
        "status": "completed",
        "start_time": "2026-01-09T10:00:00Z",
        "end_time": "2026-01-09T10:02:00Z",
        "total_rows": 2000,
        "processed_rows": 2000,
        "error_count": 0
      }
    }
  }
}
```

**Requirements:** 5.1 - Real-time display of sync progress and status

**Implementation:** `server.go:getSyncJobProgress()`

---

## Backend Services

The API endpoints are backed by the following services:

### MonitoringService

Located in `internal/sync/monitoring.go`, this service provides:

- **StartJobMonitoring**: Initializes monitoring for a new sync job
- **UpdateJobProgress**: Updates overall job progress
- **UpdateTableProgress**: Updates progress for individual tables
- **GetJobProgress**: Retrieves current job progress
- **FinishJobMonitoring**: Completes monitoring and archives job data
- **GetSyncHistory**: Retrieves historical job records
- **GetSyncStatistics**: Calculates and caches overall statistics
- **GetActiveJobs**: Returns currently running jobs
- **GetJobLogs**: Retrieves logs for a specific job
- **LogJobEvent**: Records events during sync execution

### Key Features

1. **Real-time Progress Tracking**: Active jobs are tracked in memory with detailed table-level progress
2. **Statistics Caching**: Overall statistics are cached for 5 minutes to improve performance
3. **Comprehensive Logging**: All sync events are logged with timestamps and severity levels
4. **Error Tracking**: Errors are tracked per table and per job with detailed messages
5. **Performance Metrics**: Average duration, error rates, and sync frequency are calculated

---

## Testing

Comprehensive unit tests are provided in `sync_monitoring_api_test.go`:

- **TestGetSyncStatus**: Tests sync system health check
- **TestGetSyncStats**: Tests statistics retrieval
- **TestGetSyncJobHistory**: Tests historical records with pagination
- **TestGetSyncJobHistoryWithPagination**: Tests pagination parameters
- **TestGetSyncJobLogs**: Tests log retrieval
- **TestGetSyncStatusUnavailable**: Tests error handling when sync system is unavailable
- **TestGetSyncStatsError**: Tests error handling for statistics failures

All tests use mocks to isolate API layer testing from service implementation.

---

## Error Handling

All endpoints follow consistent error response format:

```json
{
  "success": false,
  "error": "Error message describing what went wrong"
}
```

Common HTTP status codes:
- `200 OK`: Successful request
- `404 Not Found`: Resource not found (job, config, etc.)
- `500 Internal Server Error`: Server-side error
- `503 Service Unavailable`: Sync system not initialized

---

## Integration

The monitoring API endpoints are automatically registered when the sync system is initialized. They are available under the `/api/sync/` path prefix:

- `/api/sync/status` - System health
- `/api/sync/stats` - Overall statistics
- `/api/sync/jobs/history` - Historical records
- `/api/sync/jobs/active` - Active jobs
- `/api/sync/jobs/:id/progress` - Job progress
- `/api/sync/jobs/:id/logs` - Job logs

All endpoints require the sync system to be properly initialized and will return `503 Service Unavailable` if the sync system is not available.
