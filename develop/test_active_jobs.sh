#!/bin/bash

# Test script to verify active jobs API returns config_id

echo "Testing active jobs API..."
echo ""

# Start the server in background if not running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "Server not running. Please start the server first with: make run"
    exit 1
fi

echo "Fetching active jobs..."
response=$(curl -s http://localhost:8080/api/sync/jobs/active)

echo "Response:"
echo "$response" | jq '.'

echo ""
echo "Checking if config_id is present in active jobs..."
config_id_count=$(echo "$response" | jq '.data[].config_id' 2>/dev/null | wc -l)

if [ "$config_id_count" -gt 0 ]; then
    echo "✓ config_id field is present in active jobs"
else
    echo "✗ config_id field is missing in active jobs"
fi
