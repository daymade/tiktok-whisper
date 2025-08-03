# Zero-Downtime Database Migration Plan for tiktok-whisper

## Executive Summary

This document outlines a comprehensive zero-downtime migration strategy for the tiktok-whisper SQLite database. The migration follows the "Expand and Contract" pattern, ensuring continuous service availability while progressively enhancing the database schema.

**Key Principles:**
- ✅ No existing data modification
- ✅ Zero service interruption
- ✅ Rollback capability at every stage
- ✅ Progressive migration with monitoring
- ✅ Backwards compatibility throughout

**Timeline:** 4-6 weeks total
- Week 1: Index additions and performance optimizations
- Week 2-3: Schema expansion with shadow tables
- Week 4-5: Application deployment and traffic migration
- Week 6: Validation and cleanup

**Risk Level:** Low to Medium
- All changes are additive
- Each stage independently reversible
- Extensive testing before production

## Migration Architecture

### Overview

```
Current State          Transition State              Final State
┌─────────────┐       ┌─────────────┐              ┌─────────────┐
│ Old Schema  │       │ Old Schema  │              │ New Schema  │
│             │       │      +      │              │             │
│ App v1.0    │ ───>  │ New Schema  │ ───>         │ App v2.0    │
│             │       │             │              │             │
│             │       │ App v1.5    │              │             │
└─────────────┘       └─────────────┘              └─────────────┘
                      (Dual Compatible)
```

### Migration Stages

1. **Expand Phase**: Add new structures alongside existing ones
2. **Transition Phase**: Deploy dual-compatible application
3. **Migration Phase**: Gradually shift traffic to new schema
4. **Contract Phase**: Remove old structures after validation

## Stage 1: Performance Optimizations (Day 1-2)

### 1.1 Enable WAL Mode

```sql
-- Enable Write-Ahead Logging for better concurrency
-- This is safe and improves performance immediately
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
```

### 1.2 Add Indexes (Non-blocking)

```sql
-- These can be added while the application is running
-- SQLite allows concurrent reads during index creation

-- Step 1: Create indexes with IF NOT EXISTS
CREATE INDEX IF NOT EXISTS idx_file_name_error 
ON transcriptions(file_name, has_error);

CREATE INDEX IF NOT EXISTS idx_user_error 
ON transcriptions(user, has_error);

CREATE INDEX IF NOT EXISTS idx_conversion_time 
ON transcriptions(last_conversion_time);

CREATE INDEX IF NOT EXISTS idx_user_time 
ON transcriptions(user, last_conversion_time DESC);

-- Step 2: Analyze to update statistics
ANALYZE transcriptions;
```

### 1.3 Monitoring Setup

```go
// Add performance monitoring to existing code
type QueryMetrics struct {
    QueryType    string
    Duration     time.Duration
    RowsAffected int
}

func (db *SQLiteDB) measureQuery(queryType string, fn func() error) error {
    start := time.Now()
    err := fn()
    duration := time.Since(start)
    
    // Log metrics
    log.Printf("Query: %s, Duration: %v", queryType, duration)
    
    // Send to monitoring system
    metrics.RecordQueryDuration(queryType, duration)
    
    return err
}
```

### 1.4 Validation

```sql
-- Verify indexes were created
SELECT name FROM sqlite_master 
WHERE type = 'index' 
AND tbl_name = 'transcriptions';

-- Check query performance
EXPLAIN QUERY PLAN 
SELECT * FROM transcriptions 
WHERE user = 'test' AND has_error = 0;
```

## Stage 2: Schema Expansion (Day 3-7)

### 2.1 Add New Columns (Safe Operation)

```sql
-- Add columns one by one to minimize lock time
-- Each ALTER TABLE is a separate transaction

ALTER TABLE transcriptions 
ADD COLUMN file_hash TEXT;

ALTER TABLE transcriptions 
ADD COLUMN file_size INTEGER DEFAULT 0;

ALTER TABLE transcriptions 
ADD COLUMN provider_type TEXT DEFAULT 'whisper_cpp';

ALTER TABLE transcriptions 
ADD COLUMN language TEXT DEFAULT 'zh';

ALTER TABLE transcriptions 
ADD COLUMN model_name TEXT;

ALTER TABLE transcriptions 
ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE transcriptions 
ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE transcriptions 
ADD COLUMN deleted_at DATETIME;

-- Add index for new columns
CREATE INDEX idx_file_hash 
ON transcriptions(file_hash) 
WHERE file_hash IS NOT NULL;

CREATE INDEX idx_provider_type 
ON transcriptions(provider_type);

CREATE INDEX idx_deleted_at 
ON transcriptions(deleted_at) 
WHERE deleted_at IS NULL;
```

