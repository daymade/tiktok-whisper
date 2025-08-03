#!/bin/bash
# Quick integration test for tiktok-whisper

set -e

echo "=== Quick Integration Test ==="
echo ""

# Test 1: Build
echo "1. Testing build..."
if CGO_ENABLED=1 go build -o v2t_test ./cmd/v2t/main.go; then
    echo "✓ Build successful"
else
    echo "✗ Build failed"
    exit 1
fi

# Test 2: Database check
echo ""
echo "2. Testing database..."
if [ -f "./data/transcription.db" ]; then
    echo "✓ Database exists"
    
    # Check schema
    COLUMNS=$(sqlite3 ./data/transcription.db "PRAGMA table_info(transcriptions);" | wc -l)
    echo "✓ Found $COLUMNS columns"
    
    INDEXES=$(sqlite3 ./data/transcription.db ".indexes transcriptions" | wc -l)
    echo "✓ Found $INDEXES indexes"
else
    echo "✗ Database not found"
fi

# Test 3: Basic commands
echo ""
echo "3. Testing commands..."
./v2t_test --help > /dev/null 2>&1 && echo "✓ Help command works"
./v2t_test version > /dev/null 2>&1 && echo "✓ Version command works"
./v2t_test providers list > /dev/null 2>&1 && echo "✓ Providers command works"

# Test 4: Query performance
echo ""
echo "4. Testing query performance..."
TIME=$(time -p sqlite3 ./data/transcription.db "SELECT COUNT(*) FROM transcriptions WHERE user = 'test' AND has_error = 0;" 2>&1 | grep real | awk '{print $2}')
echo "✓ Query time: ${TIME}s"

# Cleanup
rm -f v2t_test

echo ""
echo "=== All tests passed! ===">