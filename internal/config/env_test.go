package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tiktok-whisper/internal/app/util/files"
)

func TestGetAPIKeys(t *testing.T) {
	// Save original environment
	originalOpenAI := os.Getenv("OPENAI_API_KEY")
	originalGemini := os.Getenv("GEMINI_API_KEY")
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalOpenAI)
		os.Setenv("GEMINI_API_KEY", originalGemini)
	}()

	testCases := []struct {
		name          string
		openaiKey     string
		geminiKey     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid OpenAI key",
			openaiKey:   "sk-1234567890abcdef1234567890abcdef",
			geminiKey:   "",
			expectError: false,
		},
		{
			name:        "valid Gemini key",
			openaiKey:   "",
			geminiKey:   "AIzaTest-1234567890abcdef1234567890",
			expectError: false,
		},
		{
			name:        "both valid keys",
			openaiKey:   "sk-1234567890abcdef1234567890abcdef",
			geminiKey:   "AIzaTest-1234567890abcdef1234567890",
			expectError: false,
		},
		{
			name:          "invalid OpenAI key format",
			openaiKey:     "invalid-key",
			geminiKey:     "",
			expectError:   true,
			errorContains: "invalid OPENAI_API_KEY format",
		},
		{
			name:          "OpenAI key too short",
			openaiKey:     "sk-short",
			geminiKey:     "",
			expectError:   true,
			errorContains: "too short",
		},
		{
			name:          "invalid Gemini key format",
			openaiKey:     "",
			geminiKey:     "invalid-key",
			expectError:   true,
			errorContains: "invalid GEMINI_API_KEY format",
		},
		{
			name:          "Gemini key too short",
			openaiKey:     "",
			geminiKey:     "AIza-short",
			expectError:   true,
			errorContains: "too short",
		},
		{
			name:        "empty keys are allowed",
			openaiKey:   "",
			geminiKey:   "",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("OPENAI_API_KEY", tc.openaiKey)
			os.Setenv("GEMINI_API_KEY", tc.geminiKey)

			// Test GetAPIKeys
			apiKeys, err := GetAPIKeys()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, apiKeys)
				assert.Equal(t, tc.openaiKey, apiKeys.OpenAI)
				assert.Equal(t, tc.geminiKey, apiKeys.Gemini)
			}
		})
	}
}

func TestValidateAPIKeys(t *testing.T) {
	testCases := []struct {
		name          string
		apiKeys       *APIKeys
		expectError   bool
		errorContains string
	}{
		{
			name: "OpenAI key only",
			apiKeys: &APIKeys{
				OpenAI: "sk-1234567890abcdef1234567890abcdef",
				Gemini: "",
			},
			expectError: false,
		},
		{
			name: "Gemini key only",
			apiKeys: &APIKeys{
				OpenAI: "",
				Gemini: "AIzaTest-1234567890abcdef1234567890",
			},
			expectError: false,
		},
		{
			name: "both keys",
			apiKeys: &APIKeys{
				OpenAI: "sk-1234567890abcdef1234567890abcdef",
				Gemini: "AIzaTest-1234567890abcdef1234567890",
			},
			expectError: false,
		},
		{
			name: "no keys - should not error (just info message)",
			apiKeys: &APIKeys{
				OpenAI: "",
				Gemini: "",
			},
			expectError: false, // Changed: ValidateAPIKeys no longer fails
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAPIKeys(tc.apiKeys)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequireAPIKeys(t *testing.T) {
	testCases := []struct {
		name          string
		apiKeys       *APIKeys
		expectError   bool
		errorContains string
	}{
		{
			name: "OpenAI key only",
			apiKeys: &APIKeys{
				OpenAI: "sk-1234567890abcdef1234567890abcdef",
				Gemini: "",
			},
			expectError: false,
		},
		{
			name: "Gemini key only",
			apiKeys: &APIKeys{
				OpenAI: "",
				Gemini: "AIzaTest-1234567890abcdef1234567890",
			},
			expectError: false,
		},
		{
			name: "both keys",
			apiKeys: &APIKeys{
				OpenAI: "sk-1234567890abcdef1234567890abcdef",
				Gemini: "AIzaTest-1234567890abcdef1234567890",
			},
			expectError: false,
		},
		{
			name: "no keys - should fail for embedding operations",
			apiKeys: &APIKeys{
				OpenAI: "",
				Gemini: "",
			},
			expectError:   true,
			errorContains: "embedding operations require at least one API key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireAPIKeys(tc.apiKeys)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetProjectRoot(t *testing.T) {
	root, err := files.GetProjectRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)

	// Verify go.mod exists in the found root
	_, err = os.Stat(root + "/go.mod")
	assert.NoError(t, err, "go.mod should exist in project root")
}
