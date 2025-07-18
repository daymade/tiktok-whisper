package vector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TDD Cycle 3: RED - Test VectorStorage interface
func TestVectorStorageInterface(t *testing.T) {
	// Arrange
	var storage VectorStorage
	storage = NewMockVectorStorage()
	ctx := context.Background()
	
	transcriptionID := 1
	provider := "openai"
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	// Act
	err := storage.StoreEmbedding(ctx, transcriptionID, provider, embedding)
	
	// Assert
	assert.NoError(t, err)
	
	// Test retrieval
	retrieved, err := storage.GetEmbedding(ctx, transcriptionID, provider)
	assert.NoError(t, err)
	assert.Equal(t, embedding, retrieved)
}

// Test dual embedding storage
func TestDualEmbeddingStorage(t *testing.T) {
	// Arrange
	storage := NewMockVectorStorage()
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

	// Act - Store both embeddings
	err := storage.StoreDualEmbeddings(ctx, transcriptionID, openaiEmbedding, geminiEmbedding)
	assert.NoError(t, err)

	// Assert - Retrieve both embeddings
	dualEmbedding, err := storage.GetDualEmbeddings(ctx, transcriptionID)
	assert.NoError(t, err)
	assert.Equal(t, openaiEmbedding, dualEmbedding.OpenAI)
	assert.Equal(t, geminiEmbedding, dualEmbedding.Gemini)
}

// Test embedding not found
func TestEmbeddingNotFound(t *testing.T) {
	// Arrange
	storage := NewMockVectorStorage()
	ctx := context.Background()

	// Act
	_, err := storage.GetEmbedding(ctx, 999, "openai")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Test get transcriptions without embeddings
func TestGetTranscriptionsWithoutEmbeddings(t *testing.T) {
	// Arrange
	storage := NewMockVectorStorage()
	ctx := context.Background()
	
	// Add some test transcriptions
	storage.AddMockTranscription(1, "Test transcription 1")
	storage.AddMockTranscription(2, "Test transcription 2")
	storage.AddMockTranscription(3, "Test transcription 3")
	
	// Store embedding for one transcription
	embedding := []float32{0.1, 0.2, 0.3}
	storage.StoreEmbedding(ctx, 1, "openai", embedding)

	// Act
	transcriptions, err := storage.GetTranscriptionsWithoutEmbeddings(ctx, "openai", 10)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, transcriptions, 2) // Should return 2 transcriptions without embeddings
	
	// Check that transcription 1 is not in the results
	for _, transcription := range transcriptions {
		assert.NotEqual(t, 1, transcription.ID)
	}
}