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

// Test OpenAI provider interface compliance
func TestOpenAIProviderInterface(t *testing.T) {
	// Arrange
	var provider EmbeddingProvider
	provider = NewOpenAIProvider("test-key")

	// Act
	info := provider.GetProviderInfo()

	// Assert
	assert.Equal(t, "openai", info.Name)
	assert.Equal(t, "text-embedding-ada-002", info.Model)
	assert.Equal(t, 1536, info.Dimension)

	// Verify interface methods are implemented
	_, ok := provider.(EmbeddingProvider)
	assert.True(t, ok, "OpenAIProvider should implement EmbeddingProvider interface")
}

// Test OpenAI constructor and configuration
func TestOpenAIProviderConstructor(t *testing.T) {
	testCases := []struct {
		name   string
		apiKey string
	}{
		{"with valid API key", "sk-test123"},
		{"with empty API key", ""},
		{"with whitespace API key", "   "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			provider := NewOpenAIProvider(tc.apiKey)

			// Assert
			assert.NotNil(t, provider)
			assert.NotNil(t, provider.client)
			assert.Equal(t, "text-embedding-ada-002", string(provider.model))
		})
	}
}

// Test OpenAI embedding generation with various inputs
func TestOpenAIEmbeddingGeneration(t *testing.T) {
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

// Test OpenAI error scenarios
func TestOpenAIErrorScenarios(t *testing.T) {
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
		{
			name:          "empty text",
			apiKey:        "dummy-key",
			input:         "",
			errorContains: "empty text",
		},
		{
			name:          "whitespace only text",
			apiKey:        "dummy-key",
			input:         "   \t\n  ",
			errorContains: "empty text",
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
			if tc.errorContains != "" {
				assert.Contains(t, err.Error(), tc.errorContains)
			}
		})
	}
}

// Test context cancellation
func TestOpenAIContextCancellation(t *testing.T) {
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
func TestOpenAIConcurrentRequests(t *testing.T) {
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
func TestOpenAIEmbeddingConsistency(t *testing.T) {
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

// Test extremely long text handling
func TestOpenAIExtremelyLongText(t *testing.T) {
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
func TestOpenAIUTF8EdgeCases(t *testing.T) {
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

// Helper functions
func isNaN(f float32) bool {
	return f != f
}

func isInf(f float32) bool {
	return f > 1e30 || f < -1e30
}
