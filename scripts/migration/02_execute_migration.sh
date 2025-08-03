#!/bin/bash
# Main migration execution script
# Performs the actual database migration with safety checks

set -e  # Exit on error

# Configuration
DB_PATH="${DB_PATH:-/Volumes/SSD2T/workspace/go/tiktok-whisper/data/transcription.db}"
MIGRATION_BASE_DIR="${MIGRATION_BASE_DIR:-/Volumes/SSD2T/workspace/go/tiktok-whisper/data/migration}"
MIGRATION_DIR="$MIGRATION_BASE_DIR/$(date +%Y%m%d_%H%M%S)"
BACKUP_KEEP_DAYS=30

echo "=== Database Migration Execution ==="
echo "Start time: $(date)"
echo ""

# Step 1: Create migration directory and backups
echo "Step 1: Creating backups..."
mkdir -p "$MIGRATION_DIR"
cd "$MIGRATION_DIR"

# Create checksums for verification
echo "  - Calculating checksum of original database..."
sha256sum "$DB_PATH" > original_checksum.txt

# Create backup copies
echo "  - Creating backup copy..."
cp -v "$DB_PATH" ./transcription_original.db
cp -v ./transcription_original.db ./transcription_migrate.db

# Verify backups
echo "  - Verifying backup integrity..."
ORIGINAL_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM transcriptions;")
BACKUP_COUNT=$(sqlite3 ./transcription_original.db "SELECT COUNT(*) FROM transcriptions;")
MIGRATE_COUNT=$(sqlite3 ./transcription_migrate.db "SELECT COUNT(*) FROM transcriptions;")

if [ "$ORIGINAL_COUNT" != "$BACKUP_COUNT" ] || [ "$ORIGINAL_COUNT" != "$MIGRATE_COUNT" ]; then
    echo "ERROR: Backup verification failed!"
    echo "Original: $ORIGINAL_COUNT, Backup: $BACKUP_COUNT, Migrate: $MIGRATE_COUNT"
    exit 1
fi
echo "  ✓ Backups verified: $ORIGINAL_COUNT records"

# Step 2: Stop application (if running)
echo ""
echo "Step 2: Checking for running application..."
# 检查是否有 v2t 进程在运行
if pgrep -f "v2t|tiktok-whisper" > /dev/null; then
    echo "  ⚠️  Found running v2t/tiktok-whisper processes"
    echo "  Please stop them manually before continuing"
    echo "  Press Ctrl+C to cancel, or Enter to continue if already stopped"
    read -p "  > "
else
    echo "  ✓ No running processes found"
fi

# Step 3: Execute migration
echo ""
echo "Step 3: Executing migration on work copy..."

# Create migration SQL script
cat > migration.sql << 'MIGRATION_SQL'
-- Enable WAL mode for better performance
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -10000;
PRAGMA busy_timeout = 10000;

-- Start transaction
BEGIN TRANSACTION;

-- Add performance indexes
CREATE INDEX IF NOT EXISTS idx_file_name_error ON transcriptions(file_name, has_error);
CREATE INDEX IF NOT EXISTS idx_user_error ON transcriptions(user, has_error);
CREATE INDEX IF NOT EXISTS idx_conversion_time ON transcriptions(last_conversion_time);
CREATE INDEX IF NOT EXISTS idx_user_time ON transcriptions(user, last_conversion_time DESC);

-- Add new columns (each in separate statement for SQLite compatibility)
ALTER TABLE transcriptions ADD COLUMN file_hash TEXT;
ALTER TABLE transcriptions ADD COLUMN file_size INTEGER DEFAULT 0;
ALTER TABLE transcriptions ADD COLUMN provider_type TEXT DEFAULT 'whisper_cpp';
ALTER TABLE transcriptions ADD COLUMN language TEXT DEFAULT 'zh';
ALTER TABLE transcriptions ADD COLUMN model_name TEXT;
ALTER TABLE transcriptions ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE transcriptions ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE transcriptions ADD COLUMN deleted_at DATETIME;

-- Create indexes for new columns
CREATE INDEX idx_file_hash ON transcriptions(file_hash) WHERE file_hash IS NOT NULL;
CREATE INDEX idx_provider_type ON transcriptions(provider_type);
CREATE INDEX idx_deleted_at ON transcriptions(deleted_at) WHERE deleted_at IS NULL;

-- Update timestamps for existing records
UPDATE transcriptions 
SET created_at = last_conversion_time,
    updated_at = last_conversion_time
