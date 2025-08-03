#!/bin/bash
# Post-migration validation script
# Verifies the migration was successful and the application is working

set -e

# Configuration
DB_PATH="${DB_PATH:-/Volumes/SSD2T/workspace/go/tiktok-whisper/data/transcription.db}"
APP_PORT="${APP_PORT:-8081}"

echo "=== Post-Migration Validation ==="
echo "Start time: $(date)"
echo ""

# Step 1: Database integrity check
echo "Step 1: Checking database integrity..."
INTEGRITY=$(sqlite3 "$DB_PATH" "PRAGMA integrity_check;")
if [ "$INTEGRITY" = "ok" ]; then
    echo "  ✓ Database integrity check passed"
else
    echo "  ✗ Database integrity check failed!"
    echo "  $INTEGRITY"
    exit 1
fi

# Step 2: Schema validation
echo ""
echo "Step 2: Validating schema..."

# Check all expected columns exist
EXPECTED_COLUMNS="id user input_dir file_name mp3_file_name audio_duration transcription last_conversion_time has_error error_message file_hash file_size provider_type language model_name created_at updated_at deleted_at"
MISSING_COLUMNS=""

for col in $EXPECTED_COLUMNS; do
    if ! sqlite3 "$DB_PATH" "PRAGMA table_info(transcriptions);" | grep -q "|$col|"; then
        MISSING_COLUMNS="$MISSING_COLUMNS $col"
    fi
done

if [ -n "$MISSING_COLUMNS" ]; then
    echo "  ✗ Missing columns:$MISSING_COLUMNS"
    exit 1
else
    echo "  ✓ All expected columns present"
fi

# Step 3: Index validation
echo ""
echo "Step 3: Validating indexes..."
INDEXES=$(sqlite3 "$DB_PATH" ".indexes transcriptions" | wc -l)
echo "  - Found $INDEXES indexes"

# Check specific important indexes
for idx in "idx_file_name_error" "idx_user_error" "idx_conversion_time" "idx_user_time"; do
    if sqlite3 "$DB_PATH" ".indexes transcriptions" | grep -q "$idx"; then
        echo "  ✓ Index $idx exists"
    else
        echo "  ✗ Missing index: $idx"
    fi
done

# Step 4: Data validation
echo ""
echo "Step 4: Validating data..."

# Check record counts
sqlite3 "$DB_PATH" <<EOF
.mode column
.headers on
SELECT 
    'Data Statistics' as metric, 
    COUNT(*) as value 
FROM transcriptions
UNION ALL
SELECT 
    'Unique Users', 
    COUNT(DISTINCT user) 
FROM transcriptions
UNION ALL
SELECT 
    'Error Records', 
    COUNT(*) 
FROM transcriptions 
WHERE has_error = 1
UNION ALL
SELECT 
    'Records with Provider Type', 
    COUNT(*) 
FROM transcriptions 
WHERE provider_type IS NOT NULL
UNION ALL
SELECT 
    'Records with Timestamps', 
    COUNT(*) 
FROM transcriptions 
WHERE created_at IS NOT NULL AND updated_at IS NOT NULL;
EOF

# Step 5: Query performance test
echo ""
echo "Step 5: Testing query performance..."

# Test indexed queries
echo "  - Testing user query with index..."
START_TIME=$(date +%s.%N)
sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM transcriptions WHERE user = (SELECT user FROM transcriptions LIMIT 1) AND has_error = 0;" > /dev/null
END_TIME=$(date +%s.%N)
QUERY_TIME=$(echo "$END_TIME - $START_TIME" | bc)
echo "  ✓ Query completed in ${QUERY_TIME}s"

# Check query plan uses index
echo ""
echo "  - Checking query plan..."
sqlite3 "$DB_PATH" "EXPLAIN QUERY PLAN SELECT * FROM transcriptions WHERE file_name = 'test.mp3' AND has_error = 0;" | sed 's/^/    /'

# Step 6: Application health check
echo ""
echo "Step 6: Checking application health..."

# Check if application is running
if pgrep -f "v2t|tiktok-whisper" > /dev/null; then
    echo "  ✓ Application processes found"
else
    echo "  - No application processes running"
fi

# Try to connect to web interface
if command -v curl &> /dev/null; then
    echo "  - Testing web interface..."
    if curl -s -f "http://localhost:$APP_PORT/health" > /dev/null 2>&1; then
        echo "  ✓ Web interface responding"
    else
        echo "  - Web interface not responding (may not be running)"
    fi
fi

# Step 7: Sample data test
echo ""
echo "Step 7: Sampling migrated data..."
echo "  - Random sample of migrated records:"
sqlite3 "$DB_PATH" <<EOF
.mode box
.headers on
SELECT 
    id,
    substr(file_name, 1, 30) as file_name,
    provider_type,
    language,
    CASE WHEN file_hash IS NOT NULL THEN 'SET' ELSE 'NULL' END as hash_status,
    datetime(created_at) as created_at
FROM transcriptions 
ORDER BY RANDOM() 
LIMIT 5;
EOF

# Step 8: Migration artifacts
echo ""
echo "Step 8: Migration artifacts..."
MIGRATION_DIRS=$(find /Volumes/SSD2T/workspace/go/tiktok-whisper/data/migration -type d -name "20*" 2>/dev/null | tail -5)
if [ -n "$MIGRATION_DIRS" ]; then
    echo "  Recent migration directories:"
    echo "$MIGRATION_DIRS" | sed 's/^/    - /'
fi

# Summary
echo ""
echo "=== Post-Migration Validation Complete ==="
echo "End time: $(date)"
echo ""

# Check for any warnings
WARNINGS=""
if [ "$INDEXES" -lt 7 ]; then
    WARNINGS="$WARNINGS\n  - Fewer indexes than expected ($INDEXES < 7)"
fi

if [ -n "$WARNINGS" ]; then
    echo "Warnings:$WARNINGS"
    echo ""
fi

echo "Migration Status: SUCCESS"
echo ""
echo "Recommendations:"
echo "1. Monitor application logs for the next 24 hours"
echo "2. Run a test transcription to verify functionality"
echo "3. Check application performance metrics"
echo "4. Keep backup for at least 30 days"