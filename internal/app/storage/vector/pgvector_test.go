package vector

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TDD Cycle 4: RED - Test PostgreSQL vector storage implementation
func TestPgVectorStorage(t *testing.T) {
	// Skip if running in CI or no postgres available
	if testing.Short() {
		t.Skip("Skipping PostgreSQL tests in short mode")
	}

	// Setup test database connection
	db, err := sql.Open("postgres", "user=postgres password=passwd dbname=postgres sslmode=disable host=localhost")
	require.NoError(t, err)
	defer db.Close()

	// Test connection
	err = db.Ping()
	require.NoError(t, err)

	// Create storage instance
	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()
	transcriptionID := 1
	openaiEmbedding := make([]float32, 1536)
	geminiEmbedding := make([]float32, 768)

	// Fill with test data
	for i := range openaiEmbedding {
		openaiEmbedding[i] = float32(i) / 1536.0
	}
	for i := range geminiEmbedding {
		geminiEmbedding[i] = float32(i) / 768.0
	}

	// Test dual embedding storage
	err = storage.StoreDualEmbeddings(ctx, transcriptionID, openaiEmbedding, geminiEmbedding)
	assert.NoError(t, err)

	// Test dual embedding retrieval
	retrieved, err := storage.GetDualEmbeddings(ctx, transcriptionID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, len(openaiEmbedding), len(retrieved.OpenAI))
	assert.Equal(t, len(geminiEmbedding), len(retrieved.Gemini))
}

// Test single embedding operations
func TestPgVectorSingleEmbedding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL tests in short mode")
	}

	db, err := sql.Open("postgres", "user=postgres password=passwd dbname=postgres sslmode=disable host=localhost")
	require.NoError(t, err)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()
	transcriptionID := 2
	provider := "openai"
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) / 1536.0
	}

	// Test store single embedding
	err = storage.StoreEmbedding(ctx, transcriptionID, provider, embedding)
	assert.NoError(t, err)

	// Test retrieve single embedding
	retrieved, err := storage.GetEmbedding(ctx, transcriptionID, provider)
	assert.NoError(t, err)
	assert.Equal(t, len(embedding), len(retrieved))
}

// Test getting transcriptions without embeddings
func TestPgVectorGetTranscriptionsWithoutEmbeddings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL tests in short mode")
	}

	db, err := sql.Open("postgres", "user=postgres password=passwd dbname=postgres sslmode=disable host=localhost")
	require.NoError(t, err)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()

	// Test getting transcriptions without embeddings
	transcriptions, err := storage.GetTranscriptionsWithoutEmbeddings(ctx, "openai", 10)
	assert.NoError(t, err)
	assert.NotNil(t, transcriptions)
	
	// Should have transcriptions since we have 1060 records
	assert.Greater(t, len(transcriptions), 0)
}