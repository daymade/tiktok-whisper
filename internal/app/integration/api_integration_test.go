//go:build integration
// +build integration

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"tiktok-whisper/internal/app/api/openai/whisper"
	"tiktok-whisper/internal/app/testutil"
)

// TestOpenAIAPIIntegration tests OpenAI API integration with real and mocked scenarios
func TestOpenAIAPIIntegration(t *testing.T) {
	// Check if we should run real API tests
	apiKey := os.Getenv("OPENAI_API_KEY")
	runRealAPITests := apiKey != "" && !testing.Short()

	t.Run("MockedAPITests", func(t *testing.T) {
		testOpenAIMockedAPI(t)
	})

	if runRealAPITests {
		t.Run("RealAPITests", func(t *testing.T) {
			testOpenAIRealAPI(t, apiKey)
		})
	} else {
		t.Skip("Skipping real OpenAI API tests (no API key or running in short mode)")
	}
}

// testOpenAIMockedAPI tests OpenAI API integration with mocked responses
func testOpenAIMockedAPI(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
		expectedResult string
	}{
		{
			name:           "SuccessfulTranscription",
			responseStatus: http.StatusOK,
			responseBody:   `{"text": "This is a successful mock transcription."}`,
			expectError:    false,
			expectedResult: "This is a successful mock transcription.",
		},
		{
			name:           "InvalidAPIKey",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`,
			expectError:    true,
			errorContains:  "invalid",
		},
		{
			name:           "RateLimitExceeded",
			responseStatus: http.StatusTooManyRequests,
			responseBody:   `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			expectError:    true,
			errorContains:  "rate limit",
		},
		{
			name:           "InternalServerError",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"error": {"message": "Internal server error", "type": "server_error"}}`,
			expectError:    true,
			errorContains:  "server error",
		},
		{
			name:           "InvalidFileFormat",
			responseStatus: http.StatusBadRequest,
			responseBody:   `{"error": {"message": "Invalid file format", "type": "invalid_request_error"}}`,
			expectError:    true,
			errorContains:  "invalid",
		},
		{
			name:           "FileTooLarge",
			responseStatus: http.StatusRequestEntityTooLarge,
			responseBody:   `{"error": {"message": "File too large", "type": "invalid_request_error"}}`,
			expectError:    true,
			errorContains:  "too large",
		},
		{
			name:           "MalformedJSONResponse",
			responseStatus: http.StatusOK,
			responseBody:   `{"text": "Missing closing quote and brace"`,
			expectError:    true,
			errorContains:  "json",
		},
		{
			name:           "EmptyResponse",
			responseStatus: http.StatusOK,
			responseBody:   ``,
			expectError:    true,
		},
		{
			name:           "UnexpectedResponseFormat",
			responseStatus: http.StatusOK,
			responseBody:   `{"unexpected_field": "value", "missing_text": true}`,
			expectError:    false,
			expectedResult: "", // Should handle missing "text" field gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := openai.DefaultConfig("test-api-key")
			config.BaseURL = server.URL
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "api_test.wav")
			defer testutil.CleanupFile(t, testFile)

			result, err := transcriber.Transcript(testFile)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
				if tt.expectedResult != "" {
					assert.Equal(t, tt.expectedResult, result)
				}
			}
		})
	}
}

// testOpenAIRealAPI tests against the real OpenAI API (when API key is available)
func testOpenAIRealAPI(t *testing.T, apiKey string) {
	client := openai.NewClient(apiKey)
	transcriber := whisper.NewRemoteTranscriber(client)

	t.Run("RealAPISuccessfulTranscription", func(t *testing.T) {
		testFile := testutil.CreateTestAudioFile(t, "real_api_test.wav")
		defer testutil.CleanupFile(t, testFile)

		result, err := transcriber.Transcript(testFile)

		// Real API should work with our test file
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
		t.Logf("Real API transcription result: %s", result)
	})

	t.Run("RealAPIInvalidFile", func(t *testing.T) {
		corruptedFile := testutil.CreateCorruptedAudioFile(t, "corrupted.wav")
		defer testutil.CleanupFile(t, corruptedFile)

		_, err := transcriber.Transcript(corruptedFile)

		// Should fail with invalid file
		assert.Error(t, err)
		t.Logf("Expected error for corrupted file: %v", err)
	})

	t.Run("RealAPINonExistentFile", func(t *testing.T) {
		_, err := transcriber.Transcript("/nonexistent/file.wav")

		// Should fail with file not found
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "no such file")
	})
}

// TestAPIResponseValidation tests various API response validation scenarios
func TestAPIResponseValidation(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		expectError  bool
	}{
		{
			name:         "ValidResponse",
			responseBody: `{"text": "Valid transcription text"}`,
			expectError:  false,
		},
		{
			name:         "EmptyText",
			responseBody: `{"text": ""}`,
			expectError:  false,
		},
		{
			name:         "NullText",
			responseBody: `{"text": null}`,
			expectError:  false, // Should handle null gracefully
		},
		{
			name:         "MissingTextField",
			responseBody: `{"result": "transcription", "status": "success"}`,
			expectError:  false, // Should handle missing text field
		},
		{
			name:         "InvalidJSON",
			responseBody: `{"text": "unclosed string`,
			expectError:  true,
		},
		{
			name:         "NonStringText",
			responseBody: `{"text": 12345}`,
			expectError:  true,
		},
		{
			name:         "ArrayInsteadOfObject",
			responseBody: `["text", "array", "response"]`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := openai.DefaultConfig("test-api-key")
			config.BaseURL = server.URL
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "validation_test.wav")
			defer testutil.CleanupFile(t, testFile)

			_, err := transcriber.Transcript(testFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAPIRetryMechanism tests retry logic for transient failures
func TestAPIRetryMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retry mechanism test in short mode")
	}

	tests := []struct {
		name              string
		failureCount      int
		expectedCallCount int
		finalResponse     string
		expectSuccess     bool
	}{
		{
			name:              "SuccessOnFirstTry",
			failureCount:      0,
			expectedCallCount: 1,
			finalResponse:     `{"text": "Success on first try"}`,
			expectSuccess:     true,
		},
		{
			name:              "SuccessOnSecondTry",
			failureCount:      1,
			expectedCallCount: 2,
			finalResponse:     `{"text": "Success on second try"}`,
			expectSuccess:     true,
		},
		{
			name:              "SuccessOnThirdTry",
			failureCount:      2,
			expectedCallCount: 3,
			finalResponse:     `{"text": "Success on third try"}`,
			expectSuccess:     true,
		},
		{
			name:              "AlwaysFails",
			failureCount:      10,
			expectedCallCount: 1, // No retry implemented in current code
			finalResponse:     "",
			expectSuccess:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++

				if callCount <= tt.failureCount {
					// Simulate transient failure
					w.WriteHeader(http.StatusServiceUnavailable)
					w.Write([]byte(`{"error": {"message": "Service temporarily unavailable"}}`))
					return
				}

				// Success response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.finalResponse))
			}))
			defer server.Close()

			config := openai.DefaultConfig("test-api-key")
			config.BaseURL = server.URL
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "retry_test.wav")
			defer testutil.CleanupFile(t, testFile)

			result, err := transcriber.Transcript(testFile)

			// Note: Current implementation doesn't have retry logic
			// This test documents the current behavior and can be updated
			// when retry logic is implemented

			if tt.expectSuccess && tt.failureCount == 0 {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			} else if tt.failureCount > 0 {
				// Current implementation will fail on first transient error
				assert.Error(t, err)
			}

			t.Logf("Call count: %d (expected: %d)", callCount, tt.expectedCallCount)
		})
	}
}

