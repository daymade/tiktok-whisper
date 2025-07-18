package provider

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test basic interface compliance for all providers
func TestEmbeddingProviderInterfaceCompliance(t *testing.T) {
	// Create instances of all providers
	providers := []struct {
		name     string
		provider EmbeddingProvider
	}{
		{"MockProvider", NewMockProvider(768)},
		{"OpenAIProvider", NewOpenAIProvider("test-key")},
		{"GeminiProvider", NewGeminiProvider("test-key")},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			// Verify the provider implements the interface
			var _ EmbeddingProvider = p.provider
			
			// Use reflection to verify interface methods
			providerType := reflect.TypeOf(p.provider)
			interfaceType := reflect.TypeOf((*EmbeddingProvider)(nil)).Elem()
			
			assert.True(t, providerType.Implements(interfaceType),
				"%s should implement EmbeddingProvider interface", p.name)
		})
	}
}

// Test that all providers have required methods with correct signatures
func TestEmbeddingProviderMethodSignatures(t *testing.T) {
	providers := []EmbeddingProvider{
		NewMockProvider(768),
		NewOpenAIProvider("test-key"),
		NewGeminiProvider("test-key"),
	}

	for _, provider := range providers {
		providerType := reflect.TypeOf(provider)
		providerName := providerType.String()
		
		t.Run(providerName, func(t *testing.T) {
			// Check GenerateEmbedding method
			method, ok := providerType.MethodByName("GenerateEmbedding")
			require.True(t, ok, "%s should have GenerateEmbedding method", providerName)
			
			// Verify method signature
			methodType := method.Type
			assert.Equal(t, 3, methodType.NumIn(), "GenerateEmbedding should have 3 inputs (receiver, context, string)")
			assert.Equal(t, 2, methodType.NumOut(), "GenerateEmbedding should have 2 outputs ([]float32, error)")
			
			// Check GetProviderInfo method
			method, ok = providerType.MethodByName("GetProviderInfo")
			require.True(t, ok, "%s should have GetProviderInfo method", providerName)
			
			// Verify method signature
			methodType = method.Type
			assert.Equal(t, 1, methodType.NumIn(), "GetProviderInfo should have 1 input (receiver)")
			assert.Equal(t, 1, methodType.NumOut(), "GetProviderInfo should have 1 output (ProviderInfo)")
		})
	}
}

// Test that provider metadata is valid for all providers
func TestEmbeddingProviderMetadata(t *testing.T) {
	testCases := []struct {
		name             string
		provider         EmbeddingProvider
		expectedName     string
		expectedModel    string
		expectedDimension int
	}{
		{
			name:             "MockProvider",
			provider:         NewMockProvider(768),
			expectedName:     "mock",
			expectedModel:    "mock-model",
			expectedDimension: 768,
		},
		{
			name:             "OpenAIProvider",
			provider:         NewOpenAIProvider("test-key"),
			expectedName:     "openai",
			expectedModel:    "text-embedding-ada-002",
			expectedDimension: 1536,
		},
		{
			name:             "GeminiProvider",
			provider:         NewGeminiProvider("test-key"),
			expectedName:     "gemini",
			expectedModel:    "models/embedding-001",
			expectedDimension: 768,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			info := tc.provider.GetProviderInfo()

			// Assert
			assert.Equal(t, tc.expectedName, info.Name)
			assert.Equal(t, tc.expectedModel, info.Model)
			assert.Equal(t, tc.expectedDimension, info.Dimension)
			
			// Additional validations
			assert.NotEmpty(t, info.Name, "Provider name should not be empty")
			assert.NotEmpty(t, info.Model, "Model name should not be empty")
			assert.Greater(t, info.Dimension, 0, "Dimension should be positive")
		})
	}
}

// Test basic embedding generation for providers that don't require API keys
func TestEmbeddingProviderBasicGeneration(t *testing.T) {
	// Only test providers that work without real API keys
	providers := []struct {
		name     string
		provider EmbeddingProvider
	}{
		{"MockProvider", NewMockProvider(768)},
		{"GeminiProvider", NewGeminiProvider("test-key")}, // Mock implementation
	}

	ctx := context.Background()
	testText := "test text for embedding"

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			// Act
			embedding, err := p.provider.GenerateEmbedding(ctx, testText)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, embedding)
			
			// Verify dimension matches provider info
			info := p.provider.GetProviderInfo()
			assert.Len(t, embedding, info.Dimension)
		})
	}
}