### 2.2 Create Shadow Tables

```sql
-- Create new table with enhanced schema
-- This doesn't affect existing operations

CREATE TABLE transcriptions_v2 (
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
    deleted_at DATETIME,
    UNIQUE(file_hash) -- Prevent duplicate files
);

-- Create all indexes on new table
CREATE INDEX idx_v2_file_name_error ON transcriptions_v2(file_name, has_error);
CREATE INDEX idx_v2_user_error ON transcriptions_v2(user, has_error);
CREATE INDEX idx_v2_conversion_time ON transcriptions_v2(last_conversion_time);
CREATE INDEX idx_v2_user_time ON transcriptions_v2(user, last_conversion_time DESC);
CREATE INDEX idx_v2_file_hash ON transcriptions_v2(file_hash);
CREATE INDEX idx_v2_provider_type ON transcriptions_v2(provider_type);
CREATE INDEX idx_v2_deleted_at ON transcriptions_v2(deleted_at) WHERE deleted_at IS NULL;
```

### 2.3 Setup Bidirectional Sync

```sql
-- Trigger to sync from old to new table
CREATE TRIGGER sync_insert_to_v2
AFTER INSERT ON transcriptions
BEGIN
    INSERT INTO transcriptions_v2 (
        id, user, input_dir, file_name, mp3_file_name,
        audio_duration, transcription, last_conversion_time,
        has_error, error_message, created_at
    ) VALUES (
        NEW.id, NEW.user, NEW.input_dir, NEW.file_name, 
        NEW.mp3_file_name, NEW.audio_duration, NEW.transcription,
        NEW.last_conversion_time, NEW.has_error, NEW.error_message,
        datetime('now')
    );
END;

CREATE TRIGGER sync_update_to_v2
AFTER UPDATE ON transcriptions
BEGIN
    UPDATE transcriptions_v2 SET
        user = NEW.user,
        input_dir = NEW.input_dir,
        file_name = NEW.file_name,
        mp3_file_name = NEW.mp3_file_name,
        audio_duration = NEW.audio_duration,
        transcription = NEW.transcription,
        last_conversion_time = NEW.last_conversion_time,
        has_error = NEW.has_error,
        error_message = NEW.error_message,
        updated_at = datetime('now')
    WHERE id = NEW.id;
END;

-- Trigger to sync from new to old table
CREATE TRIGGER sync_insert_to_v1
AFTER INSERT ON transcriptions_v2
WHEN NEW.id NOT IN (SELECT id FROM transcriptions)
BEGIN
    INSERT INTO transcriptions (
        id, user, input_dir, file_name, mp3_file_name,
        audio_duration, transcription, last_conversion_time,
        has_error, error_message
    ) VALUES (
        NEW.id, NEW.user, NEW.input_dir, NEW.file_name, 
        NEW.mp3_file_name, NEW.audio_duration, NEW.transcription,
        NEW.last_conversion_time, NEW.has_error, NEW.error_message
    );
END;

-- Copy existing data to shadow table
INSERT INTO transcriptions_v2 (
    id, user, input_dir, file_name, mp3_file_name,
    audio_duration, transcription, last_conversion_time,
    has_error, error_message, created_at, updated_at
)
SELECT 
    id, user, input_dir, file_name, mp3_file_name,
    audio_duration, transcription, last_conversion_time,
    has_error, error_message, 
    last_conversion_time as created_at,
    last_conversion_time as updated_at
FROM transcriptions;
```

### 2.4 Data Validation

```sql
-- Verify data consistency
SELECT 
    (SELECT COUNT(*) FROM transcriptions) as v1_count,
    (SELECT COUNT(*) FROM transcriptions_v2) as v2_count;

-- Check for any discrepancies
SELECT t1.id, 'v1_only' as status
FROM transcriptions t1
LEFT JOIN transcriptions_v2 t2 ON t1.id = t2.id
WHERE t2.id IS NULL
UNION ALL
SELECT t2.id, 'v2_only' as status
FROM transcriptions_v2 t2
LEFT JOIN transcriptions t1 ON t2.id = t1.id
WHERE t1.id IS NULL;
```

## Stage 3: Application Changes (Week 2-3)

### 3.1 Feature Flag System

```go
// config/features.go
package config

import (
    "os"
    "strconv"
)

type FeatureFlags struct {
    UseNewSchema      bool
    MigrationPercent  int
    EnableMetrics     bool
}

func LoadFeatureFlags() *FeatureFlags {
    return &FeatureFlags{
        UseNewSchema:     getEnvBool("USE_NEW_SCHEMA", false),
        MigrationPercent: getEnvInt("MIGRATION_PERCENT", 0),
        EnableMetrics:    getEnvBool("ENABLE_METRICS", true),
    }
}

func getEnvBool(key string, defaultVal bool) bool {
    if val, exists := os.LookupEnv(key); exists {
        if b, err := strconv.ParseBool(val); err == nil {
            return b
        }
    }
    return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
    if val, exists := os.LookupEnv(key); exists {
        if i, err := strconv.Atoi(val); err == nil {
            return i
        }
    }
    return defaultVal
}
```