// TestConcurrentAPIRequests tests concurrent API requests
func TestConcurrentAPIRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent API requests test in short mode")
	}

	numRequests := 10
	responseDelay := 100 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add small delay to simulate real API latency
		time.Sleep(responseDelay)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "Concurrent transcription response"}`))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL
	client := openai.NewClientWithConfig(config)

	// Test concurrent requests
	results := make(chan error, numRequests)
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			transcriber := whisper.NewRemoteTranscriber(client)
			testFile := testutil.CreateTestAudioFile(t, fmt.Sprintf("concurrent_test_%d.wav", id))
			defer testutil.CleanupFile(t, testFile)

			_, err := transcriber.Transcript(testFile)
			results <- err
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < numRequests; i++ {
		err := <-results
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	duration := time.Since(start)

	t.Logf("Concurrent requests completed in %v", duration)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)

	// All requests should succeed
	assert.Equal(t, numRequests, successCount)
	assert.Equal(t, 0, errorCount)

	// Should be faster than sequential execution
	maxExpectedDuration := time.Duration(numRequests) * responseDelay
	assert.Less(t, duration, maxExpectedDuration, "Concurrent requests should be faster than sequential")
}

// TestAPIContextCancellation tests context cancellation during API calls
func TestAPIContextCancellation(t *testing.T) {
	longDelay := 2 * time.Second
	shortTimeout := 500 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request was cancelled
		select {
		case <-r.Context().Done():
			return
		case <-time.After(longDelay):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"text": "Should not reach here"}`))
		}
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL
	config.HTTPClient = &http.Client{
		Timeout: shortTimeout,
	}
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "timeout_test.wav")
	defer testutil.CleanupFile(t, testFile)

	start := time.Now()
	_, err := transcriber.Transcript(testFile)
	duration := time.Since(start)

	// Should timeout and error
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "timeout")
	assert.Less(t, duration, longDelay, "Should timeout before server response")
}

