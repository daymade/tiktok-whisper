//go:build integration
// +build integration

package provider

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test OpenAI embedding generation with various inputs
func TestOpenAIEmbeddingGeneration_Integration(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration tests")
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
			input:       strings.Repeat("This is a test sentence. ", 100),
			expectError: false,
		},
		{
			name:        "unicode text",
			input:       "Hello 世界 🌍 مرحبا",
			expectError: false,
		},
		{
			name:        "text with special characters",
			input:       "Test @#$%^&*()_+-=[]{}|;':,.<>?/`~",
			expectError: false,
		},
		{
			name:        "text with newlines and tabs",
			input:       "Line 1\nLine 2\tTabbed\rCarriage return",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			provider := NewOpenAIProvider(apiKey)
			ctx := context.Background()

			// Act
			embedding, err := provider.GenerateEmbedding(ctx, tc.input)

			// Assert
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, embedding)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, embedding)
				assert.Len(t, embedding, 1536)
				// Verify all values are valid floats
				for i, val := range embedding {
					assert.False(t, isNaN(val), "Value at index %d is NaN", i)
					assert.False(t, isInf(val), "Value at index %d is Inf", i)
				}
			}
		})
	}
}

// Test OpenAI error scenarios with real API
func TestOpenAIErrorScenarios_Integration(t *testing.T) {
	testCases := []struct {
		name          string
		apiKey        string
		input         string
		errorContains string
	}{
		{
			name:          "invalid API key",
			apiKey:        "invalid-key",
			input:         "test",
			errorContains: "", // OpenAI API error message varies
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			provider := NewOpenAIProvider(tc.apiKey)
			ctx := context.Background()

			// Act
			embedding, err := provider.GenerateEmbedding(ctx, tc.input)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, embedding)
		})
	}
}

// Test context cancellation with real API
func TestOpenAIContextCancellation_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration tests")
	}

	// Arrange
	provider := NewOpenAIProvider(apiKey)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "test")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, embedding)
}

// Test concurrent embedding generation
func TestOpenAIConcurrentRequests_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration tests")
	}

	// Arrange
	provider := NewOpenAIProvider(apiKey)
	ctx := context.Background()
	numRequests := 5
	texts := []string{
		"First text",
		"Second text",
		"Third text",
		"Fourth text",
		"Fifth text",
	}

	// Act
	var wg sync.WaitGroup
	results := make([]struct {
		embedding []float32
		err       error
	}, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			embedding, err := provider.GenerateEmbedding(ctx, texts[idx])
			results[idx].embedding = embedding
			results[idx].err = err
		}(i)
	}

	wg.Wait()

	// Assert
	for i, result := range results {
		assert.NoError(t, result.err, "Request %d failed", i)
		assert.NotNil(t, result.embedding, "Request %d returned nil embedding", i)
		assert.Len(t, result.embedding, 1536, "Request %d returned wrong dimension", i)
	}

	// Verify different texts produce different embeddings
	for i := 0; i < numRequests-1; i++ {
		for j := i + 1; j < numRequests; j++ {
			assert.NotEqual(t, results[i].embedding, results[j].embedding,
				"Different texts should produce different embeddings")
		}
	}
}

// Test embedding consistency
func TestOpenAIEmbeddingConsistency_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration tests")
	}

	// Arrange
	provider := NewOpenAIProvider(apiKey)
	ctx := context.Background()
	testText := "Consistency test text"

	// Act - Generate embedding twice for the same text
	embedding1, err1 := provider.GenerateEmbedding(ctx, testText)
	time.Sleep(100 * time.Millisecond) // Small delay to avoid rate limiting
	embedding2, err2 := provider.GenerateEmbedding(ctx, testText)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, embedding1, embedding2, "Same text should produce identical embeddings")
}

// Test extremely long text handling
func TestOpenAIExtremelyLongText_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration tests")
	}

	// Arrange
	provider := NewOpenAIProvider(apiKey)
	ctx := context.Background()
	// OpenAI has a token limit, typically around 8191 tokens
	// Each word is roughly 1-2 tokens, so we'll test with a very long text
	longText := strings.Repeat("This is a test sentence with multiple words. ", 1000)

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, longText)

	// Assert - OpenAI should handle long text gracefully
	// It might truncate or return an error depending on the implementation
	if err != nil {
		// If error, it should be about text length
		t.Logf("Long text resulted in error (expected): %v", err)
	} else {
		assert.NotNil(t, embedding)
		assert.Len(t, embedding, 1536)
	}
}

// Test various UTF-8 edge cases
func TestOpenAIUTF8EdgeCases_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration tests")
	}

	testCases := []struct {
		name  string
		input string
	}{
		{"emoji", "🚀🌟💻🎉🔥"},
		{"chinese", "这是一个中文测试"},
		{"japanese", "これは日本語のテストです"},
		{"arabic", "هذا اختبار باللغة العربية"},
		{"mixed scripts", "Hello мир 世界 🌍"},
		{"zero width characters", "Hello\u200Bworld"}, // Zero-width space
		{"combining characters", "é = e\u0301"},       // e + combining acute accent
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			provider := NewOpenAIProvider(apiKey)
			ctx := context.Background()

			// Verify input is valid UTF-8
			require.True(t, utf8.ValidString(tc.input), "Test input must be valid UTF-8")

			// Act
			embedding, err := provider.GenerateEmbedding(ctx, tc.input)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, embedding)
			assert.Len(t, embedding, 1536)
		})
	}
}

// Benchmark OpenAI embedding generation
func BenchmarkOpenAIEmbeddingGeneration(b *testing.B) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_API_KEY not set, skipping benchmarks")
	}

	provider := NewOpenAIProvider(apiKey)
	ctx := context.Background()
	testText := "This is a benchmark test for OpenAI embedding generation performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateEmbedding(ctx, testText)
		if err != nil {
			b.Fatalf("Failed to generate embedding: %v", err)
		}
	}
}

// Helper functions are defined in gemini_integration_test.go