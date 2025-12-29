#!/bin/bash

cd .. && go build -o .

# Local development startup script
echo "Starting DB-Taxi in local development mode..."

# Set environment variables for local development
export DBT_LOGGING_LEVEL=debug

# Start with local configuration
./db-taxi -config ./configs/config.local.yaml

echo "DB-Taxi stopped."