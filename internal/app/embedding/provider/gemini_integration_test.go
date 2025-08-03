//go:build integration
// +build integration

package provider

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test API key validation behavior (integration test - makes real API calls)
func TestGeminiAPIKeyValidation_Integration(t *testing.T) {
	testCases := []struct {
		name        string
		apiKey      string
		expectError bool
	}{
		{"valid looking key", "test-gemini-key", false},
		{"empty key", "", true},
		{"whitespace key", "   ", true},
		{"obviously invalid key", "invalid-key-123", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip if no real API key is available
			realAPIKey := os.Getenv("GEMINI_API_KEY")
			if realAPIKey == "" && !tc.expectError {
				t.Skip("GEMINI_API_KEY not set, skipping API key validation tests")
			}

			// Arrange
			provider := NewGeminiProvider(tc.apiKey)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Act
			_, err := provider.GenerateEmbedding(ctx, "test text")

			// Assert
			if tc.expectError {
				assert.Error(t, err, "Expected error for %s", tc.name)
			} else {
				// For real API testing, use the real key instead
				if realAPIKey != "" {
					provider = NewGeminiProvider(realAPIKey)
					_, err = provider.GenerateEmbedding(ctx, "test text")
				}
				assert.NoError(t, err, "Expected no error for %s", tc.name)
			}
		})
	}
}

// Test Gemini embedding generation with real API (integration test)
func TestGeminiEmbeddingGeneration_Integration(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration tests")
	}

	testCases := []struct {
		name          string
		input         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "simple text",
			input:       "Hello, world!",
			expectError: false,
		},
		{
			name:        "long text",
			input:       strings.Repeat("This is a test sentence. ", 10),
			expectError: false,
		},
		{
			name:        "unicode text",
			input:       "Hello 世界 🌍",
			expectError: false,
		},
		{
			name:          "empty text",
			input:         "",
			expectError:   true,
			errorContains: "empty",
		},
	}

	provider := NewGeminiProvider(apiKey)
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			embedding, err := provider.GenerateEmbedding(ctx, tc.input)

			// Assert
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, embedding)
				assert.Len(t, embedding, 768, "Gemini embeddings should have 768 dimensions")
				
				// Verify embedding values are reasonable
				for i, val := range embedding {
					assert.False(t, isNaN(val), "Embedding value at index %d should not be NaN", i)
					assert.False(t, isInf(val), "Embedding value at index %d should not be infinite", i)
				}
			}
		})
	}
}

// Helper functions
func isNaN(f float32) bool {
	return f != f
}

func isInf(f float32) bool {
	return f > 3.4028235e+38 || f < -3.4028235e+38
}