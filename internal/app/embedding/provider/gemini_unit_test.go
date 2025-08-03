package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test Gemini provider interface compliance (unit test - no API calls)
func TestGeminiProviderInterface_Unit(t *testing.T) {
	// Arrange
	var provider EmbeddingProvider
	provider = NewGeminiProvider("test-key")

	// Act
	info := provider.GetProviderInfo()

	// Assert
	assert.Equal(t, "gemini", info.Name)
	assert.Equal(t, "gemini-embedding-001", info.Model)
	assert.Equal(t, 768, info.Dimension)

	// Verify interface methods are implemented
	_, ok := provider.(EmbeddingProvider)
	assert.True(t, ok, "GeminiProvider should implement EmbeddingProvider interface")
}

// Test Gemini constructor and configuration (unit test)
func TestGeminiProviderConstructor_Unit(t *testing.T) {
	testCases := []struct {
		name   string
		apiKey string
	}{
		{"with valid API key", "test-api-key-123"},
		{"with empty API key", ""},
		{"with whitespace API key", "   "},
		{"with special characters", "key-with-@#$%^&*()_+="},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			provider := NewGeminiProvider(tc.apiKey)

			// Assert
			assert.NotNil(t, provider)
			assert.Equal(t, tc.apiKey, provider.apiKey)
			assert.Equal(t, "gemini-embedding-001", provider.model)
		})
	}
}