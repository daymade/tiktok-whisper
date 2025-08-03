#!/bin/bash

# Test whisper-server via SSH port forwarding
set -e

echo "Setting up SSH port forwarding to whisper-server..."

# Kill any existing SSH forwarding on port 8081
lsof -ti:8081 | xargs kill -9 2>/dev/null || true

# Set up SSH port forwarding
ssh -f -N -L 8081:localhost:8080 daymade@mac-mini-m4-1.local
sleep 2

echo "Testing whisper-server through SSH tunnel on localhost:8081..."

# Test with a simple file
TEST_FILE="./test/data/jfk.wav"

echo "Testing transcription..."
response=$(curl -s -X POST \
    -F "file=@$TEST_FILE" \
    -F "response_format=text" \
    "http://localhost:8081/inference")

echo "Response: $response"

# Clean up SSH forwarding
echo "Cleaning up SSH port forwarding..."
lsof -ti:8081 | xargs kill -9 2>/dev/null || true

echo "Test complete!"