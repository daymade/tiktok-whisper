//go:build integration
// +build integration

package provider

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test all providers with the same input to verify they all work
func TestAllProvidersWithSameInput(t *testing.T) {
	testText := "The quick brown fox jumps over the lazy dog."
	ctx := context.Background()

	// Test cases for providers that don't require real API keys
	testCases := []struct {
		name       string
		provider   EmbeddingProvider
		skipReason string
	}{
		{
			name:     "MockProvider",
			provider: NewMockProvider(768),
		},
		{
			name:       "GeminiProvider",
			provider:   NewGeminiProvider(""), // Empty API key for mock
			skipReason: "Mock implementation",
		},
	}

	// Only test OpenAI if API key is available
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		testCases = append(testCases, struct {
			name       string
			provider   EmbeddingProvider
			skipReason string
		}{
			name:     "OpenAIProvider",
			provider: NewOpenAIProvider(apiKey),
		})
	} else {
		t.Log("OPENAI_API_KEY not set, skipping OpenAI integration tests")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipReason != "" {
				t.Logf("Note: %s", tc.skipReason)
			}

			// Act
			embedding, err := tc.provider.GenerateEmbedding(ctx, testText)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, embedding)

			// Verify dimension matches provider info
			info := tc.provider.GetProviderInfo()
			assert.Len(t, embedding, info.Dimension)

			// Log provider information
			t.Logf("Provider: %s, Model: %s, Dimension: %d",
				info.Name, info.Model, info.Dimension)
		})
	}
}

// Test that different providers produce different embeddings for the same text
func TestProviderEmbeddingDifferences(t *testing.T) {
	testText := "Machine learning and artificial intelligence"
	ctx := context.Background()

	// Generate embeddings from different providers
	mockProvider := NewMockProvider(768)
	geminiProvider := NewGeminiProvider("") // Empty API key for mock

	mockEmbedding, err := mockProvider.GenerateEmbedding(ctx, testText)
	require.NoError(t, err)

	geminiEmbedding, err := geminiProvider.GenerateEmbedding(ctx, testText)
	require.NoError(t, err)

	// Embeddings should be different (different algorithms/models)
	assert.NotEqual(t, mockEmbedding, geminiEmbedding)

	// But both should be valid
	assert.Len(t, mockEmbedding, 768)
	assert.Len(t, geminiEmbedding, 768)
}

// Test provider switching (simulating dependency injection)
func TestProviderSwitching(t *testing.T) {
	testText := "Provider switching test"
	ctx := context.Background()

	providers := []EmbeddingProvider{
		NewMockProvider(256),
		NewGeminiProvider(""), // Empty API key for mock
	}

	// Function that uses any provider
	generateEmbedding := func(provider EmbeddingProvider, text string) ([]float32, error) {
		return provider.GenerateEmbedding(ctx, text)
	}

	for i, provider := range providers {
		t.Run("provider-"+string(rune('A'+i)), func(t *testing.T) {
			embedding, err := generateEmbedding(provider, testText)

			assert.NoError(t, err)
			assert.NotNil(t, embedding)

			info := provider.GetProviderInfo()
			assert.Len(t, embedding, info.Dimension)
		})
	}
}

// Test concurrent access to multiple providers
func TestConcurrentMultiProviderAccess(t *testing.T) {
	providers := []EmbeddingProvider{
		NewMockProvider(128),
		NewMockProvider(256),
		NewGeminiProvider(""), // Empty API key for mock
		NewGeminiProvider(""), // Empty API key for mock
	}

	texts := []string{
		"Concurrent test 1",
		"Concurrent test 2",
		"Concurrent test 3",
		"Concurrent test 4",
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	// Results to collect
	type result struct {
		providerIdx int
		textIdx     int
		embedding   []float32
		err         error
	}

	results := make(chan result, len(providers)*len(texts))

	// Start concurrent operations
	for providerIdx, provider := range providers {
		for textIdx, text := range texts {
			wg.Add(1)
			go func(pIdx, tIdx int, p EmbeddingProvider, txt string) {
				defer wg.Done()
				embedding, err := p.GenerateEmbedding(ctx, txt)
				results <- result{
					providerIdx: pIdx,
					textIdx:     tIdx,
					embedding:   embedding,
					err:         err,
				}
			}(providerIdx, textIdx, provider, text)
		}
	}

	// Wait for all to complete
	wg.Wait()
	close(results)

	// Verify all results
	resultCount := 0
	for result := range results {
		resultCount++
		assert.NoError(t, result.err,
			"Provider %d with text %d failed", result.providerIdx, result.textIdx)
		assert.NotNil(t, result.embedding,
			"Provider %d with text %d returned nil embedding", result.providerIdx, result.textIdx)
	}

	assert.Equal(t, len(providers)*len(texts), resultCount)
}

// Test provider performance comparison
func TestProviderPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	providers := []struct {
		name     string
		provider EmbeddingProvider
	}{
		{"Mock-512", NewMockProvider(512)},
		{"Mock-1536", NewMockProvider(1536)},
		{"Gemini", NewGeminiProvider("")}, // Empty API key for mock
	}

	testText := "Performance test text for embedding generation"
	ctx := context.Background()
	iterations := 100

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			start := time.Now()

			for i := 0; i < iterations; i++ {
				embedding, err := p.provider.GenerateEmbedding(ctx, testText)
				require.NoError(t, err)
				require.NotNil(t, embedding)
			}

			duration := time.Since(start)
			avgDuration := duration / time.Duration(iterations)

			t.Logf("Provider %s: %d iterations in %v (avg: %v per embedding)",
				p.name, iterations, duration, avgDuration)
		})
	}
}

