# Database Optimization Plan for tiktok-whisper

## Executive Summary

This document outlines a comprehensive optimization plan for the tiktok-whisper SQLite database. The plan is divided into three phases with increasing complexity and risk, allowing for gradual implementation with minimal disruption.

**Current Status:**
- Database: SQLite with 7,070+ records
- Users: 36 unique users
- Issues: No indexes, suboptimal schema, missing constraints
- Performance: Full table scans on every query

**Expected Benefits:**
- 10-100x query performance improvement
- Better data integrity and consistency
- Enhanced scalability and maintainability
- Support for new features (provider tracking, metadata, search)

## Phase 1: Immediate Optimizations (Low Risk)

### 1.1 Add Critical Indexes

These indexes will provide immediate performance benefits without any data changes:

```sql
-- Index for CheckIfFileProcessed queries
CREATE INDEX idx_file_name_error ON transcriptions(file_name, has_error);

-- Index for GetAllByUser queries
CREATE INDEX idx_user_error ON transcriptions(user, has_error);

-- Index for time-based queries
CREATE INDEX idx_conversion_time ON transcriptions(last_conversion_time);

-- Composite index for common query patterns
CREATE INDEX idx_user_time ON transcriptions(user, last_conversion_time DESC);
```

**Impact:** Query performance improvement from O(n) to O(log n)

### 1.2 SQLite Optimizations

```sql
-- Enable Write-Ahead Logging for better concurrency
PRAGMA journal_mode = WAL;

-- Optimize synchronization for performance
PRAGMA synchronous = NORMAL;

-- Increase cache size (in pages, -10000 = ~10MB)
PRAGMA cache_size = -10000;

-- Run optimization
VACUUM;
ANALYZE;
```

### 1.3 Implementation Script

```sql
-- Phase 1 Migration Script
BEGIN TRANSACTION;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_file_name_error ON transcriptions(file_name, has_error);
CREATE INDEX IF NOT EXISTS idx_user_error ON transcriptions(user, has_error);
CREATE INDEX IF NOT EXISTS idx_conversion_time ON transcriptions(last_conversion_time);
CREATE INDEX IF NOT EXISTS idx_user_time ON transcriptions(user, last_conversion_time DESC);

-- Update statistics
ANALYZE;

COMMIT;

-- Apply optimizations (outside transaction)
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -10000;
VACUUM;
```

## Phase 2: Schema Enhancements (Medium Risk)

### 2.1 Add New Columns

```sql
-- Add metadata columns
ALTER TABLE transcriptions ADD COLUMN file_hash TEXT;
ALTER TABLE transcriptions ADD COLUMN file_size INTEGER DEFAULT 0;
ALTER TABLE transcriptions ADD COLUMN provider_type TEXT DEFAULT 'whisper_cpp';
ALTER TABLE transcriptions ADD COLUMN language TEXT DEFAULT 'zh';
ALTER TABLE transcriptions ADD COLUMN model_name TEXT;
ALTER TABLE transcriptions ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE transcriptions ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE transcriptions ADD COLUMN deleted_at DATETIME;

-- Add unique constraint on file_hash to prevent duplicates
CREATE UNIQUE INDEX idx_file_hash ON transcriptions(file_hash) WHERE file_hash IS NOT NULL AND deleted_at IS NULL;

-- Add index for provider queries
CREATE INDEX idx_provider_type ON transcriptions(provider_type);
```

### 2.2 Add Constraints and Triggers

```sql
-- Add check constraints
CREATE TABLE transcriptions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user TEXT NOT NULL DEFAULT 'default',
    input_dir TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mp3_file_name TEXT NOT NULL,
    audio_duration INTEGER NOT NULL CHECK(audio_duration >= 0),
    transcription TEXT NOT NULL,
    last_conversion_time DATETIME NOT NULL,
    has_error INTEGER NOT NULL CHECK(has_error IN (0, 1)) DEFAULT 0,
    error_message TEXT,
    file_hash TEXT,
    file_size INTEGER DEFAULT 0 CHECK(file_size >= 0),
    provider_type TEXT DEFAULT 'whisper_cpp',
    language TEXT DEFAULT 'zh',
    model_name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

-- Trigger to update updated_at
CREATE TRIGGER update_transcriptions_timestamp 
AFTER UPDATE ON transcriptions_new
BEGIN
    UPDATE transcriptions_new SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### 2.3 Data Migration Script

```sql
-- Phase 2 Migration
BEGIN TRANSACTION;

