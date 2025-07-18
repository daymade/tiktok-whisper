package provider

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test OpenAI provider interface compliance
func TestOpenAIProviderInterface(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI tests")
	}

	// Arrange
	var provider EmbeddingProvider
	provider = NewOpenAIProvider(apiKey)

	// Act
	info := provider.GetProviderInfo()

	// Assert
	assert.Equal(t, "openai", info.Name)
	assert.Equal(t, "text-embedding-ada-002", info.Model)
	assert.Equal(t, 1536, info.Dimension)
}

// Test OpenAI embedding generation (integration test)
func TestOpenAIEmbeddingGeneration(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI tests")
	}

	// Arrange
	provider := NewOpenAIProvider(apiKey)
	ctx := context.Background()

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "Hello, world!")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, embedding)
	assert.Len(t, embedding, 1536)
}

// Test OpenAI error handling
func TestOpenAIErrorHandling(t *testing.T) {
	// Arrange
	provider := NewOpenAIProvider("invalid-key")
	ctx := context.Background()

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "test")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, embedding)
}

// Test OpenAI empty text handling
func TestOpenAIEmptyText(t *testing.T) {
	// Arrange
	provider := NewOpenAIProvider("dummy-key")
	ctx := context.Background()

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "empty text")
}