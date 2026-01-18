# Migration Quick Start Guide

## TL;DR

Migrations run automatically when you start the application. For manual control, use the CLI tool or convenience scripts.

## Quick Commands

### Run Migrations

```bash
# Automatic (recommended)
./db-taxi -host localhost -user root -password secret -database mydb

# Manual
make migrate HOST=localhost USER=root PASSWORD=secret DB=mydb

# Or using script
./scripts/migrate.sh -h localhost -u root -P secret -d mydb

# Or using Go directly
go run cmd/migrate/main.go -host localhost -user root -password secret -database mydb
```

### Check Status

```bash
make migrate-status HOST=localhost USER=root PASSWORD=secret DB=mydb
```

### Get Version

```bash
make migrate-version HOST=localhost USER=root PASSWORD=secret DB=mydb
```

## Environment Variables

```bash
export DBT_DATABASE_HOST=localhost
export DBT_DATABASE_PORT=3306
export DBT_DATABASE_USERNAME=root
export DBT_DATABASE_PASSWORD=secret
export DBT_DATABASE_DATABASE=mydb

# Then just run
make migrate
```

## Create New Migration

1. Create file: `internal/migration/sql/00X_description.sql`
2. Add metadata:
   ```sql
   -- Version: X
   -- Name: description
   -- Description: What this does
   
   CREATE TABLE IF NOT EXISTS ...
   ```
3. Test: `make migrate HOST=localhost USER=root DB=test_db`
4. Commit: `git add internal/migration/sql/00X_description.sql`

## Current Migrations

1. **b-002_create_sync_tables.sql** - Core sync system tables
2. **002_add_initial_data.sql** - Initial data and indexes

## Common Issues

### "Sync tables not found"
**Solution**: Run migrations: `make migrate`

### "Migration failed"
**Solution**: Check logs, fix SQL, retry

### "Version conflict"
**Solution**: Renumber migration file

## Production Deployment

1. Backup database: `mysqldump -u root -p mydb > backup.sql`
2. Deploy application (migrations run automatically)
3. Verify: `make migrate-status CONFIG=production.yaml`
4. Monitor logs

## Need Help?

See full documentation: [docs/MIGRATIONS.md](./MIGRATIONS.md)