-- Create new table with constraints
CREATE TABLE transcriptions_new (
    -- schema as above
);

-- Copy data
INSERT INTO transcriptions_new (
    id, user, input_dir, file_name, mp3_file_name, audio_duration,
    transcription, last_conversion_time, has_error, error_message
)
SELECT 
    id, 
    COALESCE(user, 'default'),
    input_dir, 
    file_name, 
    mp3_file_name, 
    audio_duration,
    transcription, 
    last_conversion_time, 
    has_error, 
    error_message
FROM transcriptions;

-- Rename tables
ALTER TABLE transcriptions RENAME TO transcriptions_old;
ALTER TABLE transcriptions_new RENAME TO transcriptions;

-- Recreate indexes
CREATE INDEX idx_file_name_error ON transcriptions(file_name, has_error);
CREATE INDEX idx_user_error ON transcriptions(user, has_error);
CREATE INDEX idx_conversion_time ON transcriptions(last_conversion_time);
CREATE INDEX idx_user_time ON transcriptions(user, last_conversion_time DESC);
CREATE UNIQUE INDEX idx_file_hash ON transcriptions(file_hash) WHERE file_hash IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_provider_type ON transcriptions(provider_type);

COMMIT;
```

### 2.4 Go Code Updates

```go
// Update TranscriptionDAO interface
type TranscriptionDAO interface {
    // Existing methods...
    
    // New methods
    RecordToDBExtended(user, inputDir, fileName, mp3FileName string, 
        audioDuration int, transcription string, lastConversionTime time.Time, 
        hasError int, errorMessage string, fileHash string, fileSize int64, 
        providerType string, language string, modelName string) error
    
    CheckIfFileProcessedByHash(fileHash string) (int, error)
    GetTranscriptionsByProvider(provider string) ([]Transcription, error)
    SoftDeleteTranscription(id int) error
}