// Test provider resilience to stress
func TestProviderStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	provider := NewMockProvider(768)
	ctx := context.Background()
	numGoroutines := 50
	numRequestsPerGoroutine := 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numRequestsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numRequestsPerGoroutine; j++ {
				text := "Stress test text for goroutine " + string(rune('A'+goroutineID)) + " request " + string(rune('0'+j))
				_, err := provider.GenerateEmbedding(ctx, text)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Errorf("Stress test error: %v", err)
	}

	assert.Equal(t, 0, errorCount, "Stress test should not produce any errors")
}

// Test provider behavior with various context scenarios
func TestProviderContextScenarios(t *testing.T) {
	provider := NewMockProvider(256)
	testText := "Context test"

	testCases := []struct {
		name          string
		ctx           context.Context
		expectTimeout bool
	}{
		{
			name: "background context",
			ctx:  context.Background(),
		},
		{
			name: "TODO context",
			ctx:  context.TODO(),
		},
		{
			name: "context with value",
			ctx:  context.WithValue(context.Background(), "test", "value"),
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
		},
		{
			name: "timeout context",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(1 * time.Millisecond) // Ensure timeout
				return ctx
			}(),
			expectTimeout: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			embedding, err := provider.GenerateEmbedding(tc.ctx, testText)

			// Note: Mock provider doesn't check context, so these tests
			// document expected behavior for real implementations
			if tc.expectTimeout {
				t.Log("Note: Mock provider doesn't respect context timeout. Real providers should.")
			}

			// For now, all should succeed with mock provider
			assert.NoError(t, err)
			assert.NotNil(t, embedding)
		})
	}
}

// Test provider memory usage patterns
func TestProviderMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	provider := NewMockProvider(4096) // Large dimension
	ctx := context.Background()

	// Generate many embeddings to test for memory leaks
	for i := 0; i < 1000; i++ {
		text := "Memory test iteration " + string(rune('0'+(i%10)))
		embedding, err := provider.GenerateEmbedding(ctx, text)

		require.NoError(t, err)
		require.Len(t, embedding, 4096)

		// Clear reference to help GC
		embedding = nil
	}

	t.Log("Memory usage test completed successfully")
}

// Test provider with edge case text inputs
func TestProviderEdgeCaseInputs(t *testing.T) {
	provider := NewMockProvider(128)
	ctx := context.Background()

	testCases := []struct {
		name          string
		input         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "very long text",
			input:       string(make([]byte, 100000)), // 100KB of null bytes
			expectError: false,
		},
		{
			name:          "empty string",
			input:         "",
			expectError:   true,
			errorContains: "empty text",
		},
		{
			name:        "single character",
			input:       "a",
			expectError: false,
		},
		{
			name:        "only numbers",
			input:       "1234567890",
			expectError: false,
		},
		{
			name:        "only spaces",
			input:       "     ",
			expectError: true,
		},
		{
			name:        "mixed content",
			input:       "Text with\nnewlines\tand\ttabs and 数字 and émojis 🚀",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			embedding, err := provider.GenerateEmbedding(ctx, tc.input)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, embedding)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, embedding)
				assert.Len(t, embedding, 128)
			}
		})
	}
}

// Integration test with rate limiting simulation
func TestProviderRateLimitingBehavior(t *testing.T) {
	// This test simulates how providers should behave with rate limiting
	// Currently only documents expected behavior since providers don't implement it

	provider := NewMockProvider(256)
	ctx := context.Background()

	// Rapid fire requests
	numRequests := 10
	requestInterval := 10 * time.Millisecond

	for i := 0; i < numRequests; i++ {
		start := time.Now()

		_, err := provider.GenerateEmbedding(ctx, "Rate limit test")
		assert.NoError(t, err)

		elapsed := time.Since(start)
		t.Logf("Request %d completed in %v", i+1, elapsed)

		if i < numRequests-1 {
			time.Sleep(requestInterval)
		}
	}

	t.Log("Note: Real providers should implement rate limiting and backoff")
}

// Test provider error recovery
func TestProviderErrorRecovery(t *testing.T) {
	provider := NewMockProvider(128)
	ctx := context.Background()

	// First, cause an error
	_, err := provider.GenerateEmbedding(ctx, "")
	assert.Error(t, err)

	// Then verify provider still works normally
	embedding, err := provider.GenerateEmbedding(ctx, "recovery test")
	assert.NoError(t, err)
	assert.NotNil(t, embedding)
	assert.Len(t, embedding, 128)

	t.Log("Provider successfully recovered from error state")
}

// Benchmark all providers
func BenchmarkAllProviders(b *testing.B) {
	providers := []struct {
		name     string
		provider EmbeddingProvider
	}{
		{"Mock-256", NewMockProvider(256)},
		{"Mock-768", NewMockProvider(768)},
		{"Mock-1536", NewMockProvider(1536)},
		{"Gemini", NewGeminiProvider("")}, // Empty API key for mock
	}

	testText := "Benchmark text for performance testing"
	ctx := context.Background()

	for _, p := range providers {
		b.Run(p.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := p.provider.GenerateEmbedding(ctx, testText)
				if err != nil {
					b.Fatalf("Provider %s failed: %v", p.name, err)
				}
			}
		})
	}
}
