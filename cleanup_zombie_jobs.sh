#!/bin/bash

echo "Cleaning up zombie jobs (completed jobs still in active list)..."
echo "================================================================"
echo ""

# Get active jobs
active_response=$(curl -s http://localhost:8080/api/sync/jobs/active)
active_jobs=$(echo "$active_response" | jq -r '.data[].job_id')

if [ -z "$active_jobs" ]; then
    echo "No active jobs found."
    exit 0
fi

echo "Found active jobs:"
echo "$active_jobs"
echo ""

# Get job history
history_response=$(curl -s "http://localhost:8080/api/sync/jobs/history?limit=100&offset=0")

# Check each active job
for job_id in $active_jobs; do
    # Check if this job is completed in history
    status=$(echo "$history_response" | jq -r ".data[] | select(.id == \"$job_id\") | .status")
    
    if [ "$status" = "completed" ] || [ "$status" = "failed" ] || [ "$status" = "cancelled" ]; then
        echo "⚠️  Zombie job found: $job_id (status: $status)"
        echo "   This job is completed but still in active list"
    else
        echo "✓  Job $job_id is genuinely active"
    fi
done

echo ""
echo "To fix this issue:"
echo "1. Restart the db-taxi service"
echo "2. Or wait for the fix to be deployed"