// Update model
type Transcription struct {
    // Existing fields...
    
    FileHash     string    `json:"file_hash,omitempty"`
    FileSize     int64     `json:"file_size,omitempty"`
    ProviderType string    `json:"provider_type,omitempty"`
    Language     string    `json:"language,omitempty"`
    ModelName    string    `json:"model_name,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}
```

## Phase 3: Advanced Optimizations (High Risk)

### 3.1 Table Normalization

```sql
-- Separate large text data
CREATE TABLE transcription_texts (
    transcription_id INTEGER PRIMARY KEY,
    text TEXT NOT NULL,
    FOREIGN KEY (transcription_id) REFERENCES transcriptions(id)
);

-- Users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    total_transcriptions INTEGER DEFAULT 0,
    total_duration_seconds INTEGER DEFAULT 0
);

-- Providers table
CREATE TABLE providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    is_active BOOLEAN DEFAULT 1,
    config JSON
);

-- Processing queue table
CREATE TABLE processing_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL,
    user_id INTEGER,
    provider_id INTEGER,
    status TEXT DEFAULT 'pending',
    priority INTEGER DEFAULT 5,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME,
    error_message TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (provider_id) REFERENCES providers(id)
);
```

### 3.2 Full-Text Search

```sql
-- Create FTS5 table for transcription search
CREATE VIRTUAL TABLE transcription_search USING fts5(
    transcription_id,
    text,
    content=transcription_texts,
    content_rowid=transcription_id
);

-- Populate FTS table
INSERT INTO transcription_search(transcription_id, text)
SELECT transcription_id, text FROM transcription_texts;

-- Create triggers to keep FTS in sync
CREATE TRIGGER transcription_search_insert 
AFTER INSERT ON transcription_texts
BEGIN
    INSERT INTO transcription_search(transcription_id, text) 
    VALUES (NEW.transcription_id, NEW.text);
END;
```

### 3.3 Archive Strategy

```sql
-- Archive table for old records
CREATE TABLE transcriptions_archive (
    -- Same schema as transcriptions
    archived_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Archive records older than 1 year
INSERT INTO transcriptions_archive
SELECT *, CURRENT_TIMESTAMP as archived_at
FROM transcriptions
WHERE created_at < datetime('now', '-1 year')
AND deleted_at IS NULL;

-- Delete archived records from main table
DELETE FROM transcriptions
WHERE created_at < datetime('now', '-1 year')
AND deleted_at IS NULL;
```

## Implementation Guide

### Prerequisites

1. **Backup Database**
   ```bash
   cp data/transcription.db data/transcription_backup_$(date +%Y%m%d).db
   ```

2. **Test Environment**
   - Create test database with sample data
   - Run all migrations on test database first
   - Verify application functionality

### Migration Steps

#### Phase 1 (Immediate - 1 day)
1. Backup database
2. Run Phase 1 migration script
3. Test application functionality
4. Monitor query performance

#### Phase 2 (Week 2-3)
1. Update Go code to support new fields
2. Deploy code changes
3. Run Phase 2 migration during low-usage period
4. Verify data integrity
5. Update application to use new fields

#### Phase 3 (Month 2-3)
1. Implement new DAO methods
2. Create data migration utilities
3. Run table normalization in stages
4. Implement archival process
5. Consider PostgreSQL migration for scale

### Rollback Plans

Each phase includes rollback capability:

```sql
-- Phase 1 Rollback
DROP INDEX IF EXISTS idx_file_name_error;
DROP INDEX IF EXISTS idx_user_error;
DROP INDEX IF EXISTS idx_conversion_time;
DROP INDEX IF EXISTS idx_user_time;

-- Phase 2 Rollback
BEGIN TRANSACTION;
ALTER TABLE transcriptions RENAME TO transcriptions_failed;
ALTER TABLE transcriptions_old RENAME TO transcriptions;
COMMIT;

-- Phase 3 Rollback
-- Restore from backup before normalization
```

## Performance Monitoring

### Query Performance Metrics

```sql
-- Create performance monitoring table
CREATE TABLE query_performance (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query_type TEXT NOT NULL,
    execution_time_ms INTEGER NOT NULL,
    row_count INTEGER,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Monitor key queries
-- In Go code, wrap queries with timing
```

### Health Checks

```sql
-- Database health view
CREATE VIEW database_health AS
SELECT 
    (SELECT COUNT(*) FROM transcriptions) as total_records,
    (SELECT COUNT(DISTINCT user) FROM transcriptions) as total_users,
    (SELECT COUNT(*) FROM transcriptions WHERE has_error = 1) as error_count,
    (SELECT AVG(audio_duration) FROM transcriptions WHERE has_error = 0) as avg_duration,
    (SELECT COUNT(*) FROM transcriptions WHERE created_at > datetime('now', '-1 day')) as daily_count;
```

## Maintenance Scripts

### Daily Maintenance

```sql
-- daily_maintenance.sql
PRAGMA optimize;
ANALYZE;

-- Clean up soft-deleted records older than 30 days
DELETE FROM transcriptions 
WHERE deleted_at < datetime('now', '-30 days');
```

### Weekly Maintenance

```sql
-- weekly_maintenance.sql
VACUUM;
REINDEX;

-- Archive old records
INSERT INTO transcriptions_archive
SELECT * FROM transcriptions
WHERE created_at < datetime('now', '-365 days');

DELETE FROM transcriptions
WHERE id IN (SELECT id FROM transcriptions_archive);
```

## Success Metrics

- **Query Performance**: 90% of queries under 10ms
- **Data Integrity**: Zero data corruption incidents
- **Uptime**: 99.9% availability during migration
- **Storage Efficiency**: 20-30% reduction after text separation
- **Maintenance Time**: Automated maintenance under 5 minutes

## Risk Mitigation

1. **Testing**: Each phase tested in isolation
2. **Backups**: Automated daily backups with 30-day retention
3. **Monitoring**: Real-time query performance tracking
4. **Gradual Rollout**: Phase implementation over 3 months
5. **Fallback**: Keep old schema accessible for 90 days

## Conclusion

This optimization plan provides a roadmap for transforming the tiktok-whisper database from a simple flat structure to a robust, scalable system. The phased approach ensures minimal risk while delivering immediate performance benefits.

**Next Steps:**
1. Review and approve plan
2. Set up test environment
3. Begin Phase 1 implementation
4. Schedule regular review meetings