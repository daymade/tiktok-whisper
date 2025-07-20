-- Migration script to add dual embedding columns to transcriptions table
-- This script supports both OpenAI (1536 dimensions) and Gemini (768 dimensions) embeddings

-- Add dual embedding columns with separate dimensions
ALTER TABLE transcriptions ADD COLUMN embedding_openai vector(1536);
ALTER TABLE transcriptions ADD COLUMN embedding_gemini vector(3072);

-- Metadata for OpenAI embeddings
ALTER TABLE transcriptions ADD COLUMN embedding_openai_model varchar(50);
ALTER TABLE transcriptions ADD COLUMN embedding_openai_created_at timestamp;
ALTER TABLE transcriptions ADD COLUMN embedding_openai_version integer DEFAULT 1;
ALTER TABLE transcriptions ADD COLUMN embedding_openai_status varchar(20) DEFAULT 'pending';
ALTER TABLE transcriptions ADD COLUMN embedding_openai_error text;

-- Metadata for Gemini embeddings  
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_model varchar(50);
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_created_at timestamp;
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_version integer DEFAULT 1;
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_status varchar(20) DEFAULT 'pending';
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_error text;

-- Search configuration
ALTER TABLE transcriptions ADD COLUMN primary_embedding_provider varchar(20) DEFAULT 'openai';
ALTER TABLE transcriptions ADD COLUMN embedding_sync_status varchar(20) DEFAULT 'pending';

-- Performance optimization indexes
CREATE INDEX transcriptions_embedding_openai_idx ON transcriptions 
USING hnsw (embedding_openai vector_cosine_ops) WHERE embedding_openai IS NOT NULL;

CREATE INDEX transcriptions_embedding_gemini_idx ON transcriptions 
USING hnsw (embedding_gemini vector_cosine_ops) WHERE embedding_gemini IS NOT NULL;

-- Status indexes for filtering
CREATE INDEX transcriptions_openai_status_idx ON transcriptions (embedding_openai_status);
CREATE INDEX transcriptions_gemini_status_idx ON transcriptions (embedding_gemini_status);

-- Composite indexes for user-specific queries
CREATE INDEX transcriptions_user_openai_status_idx ON transcriptions (user_nickname, embedding_openai_status);
CREATE INDEX transcriptions_user_gemini_status_idx ON transcriptions (user_nickname, embedding_gemini_status);

-- Embedding provider configuration table
CREATE TABLE IF NOT EXISTS embedding_providers (
    id serial PRIMARY KEY,
    provider_name varchar(50) NOT NULL UNIQUE,
    model_name varchar(100) NOT NULL,
    dimension integer NOT NULL,
    is_active boolean DEFAULT true,
    cost_per_1k_tokens decimal(10,6),
    rate_limit_per_minute integer,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now()
);

-- Insert default providers
INSERT INTO embedding_providers (provider_name, model_name, dimension, cost_per_1k_tokens, rate_limit_per_minute) 
VALUES 
('openai', 'text-embedding-ada-002', 1536, 0.0001, 3000),
('gemini', 'gemini-embedding-001', 3072, 0.0001, 1500)
ON CONFLICT (provider_name) DO UPDATE SET
    model_name = EXCLUDED.model_name,
    dimension = EXCLUDED.dimension,
    cost_per_1k_tokens = EXCLUDED.cost_per_1k_tokens,
    rate_limit_per_minute = EXCLUDED.rate_limit_per_minute,
    updated_at = now();

-- Embedding generation batches for tracking
CREATE TABLE IF NOT EXISTS embedding_batches (
    id serial PRIMARY KEY,
    batch_id varchar(50) NOT NULL UNIQUE,
    provider varchar(50) NOT NULL,
    started_at timestamp DEFAULT now(),
    completed_at timestamp,
    total_items integer NOT NULL,
    processed_items integer DEFAULT 0,
    failed_items integer DEFAULT 0,
    status varchar(20) DEFAULT 'running',
    error_message text
);

-- Add comments for documentation
COMMENT ON COLUMN transcriptions.embedding_openai IS 'OpenAI text-embedding-ada-002 vector (1536 dimensions)';
COMMENT ON COLUMN transcriptions.embedding_gemini IS 'Google Gemini embedding-001 vector (3072 dimensions)';
COMMENT ON COLUMN transcriptions.primary_embedding_provider IS 'Which embedding to use for primary search (openai, gemini, both)';
COMMENT ON COLUMN transcriptions.embedding_sync_status IS 'Status of embedding synchronization (pending, syncing, completed, failed)';

-- Update any existing transcriptions to have pending status
UPDATE transcriptions 
SET embedding_openai_status = 'pending',
    embedding_gemini_status = 'pending',
    embedding_sync_status = 'pending'
WHERE embedding_openai_status IS NULL OR embedding_gemini_status IS NULL;