### 3.2 DAO Interface Versioning

```go
// repository/dao_v2.go
package repository

import (
    "time"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "os"
)

// Extended model for v2 schema
type TranscriptionV2 struct {
    ID                 int        `json:"id"`
    User               string     `json:"user"`
    InputDir           string     `json:"input_dir"`
    FileName           string     `json:"file_name"`
    Mp3FileName        string     `json:"mp3_file_name"`
    AudioDuration      int        `json:"audio_duration"`
    Transcription      string     `json:"transcription"`
    LastConversionTime time.Time  `json:"last_conversion_time"`
    HasError           int        `json:"has_error"`
    ErrorMessage       string     `json:"error_message"`
    // New fields
    FileHash           string     `json:"file_hash,omitempty"`
    FileSize           int64      `json:"file_size,omitempty"`
    ProviderType       string     `json:"provider_type,omitempty"`
    Language           string     `json:"language,omitempty"`
    ModelName          string     `json:"model_name,omitempty"`
    CreatedAt          time.Time  `json:"created_at"`
    UpdatedAt          time.Time  `json:"updated_at"`
    DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

// Extended DAO interface
type TranscriptionDAOV2 interface {
    TranscriptionDAO // Embed v1 interface
    
    // New methods
    RecordToDBV2(t *TranscriptionV2) error
    GetByHashV2(fileHash string) (*TranscriptionV2, error)
    GetActiveTranscriptionsV2() ([]TranscriptionV2, error)
    SoftDeleteV2(id int) error
}

// Calculate file hash
func CalculateFileHash(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }

    return hex.EncodeToString(hash.Sum(nil)), nil
}
```

### 3.3 Dual-Compatible DAO Implementation

```go
// repository/sqlite/dao_dual.go
package sqlite

import (
    "database/sql"
    "fmt"
    "hash/fnv"
)

type DualSchemaDAO struct {
    db           *sql.DB
    features     *config.FeatureFlags
    v1TableName  string
    v2TableName  string
}

func NewDualSchemaDAO(db *sql.DB, features *config.FeatureFlags) *DualSchemaDAO {
    return &DualSchemaDAO{
        db:          db,
        features:    features,
        v1TableName: "transcriptions",
        v2TableName: "transcriptions_v2",
    }
}

// Route to appropriate table based on feature flags
func (d *DualSchemaDAO) shouldUseV2(key string) bool {
    if !d.features.UseNewSchema {
        return false
    }
    
    // Use hash-based routing for gradual migration
    if d.features.MigrationPercent > 0 && d.features.MigrationPercent < 100 {
        h := fnv.New32a()
        h.Write([]byte(key))
        hashPercent := int(h.Sum32() % 100)
        return hashPercent < d.features.MigrationPercent
    }
    
    return d.features.MigrationPercent == 100
}

// Implement v1 interface with routing
func (d *DualSchemaDAO) CheckIfFileProcessed(fileName string) (int, error) {
    tableName := d.v1TableName
    if d.shouldUseV2(fileName) {
        tableName = d.v2TableName
    }
    
    query := fmt.Sprintf(`
        SELECT COUNT(*) 
        FROM %s 
        WHERE file_name = ? 
        AND has_error = 0
        AND (deleted_at IS NULL OR deleted_at = '')
    `, tableName)
    
    var count int
    err := d.db.QueryRow(query, fileName).Scan(&count)
    return count, err
}

// Implement dual write for data consistency
func (d *DualSchemaDAO) RecordToDB(user, inputDir, fileName, mp3FileName string,
    audioDuration int, transcription string, lastConversionTime time.Time,
    hasError int, errorMessage string) error {
    
    // Always write to v1 for compatibility
    err := d.recordToV1(user, inputDir, fileName, mp3FileName, audioDuration,
        transcription, lastConversionTime, hasError, errorMessage)
    if err != nil {
        return err
    }
    
    // Optionally write to v2 if migration is active
    if d.features.UseNewSchema {
        // Calculate file hash if available
        fileHash := ""
        fileSize := int64(0)
        if filePath := fmt.Sprintf("%s/%s", inputDir, fileName); fileExists(filePath) {
            fileHash, _ = CalculateFileHash(filePath)
            if info, err := os.Stat(filePath); err == nil {
                fileSize = info.Size()
            }
        }
        
        v2Record := &TranscriptionV2{
            User:               user,
            InputDir:           inputDir,
            FileName:           fileName,
            Mp3FileName:        mp3FileName,
            AudioDuration:      audioDuration,
            Transcription:      transcription,
            LastConversionTime: lastConversionTime,
            HasError:           hasError,
            ErrorMessage:       errorMessage,
            FileHash:           fileHash,
            FileSize:           fileSize,
            ProviderType:       "whisper_cpp", // Default, should be passed in
            Language:           "zh",
            CreatedAt:          time.Now(),
            UpdatedAt:          time.Now(),
        }
        
        return d.recordToV2(v2Record)
    }
    
    return nil
}
```

