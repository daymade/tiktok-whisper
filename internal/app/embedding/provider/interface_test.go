package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TDD Cycle 1: GREEN - Test basic interface compliance
func TestEmbeddingProviderInterface(t *testing.T) {
	// Arrange
	var provider EmbeddingProvider
	provider = NewMockProvider(768)
	ctx := context.Background()
	testText := "test text for embedding"

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, testText)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, embedding)
	assert.Greater(t, len(embedding), 0)
}

// Test that provider returns metadata
func TestEmbeddingProviderMetadata(t *testing.T) {
	// Arrange
	var provider EmbeddingProvider
	provider = NewMockProvider(768)

	// Act
	info := provider.GetProviderInfo()

	// Assert
	assert.NotEmpty(t, info.Name)
	assert.NotEmpty(t, info.Model)
	assert.Greater(t, info.Dimension, 0)
}