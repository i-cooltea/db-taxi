# Database Migrations

This directory contains database migration scripts for the DB-Taxi synchronization system.

## Overview

The migration system provides:
- **Version Management**: Track which migrations have been applied
- **Automatic Execution**: Migrations run automatically on application startup
- **Manual Control**: CLI tool for manual migration management
- **Rollback Safety**: Transactions ensure atomic migration application

## Migration Files

Migrations are stored in `internal/migration/sql/` and are embedded in the application binary.

### Current Migrations

1. **001_create_sync_tables.sql** - Creates all core synchronization tables
   - connections
   - sync_configs
   - table_mappings
   - sync_jobs
   - sync_logs
   - sync_checkpoints
   - database_mappings

2. **002_add_initial_data.sql** - Adds initial data and optimizes indexes
   - Additional composite indexes
   - Performance optimization indexes

## Migration File Format

Each migration file must include metadata in SQL comments:

```sql
-- Version: 1
-- Name: create_sync_tables
-- Description: Creates all tables required for the database synchronization system

-- SQL statements here...
```

## Automatic Migrations

Migrations run automatically when the application starts. The sync system will:
1. Create the `schema_migrations` table if it doesn't exist
2. Check for pending migrations
3. Apply migrations in order
4. Record each migration with timestamp and checksum

## Manual Migration Management

Use the migration CLI tool for manual control:

### Run All Pending Migrations

```bash
go run cmd/migrate/main.go \
  -host localhost \
  -port 3306 \
  -user root \
  -password secret \
  -database mydb
```

### Check Migration Status

```bash
go run cmd/migrate/main.go \
  -command status \
  -host localhost \
  -user root \
  -database mydb
```

### Get Current Version

```bash
go run cmd/migrate/main.go \
  -command version \
  -host localhost \
  -user root \
  -database mydb
```

### Migrate to Specific Version

```bash
go run cmd/migrate/main.go \
  -version 1 \
  -host localhost \
  -user root \
  -database mydb
```

### Using Configuration File

```bash
go run cmd/migrate/main.go \
  -config configs/config.yaml \
  -command status
```

## Schema Migrations Table

The system creates a `schema_migrations` table to track applied migrations:

```sql
CREATE TABLE schema_migrations (
    version INT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64) NOT NULL,
    execution_time_ms INT NOT NULL
);
```

## Creating New Migrations

To create a new migration:

1. Create a new SQL file in `internal/migration/sql/`
2. Name it with the next version number: `00X_description.sql`
3. Add metadata comments at the top:
   ```sql
   -- Version: X
   -- Name: description
   -- Description: What this migration does
   ```
4. Write your SQL statements
5. Test the migration in a development environment
6. Commit the file to version control

## Best Practices

1. **Incremental Changes**: Each migration should be small and focused
2. **Idempotent**: Use `IF NOT EXISTS` where possible
3. **Reversible**: Consider how to undo changes if needed
4. **Test First**: Always test migrations in development
5. **Backup**: Backup production databases before running migrations
6. **Transactions**: Migrations run in transactions for safety

## Troubleshooting

### Migration Fails

If a migration fails:
1. Check the error message in the logs
2. Verify database connectivity
3. Check for syntax errors in the SQL
4. Ensure the database user has sufficient privileges

### Reset Migrations (Development Only)

To reset migrations in development:

```sql
-- Drop all sync tables
DROP TABLE IF EXISTS sync_logs;
DROP TABLE IF EXISTS sync_checkpoints;
DROP TABLE IF EXISTS sync_jobs;
DROP TABLE IF EXISTS table_mappings;
DROP TABLE IF EXISTS database_mappings;
DROP TABLE IF EXISTS sync_configs;
DROP TABLE IF EXISTS connections;
DROP TABLE IF EXISTS schema_migrations;
```

Then run migrations again.

## Environment Variables

All database configuration can be set via environment variables:

```bash
export DBT_DATABASE_HOST=localhost
export DBT_DATABASE_PORT=3306
export DBT_DATABASE_USERNAME=root
export DBT_DATABASE_PASSWORD=secret
export DBT_DATABASE_DATABASE=mydb

go run cmd/migrate/main.go
```

## Integration with Application

The migration system is integrated into the main application:

1. On startup, the sync manager calls `runMigrations()`
2. The migration manager checks for pending migrations
3. Migrations are applied automatically
4. Application continues startup after migrations complete

This ensures the database schema is always up-to-date.
