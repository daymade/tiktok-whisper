#!/bin/bash

# Test script for ETL pipeline
# This script tests the complete ETL workflow from YouTube URL to transcription

set -e

echo "=== V2T Distributed ETL Pipeline Test ==="
echo

# Check if services are running
echo "Checking services..."
if ! curl -s http://localhost:7233/health > /dev/null 2>&1; then
    echo "❌ Temporal is not running. Please start it with: docker-compose up -d"
    exit 1
fi
echo "✅ Temporal is running"

if ! curl -s http://localhost:9000/minio/health/live > /dev/null 2>&1; then
    echo "❌ MinIO is not running. Please start it with: docker-compose up -d"
    exit 1
fi
echo "✅ MinIO is running"

# Test with a short YouTube video (10 seconds)
TEST_URL="https://www.youtube.com/watch?v=aqz-KE-bpKQ"  # Big Buck Bunny trailer
echo
echo "Testing with sample video: $TEST_URL"
echo

# Submit ETL job
echo "Submitting ETL job..."
OUTPUT=$(v2t etl --url "$TEST_URL" --language en 2>&1)
echo "$OUTPUT"

# Extract workflow ID
WORKFLOW_ID=$(echo "$OUTPUT" | grep "Workflow ID:" | awk '{print $3}')
if [ -z "$WORKFLOW_ID" ]; then
    echo "❌ Failed to extract workflow ID"
    exit 1
fi

echo
echo "Workflow ID: $WORKFLOW_ID"
echo "Waiting for completion..."

# Poll for status
MAX_ATTEMPTS=60  # 5 minutes timeout
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    sleep 5
    
    STATUS_OUTPUT=$(v2t job status --workflow-id "$WORKFLOW_ID" 2>&1)
    STATUS=$(echo "$STATUS_OUTPUT" | grep "Status:" | awk '{print $2}')
    
    echo -n "."
    
    if [ "$STATUS" = "completed" ]; then
        echo
        echo "✅ Transcription completed successfully!"
        echo
        echo "$STATUS_OUTPUT"
        exit 0
    elif [ "$STATUS" = "failed" ]; then
        echo
        echo "❌ Transcription failed!"
        echo
        echo "$STATUS_OUTPUT"
        exit 1
    fi
    
    ATTEMPT=$((ATTEMPT + 1))
done

echo
echo "❌ Timeout waiting for transcription"
exit 1