### 3.4 Monitoring and Metrics

```go
// monitoring/metrics.go
package monitoring

import (
    "sync"
    "time"
)

type MigrationMetrics struct {
    mu                 sync.RWMutex
    v1Queries          int64
    v2Queries          int64
    v1Errors           int64
    v2Errors           int64
    v1Duration         time.Duration
    v2Duration         time.Duration
    lastResetTime      time.Time
}

func NewMigrationMetrics() *MigrationMetrics {
    return &MigrationMetrics{
        lastResetTime: time.Now(),
    }
}

func (m *MigrationMetrics) RecordQuery(version string, duration time.Duration, err error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if version == "v1" {
        m.v1Queries++
        m.v1Duration += duration
        if err != nil {
            m.v1Errors++
        }
    } else {
        m.v2Queries++
        m.v2Duration += duration
        if err != nil {
            m.v2Errors++
        }
    }
}

func (m *MigrationMetrics) GetStats() map[string]interface{} {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    return map[string]interface{}{
        "v1_queries":      m.v1Queries,
        "v2_queries":      m.v2Queries,
        "v1_errors":       m.v1Errors,
        "v2_errors":       m.v2Errors,
        "v1_avg_duration": m.getAvgDuration(m.v1Duration, m.v1Queries),
        "v2_avg_duration": m.getAvgDuration(m.v2Duration, m.v2Queries),
        "uptime":          time.Since(m.lastResetTime),
    }
}

func (m *MigrationMetrics) getAvgDuration(total time.Duration, count int64) time.Duration {
    if count == 0 {
        return 0
    }
    return total / time.Duration(count)
}
```

## Stage 4: Progressive Migration (Week 4-5)

### 4.1 Migration Control Script

```bash
#!/bin/bash
# migration_control.sh

# Function to update migration percentage
update_migration_percent() {
    local percent=$1
    echo "Setting migration to ${percent}%"
    
    # Update environment variable
    export MIGRATION_PERCENT=$percent
    
    # Update config file
    echo "MIGRATION_PERCENT=$percent" > /etc/tiktok-whisper/migration.conf
    
    # Reload application (graceful restart)
    systemctl reload tiktok-whisper
}

# Progressive migration schedule
echo "Starting progressive migration..."

# Week 1: Canary testing
update_migration_percent 1
sleep 3600  # Monitor for 1 hour

update_migration_percent 5
sleep 86400  # Monitor for 1 day

# Week 2: Gradual increase
for percent in 10 20 30 40 50; do
    update_migration_percent $percent
    sleep 86400  # 1 day per stage
done

# Week 3: Majority migration
for percent in 60 70 80 90; do
    update_migration_percent $percent
    sleep 43200  # 12 hours per stage
done

# Final migration
update_migration_percent 100
echo "Migration complete!"
```

### 4.2 Health Check System

```go
// healthcheck/migration_health.go
package healthcheck

import (
    "context"
    "fmt"
    "time"
)

type MigrationHealthChecker struct {
    dao        repository.TranscriptionDAO
    daoV2      repository.TranscriptionDAOV2
    metrics    *monitoring.MigrationMetrics
    thresholds HealthThresholds
}

type HealthThresholds struct {
    MaxErrorRate     float64
    MaxLatencyMs     int64
    MinSuccessRate   float64
}

func (h *MigrationHealthChecker) CheckHealth(ctx context.Context) error {
    stats := h.metrics.GetStats()
    
    // Check error rates
    v1ErrorRate := float64(stats["v1_errors"].(int64)) / float64(stats["v1_queries"].(int64))
    v2ErrorRate := float64(stats["v2_errors"].(int64)) / float64(stats["v2_queries"].(int64))
    
    if v2ErrorRate > h.thresholds.MaxErrorRate {
        return fmt.Errorf("v2 error rate too high: %.2f%%", v2ErrorRate*100)
    }
    
    // Check latency
    v2AvgDuration := stats["v2_avg_duration"].(time.Duration)
    if v2AvgDuration.Milliseconds() > h.thresholds.MaxLatencyMs {
        return fmt.Errorf("v2 latency too high: %v", v2AvgDuration)
    }
    
    // Check data consistency
    if err := h.checkDataConsistency(ctx); err != nil {
        return fmt.Errorf("data consistency check failed: %w", err)
    }
    
    return nil
}

func (h *MigrationHealthChecker) checkDataConsistency(ctx context.Context) error {
    // Sample recent records for consistency check
    query := `
        SELECT t1.id, t1.file_name, t1.transcription
        FROM transcriptions t1
        LEFT JOIN transcriptions_v2 t2 ON t1.id = t2.id
        WHERE t1.last_conversion_time > datetime('now', '-1 hour')
        AND (
            t2.id IS NULL OR
            t1.transcription != t2.transcription OR
            t1.audio_duration != t2.audio_duration
        )
        LIMIT 10
    `
    
    // Execute consistency check
    // Return error if inconsistencies found
    return nil
}
```

