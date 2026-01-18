# Database Migration System

## Overview

DB-Taxi includes a comprehensive database migration system that manages schema changes and ensures database consistency across environments. The system provides automatic migration execution on application startup and manual control through CLI tools.

## Features

- **Automatic Execution**: Migrations run automatically when the application starts
- **Version Tracking**: Track which migrations have been applied with timestamps
- **Transaction Safety**: Each migration runs in a transaction for atomicity
- **Checksum Validation**: Detect if migration files have been modified
- **Manual Control**: CLI tool for manual migration management
- **Embedded Migrations**: Migration files are embedded in the binary
- **Rollback Safety**: Failed migrations are automatically rolled back

## Architecture

### Components

1. **Migration Manager** (`internal/migration/migration.go`)
   - Core migration logic
   - Version tracking
   - Transaction management
   - Checksum validation

2. **Migration Files** (`internal/migration/sql/*.sql`)
   - SQL migration scripts
   - Embedded in application binary
   - Versioned and ordered

3. **CLI Tool** (`cmd/migrate/main.go`)
   - Manual migration control
   - Status checking
   - Version management

4. **Schema Migrations Table**
   - Tracks applied migrations
   - Stores checksums and execution times
   - Provides audit trail

### Migration Flow

```
Application Start
    ↓
Initialize Migration System
    ↓
Create schema_migrations table (if not exists)
    ↓
Load Available Migrations (from embedded files)
    ↓
Query Applied Migrations (from database)
    ↓
Calculate Pending Migrations
    ↓
Apply Each Pending Migration in Transaction
    ↓
Record Migration in schema_migrations
    ↓
Continue Application Startup
```

## Migration Files

### File Location

Migrations are stored in `internal/migration/sql/` and embedded in the application binary using Go's `embed` package.

### File Naming Convention

```
XXX_description.sql
```

- `XXX`: Three-digit version number (001, 002, 003, etc.)
- `description`: Brief description using underscores

Examples:
- `b-002_create_sync_tables.sql`
- `002_add_initial_data.sql`
- `003_add_performance_indexes.sql`

### File Format

Each migration file must include metadata in SQL comments:

```sql
-- Version: 1
-- Name: create_sync_tables
-- Description: Creates all tables required for the database synchronization system

-- SQL statements here
CREATE TABLE IF NOT EXISTS connections (
    id VARCHAR(36) PRIMARY KEY,
    ...
);
```

**Required Metadata:**
- `Version`: Integer version number (must match filename)
- `Name`: Short name for the migration
- `Description`: Detailed description of what the migration does

## Schema Migrations Table

The system automatically creates a `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version INT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64) NOT NULL,
    execution_time_ms INT NOT NULL,
    INDEX idx_applied_at (applied_at)
);
```

**Columns:**
- `version`: Migration version number (primary key)
- `name`: Migration name
- `description`: Migration description
- `applied_at`: When the migration was applied
- `checksum`: Checksum of migration content
- `execution_time_ms`: How long the migration took to execute

## Usage

### Automatic Migrations (Recommended)

Migrations run automatically when the application starts:

```bash
./db-taxi -host localhost -user root -password secret -database mydb
```

The application will:
1. Connect to the database
2. Initialize the migration system
3. Apply any pending migrations
4. Continue normal startup

### Manual Migration CLI

For manual control, use the migration CLI tool:

#### Run All Pending Migrations

```bash
go run cmd/migrate/main.go \
  -host localhost \
  -port 3306 \
  -user root \
  -password secret \
  -database mydb
```

#### Check Migration Status

```bash
go run cmd/migrate/main.go \
  -command status \
  -host localhost \
  -user root \
  -password secret \
  -database mydb
```

Output:
```
Current version: 2
Applied migrations: 2
Pending migrations: 0
```

#### Get Current Version

```bash
go run cmd/migrate/main.go \
  -command version \
  -host localhost \
  -user root \
  -database mydb
```

#### Migrate to Specific Version

```bash
go run cmd/migrate/main.go \
  -version 1 \
  -host localhost \
  -user root \
  -database mydb
```

#### Using Configuration File

```bash
go run cmd/migrate/main.go \
  -config configs/config.yaml \
  -command status
```

### Convenience Script

Use the shell script for easier migration management:

```bash
# Run migrations
./scripts/migrate.sh -h localhost -u root -P secret -d mydb

# Check status
./scripts/migrate.sh status -h localhost -u root -P secret -d mydb

# Get version
./scripts/migrate.sh version -c config.yaml
```

### Makefile Targets

Use make commands for common operations:

```bash
# Run migrations
make migrate HOST=localhost USER=root PASSWORD=secret DB=mydb

# Check status
make migrate-status CONFIG=config.yaml

# Get version
make migrate-version HOST=localhost USER=root DB=mydb
```

### Environment Variables

Set database configuration via environment variables:

```bash
export DBT_DATABASE_HOST=localhost
export DBT_DATABASE_PORT=3306
export DBT_DATABASE_USERNAME=root
export DBT_DATABASE_PASSWORD=secret
export DBT_DATABASE_DATABASE=mydb

# Then run migrations
go run cmd/migrate/main.go
# or
./scripts/migrate.sh migrate
# or
make migrate
```

## Creating New Migrations

### Step 1: Create Migration File

Create a new file in `internal/migration/sql/`:

```bash
touch internal/migration/sql/003_add_new_feature.sql
```

### Step 2: Add Metadata and SQL

```sql
-- Version: 3
-- Name: add_new_feature
-- Description: Adds tables and indexes for new feature

CREATE TABLE IF NOT EXISTS new_feature (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_new_feature_name ON new_feature(name);
```

### Step 3: Test in Development

