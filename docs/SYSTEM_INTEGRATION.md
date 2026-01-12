# System Integration Documentation

## Overview

This document describes how all components of the DB-Taxi database synchronization system are integrated and work together.

## Architecture

The system follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────┐
│                     Main Application                     │
│                      (main.go)                          │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                    HTTP Server Layer                     │
│                 (internal/server/server.go)             │
│  - REST API endpoints                                   │
│  - Request routing                                      │
│  - Middleware (CORS, logging, auth)                     │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                  Sync System Manager                     │
│                 (internal/sync/sync.go)                 │
│  - Component lifecycle management                       │
│  - Dependency injection                                 │
│  - Health checks and monitoring                         │
└────────────────────────┬────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┬────────────────┐
         ▼               ▼               ▼                ▼
┌─────────────┐  ┌─────────────┐  ┌──────────┐  ┌──────────────┐
│ Connection  │  │    Sync     │  │ Mapping  │  │     Job      │
│  Manager    │  │   Manager   │  │ Manager  │  │   Engine     │
└─────────────┘  └─────────────┘  └──────────┘  └──────────────┘
         │               │               │                │
         └───────────────┴───────────────┴────────────────┘
                         │
                         ▼
                ┌─────────────────┐
                │  Sync Engine    │
                │  (Data Transfer)│
                └─────────────────┘
                         │
                         ▼
                ┌─────────────────┐
                │    Repository   │
                │  (Data Access)  │
                └─────────────────┘
                         │
                         ▼
                ┌─────────────────┐
                │  MySQL Database │
                └─────────────────┘
```

## Component Integration

### 1. Main Application (main.go)

**Responsibilities:**
- Parse command-line arguments
- Load configuration from files and environment variables
- Create and start the HTTP server
- Handle graceful shutdown on SIGINT/SIGTERM

**Lifecycle:**
```go
1. Parse flags
2. Load configuration (config file + env vars + CLI overrides)
3. Create server instance
4. Start server in goroutine
5. Wait for shutdown signal
6. Gracefully shutdown with timeout
```

### 2. HTTP Server (internal/server/server.go)

**Responsibilities:**
- Initialize all subsystems (database, sync system)
- Register HTTP routes and middleware
- Handle HTTP requests and responses
- Coordinate graceful shutdown

**Initialization Order:**
```go
1. Create logger
2. Create Gin engine with middleware
3. Initialize database connection pool
4. Initialize sync system manager
5. Register all routes (API + static files)
```

**Shutdown Order:**
```go
1. Shutdown sync system (stop jobs, close connections)
2. Close database connection pool
3. Shutdown HTTP server
```

### 3. Sync System Manager (internal/sync/sync.go)

**Responsibilities:**
- Create and wire all sync components
- Run database migrations
- Start/stop job engine
- Provide health checks and statistics
- Coordinate component lifecycle

**Component Initialization:**
```go
1. Create repository (data access layer)
2. Create monitoring service
3. Create sync engine (data transfer)
4. Create job engine (task scheduling)
5. Create connection manager (remote DB connections)
6. Create sync manager (sync configuration)
7. Create mapping manager (table mappings)
8. Wire dependencies between components
```

**Initialization Steps:**
```go
1. Run database migrations
2. Start job engine
3. Log successful initialization
```

**Shutdown Steps:**
```go
1. Stop job engine (cancel running jobs)
2. Close connection manager (close all DB connections)
3. Log shutdown completion
```

### 4. Core Components

#### Connection Manager
- Manages remote database connections
- Handles connection pooling and health checks
- Provides connection CRUD operations

#### Sync Manager
- Manages sync configurations
- Controls sync job execution
- Tracks sync status and history

#### Mapping Manager
- Manages database and table mappings
- Handles configuration import/export
- Validates mapping configurations

#### Job Engine
- Schedules and executes sync jobs
- Manages job queue and concurrency
- Handles job lifecycle (start, stop, cancel)

#### Sync Engine
- Performs actual data synchronization
- Handles full and incremental sync
- Manages transactions and error recovery

## Dependency Injection

All components are created and wired together in the `sync.NewManager()` function:

```go
// Create shared dependencies
repo := NewMySQLRepository(db, logger)
monitoring := NewMonitoringService(repo, logger)

// Create engines
syncEngine := NewSyncEngine(db, repo, logger)
jobEngine := NewJobEngine(repo, logger, monitoring, syncEngine)

// Create managers
connectionManager := NewConnectionManager(repo, logger, db)
syncManager := NewSyncManager(repo, logger, db)
mappingManager := NewMappingManager(db, repo, logger)

// Wire dependencies
syncManager.jobEngine = jobEngine
```

This ensures:
- Single source of truth for dependencies
- Proper initialization order
- Easy testing with mock dependencies
- Clear component boundaries

## Configuration

Configuration is loaded in layers with the following precedence (highest to lowest):

1. Command-line flags
2. Environment variables (with `DBT_` prefix)
3. Configuration file (YAML)
4. Default values

Example configuration structure:
```yaml
server:
  port: 8080
  host: 0.0.0.0
  read_timeout: 30s
  write_timeout: 30s