// Test error handling across providers
func TestEmbeddingProviderErrorHandling(t *testing.T) {
	providers := []struct {
		name     string
		provider EmbeddingProvider
	}{
		{"MockProvider", NewMockProvider(768)},
		{"OpenAIProvider", NewOpenAIProvider("test-key")},
		{"GeminiProvider", NewGeminiProvider("test-key")},
	}

	ctx := context.Background()

	for _, p := range providers {
		t.Run(p.name+" - empty text", func(t *testing.T) {
			// Act
			embedding, err := p.provider.GenerateEmbedding(ctx, "")

			// Assert
			assert.Error(t, err)
			assert.Nil(t, embedding)
			assert.Contains(t, err.Error(), "empty text")
		})

		t.Run(p.name+" - whitespace text", func(t *testing.T) {
			// Act
			embedding, err := p.provider.GenerateEmbedding(ctx, "   \t\n  ")

			// Assert
			assert.Error(t, err)
			assert.Nil(t, embedding)
			assert.Contains(t, err.Error(), "empty text")
		})
	}
}

// Test provider info consistency
func TestProviderInfoConsistency(t *testing.T) {
	providers := []EmbeddingProvider{
		NewMockProvider(768),
		NewOpenAIProvider("test-key"),
		NewGeminiProvider("test-key"),
	}

	for _, provider := range providers {
		providerType := reflect.TypeOf(provider).String()
		t.Run(providerType, func(t *testing.T) {
			// Get info multiple times to ensure consistency
			info1 := provider.GetProviderInfo()
			info2 := provider.GetProviderInfo()
			info3 := provider.GetProviderInfo()

			// Assert all calls return the same info
			assert.Equal(t, info1, info2)
			assert.Equal(t, info2, info3)
		})
	}
}

// Test that ProviderInfo struct has expected fields
func TestProviderInfoStructure(t *testing.T) {
	// Create a sample ProviderInfo
	info := ProviderInfo{
		Name:      "test",
		Model:     "test-model",
		Dimension: 512,
	}

	// Use reflection to verify struct fields
	infoType := reflect.TypeOf(info)
	assert.Equal(t, 3, infoType.NumField(), "ProviderInfo should have 3 fields")

	// Verify field names and types
	nameField, ok := infoType.FieldByName("Name")
	assert.True(t, ok, "ProviderInfo should have Name field")
	assert.Equal(t, "string", nameField.Type.String())

	modelField, ok := infoType.FieldByName("Model")
	assert.True(t, ok, "ProviderInfo should have Model field")
	assert.Equal(t, "string", modelField.Type.String())

	dimensionField, ok := infoType.FieldByName("Dimension")
	assert.True(t, ok, "ProviderInfo should have Dimension field")
	assert.Equal(t, "int", dimensionField.Type.String())
}

// Test provider factory pattern (if we add one in the future)
func TestProviderFactory(t *testing.T) {
	// This test documents the expected behavior if we add a factory function
	t.Run("factory pattern documentation", func(t *testing.T) {
		// Example of what a factory might look like:
		// provider, err := NewProvider("openai", "api-key")
		// assert.NoError(t, err)
		// assert.IsType(t, &OpenAIProvider{}, provider)
		
		t.Log("Factory pattern not yet implemented. This test documents expected behavior.")
	})
}

// Benchmark interface method calls
func BenchmarkProviderInfoRetrieval(b *testing.B) {
	providers := []struct {
		name     string
		provider EmbeddingProvider
	}{
		{"MockProvider", NewMockProvider(768)},
		{"OpenAIProvider", NewOpenAIProvider("test-key")},
		{"GeminiProvider", NewGeminiProvider("test-key")},
	}

	for _, p := range providers {
		b.Run(p.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.provider.GetProviderInfo()
			}
		})
	}
}

// Test nil and edge cases
func TestEmbeddingProviderEdgeCases(t *testing.T) {
	t.Run("nil context handling", func(t *testing.T) {
		// Note: passing nil context is generally bad practice,
		// but we test to ensure providers handle it gracefully
		provider := NewMockProvider(128)
		
		// This should not panic
		assert.NotPanics(t, func() {
			_, _ = provider.GenerateEmbedding(nil, "test")
		})
	})
}

// Example test showing how to use the interface
func ExampleEmbeddingProvider() {
	// Create a provider
	provider := NewMockProvider(768)
	
	// Get provider information
	info := provider.GetProviderInfo()
	fmt.Printf("Provider: %s, Model: %s, Dimension: %d\n",
		info.Name, info.Model, info.Dimension)
	
	// Generate an embedding
	ctx := context.Background()
	embedding, err := provider.GenerateEmbedding(ctx, "Hello, world!")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Generated embedding with %d dimensions\n", len(embedding))
	// Output:
	// Provider: mock, Model: mock-model, Dimension: 768
	// Generated embedding with 768 dimensions
}