package provider

import (
	"context"
	"encoding/hex"
	"math"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test MockProvider interface compliance
func TestMockProviderInterface(t *testing.T) {
	// Arrange
	var provider EmbeddingProvider
	provider = NewMockProvider(768)

	// Act
	info := provider.GetProviderInfo()

	// Assert
	assert.Equal(t, "mock", info.Name)
	assert.Equal(t, "mock-model", info.Model)
	assert.Equal(t, 768, info.Dimension)
	
	// Verify interface methods are implemented
	_, ok := provider.(EmbeddingProvider)
	assert.True(t, ok, "MockProvider should implement EmbeddingProvider interface")
}

// Test MockProvider constructor with various dimensions
func TestMockProviderConstructor(t *testing.T) {
	testCases := []struct {
		name      string
		dimension int
	}{
		{"zero dimension", 0},
		{"negative dimension", -1},
		{"small dimension", 1},
		{"standard dimension", 768},
		{"large dimension", 4096},
		{"very large dimension", 10000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			provider := NewMockProvider(tc.dimension)

			// Assert
			assert.NotNil(t, provider)
			assert.Equal(t, tc.dimension, provider.dimension)
			
			// Verify info reports correct dimension
			info := provider.GetProviderInfo()
			assert.Equal(t, tc.dimension, info.Dimension)
		})
	}
}

// Test deterministic behavior of MockProvider
func TestMockProviderDeterministic(t *testing.T) {
	// Arrange
	provider := NewMockProvider(768)
	ctx := context.Background()

	testCases := []struct {
		name   string
		text1  string
		text2  string
		equal  bool
	}{
		{
			name:  "identical text produces identical embeddings",
			text1: "hello world",
			text2: "hello world",
			equal: true,
		},
		{
			name:  "different text produces different embeddings",
			text1: "hello world",
			text2: "goodbye world",
			equal: false,
		},
		{
			name:  "case sensitivity",
			text1: "Hello",
			text2: "hello",
			equal: false,
		},
		{
			name:  "whitespace differences",
			text1: "hello world",
			text2: "hello  world",
			equal: false,
		},
		{
			name:  "special characters",
			text1: "test!",
			text2: "test?",
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
		{"Single dimension", 1},
		{"Odd dimension", 777},
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
	provider := NewMockProvider(1000)
	ctx := context.Background()
	
	// Test various inputs to ensure range is always [-1, 1]
	testInputs := []string{
		"a",
		"test normalization",
		"Very long text with many characters to ensure we test different hash values",
		strings.Repeat("x", 1000),
		"\x00\x01\x02\x03\x04\x05", // Binary-like data
		"￿￾�",              // High unicode values
	}

	for _, input := range testInputs {
		t.Run("input: "+input[:min(20, len(input))]+"...", func(t *testing.T) {
			// Act
			embedding, err := provider.GenerateEmbedding(ctx, input)

			// Assert
			assert.NoError(t, err)
			
			var minVal, maxVal float32 = 1.0, -1.0
			for i, value := range embedding {
				assert.GreaterOrEqual(t, value, float32(-1.0), "Value at index %d should be >= -1", i)
				assert.LessOrEqual(t, value, float32(1.0), "Value at index %d should be <= 1", i)
				
				if value < minVal {
					minVal = value
				}
				if value > maxVal {
					maxVal = value
				}
			}
			
			// Ensure we have some variance
			assert.NotEqual(t, minVal, maxVal, "Embedding should have variance")
		})
	}
}

// Test error handling for various inputs
func TestMockProviderErrorHandling(t *testing.T) {
	provider := NewMockProvider(768)
	ctx := context.Background()

	testCases := []struct {
		name          string
		input         string
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty text",
			input:         "",
			expectError:   true,
			errorContains: "empty text",
		},
		{
			name:          "whitespace only",
			input:         "   \t\n  ",
			expectError:   true,
			errorContains: "empty text",
		},
		{
			name:        "single character",
			input:       "a",
			expectError: false,
		},
		{
			name:        "null character",
			input:       "test\x00null",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			}
		})
	}
}

// Test concurrent embedding generation
func TestMockProviderConcurrentRequests(t *testing.T) {
	// Arrange
	provider := NewMockProvider(512)
	ctx := context.Background()
	numRequests := 100

	// Act
	var wg sync.WaitGroup
	results := make([]struct {
		text      string
		embedding []float32
		err       error
	}, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			text := "test text " + hex.EncodeToString([]byte{byte(idx)})
			embedding, err := provider.GenerateEmbedding(ctx, text)
			results[idx].text = text
			results[idx].embedding = embedding
			results[idx].err = err
		}(i)
	}

	wg.Wait()

	// Assert
	for i, result := range results {
		assert.NoError(t, result.err, "Request %d failed", i)
		assert.NotNil(t, result.embedding, "Request %d returned nil embedding", i)
		assert.Len(t, result.embedding, 512, "Request %d returned wrong dimension", i)
	}

	// Verify deterministic behavior in concurrent setting
	// Generate embedding for first text again
	verifyEmbedding, err := provider.GenerateEmbedding(ctx, results[0].text)
	assert.NoError(t, err)
	assert.Equal(t, results[0].embedding, verifyEmbedding, "Concurrent execution should not affect deterministic behavior")
}

// Test hash-based implementation details
func TestMockProviderHashBasedImplementation(t *testing.T) {
	// This test verifies the SHA256-based implementation
	provider := NewMockProvider(32) // Use 32 to match SHA256 output size
	ctx := context.Background()

	testCases := []struct {
		name  string
		text1 string
		text2 string
	}{
		{
			name:  "similar texts with one character difference",
			text1: "hello world",
			text2: "hello World",
		},
		{
			name:  "anagrams",
			text1: "listen",
			text2: "silent",
		},
		{
			name:  "reversed text",
			text1: "abcd",
			text2: "dcba",
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
			
			// Even small differences should produce very different embeddings
			// due to SHA256's avalanche effect
			assert.NotEqual(t, embedding1, embedding2)
			
			// Calculate similarity (should be low due to hash properties)
			similarity := cosineSimilarity(embedding1, embedding2)
			t.Logf("Similarity between '%s' and '%s': %.4f", tc.text1, tc.text2, similarity)
		})
	}
}

