# DB-Taxi MySQL Web Explorer

A web-based MySQL database management tool built with Go and Gin framework.

## Features

- Web-based MySQL database management
- RESTful API for database operations
- Configurable via YAML files and environment variables
- Health check and monitoring endpoints
- Graceful shutdown support

## Quick Start

### Prerequisites

- Go 1.21 or later
- MySQL database (for connection testing)

### Installation

1. Clone the repository
2. Navigate to the project directory
3. Install dependencies:
   ```bash
   go mod tidy
   ```

### Configuration

1. Copy the example configuration file:
   ```bash
   cp config.yaml.example config.yaml
   ```

2. Edit `config.yaml` to match your environment

3. Alternatively, use environment variables with `DBT_` prefix:
   ```bash
   export DBT_SERVER_PORT=8080
   export DBT_LOGGING_LEVEL=debug
   ```

### Running

Start the server:
```bash
go run main.go
```

The server will start on `http://localhost:8080` by default.

### API Endpoints

- `GET /health` - Health check endpoint
- `GET /api/status` - Server status information
- `GET /` - Web interface (static files)

## Configuration Options

### Server Configuration
- `server.port`: HTTP server port (default: 8080)
- `server.host`: HTTP server host (default: 0.0.0.0)
- `server.read_timeout`: HTTP read timeout (default: 30s)
- `server.write_timeout`: HTTP write timeout (default: 30s)
- `server.enable_https`: Enable HTTPS (default: false)

### Database Configuration
- `database.max_open_conns`: Maximum open connections (default: 25)
- `database.max_idle_conns`: Maximum idle connections (default: 5)
- `database.conn_max_lifetime`: Connection max lifetime (default: 5m)
- `database.query_timeout`: Query timeout (default: 30s)

### Security Configuration
- `security.session_timeout`: Session timeout (default: 30m)
- `security.read_only_mode`: Enable read-only mode (default: false)
- `security.enable_audit`: Enable audit logging (default: true)

### Logging Configuration
- `logging.level`: Log level (debug, info, warn, error) (default: info)
- `logging.format`: Log format (json, text) (default: json)
- `logging.output`: Log output (stdout, stderr, or file path) (default: stdout)

## Development

This project follows the implementation plan defined in the specification documents. The current implementation includes:

- ✅ Project initialization and basic architecture setup
- ⏳ Database connection management (next task)
- ⏳ Session management system
- ⏳ Database metadata explorer
- ⏳ SQL query engine
- ⏳ Data export functionality
- ⏳ Web interface and user experience

## License

This project is part of the DB-Taxi MySQL Web Explorer implementation.