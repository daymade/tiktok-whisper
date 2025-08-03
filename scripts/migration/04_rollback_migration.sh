#!/bin/bash
# Rollback migration script
# Restores the database to pre-migration state

set -e

# Configuration
DB_PATH="${DB_PATH:-/data/transcription.db}"

echo "=== Database Migration Rollback ==="
echo "Start time: $(date)"
echo ""

# Get timestamp parameter or find most recent backup
if [ -n "$1" ]; then
    TIMESTAMP="$1"
else
    # Find most recent pre-migration backup
    BACKUP_FILE=$(ls -t "${DB_PATH%.db}_pre_migration_"*.db 2>/dev/null | head -1)
    if [ -z "$BACKUP_FILE" ]; then
        echo "ERROR: No backup found! Cannot rollback."
        echo "Usage: $0 [timestamp]"
        echo "Example: $0 20240115_143022"
        exit 1
    fi
    TIMESTAMP=$(basename "$BACKUP_FILE" | sed 's/.*pre_migration_\(.*\)\.db/\1/')
fi

BACKUP_DB="${DB_PATH%.db}_pre_migration_${TIMESTAMP}.db"

# Verify backup exists
if [ ! -f "$BACKUP_DB" ]; then
    echo "ERROR: Backup not found: $BACKUP_DB"
    exit 1
fi

echo "Found backup: $BACKUP_DB"
echo ""

# Step 1: Verify backup integrity
echo "Step 1: Verifying backup integrity..."
INTEGRITY=$(sqlite3 "$BACKUP_DB" "PRAGMA integrity_check;")
if [ "$INTEGRITY" = "ok" ]; then
    echo "  ✓ Backup integrity check passed"
else
    echo "  ✗ Backup integrity check failed!"
    echo "  Cannot proceed with rollback."
    exit 1
fi

# Get backup statistics
BACKUP_COUNT=$(sqlite3 "$BACKUP_DB" "SELECT COUNT(*) FROM transcriptions;")
CURRENT_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM transcriptions;" 2>/dev/null || echo "N/A")
echo "  - Backup records: $BACKUP_COUNT"
echo "  - Current records: $CURRENT_COUNT"

# Step 2: Stop application
echo ""
echo "Step 2: Stopping application..."
if systemctl is-active --quiet tiktok-whisper; then
    sudo systemctl stop tiktok-whisper
    echo "  ✓ Service stopped"
else
    echo "  - Service not running"
    echo "  - Press Ctrl+C now if you need to stop the application manually"
    sleep 5
fi

# Step 3: Create safety backup of current state
echo ""
echo "Step 3: Creating safety backup of current database..."
SAFETY_BACKUP="${DB_PATH%.db}_rollback_safety_$(date +%Y%m%d_%H%M%S).db"
if [ -f "$DB_PATH" ]; then
    cp "$DB_PATH" "$SAFETY_BACKUP"
    echo "  ✓ Safety backup created: $SAFETY_BACKUP"
fi

# Step 4: Perform rollback
echo ""
echo "Step 4: Performing rollback..."
echo "  - Moving current database..."
if [ -f "$DB_PATH" ]; then
    mv "$DB_PATH" "${DB_PATH%.db}_failed_migration_$(date +%Y%m%d_%H%M%S).db"
fi

echo "  - Restoring backup..."
cp "$BACKUP_DB" "$DB_PATH"

# Set permissions
echo "  - Setting permissions..."
chmod 644 "$DB_PATH"
chown $(stat -c %U:%G "$BACKUP_DB" 2>/dev/null || echo "$(whoami)") "$DB_PATH" 2>/dev/null || true

echo "  ✓ Database rolled back successfully"

# Step 5: Verify rollback
echo ""
echo "Step 5: Verifying rollback..."

# Check integrity
ROLLBACK_INTEGRITY=$(sqlite3 "$DB_PATH" "PRAGMA integrity_check;")
if [ "$ROLLBACK_INTEGRITY" = "ok" ]; then
    echo "  ✓ Rolled back database integrity check passed"
else
    echo "  ✗ Rolled back database integrity check failed!"
    exit 1
fi

# Check record count matches
ROLLBACK_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM transcriptions;")
if [ "$ROLLBACK_COUNT" = "$BACKUP_COUNT" ]; then
    echo "  ✓ Record count verified: $ROLLBACK_COUNT"
else
    echo "  ⚠ Record count mismatch! Expected: $BACKUP_COUNT, Got: $ROLLBACK_COUNT"
fi

# Check schema (should not have new columns)
echo "  - Checking schema reverted..."
if sqlite3 "$DB_PATH" "PRAGMA table_info(transcriptions);" | grep -q "file_hash"; then
    echo "  ⚠ Warning: New columns still present (may be from earlier migration)"
else
    echo "  ✓ Schema reverted to original"
fi

# Step 6: Restart application
echo ""
echo "Step 6: Restarting application..."
if systemctl is-enabled tiktok-whisper &>/dev/null; then
    sudo systemctl start tiktok-whisper
    sleep 2
    if systemctl is-active --quiet tiktok-whisper; then
        echo "  ✓ Service restarted successfully"
    else
        echo "  ✗ Service failed to start!"
    fi
else
    echo "  - Service not managed by systemctl"
    echo "  - Please restart the application manually"
fi

# Summary
echo ""
echo "=== Rollback Complete ==="
echo "End time: $(date)"
echo ""
echo "Summary:"
echo "- Database restored from: $BACKUP_DB"
echo "- Failed migration backed up to: ${DB_PATH%.db}_failed_migration_*.db"
echo "- Safety backup available at: $SAFETY_BACKUP"
echo ""
echo "Next steps:"
echo "1. Verify application is working correctly"
echo "2. Check application logs for any errors"
echo "3. Investigate migration failure before reattempting"
echo ""
echo "Rollback Status: SUCCESS"