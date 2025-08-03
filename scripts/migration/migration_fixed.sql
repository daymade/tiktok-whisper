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

-- Add new columns (without non-constant defaults for SQLite compatibility)
ALTER TABLE transcriptions ADD COLUMN file_hash TEXT;
ALTER TABLE transcriptions ADD COLUMN file_size INTEGER DEFAULT 0;
ALTER TABLE transcriptions ADD COLUMN provider_type TEXT DEFAULT 'whisper_cpp';
ALTER TABLE transcriptions ADD COLUMN language TEXT DEFAULT 'zh';
ALTER TABLE transcriptions ADD COLUMN model_name TEXT;
ALTER TABLE transcriptions ADD COLUMN created_at DATETIME;
ALTER TABLE transcriptions ADD COLUMN updated_at DATETIME;
ALTER TABLE transcriptions ADD COLUMN deleted_at DATETIME;

-- Create indexes for new columns
CREATE INDEX idx_file_hash ON transcriptions(file_hash) WHERE file_hash IS NOT NULL;
CREATE INDEX idx_provider_type ON transcriptions(provider_type);
CREATE INDEX idx_deleted_at ON transcriptions(deleted_at) WHERE deleted_at IS NULL;

-- Update timestamps for existing records (use last_conversion_time as created_at/updated_at)
UPDATE transcriptions 
SET created_at = last_conversion_time,
    updated_at = last_conversion_time;

COMMIT;

-- Optimize database
VACUUM;
ANALYZE;