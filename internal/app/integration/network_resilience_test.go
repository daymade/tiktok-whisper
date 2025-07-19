//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"tiktok-whisper/internal/app/api/openai/whisper"
	"tiktok-whisper/internal/app/testutil"
)

// TestNetworkTimeout tests various timeout scenarios
func TestNetworkTimeout(t *testing.T) {
	tests := []struct {
		name           string
		serverDelay    time.Duration
		clientTimeout  time.Duration
		expectedError  bool
		errorContains  string
	}{
		{
			name:          "FastResponse",
			serverDelay:   100 * time.Millisecond,
			clientTimeout: 1 * time.Second,
			expectedError: false,
		},
		{
			name:          "SlowResponseWithinTimeout",
			serverDelay:   500 * time.Millisecond,
			clientTimeout: 1 * time.Second,
			expectedError: false,
		},
		{
			name:          "TimeoutExceeded",
			serverDelay:   2 * time.Second,
			clientTimeout: 500 * time.Millisecond,
			expectedError: true,
			errorContains: "timeout",
		},
		{
			name:          "VeryShortTimeout",
			serverDelay:   100 * time.Millisecond,
			clientTimeout: 50 * time.Millisecond,
			expectedError: true,
			errorContains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server with configurable delay
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.serverDelay)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"text": "Mock transcription"}`)
			}))
			defer server.Close()

			// Create OpenAI client with custom timeout
			config := openai.DefaultConfig("test-key")
			config.BaseURL = server.URL
			config.HTTPClient = &http.Client{
				Timeout: tt.clientTimeout,
			}
			
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			// Create a test audio file
			testFile := testutil.CreateTestAudioFile(t, "timeout_test.wav")
			defer testutil.CleanupFile(t, testFile)

			// Attempt transcription
			result, err := transcriber.Transcript(testFile)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

// TestConnectionRefused tests scenarios where the server is unreachable
func TestConnectionRefused(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		expectedError string
	}{
		{
			name:          "InvalidHost",
			baseURL:       "http://invalid-host-that-should-not-exist-12345.com",
			expectedError: "no such host",
		},
		{
			name:          "UnusedPort",
			baseURL:       "http://localhost:19999",
			expectedError: "connection refused",
		},
		{
			name:          "InvalidScheme",
			baseURL:       "invalid://localhost:8080",
			expectedError: "unsupported protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := openai.DefaultConfig("test-key")
			config.BaseURL = tt.baseURL
			config.HTTPClient = &http.Client{
				Timeout: 2 * time.Second,
			}

			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "connection_test.wav")
			defer testutil.CleanupFile(t, testFile)

			result, err := transcriber.Transcript(testFile)

			assert.Error(t, err)
			assert.Empty(t, result)
			assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.expectedError))
		})
	}
}

// TestIntermittentConnectivity tests scenarios with unstable network conditions
func TestIntermittentConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping intermittent connectivity test in short mode")
	}

	tests := []struct {
		name             string
		successRate      float64 // 0.0 to 1.0
		attempts         int
		expectedFailures int
		maxFailures      int
	}{
		{
			name:             "HighSuccessRate",
			successRate:      0.8,
			attempts:         10,
			expectedFailures: 2,
			maxFailures:      3,
		},
		{
			name:             "MediumSuccessRate",
			successRate:      0.5,
			attempts:         10,
			expectedFailures: 5,
			maxFailures:      7,
		},
		{
			name:             "LowSuccessRate",
			successRate:      0.2,
			attempts:         10,
			expectedFailures: 8,
			maxFailures:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				
				// Simulate intermittent failures based on success rate
				if float64(callCount%10)/10.0 >= tt.successRate {
					http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"text": "Transcription successful"}`)
			}))
			defer server.Close()

			config := openai.DefaultConfig("test-key")
			config.BaseURL = server.URL
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "intermittent_test.wav")
			defer testutil.CleanupFile(t, testFile)

			failures := 0
			successes := 0

			for i := 0; i < tt.attempts; i++ {
				_, err := transcriber.Transcript(testFile)
				if err != nil {
					failures++
				} else {
					successes++
				}
			}

			t.Logf("Attempts: %d, Successes: %d, Failures: %d", tt.attempts, successes, failures)
			assert.LessOrEqual(t, failures, tt.maxFailures, "Too many failures")
			assert.GreaterOrEqual(t, successes, 1, "Expected at least one success")
		})
	}
}

