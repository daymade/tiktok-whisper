package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test OpenAI provider interface compliance
func TestOpenAIProviderInterface_Unit(t *testing.T) {
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
func TestOpenAIProviderConstructor_Unit(t *testing.T) {
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

// Test OpenAI error scenarios
func TestOpenAIErrorScenarios_Unit(t *testing.T) {
	testCases := []struct {
		name          string
		apiKey        string
		input         string
		errorContains string
	}{
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

// Test text validation logic
func TestOpenAITextValidation_Unit(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid text",
			input:       "This is valid text",
			shouldError: false,
		},
		{
			name:        "empty string",
			input:       "",
			shouldError: true,
			errorMsg:    "empty text",
		},
		{
			name:        "only spaces",
			input:       "     ",
			shouldError: true,
			errorMsg:    "empty text",
		},
		{
			name:        "only tabs",
			input:       "\t\t\t",
			shouldError: true,
			errorMsg:    "empty text",
		},
		{
			name:        "only newlines",
			input:       "\n\n\n",
			shouldError: true,
			errorMsg:    "empty text",
		},
		{
			name:        "mixed whitespace",
			input:       " \t\n\r ",
			shouldError: true,
			errorMsg:    "empty text",
		},
		{
			name:        "text with leading/trailing whitespace",
			input:       "  valid text  ",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the validation logic that should be in the provider
			trimmed := strings.TrimSpace(tc.input)
			isEmpty := len(trimmed) == 0

			if tc.shouldError {
				assert.True(t, isEmpty, "Expected text to be considered empty")
			} else {
				assert.False(t, isEmpty, "Expected text to be considered valid")
			}
		})
	}
}

// Test provider info consistency
func TestOpenAIProviderInfo_Unit(t *testing.T) {
	// Create multiple instances and verify they report consistent info
	providers := []*OpenAIProvider{
		NewOpenAIProvider("key1"),
		NewOpenAIProvider("key2"),
		NewOpenAIProvider(""),
	}

	for i, provider := range providers {
		info := provider.GetProviderInfo()
		assert.Equal(t, "openai", info.Name, "Provider %d name mismatch", i)
		assert.Equal(t, "text-embedding-ada-002", info.Model, "Provider %d model mismatch", i)
		assert.Equal(t, 1536, info.Dimension, "Provider %d dimension mismatch", i)
	}
}

// Test context handling without making API calls
func TestOpenAIContextHandling_Unit(t *testing.T) {
	provider := NewOpenAIProvider("test-key")

	testCases := []struct {
		name        string
		ctx         context.Context
		description string
	}{
		{
			name:        "nil context",
			ctx:         nil,
			description: "should handle nil context gracefully",
		},
		{
			name:        "background context",
			ctx:         context.Background(),
			description: "should accept background context",
		},
		{
			name:        "TODO context",
			ctx:         context.TODO(),
			description: "should accept TODO context",
		},
		{
			name:        "cancelled context",
			ctx:         func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			description: "should handle cancelled context",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Just verify the method accepts the context without panic
			// Actual API call would fail, but we're just testing the method signature
			assert.NotPanics(t, func() {
				// We expect an error due to empty text, but no panic
				_, _ = provider.GenerateEmbedding(tc.ctx, "")
			}, tc.description)
		})
	}
}