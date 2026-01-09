# Sync Configuration API Endpoints

This document describes the REST API endpoints for sync configuration management, table mapping configuration, and sync task control.

## Base URL

All endpoints are prefixed with `/api/sync`

## Sync Configuration Endpoints

### 1. Get Sync Configurations
**GET** `/api/sync/configs?connection_id={connection_id}`

Get all sync configurations for a specific connection.

**Query Parameters:**
- `connection_id` (required): The connection ID to filter configurations

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "config-123",
      "connection_id": "conn-123",
      "name": "Production Sync",
      "tables": [...],
      "sync_mode": "incremental",
      "schedule": "0 */6 * * *",
      "enabled": true,
      "options": {...},
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "total": 1
  }
}
```

### 2. Create Sync Configuration
**POST** `/api/sync/configs`

Create a new sync configuration.

**Request Body:**
```json
{
  "connection_id": "conn-123",
  "name": "Production Sync",
  "tables": [
    {
      "source_table": "users",
      "target_table": "users_local",
      "sync_mode": "full",
      "enabled": true
    }
  ],
  "sync_mode": "incremental",
  "schedule": "0 */6 * * *",
  "enabled": true,
  "options": {
    "batch_size": 1000,
    "max_concurrency": 5,
    "enable_compression": true,
    "conflict_resolution": "overwrite"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "config-123",
    ...
  }
}
```

### 3. Get Sync Configuration
**GET** `/api/sync/configs/:id`

Get a specific sync configuration by ID.

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "config-123",
    "connection_id": "conn-123",
    "name": "Production Sync",
    ...
  }
}
```

### 4. Update Sync Configuration
**PUT** `/api/sync/configs/:id`

Update an existing sync configuration.

**Request Body:** Same as Create Sync Configuration

**Response:**
```json
{
  "success": true,
  "message": "Sync config updated successfully"
}
```

### 5. Delete Sync Configuration
**DELETE** `/api/sync/configs/:id`

Delete a sync configuration.

**Response:**
```json
{
  "success": true,
  "message": "Sync config deleted successfully"
}
```

## Table Mapping Endpoints

### 6. Get Remote Tables
**GET** `/api/sync/configs/:id/tables`

Get list of available tables from the remote database.

**Response:**
```json
{
  "success": true,
  "data": ["users", "orders", "products"],
  "meta": {
    "total": 3
  }
}
```

### 7. Get Remote Table Schema
**GET** `/api/sync/configs/:id/tables/:table/schema`

Get schema information for a specific remote table.

**Response:**
```json
{
  "success": true,
  "data": {
    "name": "users",
    "columns": [
      {
        "name": "id",
        "type": "int",
        "nullable": false,
        "extra": "auto_increment"
      },
      {
        "name": "email",
        "type": "varchar(255)",
        "nullable": false
      }
    ],
    "indexes": [...],
    "keys": [...]
  }
}
```

### 8. Get Table Mappings
**GET** `/api/sync/configs/:id/mappings`

Get all table mappings for a sync configuration.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "mapping-123",
      "sync_config_id": "config-123",
      "source_table": "users",
      "target_table": "users_local",
      "sync_mode": "full",
      "enabled": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "total": 1
  }
}
```

### 9. Add Table Mapping
**POST** `/api/sync/configs/:id/mappings`

Add a new table mapping to a sync configuration.

**Request Body:**
```json
{
  "source_table": "orders",
  "target_table": "orders_local",
  "sync_mode": "incremental",
  "enabled": true,
  "where_clause": "created_at > '2024-01-01'"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "mapping-456",
    ...
  }
}
```

### 10. Update Table Mapping
**PUT** `/api/sync/configs/:id/mappings/:mapping_id`

Update an existing table mapping.

**Request Body:** Same as Add Table Mapping

**Response:**
```json
{
  "success": true,
  "message": "Table mapping updated successfully"
}
```

### 11. Remove Table Mapping
**DELETE** `/api/sync/configs/:id/mappings/:mapping_id`

Remove a table mapping from a sync configuration.

**Response:**
```json
{
  "success": true,
  "message": "Table mapping removed successfully"
}
```

### 12. Toggle Table Mapping
**POST** `/api/sync/configs/:id/mappings/:mapping_id/toggle`

Enable or disable a table mapping.

**Request Body:**
```json
{
  "enabled": false
}
```

**Response:**
```json
{
  "success": true,
  "message": "Table mapping toggled successfully"
}
```

### 13. Set Table Sync Mode
**POST** `/api/sync/configs/:id/mappings/:mapping_id/sync-mode`

Update the sync mode for a table mapping.

**Request Body:**
```json
{
  "sync_mode": "incremental"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Table sync mode updated successfully"
}
```

## Sync Task Control Endpoints

### 14. Get Sync Jobs
**GET** `/api/sync/jobs?limit=50&offset=0`

Get list of sync jobs with pagination.

**Query Parameters:**
- `limit` (optional, default: 50): Maximum number of jobs to return
- `offset` (optional, default: 0): Number of jobs to skip

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "job-123",
      "config_id": "config-123",
      "status": "completed",
      "start_time": "2024-01-01T00:00:00Z",
      "end_time": "2024-01-01T00:30:00Z",
      "total_tables": 5,
      "completed_tables": 5,
      "total_rows": 10000,
      "processed_rows": 10000
    }
  ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "count": 1
  }
}
```