### 4.3 Automatic Rollback

```go
// migration/auto_rollback.go
package migration

import (
    "log"
    "time"
)

type AutoRollbackManager struct {
    healthChecker *healthcheck.MigrationHealthChecker
    features      *config.FeatureFlags
    checkInterval time.Duration
    rollbackChan  chan bool
}

func (m *AutoRollbackManager) Start() {
    ticker := time.NewTicker(m.checkInterval)
    defer ticker.Stop()
    
    consecutiveFailures := 0
    maxFailures := 3
    
    for {
        select {
        case <-ticker.C:
            if err := m.healthChecker.CheckHealth(context.Background()); err != nil {
                consecutiveFailures++
                log.Printf("Health check failed (%d/%d): %v", 
                    consecutiveFailures, maxFailures, err)
                
                if consecutiveFailures >= maxFailures {
                    log.Println("Triggering automatic rollback!")
                    m.triggerRollback()
                    return
                }
            } else {
                consecutiveFailures = 0
            }
            
        case <-m.rollbackChan:
            return
        }
    }
}

func (m *AutoRollbackManager) triggerRollback() {
    // Disable new schema usage
    m.features.UseNewSchema = false
    m.features.MigrationPercent = 0
    
    // Notify operations team
    alert := Alert{
        Severity: "CRITICAL",
        Message:  "Automatic rollback triggered for database migration",
        Time:     time.Now(),
    }
    sendAlert(alert)
}
```

## Stage 5: Validation and Cutover (Week 5-6)

### 5.1 Final Data Validation

```sql
-- Comprehensive validation queries

-- 1. Record count validation
SELECT 
    'Record Count' as check_name,
    (SELECT COUNT(*) FROM transcriptions) as v1_count,
    (SELECT COUNT(*) FROM transcriptions_v2) as v2_count,
    CASE 
        WHEN (SELECT COUNT(*) FROM transcriptions) = 
             (SELECT COUNT(*) FROM transcriptions_v2)
        THEN 'PASS'
        ELSE 'FAIL'
    END as status;

-- 2. Data integrity validation
SELECT 
    'Data Integrity' as check_name,
    COUNT(*) as mismatched_records
FROM transcriptions t1
JOIN transcriptions_v2 t2 ON t1.id = t2.id
WHERE 
    t1.user != t2.user OR
    t1.file_name != t2.file_name OR
    t1.audio_duration != t2.audio_duration OR
    t1.transcription != t2.transcription;

-- 3. Index usage validation
EXPLAIN QUERY PLAN
SELECT * FROM transcriptions_v2 
WHERE user = 'test' 
AND has_error = 0 
AND deleted_at IS NULL;

-- 4. Performance comparison
.timer on
SELECT COUNT(*) FROM transcriptions WHERE user = 'test';
SELECT COUNT(*) FROM transcriptions_v2 WHERE user = 'test';
.timer off
```

### 5.2 Cutover Procedure

