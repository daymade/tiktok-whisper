# Database Migration Scripts

This directory contains scripts for migrating the tiktok-whisper SQLite database to add new features while preserving all existing data.

## Migration Overview

The migration adds the following enhancements:
- Performance indexes for 10-100x query speed improvement
- New metadata fields for provider tracking and file management
- Timestamp tracking (created_at, updated_at, deleted_at)
- File integrity checking (file_hash, file_size)
- Multi-language and model support

## Prerequisites

- Bash shell
- SQLite3 command-line tool
- Sufficient disk space (2x current database size)
- Application downtime window (30-60 minutes)
- Root/sudo access (for service management)

## Migration Process

### Step 1: Pre-Migration Check

```bash
chmod +x *.sh
./01_pre_migration_check.sh
```

This script will:
- Verify database integrity
- Show current statistics
- Check disk space
- Create migration directory
- Require confirmation to proceed

### Step 2: Execute Migration

```bash
# Set custom database path if needed
export DB_PATH=/path/to/your/transcription.db

./02_execute_migration.sh
```

This script will:
- Create full database backups
- Stop the application service
- Add new columns and indexes
- Deploy the migrated database
- Restart the application

### Step 3: Post-Migration Validation

```bash
./03_post_migration_check.sh
```

This script will:
- Verify database integrity
- Validate schema changes
- Test query performance
- Check application health
- Display migration statistics

### Step 4: Rollback (if needed)

```bash
# Rollback to most recent backup
./04_rollback_migration.sh

# Or rollback to specific timestamp
./04_rollback_migration.sh 20240115_143022
```

## Important Notes

1. **Always run scripts in order**: 01 → 02 → 03
2. **Backups are kept for 30 days** in the original location
3. **Migration artifacts** are stored in `/data/migration/YYYYMMDD_HHMMSS/`
4. **Original database** is preserved as `transcription_pre_migration_TIMESTAMP.db`

## Troubleshooting

### Service won't start after migration
1. Check logs: `journalctl -u tiktok-whisper -n 50`
2. Verify database permissions: `ls -la /data/transcription.db`
3. Run rollback script if needed

### Migration script fails
1. Check error messages for specific issues
2. Verify disk space: `df -h /data`
3. Ensure SQLite3 is installed: `sqlite3 --version`

### Performance issues after migration
1. Run `VACUUM` on the database
2. Verify indexes were created: `sqlite3 /data/transcription.db ".indexes transcriptions"`
3. Update application code to use new fields

## Post-Migration Tasks

After successful migration:

1. **Update Go models** to use `TranscriptionFull` struct
2. **Update DAO methods** to handle new fields
3. **Set up regular maintenance** (weekly VACUUM, monthly cleanup)
4. **Monitor performance** for the first 24-48 hours

## Schema Changes

### New Columns Added

| Column | Type | Default | Description |
|--------|------|---------|-------------|
| file_hash | TEXT | NULL | SHA256 hash of the audio file |
| file_size | INTEGER | 0 | File size in bytes |
| provider_type | TEXT | 'whisper_cpp' | Transcription provider used |
| language | TEXT | 'zh' | Language code |
| model_name | TEXT | NULL | Model used for transcription |
| created_at | DATETIME | CURRENT_TIMESTAMP | Record creation time |
| updated_at | DATETIME | CURRENT_TIMESTAMP | Last update time |
| deleted_at | DATETIME | NULL | Soft delete timestamp |

### New Indexes Added

- `idx_file_name_error` - Speeds up file processing checks
- `idx_user_error` - Speeds up user queries
- `idx_conversion_time` - Speeds up time-based queries
- `idx_user_time` - Composite index for user + time queries
- `idx_file_hash` - Speeds up duplicate detection
- `idx_provider_type` - Speeds up provider analysis
- `idx_deleted_at` - Speeds up active record queries