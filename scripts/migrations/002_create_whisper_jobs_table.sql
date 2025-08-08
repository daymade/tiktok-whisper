-- Create whisper_jobs table for job-based transcription tracking
CREATE TABLE IF NOT EXISTS whisper_jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    whisper_job_id INTEGER REFERENCES transcriptions(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    file_name TEXT NOT NULL,
    file_url TEXT NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    audio_duration INTEGER DEFAULT 0,
    provider VARCHAR(50),
    language VARCHAR(10) DEFAULT 'auto',
    output_format VARCHAR(20) DEFAULT 'text',
    transcription_text TEXT,
    transcription_url TEXT,
    credit_cost INTEGER NOT NULL DEFAULT 0,
    credit_refunded BOOLEAN NOT NULL DEFAULT FALSE,
    error TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance
CREATE INDEX idx_whisper_jobs_user_id ON whisper_jobs(user_id);
CREATE INDEX idx_whisper_jobs_status ON whisper_jobs(status);
CREATE INDEX idx_whisper_jobs_created_at ON whisper_jobs(created_at DESC);
CREATE INDEX idx_whisper_jobs_user_status ON whisper_jobs(user_id, status);

-- Create updated_at trigger
CREATE OR REPLACE FUNCTION update_whisper_jobs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER whisper_jobs_updated_at_trigger
    BEFORE UPDATE ON whisper_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_whisper_jobs_updated_at();