### 15. Start Sync Job
**POST** `/api/sync/jobs`

Start a new sync job.

**Request Body:**
```json
{
  "config_id": "config-123"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "job-456",
    "config_id": "config-123",
    "status": "pending",
    "start_time": "2024-01-01T00:00:00Z",
    "total_tables": 5,
    "completed_tables": 0
  }
}
```

### 16. Get Sync Job
**GET** `/api/sync/jobs/:id`

Get details of a specific sync job.

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "job-123",
    "config_id": "config-123",
    "status": "running",
    "progress": {
      "total_tables": 5,
      "completed_tables": 2,
      "total_rows": 10000,
      "processed_rows": 4000,
      "percentage": 40.0
    },
    ...
  }
}
```

### 17. Stop Sync Job
**POST** `/api/sync/jobs/:id/stop`

Stop a running sync job.

**Response:**
```json
{
  "success": true,
  "message": "Sync job stop requested"
}
```

### 18. Cancel Sync Job
**POST** `/api/sync/jobs/:id/cancel`

Cancel a sync job (alias for stop).

**Response:**
```json
{
  "success": true,
  "message": "Sync job cancelled successfully"
}
```

### 19. Get Sync Job Logs
**GET** `/api/sync/jobs/:id/logs`

Get logs for a specific sync job.

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
      "message": "Started syncing table users",
      "created_at": "2024-01-01T00:00:00Z"
    },
    {
      "id": 2,
      "job_id": "job-123",
      "table_name": "users",
      "level": "info",
      "message": "Completed syncing table users: 1000 rows",
      "created_at": "2024-01-01T00:05:00Z"
    }
  ],
  "meta": {
    "total": 2
  }
}
```

### 20. Get Sync Job Progress
**GET** `/api/sync/jobs/:id/progress`

Get detailed progress information for a sync job.

**Response:**
```json
{
  "success": true,
  "data": {
    "job_id": "job-123",
    "config_id": "config-123",
    "status": "running",
    "start_time": "2024-01-01T00:00:00Z",
    "total_tables": 5,
    "completed_tables": 2,
    "total_rows": 10000,
    "processed_rows": 4000,
    "progress_percent": 40.0,
    "table_progress": {
      "users": {
        "table_name": "users",
        "status": "completed",
        "total_rows": 1000,
        "processed_rows": 1000
      },
      "orders": {
        "table_name": "orders",
        "status": "running",
        "total_rows": 5000,
        "processed_rows": 2000
      }
    }
  }
}
```

### 21. Get Active Sync Jobs
**GET** `/api/sync/jobs/active`

Get all currently running sync jobs.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "job_id": "job-123",
      "config_id": "config-123",
      "status": "running",
      "progress_percent": 40.0,
      ...
    }
  ],
  "meta": {
    "total": 1
  }
}
```

### 22. Get Sync Job History
**GET** `/api/sync/jobs/history?limit=50&offset=0`

Get historical sync job records.

**Query Parameters:**
- `limit` (optional, default: 50): Maximum number of jobs to return
- `offset` (optional, default: 0): Number of jobs to skip

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "job-123",
      "config_id": "config-123",
      "config_name": "Production Sync",
      "connection_name": "Production DB",
      "status": "completed",
      "start_time": "2024-01-01T00:00:00Z",
      "end_time": "2024-01-01T00:30:00Z",
      "total_tables": 5,
      "completed_tables": 5,
      "total_rows": 10000,
      "processed_rows": 10000
    }
  ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "count": 1
  }
}
```

## Error Responses

All endpoints return error responses in the following format:

```json
{
  "success": false,
  "error": "Error message describing what went wrong"
}
```

Common HTTP status codes:
- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request body or parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Sync system not available

## Requirements Coverage

This API implementation covers the following requirements:

- **Requirement 3.1**: Browse remote database and display available tables (endpoints 6, 7)
- **Requirement 3.2**: Select tables for synchronization and save configuration (endpoints 2, 9)
- **Requirement 3.3**: Configure table sync rules (full/incremental mode) (endpoints 2, 4, 13)
- **Requirement 3.4**: Customize local table names (endpoints 2, 4, 9, 10)
- **Requirement 3.5**: Enable/disable table synchronization (endpoints 11, 12)
- **Requirement 4.1**: Start synchronization tasks (endpoint 15)
- **Requirement 4.5**: Update sync status (endpoint 16)
- **Requirement 5.1**: Real-time display of sync progress and status (endpoints 16, 20, 21)
- **Requirement 5.2**: Display historical sync records (endpoints 14, 22)
- **Requirement 5.3**: Display detailed error information (endpoint 19)
- **Requirement 5.4**: Display statistics information (endpoint 20)
