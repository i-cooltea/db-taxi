#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <job_id>"
    echo ""
    echo "This script monitors a job's lifecycle in real-time"
    exit 1
fi

JOB_ID=$1

echo "Monitoring job: $JOB_ID"
echo "================================"
echo ""

while true; do
    clear
    echo "Job Lifecycle Monitor - $(date)"
    echo "================================"
    echo ""
    
    # Check if job is in active jobs
    active_response=$(curl -s http://localhost:8080/api/sync/jobs/active)
    is_active=$(echo "$active_response" | jq -r ".data[] | select(.job_id == \"$JOB_ID\") | .job_id")
    
    if [ -n "$is_active" ]; then
        echo "✓ Job is in ACTIVE JOBS list"
        echo ""
        echo "Active Job Details:"
        echo "$active_response" | jq ".data[] | select(.job_id == \"$JOB_ID\")"
    else
        echo "✗ Job is NOT in active jobs list"
    fi
    
    echo ""
    echo "---"
    echo ""
    
    # Check job in history
    history_response=$(curl -s "http://localhost:8080/api/sync/jobs/history?limit=100&offset=0")
    job_status=$(echo "$history_response" | jq -r ".data[] | select(.id == \"$JOB_ID\") | .status")
    
    if [ -n "$job_status" ]; then
        echo "✓ Job found in HISTORY"
        echo ""
        echo "History Details:"
        echo "$history_response" | jq ".data[] | select(.id == \"$JOB_ID\")"
        
        if [ "$job_status" = "completed" ] || [ "$job_status" = "failed" ] || [ "$job_status" = "cancelled" ]; then
            echo ""
            echo "================================"
            echo "Job has finished with status: $job_status"
            
            if [ -n "$is_active" ]; then
                echo ""
                echo "⚠️  WARNING: Job is finished but still in active list!"
                echo "   This is a ZOMBIE JOB!"
            else
                echo ""
                echo "✓ Job correctly removed from active list"
            fi
            
            echo ""
            echo "Press Ctrl+C to exit or wait 5 seconds for final check..."
            sleep 5
            break
        fi
    else
        echo "✗ Job NOT found in history yet"
    fi
    
    echo ""
    echo "Refreshing in 2 seconds... (Press Ctrl+C to stop)"
    sleep 2
done

echo ""
echo "Monitoring complete."
