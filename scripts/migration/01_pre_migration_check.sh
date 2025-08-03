#!/bin/bash
# Pre-migration validation script
# Ensures database is ready for migration

set -e  # Exit on error

# Configuration
DB_PATH="${DB_PATH:-/Volumes/SSD2T/workspace/go/tiktok-whisper/data/transcription.db}"
MIGRATION_BASE_DIR="${MIGRATION_BASE_DIR:-/Volumes/SSD2T/workspace/go/tiktok-whisper/data/migration}"
MIGRATION_DIR="$MIGRATION_BASE_DIR/$(date +%Y%m%d_%H%M%S)"

echo "=== Pre-Migration Validation ==="
echo "Database: $DB_PATH"
echo "Migration directory: $MIGRATION_DIR"
echo ""

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    echo "ERROR: Database not found at $DB_PATH"
    exit 1
fi

# Check database integrity
echo "1. Checking database integrity..."
sqlite3 "$DB_PATH" "PRAGMA integrity_check;" > /tmp/integrity_check.log
if grep -q "ok" /tmp/integrity_check.log; then
    echo "   ✓ Database integrity check passed"
else
    echo "   ✗ Database integrity check failed!"
    cat /tmp/integrity_check.log
    exit 1
fi

# Get database statistics
echo ""
echo "2. Database statistics:"
sqlite3 "$DB_PATH" <<EOF
.mode column
.headers on
SELECT 
    COUNT(*) as total_records,
    COUNT(DISTINCT user) as unique_users,
    COUNT(CASE WHEN has_error = 1 THEN 1 END) as error_records,
    ROUND(AVG(audio_duration)) as avg_duration_sec,
    MIN(last_conversion_time) as oldest_record,
    MAX(last_conversion_time) as newest_record
FROM transcriptions;
EOF

# Check disk space
echo ""
echo "3. Checking disk space..."
DB_SIZE=$(ls -lh "$DB_PATH" | awk '{print $5}')
AVAILABLE_SPACE=$(df -h $(dirname "$DB_PATH") | tail -1 | awk '{print $4}')
echo "   Database size: $DB_SIZE"
echo "   Available space: $AVAILABLE_SPACE"

# Verify SQLite version
echo ""
echo "4. SQLite version:"
sqlite3 --version

# Check current schema
echo ""
echo "5. Current schema check:"
echo "   Existing columns:"
sqlite3 "$DB_PATH" "PRAGMA table_info(transcriptions);" | awk -F'|' '{print "   - " $2 " (" $3 ")"}'

# Check for existing indexes
echo ""
echo "6. Existing indexes:"
sqlite3 "$DB_PATH" ".indexes transcriptions" | sed 's/^/   - /'

# Create migration directory
echo ""
echo "7. Creating migration directory..."
mkdir -p "$MIGRATION_DIR"
echo "   ✓ Created $MIGRATION_DIR"

# Final confirmation
echo ""
echo "=== Pre-Migration Check Complete ==="
echo ""
echo "Ready to proceed with migration? This will:"
echo "- Create a backup of the database"
echo "- Add new columns and indexes"
echo "- Require approximately 30-60 minutes downtime"
echo ""
read -p "Continue? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "Migration cancelled."
    exit 0
fi

echo ""
echo "Pre-migration validation passed. Proceed to run: 02_execute_migration.sh"