// TestRateLimiting tests scenarios with API rate limiting
func TestRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting test in short mode")
	}

	tests := []struct {
		name            string
		requestsPerSec  int
		rateLimitPerSec int
		duration        time.Duration
		expectRateLimit bool
	}{
		{
			name:            "WithinLimits",
			requestsPerSec:  1,
			rateLimitPerSec: 5,
			duration:        3 * time.Second,
			expectRateLimit: false,
		},
		{
			name:            "ExceedsLimits",
			requestsPerSec:  10,
			rateLimitPerSec: 3,
			duration:        2 * time.Second,
			expectRateLimit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			requestTimes := make([]time.Time, 0)
			mutex := sync.Mutex{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutex.Lock()
				requestCount++
				now := time.Now()
				requestTimes = append(requestTimes, now)

				// Check rate limit - count requests in the last second
				cutoff := now.Add(-1 * time.Second)
				recentRequests := 0
				for _, reqTime := range requestTimes {
					if reqTime.After(cutoff) {
						recentRequests++
					}
				}
				mutex.Unlock()

				// Simulate rate limiting
				if recentRequests > tt.rateLimitPerSec {
					w.Header().Set("Retry-After", "1")
					http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"text": "Transcription %d"}`, requestCount)
			}))
			defer server.Close()

			config := openai.DefaultConfig("test-key")
			config.BaseURL = server.URL
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "ratelimit_test.wav")
			defer testutil.CleanupFile(t, testFile)

			rateLimitErrors := 0
			successCount := 0
			
			// Send requests at the specified rate
			ticker := time.NewTicker(time.Second / time.Duration(tt.requestsPerSec))
			defer ticker.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), tt.duration)
			defer cancel()

			for {
				select {
				case <-ticker.C:
					_, err := transcriber.Transcript(testFile)
					if err != nil && strings.Contains(err.Error(), "429") {
						rateLimitErrors++
					} else if err == nil {
						successCount++
					}
				case <-ctx.Done():
					goto done
				}
			}

			done:
			t.Logf("Success: %d, Rate limit errors: %d", successCount, rateLimitErrors)

			if tt.expectRateLimit {
				assert.Greater(t, rateLimitErrors, 0, "Expected rate limit errors")
			} else {
				assert.Equal(t, 0, rateLimitErrors, "Unexpected rate limit errors")
			}
		})
	}
}

// TestNetworkPartition tests scenarios where network becomes temporarily unavailable
func TestNetworkPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network partition test in short mode")
	}

	// Create a server that will be stopped and restarted
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"text": "Transcription successful"}`)
	}))

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL
	config.HTTPClient = &http.Client{
		Timeout: 1 * time.Second,
	}
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "partition_test.wav")
	defer testutil.CleanupFile(t, testFile)

	// Test initial connectivity
	_, err := transcriber.Transcript(testFile)
	assert.NoError(t, err, "Initial connection should work")

	// Simulate network partition by closing the server
	server.Close()

	// Attempt transcription during partition
	_, err = transcriber.Transcript(testFile)
	assert.Error(t, err, "Should fail during network partition")

	// Restart server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"text": "Transcription after recovery"}`)
	}))
	defer server.Close()

	// Update client configuration for new server
	config.BaseURL = server.URL
	client = openai.NewClientWithConfig(config)
	transcriber = whisper.NewRemoteTranscriber(client)

	// Test recovery
	result, err := transcriber.Transcript(testFile)
	assert.NoError(t, err, "Should work after network recovery")
	assert.Contains(t, result, "recovery", "Should receive response from recovered server")
}