```bash
#!/bin/bash
# cutover.sh

echo "Starting final cutover procedure..."

# Step 1: Stop write traffic (maintenance mode)
echo "Enabling maintenance mode..."
touch /var/run/tiktok-whisper/maintenance.flag

# Step 2: Final sync verification
echo "Verifying final data sync..."
sqlite3 /data/transcription.db <<EOF
SELECT 
    CASE 
        WHEN v1.cnt = v2.cnt THEN 'SYNC OK: ' || v1.cnt || ' records'
        ELSE 'SYNC FAIL: v1=' || v1.cnt || ', v2=' || v2.cnt
    END as sync_status
FROM 
    (SELECT COUNT(*) as cnt FROM transcriptions) v1,
    (SELECT COUNT(*) as cnt FROM transcriptions_v2) v2;
EOF

# Step 3: Rename tables
echo "Performing table swap..."
sqlite3 /data/transcription.db <<EOF
BEGIN TRANSACTION;

-- Drop sync triggers
DROP TRIGGER IF EXISTS sync_insert_to_v2;
DROP TRIGGER IF EXISTS sync_update_to_v2;
DROP TRIGGER IF EXISTS sync_insert_to_v1;

-- Rename tables
ALTER TABLE transcriptions RENAME TO transcriptions_old;
ALTER TABLE transcriptions_v2 RENAME TO transcriptions;

-- Update application to use new schema only
UPDATE app_config SET value = 'true' WHERE key = 'use_new_schema_only';

COMMIT;
EOF

# Step 4: Update application configuration
echo "Updating application configuration..."
cat > /etc/tiktok-whisper/migration.conf <<EOF
USE_NEW_SCHEMA=true
MIGRATION_PERCENT=100
MIGRATION_COMPLETE=true
EOF

# Step 5: Restart application
echo "Restarting application..."
systemctl restart tiktok-whisper

# Step 6: Remove maintenance mode
echo "Disabling maintenance mode..."
rm -f /var/run/tiktok-whisper/maintenance.flag

echo "Cutover complete!"
```

### 5.3 Post-Migration Cleanup

```sql
-- After successful validation (Week 6+)

-- 1. Archive old table
CREATE TABLE transcriptions_archive_20240101 AS 
SELECT * FROM transcriptions_old;

-- 2. Create cleanup tracking
CREATE TABLE migration_cleanup_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action TEXT NOT NULL,
    affected_rows INTEGER,
    executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 3. Remove old triggers (if any remain)
DROP TRIGGER IF EXISTS sync_insert_to_v2;
DROP TRIGGER IF EXISTS sync_update_to_v2;
DROP TRIGGER IF EXISTS sync_insert_to_v1;

-- 4. Optimize database
VACUUM;
ANALYZE;

-- 5. Log cleanup actions
INSERT INTO migration_cleanup_log (action, affected_rows)
VALUES ('Archived old table', (SELECT COUNT(*) FROM transcriptions_old));
```

## Rollback Procedures

### Stage-Specific Rollback

#### Stage 1 Rollback (Indexes)
```sql
-- Remove indexes if performance degrades
DROP INDEX IF EXISTS idx_file_name_error;
DROP INDEX IF EXISTS idx_user_error;
DROP INDEX IF EXISTS idx_conversion_time;
DROP INDEX IF EXISTS idx_user_time;

-- Revert WAL mode if needed
PRAGMA journal_mode = DELETE;
```

#### Stage 2 Rollback (Schema)
```sql
-- Remove sync triggers
DROP TRIGGER IF EXISTS sync_insert_to_v2;
DROP TRIGGER IF EXISTS sync_update_to_v2;
DROP TRIGGER IF EXISTS sync_insert_to_v1;

-- Drop shadow table
DROP TABLE IF EXISTS transcriptions_v2;

-- Remove added columns (if needed)
-- Note: SQLite doesn't support DROP COLUMN, would need table recreation
```

#### Stage 3 Rollback (Application)
```bash
# Revert to previous application version
git checkout v1.0.0
go build -o v2t ./cmd/v2t/main.go

# Reset feature flags
export USE_NEW_SCHEMA=false
export MIGRATION_PERCENT=0

# Restart application
systemctl restart tiktok-whisper
```

#### Stage 4 Rollback (Migration)
```bash
# Immediate rollback to 0%
./migration_control.sh rollback

# Or gradual rollback
for percent in 90 70 50 30 10 0; do
    update_migration_percent $percent
    sleep 3600  # Monitor each stage
done
```

## Monitoring Dashboard

### Key Metrics to Track

```go
// monitoring/dashboard.go
package monitoring

type MigrationDashboard struct {
    // Real-time metrics
    CurrentMigrationPercent int
    V1QPS                   float64
    V2QPS                   float64
    V1ErrorRate             float64
    V2ErrorRate             float64
    V1P95Latency            time.Duration
    V2P95Latency            time.Duration
    
    // Cumulative metrics
    TotalV1Queries          int64
    TotalV2Queries          int64
    TotalV1Errors           int64
    TotalV2Errors           int64
    DataConsistencyScore    float64
    
    // System health
    DatabaseSize            int64
    ConnectionPoolUsage     float64
    WALCheckpointLag        int64
}

func (d *MigrationDashboard) GenerateReport() string {
    return fmt.Sprintf(`
Migration Status Report
======================
Migration Progress: %d%%
Active Schema: %s

Performance Metrics:
- V1 QPS: %.2f (errors: %.2f%%)
- V2 QPS: %.2f (errors: %.2f%%)
- V1 P95 Latency: %v
- V2 P95 Latency: %v