// Test various UTF-8 inputs
func TestMockProviderUTF8Handling(t *testing.T) {
	provider := NewMockProvider(256)
	ctx := context.Background()

	testCases := []struct {
		name  string
		input string
	}{
		{"ASCII", "Hello, World!"},
		{"Latin-1 Supplement", "àéîöü"},
		{"Greek", "Αθήνα"},
		{"Cyrillic", "Москва"},
		{"CJK", "中文日本語한국어"},
		{"Emoji", "😀🚀🌟💻"},
		{"Mixed", "Hello 世界 🌍!"},
		{"RTL", "شلوم عليكم"},
		{"Combining marks", "n\u0303 e\u0301 a\u0300"},
		{"Zero-width joiners", "👨\u200d💻"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify input is valid UTF-8
			require.True(t, utf8.ValidString(tc.input), "Test input must be valid UTF-8")

			// Act
			embedding, err := provider.GenerateEmbedding(ctx, tc.input)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, embedding)
			assert.Len(t, embedding, 256)
		})
	}
}

// Test edge cases with dimensions
func TestMockProviderDimensionEdgeCases(t *testing.T) {
	ctx := context.Background()
	testText := "test"

	testCases := []struct {
		name      string
		dimension int
	}{
		{"dimension larger than hash size", 64}, // SHA256 produces 32 bytes
		{"dimension equal to hash size", 32},
		{"dimension smaller than hash size", 16},
		{"very small dimension", 1},
		{"prime number dimension", 97},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			provider := NewMockProvider(tc.dimension)

			// Act
			embedding, err := provider.GenerateEmbedding(ctx, testText)

			// Assert
			assert.NoError(t, err)
			assert.Len(t, embedding, tc.dimension)
			
			// Verify values are properly distributed
			valueMap := make(map[float32]int)
			for _, val := range embedding {
				valueMap[val]++
			}
			
			// When dimension > hash size, we expect repeated values
			if tc.dimension > 32 {
				assert.Less(t, len(valueMap), tc.dimension, "Should have repeated values when dimension > hash size")
			}
		})
	}
}

// Benchmark mock provider
func BenchmarkMockProviderEmbeddingGeneration(b *testing.B) {
	provider := NewMockProvider(768)
	ctx := context.Background()
	testText := "This is a benchmark test for mock embedding generation performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateEmbedding(ctx, testText)
		if err != nil {
			b.Fatalf("Failed to generate embedding: %v", err)
		}
	}
}

// Benchmark different dimensions
func BenchmarkMockProviderDimensions(b *testing.B) {
	dimensions := []int{128, 256, 512, 768, 1536, 4096}
	ctx := context.Background()
	testText := "Benchmark text"

	for _, dim := range dimensions {
		b.Run("dimension-"+string(rune(dim)), func(b *testing.B) {
			provider := NewMockProvider(dim)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := provider.GenerateEmbedding(ctx, testText)
				if err != nil {
					b.Fatalf("Failed to generate embedding: %v", err)
				}
			}
		})
	}
}

// Test context handling (even though mock doesn't use it)
func TestMockProviderContextHandling(t *testing.T) {
	provider := NewMockProvider(128)

	testCases := []struct {
		name string
		ctx  context.Context
	}{
		{"with background context", context.Background()},
		{"with TODO context", context.TODO()},
		{"with value context", context.WithValue(context.Background(), "key", "value")},
		{"with cancelled context", func() context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			embedding, err := provider.GenerateEmbedding(tc.ctx, "test")

			// Assert - Mock doesn't check context, so all should succeed
			assert.NoError(t, err)
			assert.NotNil(t, embedding)
		})
	}
}

// Test statistical properties of generated embeddings
func TestMockProviderStatisticalProperties(t *testing.T) {
	provider := NewMockProvider(1000)
	ctx := context.Background()

	// Generate embeddings for different texts
	texts := []string{
		"short",
		"medium length text with more words",
		"very long text that contains many words and should produce a different distribution of values in the embedding vector",
	}

	for _, text := range texts {
		t.Run("text: "+text[:min(20, len(text))]+"...", func(t *testing.T) {
			// Act
			embedding, err := provider.GenerateEmbedding(ctx, text)
			require.NoError(t, err)

			// Calculate statistics
			var sum, sumSquared float64
			var minVal, maxVal float32 = 1.0, -1.0

			for _, val := range embedding {
				sum += float64(val)
				sumSquared += float64(val * val)
				if val < minVal {
					minVal = val
				}
				if val > maxVal {
					maxVal = val
				}
			}

			mean := sum / float64(len(embedding))
			variance := (sumSquared / float64(len(embedding))) - (mean * mean)
			stdDev := math.Sqrt(variance)

			// Assert statistical properties
			t.Logf("Stats for '%s': mean=%.4f, stdDev=%.4f, min=%.4f, max=%.4f",
				text[:min(20, len(text))], mean, stdDev, minVal, maxVal)

			// Mean should be close to 0 (since range is -1 to 1)
			assert.InDelta(t, 0.0, mean, 0.5, "Mean should be roughly centered")
			
			// Should have reasonable variance (not all same value)
			assert.Greater(t, stdDev, 0.1, "Should have reasonable standard deviation")
		})
	}
}

// Helper function to calculate cosine similarity
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}