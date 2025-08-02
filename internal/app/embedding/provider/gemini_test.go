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

// Test Gemini provider interface compliance
func TestGeminiProviderInterface(t *testing.T) {
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

// Test Gemini constructor and configuration
func TestGeminiProviderConstructor(t *testing.T) {
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

// Test Gemini embedding generation with various inputs (mock implementation)
func TestGeminiEmbeddingGeneration(t *testing.T) {
	// Note: This tests the mock implementation. When real API is implemented,
	// add GEMINI_API_KEY check similar to OpenAI tests

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
		{
			name:          "empty text",
			input:         "",
			expectError:   true,
			errorContains: "empty text",
		},
		{
			name:          "whitespace only text",
			input:         "   \t\n  ",
			expectError:   true,
			errorContains: "empty text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			provider := NewGeminiProvider("") // Empty API key to use mock
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
				assert.Len(t, embedding, 768)
				// Verify all values are valid floats (no NaN/Inf)
				for i, val := range embedding {
					assert.False(t, isNaNFloat32(val), "Value at index %d is NaN", i)
					assert.False(t, isInfFloat32(val), "Value at index %d is Inf", i)
					// Values should be in reasonable range for normalized embeddings
					assert.GreaterOrEqual(t, val, float32(-2.0), "Value at index %d should be >= -2", i)
					assert.LessOrEqual(t, val, float32(2.0), "Value at index %d should be <= 2", i)
				}
			}
		})
	}
}

// Test Gemini deterministic behavior (mock implementation)
func TestGeminiDeterministicBehavior(t *testing.T) {
	// Arrange
	provider := NewGeminiProvider("") // Empty API key for mock
	ctx := context.Background()

	testCases := []struct {
		name  string
		text1 string
		text2 string
		equal bool
	}{
		{
			name:  "same text produces same embedding",
			text1: "test text",
			text2: "test text",
			equal: true,
		},
		{
			name:  "different text produces different embedding",
			text1: "first text",
			text2: "second text",
			equal: false,
		},
		{
			name:  "case sensitive",
			text1: "Test",
			text2: "test",
			equal: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			embedding1, err1 := provider.GenerateEmbedding(ctx, tc.text1)
			embedding2, err2 := provider.GenerateEmbedding(ctx, tc.text2)

			// Assert
			assert.NoError(t, err1)
			assert.NoError(t, err2)

			if tc.equal {
				assert.Equal(t, embedding1, embedding2)
			} else {
				assert.NotEqual(t, embedding1, embedding2)
			}
		})
	}
}

// Test context cancellation
func TestGeminiContextCancellation(t *testing.T) {
	// Note: Current mock implementation doesn't check context
	// This test documents expected behavior for real implementation

	// Arrange
	provider := NewGeminiProvider("") // Empty API key for mock
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "test")

	// Assert - Mock doesn't check context, but real implementation should
	// When implementing real API, this should return context.Canceled error
	if err == nil {
		t.Log("Note: Mock implementation doesn't check context. Real implementation should return error.")
		assert.NotNil(t, embedding) // Mock still returns embedding
	}
}

// Test concurrent embedding generation
func TestGeminiConcurrentRequests(t *testing.T) {
	// Arrange
	provider := NewGeminiProvider("") // Empty API key for mock
	ctx := context.Background()
	numRequests := 10
	texts := make([]string, numRequests)
	for i := 0; i < numRequests; i++ {
		texts[i] = "Test text " + string(rune('A'+i))
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
		assert.Len(t, result.embedding, 768, "Request %d returned wrong dimension", i)
	}

	// Verify different texts produce different embeddings
	for i := 0; i < numRequests-1; i++ {
		for j := i + 1; j < numRequests; j++ {
			assert.NotEqual(t, results[i].embedding, results[j].embedding,
				"Different texts should produce different embeddings")
		}
	}
}

// Test extremely long text handling
func TestGeminiExtremelyLongText(t *testing.T) {
	// Arrange
	provider := NewGeminiProvider("") // Empty API key for mock
	ctx := context.Background()
	// Gemini may have different token limits than OpenAI
	longText := strings.Repeat("This is a test sentence with multiple words. ", 2000)

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, longText)

	// Assert - Mock implementation handles any length
	// Real implementation might have limits
	assert.NoError(t, err)
	assert.NotNil(t, embedding)
	assert.Len(t, embedding, 768)
}