Data Quality:
- Consistency Score: %.2f%%
- Total Records: %d
- Sync Lag: %d records

System Health:
- DB Size: %s
- Connection Pool: %.1f%% used
- WAL Checkpoint Lag: %d pages
`,
        d.CurrentMigrationPercent,
        d.getActiveSchema(),
        d.V1QPS, d.V1ErrorRate*100,
        d.V2QPS, d.V2ErrorRate*100,
        d.V1P95Latency,
        d.V2P95Latency,
        d.DataConsistencyScore*100,
        d.TotalV1Queries + d.TotalV2Queries,
        d.getSyncLag(),
        humanizeBytes(d.DatabaseSize),
        d.ConnectionPoolUsage*100,
        d.WALCheckpointLag,
    )
}
```

### Grafana Dashboard Configuration

```json
{
  "dashboard": {
    "title": "TikTok Whisper Migration Dashboard",
    "panels": [
      {
        "title": "Migration Progress",
        "type": "gauge",
        "targets": [
          {
            "expr": "migration_percent"
          }
        ]
      },
      {
        "title": "Query Distribution",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(queries_total{version=\"v1\"}[5m])",
            "legendFormat": "V1 Queries"
          },
          {
            "expr": "rate(queries_total{version=\"v2\"}[5m])",
            "legendFormat": "V2 Queries"
          }
        ]
      },
      {
        "title": "Error Rates",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(errors_total{version=\"v1\"}[5m]) / rate(queries_total{version=\"v1\"}[5m])",
            "legendFormat": "V1 Error Rate"
          },
          {
            "expr": "rate(errors_total{version=\"v2\"}[5m]) / rate(queries_total{version=\"v2\"}[5m])",
            "legendFormat": "V2 Error Rate"
          }
        ]
      },
      {
        "title": "Query Latency (P95)",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, query_duration_seconds{version=\"v1\"})",
            "legendFormat": "V1 P95"
          },
          {
            "expr": "histogram_quantile(0.95, query_duration_seconds{version=\"v2\"})",
            "legendFormat": "V2 P95"
          }
        ]
      }
    ]
  }
}
```

## Testing Strategy

### 1. Unit Tests for Dual DAO

```go
// repository/sqlite/dao_dual_test.go
package sqlite

import (
    "testing"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

func TestDualSchemaRouting(t *testing.T) {
    tests := []struct {
        name             string
        migrationPercent int
        testKeys         []string
        expectedV2Count  int
    }{
        {
            name:             "No migration",
            migrationPercent: 0,
            testKeys:         []string{"file1", "file2", "file3"},
            expectedV2Count:  0,
        },
        {
            name:             "50% migration",
            migrationPercent: 50,
            testKeys:         generateTestKeys(100),
            expectedV2Count:  45, // Allow ±5% variance
        },
        {
            name:             "Full migration",
            migrationPercent: 100,
            testKeys:         []string{"file1", "file2", "file3"},
            expectedV2Count:  3,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            features := &config.FeatureFlags{
                UseNewSchema:     true,
                MigrationPercent: tt.migrationPercent,
            }
            
            dao := NewDualSchemaDAO(nil, features)
            v2Count := 0
            
            for _, key := range tt.testKeys {
                if dao.shouldUseV2(key) {
                    v2Count++
                }
            }
            
            // Allow some variance for percentage-based routing
            if tt.migrationPercent > 0 && tt.migrationPercent < 100 {
                variance := len(tt.testKeys) / 20 // 5% variance
                if abs(v2Count-tt.expectedV2Count) > variance {
                    t.Errorf("Expected ~%d v2 routes, got %d", 
                        tt.expectedV2Count, v2Count)
                }
            } else if v2Count != tt.expectedV2Count {
                t.Errorf("Expected %d v2 routes, got %d", 
                    tt.expectedV2Count, v2Count)
            }
        })
    }
}
```

### 2. Integration Tests

