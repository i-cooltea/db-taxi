# Database Synchronization Features

This document describes the database synchronization features added to DB-Taxi.

## Overview

The synchronization system extends DB-Taxi with the ability to:
- Manage multiple remote database connections
- Configure selective table synchronization
- Execute full and incremental sync operations
- Monitor sync job progress and status
- Handle errors and provide detailed logging

## Architecture

The sync system is built with a modular architecture:

### Core Components

1. **Connection Manager** - Manages remote database connections
2. **Sync Manager** - Handles synchronization configurations and jobs
3. **Job Engine** - Executes sync jobs (planned for future implementation)
4. **Mapping Manager** - Manages database and table mappings (planned)
5. **Sync Engine** - Performs actual data synchronization (planned)

### Data Layer

- **Repository** - Data access layer for sync operations
- **MySQL Storage** - Stores sync configurations, jobs, and logs

## Configuration

Add sync configuration to your `config.yaml`:

```yaml
sync:
  enabled: true
  max_concurrency: 5
  batch_size: 1000
  retry_attempts: 3
  retry_delay: "30s"
  job_timeout: "1h"
  cleanup_age: "30d"
```

## Database Setup

Run the migration script to create sync tables:

```sql
-- Run the migration script
source migrations/b-002_create_sync_tables.sql
```

## API Endpoints

### Connection Management

- `GET /api/sync/connections` - List all connections
- `POST /api/sync/connections` - Create new connection
- `GET /api/sync/connections/:id` - Get connection details
- `PUT /api/sync/connections/:id` - Update connection
- `DELETE /api/sync/connections/:id` - Delete connection
- `POST /api/sync/connections/:id/test` - Test connection

### Sync Configuration

- `GET /api/sync/configs?connection_id=:id` - List sync configs for connection
- `POST /api/sync/configs` - Create sync configuration
- `GET /api/sync/configs/:id` - Get sync configuration
- `PUT /api/sync/configs/:id` - Update sync configuration
- `DELETE /api/sync/configs/:id` - Delete sync configuration

### Job Management

- `GET /api/sync/jobs` - List sync jobs (planned)
- `POST /api/sync/jobs` - Start sync job
- `GET /api/sync/jobs/:id` - Get job status
- `POST /api/sync/jobs/:id/stop` - Stop sync job
- `GET /api/sync/jobs/:id/logs` - Get job logs (planned)

### System Status

- `GET /api/sync/status` - Get sync system health status
- `GET /api/sync/stats` - Get sync system statistics

## Usage Example

### 1. Create a Remote Connection

```bash
curl -X POST http://localhost:8080/api/sync/connections \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production DB",
    "host": "prod-db.example.com",
    "port": 3306,
    "username": "sync_user",
    "password": "secure_password",
    "database": "production",
    "local_db_name": "prod_sync",
    "ssl": true
  }'
```

### 2. Create Sync Configuration

```bash
curl -X POST http://localhost:8080/api/sync/configs \
  -H "Content-Type: application/json" \
  -d '{
    "connection_id": "connection-uuid",
    "name": "User Data Sync",
    "sync_mode": "incremental",
    "enabled": true,
    "tables": [
      {
        "source_table": "users",
        "target_table": "users",
        "sync_mode": "incremental",
        "enabled": true
      },
      {
        "source_table": "orders",
        "target_table": "orders",
        "sync_mode": "full",
        "enabled": true
      }
    ],
    "options": {
      "batch_size": 1000,
      "max_concurrency": 3,
      "enable_compression": true,
      "conflict_resolution": "overwrite"
    }
  }'
```

### 3. Start Sync Job

```bash
curl -X POST http://localhost:8080/api/sync/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "config_id": "sync-config-uuid"
  }'
```

### 4. Monitor Job Status

```bash
curl http://localhost:8080/api/sync/jobs/job-uuid
```

## Current Implementation Status

### ✅ Completed (Task 1)
- Core interfaces and data structures
- Configuration system extension
- Database migration scripts
- Repository layer implementation
- Basic service layer
- REST API endpoints
- Server integration

### 🚧 Planned (Future Tasks)
- Job Engine implementation
- Sync Engine implementation
- Mapping Manager implementation
- Actual database connection testing
- Data synchronization logic
- Web UI components
- Advanced error handling
- Performance optimizations

## Development Notes

This implementation provides the foundation for the database synchronization system. The core interfaces, data structures, and API endpoints are in place, but the actual synchronization logic will be implemented in subsequent tasks.

The system is designed to be:
- **Modular** - Each component has clear responsibilities
- **Extensible** - Easy to add new sync modes and features
- **Robust** - Comprehensive error handling and logging
- **Scalable** - Support for concurrent operations and large datasets