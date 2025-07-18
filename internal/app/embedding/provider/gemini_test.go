package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test Gemini provider interface compliance
func TestGeminiProviderInterface(t *testing.T) {
	// Arrange
	var provider EmbeddingProvider
	provider = NewGeminiProvider("dummy-key")

	// Act
	info := provider.GetProviderInfo()

	// Assert
	assert.Equal(t, "gemini", info.Name)
	assert.Equal(t, "models/embedding-001", info.Model)
	assert.Equal(t, 768, info.Dimension)
}

// Test Gemini embedding generation (mock implementation)
func TestGeminiEmbeddingGeneration(t *testing.T) {
	// Arrange
	provider := NewGeminiProvider("dummy-key")
	ctx := context.Background()

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "Hello, world!")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, embedding)
	assert.Len(t, embedding, 768)
}

// Test Gemini empty text handling
func TestGeminiEmptyText(t *testing.T) {
	// Arrange
	provider := NewGeminiProvider("dummy-key")
	ctx := context.Background()

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "empty text")
}

// Test Gemini deterministic behavior (mock implementation)
func TestGeminiDeterministicBehavior(t *testing.T) {
	// Arrange
	provider := NewGeminiProvider("dummy-key")
	ctx := context.Background()

	// Act - Same input should produce same output
	embedding1, err1 := provider.GenerateEmbedding(ctx, "test text")
	embedding2, err2 := provider.GenerateEmbedding(ctx, "test text")

	// Assert
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, embedding1, embedding2)
}