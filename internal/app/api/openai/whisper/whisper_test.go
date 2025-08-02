package whisper

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

// TestRemoteTranscriber_Transcript tests the RemoteTranscriber implementation
func TestRemoteTranscriber_Transcript(t *testing.T) {
	tests := []struct {
		name          string
		inputFile     string
		mockResponse  string
		mockStatus    int
		expectedText  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "successful transcription",
			inputFile:    "/test/audio.mp3",
			mockResponse: `{"text": "This is a test transcription"}`,
			mockStatus:   http.StatusOK,
			expectedText: "This is a test transcription",
			expectError:  false,
		},
		{
			name:         "successful transcription with special characters",
			inputFile:    "/test/audio.wav",
			mockResponse: `{"text": "Hello, 世界! This is a test with émojis 🎵"}`,
			mockStatus:   http.StatusOK,
			expectedText: "Hello, 世界! This is a test with émojis 🎵",
			expectError:  false,
		},
		{
			name:          "API error - unauthorized",
			inputFile:     "/test/audio.mp3",
			mockResponse:  `{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`,
			mockStatus:    http.StatusUnauthorized,
			expectError:   true,
			errorContains: "401",
		},
		{
			name:          "API error - rate limit",
			inputFile:     "/test/audio.mp3",
			mockResponse:  `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			mockStatus:    http.StatusTooManyRequests,
			expectError:   true,
			errorContains: "429",
		},
		{
			name:          "API error - server error",
			inputFile:     "/test/audio.mp3",
			mockResponse:  `{"error": {"message": "Internal server error", "type": "server_error"}}`,
			mockStatus:    http.StatusInternalServerError,
			expectError:   true,
			errorContains: "500",
		},
		{
			name:          "network error",
			inputFile:     "/test/audio.mp3",
			mockStatus:    0, // This will trigger a panic in the server, simulating network error
			expectError:   true,
			errorContains: "EOF",
		},
		{
			name:          "invalid JSON response",
			inputFile:     "/test/audio.mp3",
			mockResponse:  `{"text": "incomplete JSON`,
			mockStatus:    http.StatusOK,
			expectError:   true,
			errorContains: "EOF",
		},
		{
			name:         "empty transcription",
			inputFile:    "/test/audio.mp3",
			mockResponse: `{"text": ""}`,
			mockStatus:   http.StatusOK,
			expectedText: "",
			expectError:  false,
		},
		{
			name:         "transcription with line breaks",
			inputFile:    "/test/audio.mp3",
			mockResponse: `{"text": "Line 1\nLine 2\nLine 3"}`,
			mockStatus:   http.StatusOK,
			expectedText: "Line 1\nLine 2\nLine 3",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that returns our mock responses
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Handle network error simulation
				if tt.mockStatus == 0 {
					// Close connection without writing anything to simulate network error
					hijacker, ok := w.(http.Hijacker)
					if ok {
						conn, _, _ := hijacker.Hijack()
						conn.Close()
						return
					}
				}

				// Verify request headers
				if r.Header.Get("Authorization") == "" {
					t.Error("Missing Authorization header")
				}

				// Verify request method
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST method, got %s", r.Method)
				}

				// Verify content type
				contentType := r.Header.Get("Content-Type")
				if !strings.Contains(contentType, "multipart/form-data") {
					t.Errorf("Expected multipart/form-data content type, got %s", contentType)
				}

				// Parse multipart form
				err := r.ParseMultipartForm(32 << 20) // 32MB
				if err != nil {
					t.Errorf("Failed to parse multipart form: %v", err)
				}

				// Verify model parameter
				model := r.FormValue("model")
				if model != "whisper-1" {
					t.Errorf("Expected model whisper-1, got %s", model)
				}

				// Verify file upload
				file, header, err := r.FormFile("file")
				if err != nil {
					t.Errorf("Failed to get file from form: %v", err)
				}
				defer file.Close()

				// Log file details
				t.Logf("Uploaded file: %s", header.Filename)

				// Write mock response
				if tt.mockStatus > 0 {
					w.WriteHeader(tt.mockStatus)
				}
				if tt.mockResponse != "" {
					w.Write([]byte(tt.mockResponse))
				}
			}))
			defer server.Close()

			// Create a client with custom config pointing to our test server
			config := openai.DefaultConfig("test-api-key")
			config.BaseURL = server.URL + "/v1"
			client := openai.NewClientWithConfig(config)

			// Create the transcriber
			rt := NewRemoteTranscriber(client)

			// Create a temporary test file
			tempFile := createTempTestFile(t, tt.inputFile)
			defer os.Remove(tempFile)

			// Perform transcription
			result, err := rt.Transcript(tempFile)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expectedText {
					t.Errorf("Expected text '%s', got '%s'", tt.expectedText, result)
				}
			}
		})
	}
}

// TestRemoteTranscriber_FileNotFound tests handling of non-existent files
func TestRemoteTranscriber_FileNotFound(t *testing.T) {
	config := openai.DefaultConfig("test-api-key")
	client := openai.NewClientWithConfig(config)
	rt := NewRemoteTranscriber(client)

	_, err := rt.Transcript("/non/existent/file.mp3")
	if err == nil {
		t.Error("Expected error for non-existent file, got none")
	}
}

