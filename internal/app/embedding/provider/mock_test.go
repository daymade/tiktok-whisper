package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TDD Cycle 2: RED - Test deterministic behavior of MockProvider
func TestMockProviderDeterministic(t *testing.T) {
	// Arrange
	provider := NewMockProvider(768)
	ctx := context.Background()

	// Act - Same input should produce same output
	embedding1, err1 := provider.GenerateEmbedding(ctx, "hello world")
	embedding2, err2 := provider.GenerateEmbedding(ctx, "hello world")

	// Assert
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, embedding1, embedding2, "Same input should produce same embedding")

	// Act - Different input should produce different output
	embedding3, err3 := provider.GenerateEmbedding(ctx, "goodbye world")

	// Assert
	assert.NoError(t, err3)
	assert.NotEqual(t, embedding1, embedding3, "Different input should produce different embedding")
}

// Test that mock provider can generate embeddings of different dimensions
func TestMockProviderDimensions(t *testing.T) {
	testCases := []struct {
		name      string
		dimension int
	}{
		{"OpenAI dimension", 1536},
		{"Gemini dimension", 768},
		{"Small dimension", 128},
		{"Large dimension", 4096},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			provider := NewMockProvider(tc.dimension)
			ctx := context.Background()

			// Act
			embedding, err := provider.GenerateEmbedding(ctx, "test text")

			// Assert
			assert.NoError(t, err)
			assert.Len(t, embedding, tc.dimension)
			
			// Verify provider info matches
			info := provider.GetProviderInfo()
			assert.Equal(t, tc.dimension, info.Dimension)
		})
	}
}

// Test that embeddings are normalized to [-1, 1] range
func TestMockProviderNormalization(t *testing.T) {
	// Arrange
	provider := NewMockProvider(100)
	ctx := context.Background()

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "test normalization")

	// Assert
	assert.NoError(t, err)
	
	for i, value := range embedding {
		assert.GreaterOrEqual(t, value, float32(-1.0), "Value at index %d should be >= -1", i)
		assert.LessOrEqual(t, value, float32(1.0), "Value at index %d should be <= 1", i)
	}
}

// Test error handling for empty text
func TestMockProviderEmptyText(t *testing.T) {
	// Arrange
	provider := NewMockProvider(768)
	ctx := context.Background()

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "empty text")
}