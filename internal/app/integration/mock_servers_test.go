//go:build integration
// +build integration

package integration

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tiktok-whisper/internal/app/api/openai/whisper"
	"tiktok-whisper/internal/app/testutil"
)

// MockOpenAIServer provides a comprehensive mock server for OpenAI API testing
type MockOpenAIServer struct {
	server          *httptest.Server
	requestLog      []MockRequest
	responseQueue   []MockResponse
	failureMode     string
	mutex           sync.RWMutex
	requestCount    int
	rateLimitConfig RateLimitConfig
}

type MockRequest struct {
	Method    string
	Path      string
	Headers   map[string]string
	Body      []byte
	Timestamp time.Time
}

type MockResponse struct {
	StatusCode  int
	Body        string
	Headers     map[string]string
	Delay       time.Duration
	ContentType string
}

type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerDay    int
	TokensPerMinute   int
	Enabled           bool
	WindowStart       time.Time
	RequestsInWindow  int
}

// NewMockOpenAIServer creates a new mock OpenAI server
func NewMockOpenAIServer() *MockOpenAIServer {
	mock := &MockOpenAIServer{
		requestLog:    make([]MockRequest, 0),
		responseQueue: make([]MockResponse, 0),
		rateLimitConfig: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerDay:    1000,
			TokensPerMinute:   10000,
			Enabled:           false,
		},
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleRequest))
	return mock
}

// Close shuts down the mock server
func (m *MockOpenAIServer) Close() {
	m.server.Close()
}

// URL returns the mock server URL
func (m *MockOpenAIServer) URL() string {
	return m.server.URL
}