// TestRemoteTranscriber_LargeFile tests handling of large audio files
func TestRemoteTranscriber_LargeFile(t *testing.T) {
	// Create a test server that tracks upload size
	var uploadedSize int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track content length
		uploadedSize = r.ContentLength

		// Parse multipart to verify file can be read
		err := r.ParseMultipartForm(100 << 20) // 100MB limit
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": {"message": "File too large"}}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "Large file transcription"}`))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(config)
	rt := NewRemoteTranscriber(client)

	// Create a temporary large test file (simulate with small file for testing)
	tempFile := createTempTestFile(t, "/test/large_audio.mp3")
	defer os.Remove(tempFile)

	result, err := rt.Transcript(tempFile)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "Large file transcription" {
		t.Errorf("Expected 'Large file transcription', got '%s'", result)
	}
	if uploadedSize == 0 {
		t.Error("Upload size was not tracked")
	}
}

// TestRemoteTranscriber_Timeout tests request timeout handling
func TestRemoteTranscriber_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than typical timeout
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "Should timeout"}`))
	}))
	defer server.Close()

	// Create client with short timeout
	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL + "/v1"
	config.HTTPClient = &http.Client{
		Timeout: 100 * time.Millisecond,
	}
	client := openai.NewClientWithConfig(config)
	rt := NewRemoteTranscriber(client)

	tempFile := createTempTestFile(t, "/test/audio.mp3")
	defer os.Remove(tempFile)

	_, err := rt.Transcript(tempFile)
	if err == nil {
		t.Error("Expected timeout error, got none")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestRemoteTranscriber_ConcurrentRequests tests concurrent transcription requests
func TestRemoteTranscriber_ConcurrentRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"text": "Transcription %d"}`, requestCount)))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(config)
	rt := NewRemoteTranscriber(client)

	// Create multiple temp files
	numRequests := 5
	tempFiles := make([]string, numRequests)
	for i := 0; i < numRequests; i++ {
		tempFiles[i] = createTempTestFile(t, fmt.Sprintf("/test/audio%d.mp3", i))
		defer os.Remove(tempFiles[i])
	}

	// Run concurrent transcriptions
	results := make(chan string, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			result, err := rt.Transcript(tempFiles[index])
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-errors:
			t.Errorf("Unexpected error in concurrent request: %v", err)
		case result := <-results:
			if !strings.Contains(result, "Transcription") {
				t.Errorf("Unexpected result: %s", result)
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for concurrent requests")
		}
	}

	if requestCount != numRequests {
		t.Errorf("Expected %d requests, got %d", numRequests, requestCount)
	}
}

// TestRemoteTranscriber_RetryableErrors tests handling of retryable errors
func TestRemoteTranscriber_RetryableErrors(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		// Fail first attempt with retryable error
		if attemptCount == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": {"message": "Service temporarily unavailable"}}`))
			return
		}

		// Succeed on second attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "Success after retry"}`))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(config)
	rt := NewRemoteTranscriber(client)

	tempFile := createTempTestFile(t, "/test/audio.mp3")
	defer os.Remove(tempFile)

	// Note: The OpenAI client may not automatically retry,
	// so we expect an error on the first attempt
	_, err := rt.Transcript(tempFile)
	if err == nil {
		t.Error("Expected error for service unavailable")
	}
}

// TestRemoteTranscriber_EmptyAPIKey tests behavior with empty API key
func TestRemoteTranscriber_EmptyAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || auth == "Bearer " {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": {"message": "Missing API key"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "Should not reach here"}`))
	}))
	defer server.Close()

	config := openai.DefaultConfig("")
	config.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(config)
	rt := NewRemoteTranscriber(client)

	tempFile := createTempTestFile(t, "/test/audio.mp3")
	defer os.Remove(tempFile)

	_, err := rt.Transcript(tempFile)
	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

// Helper function to create temporary test files
func createTempTestFile(t *testing.T, name string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, filepath.Base(name))

	// Create a minimal valid audio file (WAV header)
	wavHeader := []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x24, 0x00, 0x00, 0x00, // File size
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Chunk size
		0x01, 0x00, // Audio format (PCM)
		0x01, 0x00, // Channels (mono)
		0x80, 0x3E, 0x00, 0x00, // Sample rate (16000)
		0x00, 0x7D, 0x00, 0x00, // Byte rate
		0x02, 0x00, // Block align
		0x10, 0x00, // Bits per sample
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x00, 0x00, 0x00, // Data size
	}

	err := os.WriteFile(tempFile, wavHeader, 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return tempFile
}

// Helper function to create temporary test files for benchmarks
func createTempTestFileForBenchmark(b *testing.B, name string) string {
	b.Helper()

	tempDir := b.TempDir()
	tempFile := filepath.Join(tempDir, filepath.Base(name))

	// Create a minimal valid audio file (WAV header)
	wavHeader := []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x24, 0x00, 0x00, 0x00, // File size
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Chunk size
		0x01, 0x00, // Audio format (PCM)
		0x01, 0x00, // Channels (mono)
		0x80, 0x3E, 0x00, 0x00, // Sample rate (16000)
		0x00, 0x7D, 0x00, 0x00, // Byte rate
		0x02, 0x00, // Block align
		0x10, 0x00, // Bits per sample
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x00, 0x00, 0x00, // Data size
	}

	err := os.WriteFile(tempFile, wavHeader, 0644)
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}

	return tempFile
}

// BenchmarkRemoteTranscriber_Transcript benchmarks the transcription performance
func BenchmarkRemoteTranscriber_Transcript(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate processing time
		time.Sleep(50 * time.Millisecond)

		// Read the uploaded file to simulate real processing
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Read file content (simulate processing)
		_, err = io.ReadAll(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "Benchmark transcription result"}`))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(config)
	rt := NewRemoteTranscriber(client)

	tempFile := createTempTestFileForBenchmark(b, "/test/benchmark.mp3")
	defer os.Remove(tempFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rt.Transcript(tempFile)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