// TestDNSFailures tests various DNS-related failure scenarios
func TestDNSFailures(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		expectError bool
	}{
		{
			name:        "NonExistentDomain",
			hostname:    "this-domain-should-never-exist-12345.invalid",
			expectError: true,
		},
		{
			name:        "InvalidTLD",
			hostname:    "api.openai.invalidtld",
			expectError: true,
		},
		{
			name:        "LocalhostValid",
			hostname:    "localhost",
			expectError: true, // Will fail because there's no server, but DNS resolves
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := openai.DefaultConfig("test-key")
			config.BaseURL = fmt.Sprintf("https://%s", tt.hostname)
			config.HTTPClient = &http.Client{
				Timeout: 2 * time.Second,
			}

			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			testFile := testutil.CreateTestAudioFile(t, "dns_test.wav")
			defer testutil.CleanupFile(t, testFile)

			_, err := transcriber.Transcript(testFile)

			if tt.expectError {
				assert.Error(t, err)
				t.Logf("Expected error received: %v", err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConcurrentNetworkFailures tests how the system handles concurrent requests during network issues
func TestConcurrentNetworkFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent network failures test in short mode")
	}

	// Create a server that fails intermittently
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		
		// Fail every third request
		if callCount%3 == 0 {
			http.Error(w, "Simulated failure", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"text": "Concurrent transcription %d"}`, callCount)
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL
	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "concurrent_test.wav")
	defer testutil.CleanupFile(t, testFile)

	// Run multiple concurrent requests
	numGoroutines := 10
	results := make(chan error, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := transcriber.Transcript(testFile)
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	errorCount := 0

	for err := range results {
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	t.Logf("Concurrent requests - Success: %d, Errors: %d", successCount, errorCount)
	
	// We expect some successes and some failures based on our 1/3 failure rate
	assert.Greater(t, successCount, 0, "Should have some successful requests")
	assert.Greater(t, errorCount, 0, "Should have some failed requests")
	assert.Equal(t, numGoroutines, successCount+errorCount, "All requests should complete")
}

// TestGracefulDegradation tests the system's ability to degrade gracefully when external services fail
func TestGracefulDegradation(t *testing.T) {
	// This test would typically involve testing fallback mechanisms
	// For now, we'll test that the system properly reports failures
	
	config := openai.DefaultConfig("test-key")
	config.BaseURL = "http://localhost:99999" // Unavailable port
	config.HTTPClient = &http.Client{
		Timeout: 1 * time.Second,
	}

	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "degradation_test.wav")
	defer testutil.CleanupFile(t, testFile)

	result, err := transcriber.Transcript(testFile)

	// System should fail gracefully with appropriate error
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, strings.ToLower(err.Error()), "connection")

	t.Logf("Graceful degradation error: %v", err)
}

// TestCustomDialer tests network failures at the connection level
func TestCustomDialer(t *testing.T) {
	// Create a custom dialer that simulates connection issues
	customDialer := &net.Dialer{
		Timeout: 100 * time.Millisecond, // Very short timeout
	}

	transport := &http.Transport{
		DialContext: customDialer.DialContext,
	}

	config := openai.DefaultConfig("test-key")
	config.BaseURL = "http://httpbin.org" // External service that should be slow enough to timeout
	config.HTTPClient = &http.Client{
		Transport: transport,
		Timeout:   200 * time.Millisecond,
	}

	client := openai.NewClientWithConfig(config)
	transcriber := whisper.NewRemoteTranscriber(client)

	testFile := testutil.CreateTestAudioFile(t, "dialer_test.wav")
	defer testutil.CleanupFile(t, testFile)

	_, err := transcriber.Transcript(testFile)

	// Should fail due to connection timeout
	assert.Error(t, err)
	t.Logf("Custom dialer error: %v", err)
}