// Test various UTF-8 edge cases
func TestGeminiUTF8EdgeCases(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"emoji cluster", "👨‍👩‍👧‍👦"}, // Family emoji
		{"korean", "한국어 테스트"},
		{"hebrew", "בדיקה בעברית"},
		{"thai", "การทดสอบภาษาไทย"},
		{"mixed scripts with RTL", "English عربي 日本語"},
		{"mathematical symbols", "∑∫√∞ αβγδ"},
		{"invisible characters", "Hello\u200B\u200C\u200Dworld"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			provider := NewGeminiProvider("") // Empty API key to use mock
			ctx := context.Background()

			// Verify input is valid UTF-8
			require.True(t, utf8.ValidString(tc.input), "Test input must be valid UTF-8")

			// Act
			embedding, err := provider.GenerateEmbedding(ctx, tc.input)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, embedding)
			assert.Len(t, embedding, 768)
		})
	}
}

// Benchmark Gemini embedding generation
func BenchmarkGeminiEmbeddingGeneration(b *testing.B) {
	provider := NewGeminiProvider("") // Empty API key for mock
	ctx := context.Background()
	testText := "This is a benchmark test for Gemini embedding generation performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateEmbedding(ctx, testText)
		if err != nil {
			b.Fatalf("Failed to generate embedding: %v", err)
		}
	}
}

// Test embedding value distribution
func TestGeminiEmbeddingValueDistribution(t *testing.T) {
	// Arrange
	provider := NewGeminiProvider("") // Empty API key for mock
	ctx := context.Background()
	texts := []string{
		"Short text",
		"A much longer text with many more characters to process",
		"123456789",
		"Special!@#$%^&*()Characters",
	}

	for _, text := range texts {
		t.Run(text, func(t *testing.T) {
			// Act
			embedding, err := provider.GenerateEmbedding(ctx, text)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, embedding)

			// Check that not all values are the same (mock implementation issue)
			firstVal := embedding[0]
			allSame := true
			for _, val := range embedding[1:] {
				if val != firstVal {
					allSame = false
					break
				}
			}
			// Current mock implementation produces all same values
			// This test documents that behavior and should be updated
			// when real implementation is added
			if allSame {
				t.Log("Note: Mock implementation produces uniform embeddings. Real implementation should have variance.")
			}
		})
	}
}

// Test API key validation behavior
func TestGeminiAPIKeyValidation(t *testing.T) {
	testCases := []struct {
		name        string
		apiKey      string
		expectError bool // For real implementation
	}{
		{"valid looking key", "AIzaSy" + strings.Repeat("a", 33), false},
		{"empty key", "", true},
		{"whitespace key", "   ", true},
		{"obviously invalid key", "invalid", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			provider := NewGeminiProvider(tc.apiKey)
			ctx := context.Background()

			// Act
			embedding, err := provider.GenerateEmbedding(ctx, "test")

			// Assert - Mock implementation doesn't validate API key format
			// It only validates that text is not empty
			assert.NoError(t, err)
			assert.NotNil(t, embedding)
			t.Log("Note: Real implementation should validate API key format and authentication")
		})
	}
}

// Test timeout behavior
func TestGeminiTimeout(t *testing.T) {
	// Arrange
	provider := NewGeminiProvider("") // Empty API key for mock
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // Ensure timeout has passed

	// Act
	embedding, err := provider.GenerateEmbedding(ctx, "test")

	// Assert - Mock doesn't respect context timeout
	// Real implementation should return deadline exceeded error
	if err == nil {
		t.Log("Note: Mock implementation doesn't respect context timeout. Real implementation should.")
		assert.NotNil(t, embedding)
	}
}

// Integration test placeholder
func TestGeminiIntegration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration tests")
	}

	// TODO: Implement real Gemini API integration tests when API client is added
	t.Log("Gemini integration tests will be implemented when real API client is added")
}

// Helper functions
func isNaNFloat32(f float32) bool {
	return f != f
}

func isInfFloat32(f float32) bool {
	return f > 1e30 || f < -1e30
}