```bash
# Test the migration
make migrate HOST=localhost USER=root PASSWORD=devpass DB=dev_db

# Verify it worked
make migrate-status HOST=localhost USER=root PASSWORD=devpass DB=dev_db
```

### Step 4: Commit to Version Control

```bash
git add internal/migration/sql/003_add_new_feature.sql
git commit -m "Add migration for new feature"
```

## Best Practices

### 1. Incremental Changes

Keep migrations small and focused:
- ✅ One feature per migration
- ✅ Related changes together
- ❌ Multiple unrelated changes

### 2. Idempotent Operations

Use `IF NOT EXISTS` and `IF EXISTS`:

```sql
-- Good
CREATE TABLE IF NOT EXISTS users (...);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255);

-- Avoid
CREATE TABLE users (...);  -- Will fail if table exists
```

### 3. Backward Compatibility

Consider backward compatibility:
- Add columns with defaults
- Don't remove columns immediately
- Use deprecation periods

### 4. Test Thoroughly

Always test migrations:
1. Test in development environment
2. Test on copy of production data
3. Verify rollback works
4. Check performance impact

### 5. Document Changes

Include clear descriptions:
```sql
-- Version: 3
-- Name: add_user_email_index
-- Description: Adds index on users.email for faster login queries
--              Expected to improve login performance by 50%
--              Safe to run on production (non-blocking index creation)
```

### 6. Handle Large Tables

For large tables, consider:
- Online schema changes
- Batched updates
- Off-peak execution

```sql
-- For large tables, use ALGORITHM=INPLACE
ALTER TABLE large_table 
ADD COLUMN new_column VARCHAR(255),
ALGORITHM=INPLACE, LOCK=NONE;
```

## Troubleshooting

### Migration Fails

**Symptom**: Migration fails with error

**Solutions**:
1. Check error message in logs
2. Verify database connectivity
3. Check SQL syntax
4. Ensure user has sufficient privileges
5. Check for conflicting data

**Recovery**:
```bash
# Check current status
make migrate-status

# Fix the migration file
# Then retry
make migrate
```

### Checksum Mismatch

**Symptom**: Warning about checksum mismatch

**Cause**: Migration file was modified after being applied

**Solution**: Don't modify applied migrations. Create a new migration instead.

### Version Conflict

**Symptom**: Version number already exists

**Cause**: Two migrations with same version number

**Solution**: Renumber the newer migration file

### Database Locked

**Symptom**: Migration times out or fails with lock error

**Cause**: Long-running queries or transactions

**Solution**:
1. Wait for queries to complete
2. Kill blocking queries
3. Retry migration

### Reset Migrations (Development Only)

To completely reset migrations in development:

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

Then run migrations again:
```bash
make migrate
```

## Production Deployment

### Pre-Deployment Checklist

- [ ] Test migrations in staging environment
- [ ] Backup production database
- [ ] Review migration SQL for performance impact
- [ ] Check for blocking operations
- [ ] Plan rollback strategy
- [ ] Schedule during maintenance window if needed

### Deployment Process

1. **Backup Database**
   ```bash
   mysqldump -u root -p mydb > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **Deploy New Application Version**
   ```bash
   # Application will run migrations automatically on startup
   ./db-taxi -config production.yaml
   ```

3. **Verify Migrations**
   ```bash
   make migrate-status CONFIG=production.yaml
   ```

4. **Monitor Application**
   - Check logs for errors
   - Verify functionality
   - Monitor performance

### Rollback Strategy

If migration causes issues:

1. **Stop Application**
2. **Restore Database Backup**
   ```bash
   mysql -u root -p mydb < backup_20240111_120000.sql
   ```
3. **Deploy Previous Application Version**

## Advanced Topics

### Custom Migration Logic

For complex migrations, you can extend the migration system:

```go
// In your application code
migrationManager := migration.NewManager(db, logger)

// Run migrations up to specific version
if err := migrationManager.MigrateToVersion(ctx, 2); err != nil {
    log.Fatal(err)
}

// Get current version
version, err := migrationManager.GetCurrentVersion(ctx)
```

### Migration Hooks

Add custom logic before/after migrations:

```go
// Before migrations
log.Info("Preparing for migrations...")

// Run migrations
if err := migrationManager.Migrate(ctx); err != nil {
    return err
}

// After migrations
log.Info("Migrations complete, running post-migration tasks...")
```

### Monitoring

Monitor migration execution:
- Check `schema_migrations` table for history
- Monitor `execution_time_ms` for performance
- Alert on failed migrations

```sql
-- Get migration history
SELECT version, name, applied_at, execution_time_ms
FROM schema_migrations
ORDER BY version DESC;

-- Find slow migrations
SELECT version, name, execution_time_ms
FROM schema_migrations
WHERE execution_time_ms > 1000
ORDER BY execution_time_ms DESC;
```

## FAQ

**Q: Do migrations run automatically?**
A: Yes, migrations run automatically when the application starts.

**Q: Can I run migrations manually?**
A: Yes, use the CLI tool: `go run cmd/migrate/main.go`

**Q: What happens if a migration fails?**
A: The migration is rolled back and the application reports an error.

**Q: Can I skip a migration?**
A: No, migrations must be applied in order.

**Q: How do I rollback a migration?**
A: Create a new migration that reverses the changes.

**Q: Are migrations transactional?**
A: Yes, each migration runs in a transaction.

**Q: Can I modify an applied migration?**
A: No, create a new migration instead.

**Q: How do I test migrations?**
A: Test in development environment first, then staging.

**Q: What if I need to run migrations on multiple databases?**
A: Run the migration tool for each database separately.

**Q: Can migrations run concurrently?**
A: No, migrations are applied sequentially for safety.