database:
  host: localhost
  port: 3306
  username: root
  password: secret
  database: db_taxi
  max_open_conns: 25
  max_idle_conns: 5

sync:
  enabled: true
  max_concurrency: 5
  batch_size: 1000
  retry_attempts: 3
  retry_delay: 30s
  job_timeout: 1h

logging:
  level: info
  format: json
```

## Startup Sequence

1. **Parse Configuration**
   - Read config file (if specified)
   - Apply environment variables
   - Apply command-line overrides

2. **Initialize Server**
   - Create logger with configured level
   - Create Gin engine with middleware
   - Initialize database connection pool
   - Test database connectivity

3. **Initialize Sync System**
   - Create sync manager with all components
   - Run database migrations
   - Start job engine
   - Verify system health

4. **Register Routes**
   - Register API endpoints
   - Register static file handlers
   - Set up error handlers

5. **Start HTTP Server**
   - Bind to configured host:port
   - Start listening for requests
   - Log startup completion

## Shutdown Sequence

1. **Receive Shutdown Signal**
   - SIGINT (Ctrl+C) or SIGTERM

2. **Stop Accepting New Requests**
   - HTTP server stops accepting connections

3. **Shutdown Sync System**
   - Stop job engine (cancel running jobs)
   - Wait for jobs to complete (with timeout)
   - Close all remote database connections

4. **Close Database Connection**
   - Close local database connection pool
   - Wait for active connections to finish

5. **Shutdown HTTP Server**
   - Wait for active requests to complete (30s timeout)
   - Close all connections
   - Exit process

## Health Checks

The system provides multiple health check endpoints:

### System Health (`/health`)
- Basic server health check
- Returns 200 OK if server is running

### Database Status (`/api/status`)
- Checks database connectivity
- Returns connection pool statistics
- Checks sync system status

### Sync System Health (`/api/sync/status`)
- Verifies sync tables exist
- Checks database connectivity
- Returns sync system statistics

## Error Handling

### Initialization Errors

If critical components fail to initialize:
- Database connection failure: Server starts but shows error in UI
- Sync system failure: Server starts with sync features disabled
- Configuration errors: Application exits with error message

### Runtime Errors

- Connection errors: Retry with exponential backoff
- Sync errors: Log error, continue with other tables
- Database errors: Rollback transaction, retry if transient

### Shutdown Errors

- Job engine stop failure: Log error, continue shutdown
- Connection close failure: Log error, continue shutdown
- Database close failure: Log error, exit anyway

## Testing

### Unit Tests
- Test individual components in isolation
- Mock dependencies for fast execution
- Focus on business logic

### Integration Tests
- Test component interaction
- Use real database (test instance)
- Verify end-to-end workflows

### System Tests
- Test complete system startup/shutdown
- Verify graceful shutdown behavior
- Test configuration loading

Run integration tests:
```bash
# Run all tests including integration tests
go test ./internal/... -v

# Run only integration tests
go test ./internal/integration_test.go -v

# Skip integration tests (fast)
go test ./internal/... -short
```

## Monitoring

### Metrics Available

- Total connections
- Connected connections
- Total sync jobs
- Running jobs
- Completed jobs
- Failed jobs

### Logging

All components use structured logging (logrus):
- JSON format for production
- Configurable log levels
- Request/response logging
- Error tracking with context

## Best Practices

1. **Always use context for cancellation**
   - Pass context through all layers
   - Respect context cancellation
   - Use timeouts for long operations

2. **Handle errors gracefully**
   - Log errors with context
   - Return meaningful error messages
   - Don't panic in production code

3. **Clean up resources**
   - Close database connections
   - Cancel goroutines
   - Release locks

4. **Use dependency injection**
   - Pass dependencies explicitly
   - Avoid global state
   - Make testing easier

5. **Follow shutdown order**
   - Stop accepting new work first
   - Wait for current work to complete
   - Clean up resources last

## Troubleshooting

### Server won't start
- Check if port is already in use
- Verify database connectivity
- Check configuration file syntax
- Review logs for error messages

### Sync system not working
- Verify sync.enabled = true in config
- Check database migrations ran successfully
- Verify sync tables exist
- Check job engine is running

### Graceful shutdown not working
- Increase shutdown timeout
- Check for stuck goroutines
- Review job cancellation logic
- Check database connection cleanup

## Future Improvements

1. **Service Discovery**
   - Support for dynamic service registration
   - Health check integration with load balancers

2. **Metrics Export**
   - Prometheus metrics endpoint
   - Custom metric collectors

3. **Distributed Tracing**
   - OpenTelemetry integration
   - Request tracing across components

4. **Configuration Hot Reload**
   - Watch configuration file for changes
   - Reload without restart (where possible)
