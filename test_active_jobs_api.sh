#!/bin/bash

echo "Testing Active Jobs API..."
echo "=========================="
echo ""

# Test 1: Get active jobs
echo "1. Getting active jobs..."
response=$(curl -s http://localhost:8080/api/sync/jobs/active)
echo "Response: $response"
echo ""

# Parse and display
echo "Active jobs count:"
echo "$response" | jq '.data | length'
echo ""

echo "Active job IDs:"
echo "$response" | jq -r '.data[].job_id'
echo ""

# Test 2: Get job history
echo "2. Getting job history..."
history_response=$(curl -s "http://localhost:8080/api/sync/jobs/history?limit=10&offset=0")
echo "History count:"
echo "$history_response" | jq '.data | length'
echo ""

echo "Recent job statuses:"
echo "$history_response" | jq -r '.data[] | "\(.id): \(.status)"'
echo ""

# Test 3: Get stats
echo "3. Getting sync stats..."
stats_response=$(curl -s http://localhost:8080/api/sync/stats)
echo "Stats:"
echo "$stats_response" | jq '.'
