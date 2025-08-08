#!/bin/bash

# Test script for v2t distributed setup

echo "=== v2t Distributed System Test ==="
echo

# Check Docker
echo "1. Checking Docker..."
if command -v docker &> /dev/null; then
    echo "✓ Docker installed: $(docker --version)"
else
    echo "✗ Docker not found"
    exit 1
fi

# Check Docker Compose
echo
echo "2. Checking Docker Compose..."
if command -v docker-compose &> /dev/null; then
    echo "✓ Docker Compose installed: $(docker-compose --version)"
else
    echo "✗ Docker Compose not found"
    exit 1
fi

# Check required files
echo
echo "3. Checking required files..."
required_files=(
    "docker-compose.yml"
    "docker-compose.worker.yml"
    "Dockerfile.worker"
    "nginx.conf"
    "activities/transcribe.go"
    "activities/storage.go"
    "workflows/single_file.go"
    "workflows/batch.go"
    "workflows/fallback.go"
    "worker/main.go"
    "client/cli.go"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ Found: $file"
    else
        echo "✗ Missing: $file"
    fi
done

# Check Go modules
echo
echo "4. Checking Go modules..."
if [ -f "go.mod" ]; then
    echo "✓ go.mod found"
    if go mod tidy &> /dev/null; then
        echo "✓ Go modules valid"
    else
        echo "✗ Go module errors"
    fi
else
    echo "✗ go.mod not found"
fi

# Check environment
echo
echo "5. Checking environment variables..."
if [ -f ".env" ]; then
    echo "✓ .env file found"
else
    echo "⚠ .env file not found (using defaults)"
fi

# Test network connectivity (if services are running)
echo
echo "6. Testing service connectivity (if running)..."
if curl -s http://localhost:7233/health &> /dev/null; then
    echo "✓ Temporal server is accessible"
else
    echo "⚠ Temporal server not accessible (may not be running)"
fi

if curl -s http://localhost:9001 &> /dev/null; then
    echo "✓ MinIO console is accessible"
else
    echo "⚠ MinIO console not accessible (may not be running)"
fi

echo
echo "=== Test Complete ==="
echo
echo "To start the system:"
echo "  1. Main node: docker-compose up -d"
echo "  2. Worker nodes: docker-compose -f docker-compose.worker.yml up -d"
echo "  3. Submit jobs: ./v2t-distributed transcribe <file>"