// handleRequest handles incoming HTTP requests
func (m *MockOpenAIServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.requestCount++

	// Log request
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()

	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[key] = strings.Join(values, ", ")
	}

	request := MockRequest{
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   headers,
		Body:      body,
		Timestamp: time.Now(),
	}
	m.requestLog = append(m.requestLog, request)

	// Check rate limiting
	if m.rateLimitConfig.Enabled && m.shouldRateLimit() {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`))
		return
	}

	// Handle failure modes
	switch m.failureMode {
	case "timeout":
		time.Sleep(30 * time.Second) // Longer than any reasonable timeout
		return
	case "connection_reset":
		// Simulate connection reset by closing without response
		if hijacker, ok := w.(http.Hijacker); ok {
			conn, _, _ := hijacker.Hijack()
			conn.Close()
		}
		return
	case "malformed_response":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"malformed": json without closing`))
		return
	case "empty_response":
		w.WriteHeader(http.StatusOK)
		return
	case "random_errors":
		if m.requestCount%3 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": {"message": "Random server error"}}`))
			return
		}
	}

	// Use queued response if available
	if len(m.responseQueue) > 0 {
		response := m.responseQueue[0]
		m.responseQueue = m.responseQueue[1:]
		m.sendResponse(w, response)
		return
	}

	// Default success response for transcription
	if strings.Contains(r.URL.Path, "transcriptions") {
		m.sendDefaultTranscriptionResponse(w)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"message": "Endpoint not found"}}`))
	}
}

// shouldRateLimit determines if request should be rate limited
func (m *MockOpenAIServer) shouldRateLimit() bool {
	now := time.Now()

	// Reset window if needed
	if now.Sub(m.rateLimitConfig.WindowStart) > time.Minute {
		m.rateLimitConfig.WindowStart = now
		m.rateLimitConfig.RequestsInWindow = 0
	}

	m.rateLimitConfig.RequestsInWindow++
	return m.rateLimitConfig.RequestsInWindow > m.rateLimitConfig.RequestsPerMinute
}

// sendResponse sends a mock response
func (m *MockOpenAIServer) sendResponse(w http.ResponseWriter, response MockResponse) {
	if response.Delay > 0 {
		time.Sleep(response.Delay)
	}

	// Set headers
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}

	if response.ContentType == "" {
		response.ContentType = "application/json"
	}
	w.Header().Set("Content-Type", response.ContentType)

	w.WriteHeader(response.StatusCode)
	w.Write([]byte(response.Body))
}

// sendDefaultTranscriptionResponse sends a default successful transcription response
func (m *MockOpenAIServer) sendDefaultTranscriptionResponse(w http.ResponseWriter) {
	response := map[string]interface{}{
		"text": fmt.Sprintf("Mock transcription response %d", m.requestCount),
	}

	jsonBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

// Configuration methods

func (m *MockOpenAIServer) SetFailureMode(mode string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.failureMode = mode
}

func (m *MockOpenAIServer) EnableRateLimit(requestsPerMinute int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.rateLimitConfig.Enabled = true
	m.rateLimitConfig.RequestsPerMinute = requestsPerMinute
	m.rateLimitConfig.WindowStart = time.Now()
	m.rateLimitConfig.RequestsInWindow = 0
}

func (m *MockOpenAIServer) DisableRateLimit() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.rateLimitConfig.Enabled = false
}

func (m *MockOpenAIServer) QueueResponse(response MockResponse) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.responseQueue = append(m.responseQueue, response)
}

func (m *MockOpenAIServer) GetRequestCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.requestCount
}

func (m *MockOpenAIServer) GetRequestLog() []MockRequest {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return append([]MockRequest{}, m.requestLog...)
}

func (m *MockOpenAIServer) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.requestLog = make([]MockRequest, 0)
	m.responseQueue = make([]MockResponse, 0)
	m.requestCount = 0
	m.failureMode = ""
}

// Test functions using the mock server

func TestMockServerBasicFunctionality(t *testing.T) {
	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "mock_basic_test.wav")
	defer testutil.CleanupFile(t, testFile)

	result, err := transcriber.Transcript(testFile)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Mock transcription response")
	assert.Equal(t, 1, mockServer.GetRequestCount())

	// Check request log
	requests := mockServer.GetRequestLog()
	assert.Len(t, requests, 1)
	assert.Equal(t, "POST", requests[0].Method)
	assert.Contains(t, requests[0].Path, "transcriptions")
}

func TestMockServerFailureModes(t *testing.T) {
	tests := []struct {
		name        string
		failureMode string
		expectError bool
		timeout     time.Duration
	}{
		{
			name:        "MalformedResponse",
			failureMode: "malformed_response",
			expectError: true,
			timeout:     5 * time.Second,
		},
		{
			name:        "EmptyResponse",
			failureMode: "empty_response",
			expectError: true,
			timeout:     5 * time.Second,
		},
		{
			name:        "RandomErrors",
			failureMode: "random_errors",
			expectError: false, // First request should succeed
			timeout:     5 * time.Second,
		},
		{
			name:        "Timeout",
			failureMode: "timeout",
			expectError: true,
			timeout:     1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := NewMockOpenAIServer()
			defer mockServer.Close()

			mockServer.SetFailureMode(tt.failureMode)

			config := openai.DefaultConfig("test-api-key")
			config.BaseURL = mockServer.URL()
			config.HTTPClient = &http.Client{Timeout: tt.timeout}
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "mock_failure_test.wav")
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

func TestMockServerRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting test in short mode")
	}

	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	// Set aggressive rate limit for testing
	mockServer.EnableRateLimit(3) // 3 requests per minute

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "rate_limit_test.wav")
	defer testutil.CleanupFile(t, testFile)

	successCount := 0
	rateLimitCount := 0

	// Make 5 requests quickly
	for i := 0; i < 5; i++ {
		_, err := transcriber.Transcript(testFile)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "rate limit") ||
				strings.Contains(strings.ToLower(err.Error()), "429") {
				rateLimitCount++
			}
		} else {
			successCount++
		}
	}

	assert.Equal(t, 3, successCount, "Should allow exactly 3 successful requests")
	assert.Equal(t, 2, rateLimitCount, "Should rate limit 2 requests")
	assert.Equal(t, 5, mockServer.GetRequestCount(), "Should have received all 5 requests")
}

func TestMockServerCustomResponses(t *testing.T) {
	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	// Queue custom responses
	mockServer.QueueResponse(MockResponse{
		StatusCode:  http.StatusOK,
		Body:        `{"text": "First custom response"}`,
		ContentType: "application/json",
	})
	mockServer.QueueResponse(MockResponse{
		StatusCode:  http.StatusOK,
		Body:        `{"text": "Second custom response"}`,
		ContentType: "application/json",
		Delay:       200 * time.Millisecond,
	})
	mockServer.QueueResponse(MockResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"error": {"message": "Custom error response"}}`,
	})

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "custom_response_test.wav")
	defer testutil.CleanupFile(t, testFile)

	// First request - should get first custom response
	result1, err1 := transcriber.Transcript(testFile)
	assert.NoError(t, err1)
	assert.Equal(t, "First custom response", result1)

	// Second request - should get second custom response with delay
	start := time.Now()
	result2, err2 := transcriber.Transcript(testFile)
	duration := time.Since(start)

	assert.NoError(t, err2)
	assert.Equal(t, "Second custom response", result2)
	assert.GreaterOrEqual(t, duration, 200*time.Millisecond)

	// Third request - should get error response
	_, err3 := transcriber.Transcript(testFile)
	assert.Error(t, err3)
	assert.Contains(t, err3.Error(), "Custom error response")

	// Fourth request - should get default response (queue exhausted)
	result4, err4 := transcriber.Transcript(testFile)
	assert.NoError(t, err4)
	assert.Contains(t, result4, "Mock transcription response")
}

func TestMockServerConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent mock server test in short mode")
	}

	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)

	numRequests := 10
	results := make(chan error, numRequests)

	// Send concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			transcriber := whisper.NewRemoteTranscriber(client)
			testFile := testutil.CreateTestAudioFile(t, fmt.Sprintf("concurrent_%d.wav", id))
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

	assert.Equal(t, numRequests, successCount)
	assert.Equal(t, 0, errorCount)
	assert.Equal(t, numRequests, mockServer.GetRequestCount())

	// Verify all requests were logged
	requests := mockServer.GetRequestLog()
	assert.Len(t, requests, numRequests)
}

func TestMockServerCompressedResponses(t *testing.T) {
	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	// Create gzip compressed response
	responseText := `{"text": "This is a compressed response that should be handled correctly by the client"}`
	var compressed strings.Builder
	gzipWriter := gzip.NewWriter(&compressed)
	gzipWriter.Write([]byte(responseText))
	gzipWriter.Close()

	mockServer.QueueResponse(MockResponse{
		StatusCode:  http.StatusOK,
		Body:        compressed.String(),
		ContentType: "application/json",
		Headers: map[string]string{
			"Content-Encoding": "gzip",
		},
	})

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "compressed_test.wav")
	defer testutil.CleanupFile(t, testFile)

	result, err := transcriber.Transcript(testFile)

	// Note: This test depends on the HTTP client automatically handling gzip
	// The result will depend on the client's behavior
	t.Logf("Compressed response result: %s, error: %v", result, err)
}

func TestMockServerLargeResponses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large response test in short mode")
	}

	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	// Create a large response (1MB)
	largeText := strings.Repeat("This is a very long transcription response. ", 25000)
	largeResponse := fmt.Sprintf(`{"text": "%s"}`, largeText)

	mockServer.QueueResponse(MockResponse{
		StatusCode:  http.StatusOK,
		Body:        largeResponse,
		ContentType: "application/json",
	})

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "large_response_test.wav")
	defer testutil.CleanupFile(t, testFile)

	result, err := transcriber.Transcript(testFile)

	assert.NoError(t, err)
	assert.Greater(t, len(result), 100000, "Should handle large response")
	t.Logf("Large response size: %d characters", len(result))
}

func TestMockServerHTTPHeaders(t *testing.T) {
	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "headers_test.wav")
	defer testutil.CleanupFile(t, testFile)

	_, err := transcriber.Transcript(testFile)
	assert.NoError(t, err)

	// Check request headers
	requests := mockServer.GetRequestLog()
	require.Len(t, requests, 1)

	request := requests[0]

	// Verify expected headers
	assert.Contains(t, request.Headers, "Authorization")
	assert.Contains(t, request.Headers["Authorization"], "Bearer")
	assert.Contains(t, request.Headers, "User-Agent")

	t.Logf("Request headers: %+v", request.Headers)
}

func TestMockServerDifferentHTTPMethods(t *testing.T) {
	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "methods_test.wav")
	defer testutil.CleanupFile(t, testFile)

	_, err := transcriber.Transcript(testFile)
	assert.NoError(t, err)

	// Verify request method
	requests := mockServer.GetRequestLog()
	require.Len(t, requests, 1)
	assert.Equal(t, "POST", requests[0].Method)
}

func TestMockServerResponseTiming(t *testing.T) {
	mockServer := NewMockOpenAIServer()
	defer mockServer.Close()

	delays := []time.Duration{
		100 * time.Millisecond,
		300 * time.Millisecond,
		500 * time.Millisecond,
	}

	for i, delay := range delays {
		mockServer.QueueResponse(MockResponse{
			StatusCode:  http.StatusOK,
			Body:        fmt.Sprintf(`{"text": "Response %d with delay"}`, i+1),
			ContentType: "application/json",
			Delay:       delay,
		})
	}

	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = mockServer.URL()
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "timing_test.wav")
	defer testutil.CleanupFile(t, testFile)

	for i, expectedDelay := range delays {
		start := time.Now()
		result, err := transcriber.Transcript(testFile)
		actualDelay := time.Since(start)

		assert.NoError(t, err)
		assert.Contains(t, result, fmt.Sprintf("Response %d", i+1))
		assert.GreaterOrEqual(t, actualDelay, expectedDelay, "Response should be delayed")

		t.Logf("Request %d: expected delay %v, actual delay %v", i+1, expectedDelay, actualDelay)
	}
}
