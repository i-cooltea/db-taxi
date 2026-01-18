#!/bin/bash

# DB-Taxi Configuration Demo Script
echo "🚀 DB-Taxi MySQL Web Explorer - Configuration Demo"
echo "=================================================="

# Build the application first
echo "📦 Building DB-Taxi..."
go build -o db-taxi . || exit 1
echo "✅ Build completed"
echo ""

# Demo 1: Show help
echo "📖 Demo 1: Command Line Help"
echo "----------------------------"
./db-taxi -help
echo ""

# Demo 2: Show default configuration (will fail to connect, but shows config loading)
echo "📋 Demo 2: Default Configuration"
echo "--------------------------------"
echo "Starting with default configuration (will show database connection warning)..."
echo "Press Ctrl+C to stop after a few seconds"
echo ""
timeout 3s ./db-taxi 2>&1 | head -10 || true
echo ""

# Demo 3: Custom config file
echo "📁 Demo 3: Custom Configuration File"
echo "------------------------------------"
echo "Using configs/local.yaml configuration..."
echo "Press Ctrl+C to stop after a few seconds"
echo ""
timeout 3s ./db-taxi -config configs/local.yaml 2>&1 | head -10 || true
echo ""

# Demo 4: Command line overrides
echo "⚙️  Demo 4: Command Line Parameter Overrides"
echo "--------------------------------------------"
echo "Overriding database connection via command line..."
echo "Press Ctrl+C to stop after a few seconds"
echo ""
timeout 3s ./db-taxi -host example.com -port 3306 -user demo -password demo123 -database testdb -server-port 9090 2>&1 | head -10 || true
echo ""

# Demo 5: Environment variables
echo "🌍 Demo 5: Environment Variable Configuration"
echo "---------------------------------------------"
echo "Setting configuration via environment variables..."
export DBT_DATABASE_HOST=env-mysql-host
export DBT_DATABASE_PORT=3306
export DBT_DATABASE_USERNAME=env-user
export DBT_DATABASE_PASSWORD=env-password
export DBT_DATABASE_DATABASE=env-database
export DBT_SERVER_PORT=8888
echo "Environment variables set:"
echo "  DBT_DATABASE_HOST=$DBT_DATABASE_HOST"
echo "  DBT_DATABASE_PORT=$DBT_DATABASE_PORT"
echo "  DBT_DATABASE_USERNAME=$DBT_DATABASE_USERNAME"
echo "  DBT_SERVER_PORT=$DBT_SERVER_PORT"
echo ""
echo "Starting with environment configuration..."
echo "Press Ctrl+C to stop after a few seconds"
echo ""
timeout 3s ./db-taxi 2>&1 | head -10 || true

# Clean up environment variables
unset DBT_DATABASE_HOST DBT_DATABASE_PORT DBT_DATABASE_USERNAME DBT_DATABASE_PASSWORD DBT_DATABASE_DATABASE DBT_SERVER_PORT
echo ""

# Demo 6: Mixed configuration (config file + command line overrides)
echo "🔀 Demo 6: Mixed Configuration (Config File + Command Line)"
echo "-----------------------------------------------------------"
echo "Using config file with command line overrides..."
echo "Press Ctrl+C to stop after a few seconds"
echo ""
timeout 3s ./db-taxi -config configs/local.yaml -password "overridden-password" -server-port 7777 2>&1 | head -10 || true
echo ""

echo "🎉 Configuration demo completed!"
echo ""
echo "💡 Tips:"
echo "   • Use -config to specify custom configuration files"
echo "   • Command line parameters override config file values"
echo "   • Environment variables override config file values"
echo "   • Command line parameters have the highest priority"
echo "   • Use -help to see all available options"
echo ""
echo "🔗 Available configuration files:"
echo "   • configs/local.yaml      - Local development"
echo "   • configs/production.yaml - Production environment"
echo "   • configs/docker.yaml     - Docker container"
echo ""
echo "🌐 Once configured, access the web interface at: http://localhost:PORT"