# DB-Taxi

[中文](README.zh.md) | English

![image](https://raw.githubusercontent.com/i-cooltea/resource/refs/heads/master/image/db-taxi-logo.png)

A convenient and practical MySQL data cloning tool.

## Features

- 🔌 **Database Connection Management** - Support for MySQL database connections, including connection pool management
- 🔄 **Database Synchronization** - Support for full database sync, table-specific sync, and conditional sync
- 📦 **Batch Operations** - Support for batch data transfer and batch processing
- 🔍 **Sync Monitoring** - Real-time monitoring of sync task status and progress
- 🌐 **Web Interface** - Modern responsive Web interface (Vue 3 + Vite)
- ⚡ **High Performance** - Built with Go and Gin framework, supports high concurrency
- 🔧 **Flexible Configuration** - Support for configuration files and environment variables

## Quick Start

### 1. Configure Database Connection

There are multiple ways to configure database connections:

#### Method 1: Using Configuration File
```bash
# Copy configuration file template
cp config.yaml.example config.yaml

# Edit configuration file
vim config.yaml

./db-taxi -config /path/to/your/config.yaml
```

#### Method 2: Using Command Line Arguments
```bash
./db-taxi -host localhost -port 3306 -user root -password secret -database mydb
```

#### Method 3: Using Environment Variables
```bash
export DBT_DATABASE_HOST=localhost
export DBT_DATABASE_PORT=3306
export DBT_DATABASE_USERNAME=root
export DBT_DATABASE_PASSWORD=secret
export DBT_DATABASE_DATABASE=mydb
./db-taxi
```

### 2. Build and Run

```bash
# Install dependencies
go mod tidy

# Build project
go build -o db-taxi .

# Run (using default configuration)
./db-taxi

# Or use command line arguments
./db-taxi -host localhost -user root -password secret -database mydb -server-port 9090
```

### 3. Access Web Interface

Open your browser and visit: http://localhost:8080

## Database Migration

DB-Taxi includes an automatic database migration system that creates and updates required database tables on application startup.

### Automatic Migration (Recommended)

Migrations run automatically when the application starts:

```bash
./db-taxi -host localhost -user root -password secret -database mydb
```

### Manual Migration

To manually control migrations, use the following commands:

```bash
# Run all pending migrations
make migrate HOST=localhost USER=root PASSWORD=secret DB=mydb

# Check migration status
make migrate-status HOST=localhost USER=root DB=mydb

# View current version
make migrate-version HOST=localhost USER=root DB=mydb
```

Or use the convenience script:

```bash
./scripts/migrate.sh -h localhost -u root -P secret -d mydb
```

For detailed documentation, please refer to:
- [Complete Migration Documentation](docs/MIGRATIONS.md)
- [Quick Start Guide](docs/MIGRATION_QUICK_START.md)

## Sync Feature Usage

DB-Taxi provides powerful database synchronization features, supporting multi-database connection management and selective table synchronization.

### Quick Start for Synchronization

1. **Add Remote Connection**: Add remote database connections on the "Connection Management" page in the Web interface
2. **Configure Sync**: Select tables to sync, set sync mode (full/incremental)
3. **Start Sync**: Click "Start Sync" to begin data synchronization
4. **Monitor Progress**: View real-time progress and logs on the "Sync Monitoring" page

### Sync Features

- ✅ Multi-database instance connection management
- ✅ Selective table synchronization
- ✅ Full and incremental sync modes
- ✅ Real-time progress monitoring
- ✅ Sync failure support with error information viewing
- ✅ Configuration import/export
- ✅ Batch operations and performance optimization
- ✅ Scheduled synchronization

For detailed usage guide, please refer to:
- [Sync Feature User Guide](docs/SYNC_USER_GUIDE.md)
- [API Documentation](docs/API.md)

## Command Line Options

```bash
db-taxi [options]

Configuration Options:
  -config string      Specify configuration file path
  -host string       Database host address
  -port int          Database port
  -user string       Database username
  -password string   Database password
  -database string   Database name
  -ssl               Enable SSL connection
  -server-port int   Web server port
  -help              Show help information
```

## Usage Examples

### Basic Usage
```bash
# Use default configuration file
./db-taxi

# Show help
./db-taxi -help
```

### Specify Configuration File
```bash
# Use custom configuration file
./db-taxi -config /etc/db-taxi/production.yaml

# Use preset configuration files
./db-taxi -config configs/local.yaml      # Local development
./db-taxi -config configs/production.yaml # Production environment
```

### Command Line Parameter Override
```bash
# Specify completely via command line
./db-taxi -host 192.168.1.100 -port 3306 -user admin -password secret123 -database myapp

# Use configuration file but override some parameters
./db-taxi -config configs/local.yaml -password newsecret -server-port 9090

# Mix environment variables and command line parameters
export DBT_DATABASE_HOST=remote-mysql
./db-taxi -user admin -password secret -database production_db
```

### Production Deployment
```bash
# Use environment variables (recommended for production)
export DBT_DATABASE_HOST=mysql-server.internal
export DBT_DATABASE_USERNAME=app_user
export DBT_DATABASE_PASSWORD=secure_password
export DBT_DATABASE_DATABASE=production_db
export DBT_SERVER_PORT=8080
./db-taxi -config configs/production.yaml
```

## Environment Variable Configuration

You can also configure the application using environment variables:

```bash
export DBT_DATABASE_HOST=localhost
export DBT_DATABASE_PORT=3306
export DBT_DATABASE_USERNAME=root
export DBT_DATABASE_PASSWORD=your_password
export DBT_DATABASE_DATABASE=your_database
export DBT_SERVER_PORT=8080
```

## API Endpoints

### Health Check
- `GET /health` - Server health check

### Database Operations
- `GET /api/status` - Get server and database status
- `GET /api/connection/test` - Test database connection
- `GET /api/databases` - Get database list
- `GET /api/databases/{database}/tables` - Get table list for specified database
- `GET /api/databases/{database}/tables/{table}` - Get table details
- `GET /api/databases/{database}/tables/{table}/data` - Get table data (supports pagination)

### Sync System API
- `GET /api/sync/status` - Get sync system status
- `GET /api/sync/stats` - Get sync system statistics

#### Connection Management
- `GET /api/sync/connections` - Get all sync connections
- `POST /api/sync/connections` - Create new sync connection
- `GET /api/sync/connections/{id}` - Get specified connection details
- `PUT /api/sync/connections/{id}` - Update connection configuration
- `DELETE /api/sync/connections/{id}` - Delete connection
- `POST /api/sync/connections/{id}/test` - Test connection

#### Sync Configuration
- `GET /api/sync/configs` - Get sync configuration list
- `POST /api/sync/configs` - Create sync configuration
- `GET /api/sync/configs/{id}` - Get configuration details
- `PUT /api/sync/configs/{id}` - Update configuration
- `DELETE /api/sync/configs/{id}` - Delete configuration

#### Job Management
- `GET /api/sync/jobs` - Get sync job list
- `POST /api/sync/jobs` - Start new sync job
- `GET /api/sync/jobs/{id}` - Get job details
- `POST /api/sync/jobs/{id}/stop` - Stop job
- `GET /api/sync/jobs/{id}/logs` - Get job logs

#### Configuration Management
- `GET /api/sync/config/export` - Export sync configuration
- `POST /api/sync/config/import` - Import sync configuration
- `POST /api/sync/config/validate` - Validate configuration file

### Query Parameters
- `limit` - Limit number of records returned (default: 10, max: 1000)
- `offset` - Offset (default: 0)

## Project Structure

```
db-taxi/
├── main.go                    # Application entry point
├── config.yaml.example       # Configuration file template
├── static/                    # Static files
│   └── index.html            # Web interface
├── frontend/                  # Vue 3 frontend application
│   ├── src/
│   │   ├── components/       # Vue components
│   │   ├── views/            # Page views
│   │   ├── stores/           # State management
│   │   └── router/           # Router configuration
│   └── package.json
├── internal/
│   ├── config/               # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── server/               # HTTP server
│   │   ├── server.go
│   │   ├── middleware.go
│   │   └── server_test.go
│   ├── database/             # Database operations
│   │   ├── connection.go     # Connection pool management
│   │   ├── schema.go         # Database schema exploration
│   │   └── connection_test.go
│   ├── sync/                 # Sync system
│   │   ├── sync.go           # Sync manager
│   │   ├── interfaces.go     # Interface definitions
│   │   ├── repository.go     # Data access layer
│   │   ├── service.go        # Business logic layer
│   │   ├── job_engine.go     # Job engine
│   │   ├── sync_engine.go    # Sync engine
│   │   └── mapping_manager.go # Mapping manager
│   ├── migration/            # Database migration
│   │   ├── migration.go
│   │   └── sql/              # SQL migration files
│   └── integration_test.go   # Integration tests
├── docs/                     # Documentation
│   ├── SYSTEM_INTEGRATION.md # System integration documentation
│   ├── MIGRATIONS.md         # Migration documentation
│   └── MIGRATION_QUICK_START.md
├── scripts/                  # Scripts
│   ├── migrate.sh            # Migration script
│   └── verify-integration.sh # Integration verification script
└── go.mod                    # Go module definition
```

## Configuration Options

### Server Configuration
- `server.port` - Server port (default: 8080)
- `server.host` - Server host (default: 0.0.0.0)
- `server.read_timeout` - Read timeout
- `server.write_timeout` - Write timeout

### Database Configuration
- `database.host` - MySQL host address
- `database.port` - MySQL port
- `database.username` - Username
- `database.password` - Password
- `database.database` - Database name
- `database.ssl` - Enable SSL
- `database.max_open_conns` - Maximum open connections
- `database.max_idle_conns` - Maximum idle connections
- `database.conn_max_lifetime` - Connection maximum lifetime

### Security Configuration
- `security.session_timeout` - Session timeout
- `security.read_only_mode` - Read-only mode
- `security.enable_audit` - Enable audit logging

### Logging Configuration
- `logging.level` - Log level (debug, info, warn, error)
- `logging.format` - Log format (json, text)
- `logging.output` - Log output (stdout, stderr, file path)

### Sync System Configuration
- `sync.enabled` - Enable sync system (default: true)
- `sync.max_concurrency` - Maximum concurrent sync tasks (default: 5)
- `sync.batch_size` - Batch operation size (default: 1000)
- `sync.retry_attempts` - Retry attempts (default: 3)
- `sync.retry_delay` - Retry delay (default: 30s)
- `sync.job_timeout` - Job timeout (default: 1h)
- `sync.cleanup_age` - History cleanup time (default: 720h)

## Development

### Running Tests
```bash
# Run all tests
go test ./...

# Run unit tests (skip integration tests)
go test ./... -short

# Run integration tests
go test ./internal/integration_test.go -v

# Run tests for specific package
go test ./internal/sync/... -v
```

### Verify System Integration
```bash
# Run integration verification script
./scripts/verify-integration.sh
```

### Frontend Development
```bash
# Enter frontend directory
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Build production version
npm run build

# Build frontend code to static folder (run from project root)
make build-frontend
```

### Docker Deployment
```bash
# Use Docker Compose (includes MySQL)
docker-compose up -d

# Or build and run separately
docker build -t db-taxi .
docker run -p 8080:8080 \
  -e DBT_DATABASE_HOST=your-mysql-host \
  -e DBT_DATABASE_USERNAME=root \
  -e DBT_DATABASE_PASSWORD=secret \
  -e DBT_DATABASE_DATABASE=mydb \
  db-taxi
```

### Quick Start Scripts
```bash
# Local development
chmod +x scripts/start-local.sh
./scripts/start-local.sh

# Production environment
export DB_PASSWORD=your_production_password
chmod +x scripts/start-production.sh
./scripts/start-production.sh
```

## Tech Stack

- **Backend**: Go 1.21+, Gin Web Framework
- **Database**: MySQL 5.7+
- **Frontend**: Vue 3, Vite, Vue Router, Pinia
- **Dependency Management**: Go Modules, npm

## Dependencies

- `github.com/gin-gonic/gin` - Web framework
- `github.com/jmoiron/sqlx` - SQL extension library
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/sirupsen/logrus` - Logging library
- `github.com/spf13/viper` - Configuration management

## Implementation Status

Based on the implementation plan in the specification document, the current implementation includes:

### Core Features
- ✅ Project initialization and infrastructure setup
- ✅ Database connection pool manager
- ✅ Database metadata explorer (Schema Explorer)
- ✅ Web interface and user experience (Vue 3 + Vite)
- ✅ REST API interface implementation

### Sync System
- ✅ Connection Manager
- ✅ Sync Manager
- ✅ Mapping Manager
- ✅ Job Engine
- ✅ Sync Engine
- ✅ Batch processing and performance optimization
- ✅ Error handling and recovery mechanisms
- ✅ Configuration import/export
- ✅ Real-time monitoring and statistics

### System Integration
- ✅ All components dependency injection
- ✅ System startup and shutdown logic
- ✅ Database migration system
- ✅ Health checks and monitoring
- ✅ Integration tests

### Features to be Implemented
- ⏳ Session management system
- ⏳ SQL query engine
- ⏳ Data export functionality

For detailed system integration documentation, please refer to: [SYSTEM_INTEGRATION.md](docs/SYSTEM_INTEGRATION.md)

## License

MIT License

## Contributing

Welcome to submit Issues and Pull Requests!

## Support

If you encounter any issues, please check:
1. Ensure MySQL service is running
2. Check database connection information in configuration file
3. View application logs for detailed error information
4. Run integration verification script: `./scripts/verify-integration.sh`
5. View system integration documentation: [SYSTEM_INTEGRATION.md](docs/SYSTEM_INTEGRATION.md)