```go
// integration/migration_test.go
package integration

func TestDataConsistencyDuringMigration(t *testing.T) {
    // Setup test database with both schemas
    db := setupTestDB(t)
    defer db.Close()
    
    // Create DAOs
    v1DAO := sqlite.NewTranscriptionDAO(db)
    dualDAO := sqlite.NewDualSchemaDAO(db, &config.FeatureFlags{
        UseNewSchema:     true,
        MigrationPercent: 50,
    })
    
    // Insert test data
    testData := []struct {
        user          string
        fileName      string
        transcription string
    }{
        {"user1", "file1.mp3", "transcription 1"},
        {"user2", "file2.mp3", "transcription 2"},
        {"user3", "file3.mp3", "transcription 3"},
    }
    
    for _, td := range testData {
        err := dualDAO.RecordToDB(
            td.user, "/data", td.fileName, td.fileName,
            120, td.transcription, time.Now(), 0, "",
        )
        if err != nil {
            t.Fatalf("Failed to insert: %v", err)
        }
    }
    
    // Verify data exists in both tables
    for _, td := range testData {
        // Check v1 table
        count, err := v1DAO.CheckIfFileProcessed(td.fileName)
        if err != nil || count == 0 {
            t.Errorf("File %s not found in v1 table", td.fileName)
        }
        
        // Check v2 table
        var v2Count int
        err = db.QueryRow(
            "SELECT COUNT(*) FROM transcriptions_v2 WHERE file_name = ?",
            td.fileName,
        ).Scan(&v2Count)
        if err != nil || v2Count == 0 {
            t.Errorf("File %s not found in v2 table", td.fileName)
        }
    }
}
```

### 3. Load Testing

```bash
#!/bin/bash
# load_test.sh

# Simulate production load during migration
echo "Starting load test..."

# Function to run queries
run_queries() {
    local num_queries=$1
    local user=$2
    
    for i in $(seq 1 $num_queries); do
        # Random operations
        case $((i % 4)) in
            0) # Insert
                curl -X POST http://localhost:8080/api/transcribe \
                    -d "{\"user\":\"$user\",\"file\":\"test$i.mp3\"}"
                ;;
            1) # Query by user
                curl http://localhost:8080/api/transcriptions?user=$user
                ;;
            2) # Check file
                curl http://localhost:8080/api/check?file=test$i.mp3
                ;;
            3) # Get all
                curl http://localhost:8080/api/transcriptions
                ;;
        esac
        
        # Random delay between requests (10-100ms)
        sleep 0.0$((RANDOM % 10))
    done
}

# Run parallel load
for user in user1 user2 user3 user4 user5; do
    run_queries 1000 $user &
done

# Monitor while load test runs
while true; do
    sleep 5
    echo "Current metrics:"
    curl -s http://localhost:8080/metrics | grep -E "(queries_total|errors_total|migration_percent)"
done
```

## Operational Runbook

### Pre-Migration Checklist

- [ ] Database backup completed
- [ ] Test environment validated
- [ ] Monitoring dashboards configured
- [ ] Alert rules configured
- [ ] Rollback procedures tested
- [ ] Team communication plan ready
- [ ] Maintenance window scheduled (if needed)

### Migration Day Procedures

1. **Morning (Start of Business)**
   - Verify database backup
   - Check system health
   - Enable enhanced monitoring
   - Begin Stage 1 (indexes)

2. **Midday Check**
   - Verify index creation success
   - Check query performance
   - Monitor error rates
   - Proceed to Stage 2 if healthy

3. **End of Day**
   - Complete Stage 2 (schema expansion)
   - Verify data sync
   - Set up overnight monitoring
   - Prepare for next day's application deployment

### Emergency Procedures

**High Error Rate:**
1. Check recent changes
2. Review error logs
3. Rollback if >5% error rate
4. Investigate root cause

**Performance Degradation:**
1. Check query plans
2. Verify index usage
3. Check for lock contention
4. Scale back migration percentage

**Data Inconsistency:**
1. Stop writes immediately
2. Run consistency checks
3. Identify affected records
4. Execute repair procedures

### Communication Templates

**Migration Start:**
```
Subject: Database Migration Starting - tiktok-whisper

Team,

We are beginning the planned zero-downtime database migration.

- Duration: 4-6 weeks
- Impact: None expected
- Monitoring: Enhanced monitoring active

Dashboard: https://monitoring/tiktok-whisper-migration
Runbook: https://wiki/tiktok-whisper-migration

Contact: [Your Name] if issues arise.
```

**Daily Update:**
```
Subject: Migration Progress - Day X

Current Status:
- Migration: XX% complete
- Performance: Normal
- Errors: X.XX% (within threshold)
- Next Steps: [Planned activities]

No action required.
```

## Success Criteria

1. **Zero Downtime**: No service interruption during migration
2. **Data Integrity**: 100% data consistency between schemas
3. **Performance**: <10ms P95 latency maintained
4. **Error Rate**: <0.1% error rate throughout
5. **Rollback Ready**: <5 minute rollback time at any stage

## Conclusion

This zero-downtime migration plan provides a safe, progressive approach to upgrading the tiktok-whisper database schema. By following the expand-and-contract pattern with comprehensive monitoring and rollback capabilities, we ensure business continuity while improving the system's capabilities.

The migration can be paused, rolled back, or accelerated based on real-time metrics, providing maximum flexibility and safety throughout the process.