WHERE created_at IS NULL;

COMMIT;

-- Optimize database
VACUUM;
ANALYZE;
MIGRATION_SQL

# Execute migration
echo "  - Running migration SQL..."
if sqlite3 ./transcription_migrate.db < migration.sql; then
    echo "  ✓ Migration SQL executed successfully"
else
    echo "  ✗ Migration failed!"
    exit 1
fi

# Step 4: Verify migration
echo ""
echo "Step 4: Verifying migration..."

# Check record count
POST_COUNT=$(sqlite3 ./transcription_migrate.db "SELECT COUNT(*) FROM transcriptions;")
if [ "$ORIGINAL_COUNT" != "$POST_COUNT" ]; then
    echo "ERROR: Record count mismatch! Original: $ORIGINAL_COUNT, Post-migration: $POST_COUNT"
    exit 1
fi
echo "  ✓ Record count verified: $POST_COUNT"

# Check new columns exist
echo "  - Checking new columns..."
NEW_COLS=$(sqlite3 ./transcription_migrate.db "PRAGMA table_info(transcriptions);" | grep -E "(file_hash|provider_type|created_at)" | wc -l)
if [ "$NEW_COLS" -lt 3 ]; then
    echo "ERROR: New columns not created properly!"
    exit 1
fi
echo "  ✓ New columns verified"

# Check indexes
echo "  - Checking indexes..."
INDEX_COUNT=$(sqlite3 ./transcription_migrate.db ".indexes transcriptions" | wc -l)
if [ "$INDEX_COUNT" -lt 7 ]; then
    echo "ERROR: Expected at least 7 indexes, found $INDEX_COUNT"
    exit 1
fi
echo "  ✓ Indexes created: $INDEX_COUNT"

# Test query performance
echo "  - Testing query performance..."
time sqlite3 ./transcription_migrate.db "SELECT COUNT(*) FROM transcriptions WHERE user = 'test' AND has_error = 0;" > /dev/null
echo "  ✓ Query test completed"

# Step 5: Deploy migrated database
echo ""
echo "Step 5: Deploying migrated database..."

# Backup current production database
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
echo "  - Backing up current production database..."
mv "$DB_PATH" "${DB_PATH%.db}_pre_migration_${TIMESTAMP}.db"

# Deploy new database
echo "  - Deploying migrated database..."
cp ./transcription_migrate.db "$DB_PATH"

# Set permissions
echo "  - Setting permissions..."
chmod 644 "$DB_PATH"
# macOS 使用不同的 stat 命令格式
if [[ "$OSTYPE" == "darwin"* ]]; then
    chown $(stat -f %u:%g "${DB_PATH%.db}_pre_migration_${TIMESTAMP}.db") "$DB_PATH" 2>/dev/null || true
else
    chown $(stat -c %U:%G "${DB_PATH%.db}_pre_migration_${TIMESTAMP}.db") "$DB_PATH" 2>/dev/null || true
fi

# Step 6: Create version marker
echo ""
echo "Step 6: Recording migration version..."
cat > "$MIGRATION_DIR/migration_info.txt" << EOF
Migration completed: $(date)
Original records: $ORIGINAL_COUNT
Post-migration records: $POST_COUNT
Database version: 2.0
Schema additions:
  - file_hash (TEXT)
  - file_size (INTEGER)
  - provider_type (TEXT)
  - language (TEXT)
  - model_name (TEXT)
  - created_at (DATETIME)
  - updated_at (DATETIME)
  - deleted_at (DATETIME)
Indexes added: 7
EOF

echo "  ✓ Migration info recorded"

# Step 7: Application restart reminder
echo ""
echo "Step 7: Application restart..."
echo "  - Database migration completed"
echo "  - Please restart your application manually when ready"
echo "  - Test command: v2t convert single --file test.mp3"

# Summary
echo ""
echo "=== Migration Complete ==="
echo "End time: $(date)"
echo ""
echo "Summary:"
echo "- Original database backed up to: ${DB_PATH%.db}_pre_migration_${TIMESTAMP}.db"
echo "- Migration artifacts saved in: $MIGRATION_DIR"
echo "- New database deployed to: $DB_PATH"
echo ""
echo "Next steps:"
echo "1. Run post-migration validation: ./03_post_migration_check.sh"
echo "2. Update application code to use new schema"
echo "3. Monitor application logs for any issues"
echo ""
echo "If issues occur, run rollback: ./04_rollback_migration.sh $TIMESTAMP"