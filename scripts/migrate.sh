#!/bin/bash

# DB-Taxi Migration Script
# This script provides a convenient way to run database migrations

set -e

# Default values
COMMAND="migrate"
CONFIG_FILE=""
HOST="${DBT_DATABASE_HOST:-localhost}"
PORT="${DBT_DATABASE_PORT:-3306}"
USER="${DBT_DATABASE_USERNAME:-root}"
PASSWORD="${DBT_DATABASE_PASSWORD:-}"
DATABASE="${DBT_DATABASE_DATABASE:-}"
VERSION="0"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
DB-Taxi Migration Script

Usage: $0 [options]

Commands:
    migrate         Run all pending migrations (default)
    status          Show migration status
    version         Show current migration version

Options:
    -c, --config FILE       Path to configuration file
    -h, --host HOST         Database host (default: localhost)
    -p, --port PORT         Database port (default: 3306)
    -u, --user USER         Database username (default: root)
    -P, --password PASS     Database password
    -d, --database DB       Database name
    -v, --version NUM       Target version for migration (0 = latest)
    --help                  Show this help message

Environment Variables:
    DBT_DATABASE_HOST       Database host
    DBT_DATABASE_PORT       Database port
    DBT_DATABASE_USERNAME   Database username
    DBT_DATABASE_PASSWORD   Database password
    DBT_DATABASE_DATABASE   Database name

Examples:
    # Run all pending migrations
    $0 -h localhost -u root -P secret -d mydb

    # Check migration status
    $0 status -c config.yaml

    # Migrate to specific version
    $0 migrate -v 1 -h localhost -u root -d mydb

    # Using environment variables
    export DBT_DATABASE_HOST=localhost
    export DBT_DATABASE_USERNAME=root
    export DBT_DATABASE_PASSWORD=secret
    export DBT_DATABASE_DATABASE=mydb
    $0 migrate

EOF
}

# Parse command line arguments
POSITIONAL=()
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        migrate|status|version)
            COMMAND="$1"
            shift
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -h|--host)
            HOST="$2"
            shift 2
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -u|--user)
            USER="$2"
            shift 2
            ;;
        -P|--password)
            PASSWORD="$2"
            shift 2
            ;;
        -d|--database)
            DATABASE="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            POSITIONAL+=("$1")
            shift
            ;;
    esac
done

# Restore positional parameters
set -- "${POSITIONAL[@]}"

# Validate required parameters
if [ -z "$DATABASE" ] && [ -z "$CONFIG_FILE" ]; then
    print_error "Database name is required (use -d or set DBT_DATABASE_DATABASE)"
    echo ""
    show_usage
    exit 1
fi

# Build command arguments
ARGS="-command $COMMAND"

if [ -n "$CONFIG_FILE" ]; then
    ARGS="$ARGS -config $CONFIG_FILE"
fi

if [ -n "$HOST" ]; then
    ARGS="$ARGS -host $HOST"
fi

if [ -n "$PORT" ]; then
    ARGS="$ARGS -port $PORT"
fi

if [ -n "$USER" ]; then
    ARGS="$ARGS -user $USER"
fi

if [ -n "$PASSWORD" ]; then
    ARGS="$ARGS -password $PASSWORD"
fi

if [ -n "$DATABASE" ]; then
    ARGS="$ARGS -database $DATABASE"
fi

if [ "$VERSION" != "0" ]; then
    ARGS="$ARGS -version $VERSION"
fi

# Print info
print_info "Running migration command: $COMMAND"
if [ -n "$CONFIG_FILE" ]; then
    print_info "Using config file: $CONFIG_FILE"
else
    print_info "Database: $USER@$HOST:$PORT/$DATABASE"
fi

# Change to project root directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
cd "$PROJECT_ROOT"

# Run migration
print_info "Executing migration..."
if go run cmd/migrate/main.go $ARGS; then
    print_info "Migration completed successfully"
    exit 0
else
    print_error "Migration failed"
    exit 1
fi