// TestAPIDifferentContentTypes tests handling of different content types
func TestAPIDifferentContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		expectError bool
	}{
		{
			name:        "ApplicationJSON",
			contentType: "application/json",
			body:        `{"text": "JSON response"}`,
			expectError: false,
		},
		{
			name:        "TextPlain",
			contentType: "text/plain",
			body:        `{"text": "Plain text response"}`,
			expectError: true, // Should expect JSON
		},
		{
			name:        "ApplicationXML",
			contentType: "application/xml",
			body:        `<response><text>XML response</text></response>`,
			expectError: true, // Should expect JSON
		},
		{
			name:        "NoContentType",
			contentType: "",
			body:        `{"text": "No content type"}`,
			expectError: false, // Should work if valid JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			config := openai.DefaultConfig("test-api-key")
			config.BaseURL = server.URL
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "content_type_test.wav")
			defer testutil.CleanupFile(t, testFile)

			_, err := transcriber.Transcript(testFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAPILargeResponse tests handling of large API responses
func TestAPILargeResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large response test in short mode")
	}

	// Generate a large response (1MB of text)
	largeText := strings.Repeat("This is a very long transcription text. ", 25000)
	responseBody := fmt.Sprintf(`{"text": "%s"}`, largeText)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "large_response_test.wav")
	defer testutil.CleanupFile(t, testFile)

	result, err := transcriber.Transcript(testFile)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Greater(t, len(result), 100000, "Should handle large responses")
	t.Logf("Large response length: %d characters", len(result))
}

// TestAPIErrorResponseParsing tests parsing of various error response formats
func TestAPIErrorResponseParsing(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		responseStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name:           "StandardErrorFormat",
			responseBody:   `{"error": {"message": "Invalid API key provided", "type": "invalid_request_error", "code": "invalid_api_key"}}`,
			responseStatus: http.StatusUnauthorized,
			expectError:    true,
			errorContains:  "invalid api key",
		},
		{
			name:           "SimpleErrorFormat",
			responseBody:   `{"error": "Simple error message"}`,
			responseStatus: http.StatusBadRequest,
			expectError:    true,
			errorContains:  "simple error",
		},
		{
			name:           "NoErrorField",
			responseBody:   `{"message": "Something went wrong", "status": "error"}`,
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:           "EmptyErrorResponse",
			responseBody:   `{}`,
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:           "PlainTextError",
			responseBody:   `Internal Server Error`,
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := openai.DefaultConfig("test-api-key")
			config.BaseURL = server.URL
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "error_parsing_test.wav")
			defer testutil.CleanupFile(t, testFile)

			_, err := transcriber.Transcript(testFile)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
