#!/bin/bash

# Production startup script
echo "Starting DB-Taxi in production mode..."

# Ensure required environment variables are set
if [ -z "$DB_PASSWORD" ]; then
    echo "Error: DB_PASSWORD environment variable is required"
    exit 1
fi

# Start with production configuration
./db-taxi -config configs/production.yaml

echo "DB-Taxi stopped."