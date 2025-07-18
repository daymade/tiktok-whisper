package testutil

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/app/api"
)

// MockTranscriber is a comprehensive mock implementation of the api.Transcriber interface
// It provides configurable behavior for testing various transcription scenarios
type MockTranscriber struct {
	mock.Mock
	mu sync.RWMutex

	// Configuration options
	DefaultLatency    time.Duration
	DefaultError      error
	DefaultResponse   string
	EnableRealistic   bool
	EnableCallTracking bool

	// State tracking
	CallCount    int
	CallHistory  []TranscriptionCall
	ErrorMap     map[string]error
	ResponseMap  map[string]string
	LatencyMap   map[string]time.Duration
}

// TranscriptionCall represents a single transcription call for tracking
type TranscriptionCall struct {
	InputFilePath string
	Timestamp     time.Time
	Duration      time.Duration
	Response      string
	Error         error
}

// NewMockTranscriber creates a new MockTranscriber with sensible defaults
func NewMockTranscriber() *MockTranscriber {
	return &MockTranscriber{
		DefaultLatency:     10 * time.Millisecond,
		DefaultResponse:    "This is a mock transcription result.",
		EnableRealistic:    true,
		EnableCallTracking: true,
		ErrorMap:           make(map[string]error),
		ResponseMap:        make(map[string]string),
		LatencyMap:         make(map[string]time.Duration),
		CallHistory:        make([]TranscriptionCall, 0),
	}
}

// Transcript implements the api.Transcriber interface
func (m *MockTranscriber) Transcript(inputFilePath string) (string, error) {
	startTime := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	// Track the call
	m.CallCount++
	
	// Check for specific file errors
	if err, exists := m.ErrorMap[inputFilePath]; exists {
		if m.EnableCallTracking {
			m.CallHistory = append(m.CallHistory, TranscriptionCall{
				InputFilePath: inputFilePath,
				Timestamp:     startTime,
				Duration:      time.Since(startTime),
				Error:         err,
			})
		}
		return "", err
	}

	// Check for default error
	if m.DefaultError != nil {
		if m.EnableCallTracking {
			m.CallHistory = append(m.CallHistory, TranscriptionCall{
				InputFilePath: inputFilePath,
				Timestamp:     startTime,
				Duration:      time.Since(startTime),
				Error:         m.DefaultError,
			})
		}
		return "", m.DefaultError
	}

	// Simulate processing latency
	latency := m.DefaultLatency
	if customLatency, exists := m.LatencyMap[inputFilePath]; exists {
		latency = customLatency
	}
	if latency > 0 {
		time.Sleep(latency)
	}

	// Generate response
	var response string
	if customResponse, exists := m.ResponseMap[inputFilePath]; exists {
		response = customResponse
	} else if m.EnableRealistic {
		response = m.generateRealisticResponse(inputFilePath)
	} else {
		response = m.DefaultResponse
	}

	// Record the call
	if m.EnableCallTracking {
		m.CallHistory = append(m.CallHistory, TranscriptionCall{
			InputFilePath: inputFilePath,
			Timestamp:     startTime,
			Duration:      time.Since(startTime),
			Response:      response,
		})
	}

	// Call the testify mock if methods are expected
	args := m.Called(inputFilePath)
	if args.Get(0) != nil {
		return args.String(0), args.Error(1)
	}

	return response, nil
}

// Configuration Methods

// WithDefaultLatency sets the default processing latency
func (m *MockTranscriber) WithDefaultLatency(latency time.Duration) *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DefaultLatency = latency
	return m
}

// WithDefaultError sets the default error to return
func (m *MockTranscriber) WithDefaultError(err error) *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DefaultError = err
	return m
}

// WithDefaultResponse sets the default response text
func (m *MockTranscriber) WithDefaultResponse(response string) *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DefaultResponse = response
	return m
}

// WithRealisticResponses enables/disables realistic response generation
func (m *MockTranscriber) WithRealisticResponses(enabled bool) *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EnableRealistic = enabled
	return m
}

// WithCallTracking enables/disables call history tracking
func (m *MockTranscriber) WithCallTracking(enabled bool) *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EnableCallTracking = enabled
	return m
}

// File-specific Configuration

// SetErrorForFile sets a specific error for a given file path
func (m *MockTranscriber) SetErrorForFile(filePath string, err error) *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorMap[filePath] = err
	return m
}

// SetResponseForFile sets a specific response for a given file path
func (m *MockTranscriber) SetResponseForFile(filePath string, response string) *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ResponseMap[filePath] = response
	return m
}

// SetLatencyForFile sets a specific latency for a given file path
func (m *MockTranscriber) SetLatencyForFile(filePath string, latency time.Duration) *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LatencyMap[filePath] = latency
	return m
}

// Common Error Scenarios

// SimulateFileNotFound simulates file not found errors for specific patterns
func (m *MockTranscriber) SimulateFileNotFound(filePattern string) *MockTranscriber {
	return m.SetErrorForFile(filePattern, fmt.Errorf("file not found: %s", filePattern))
}

// SimulateProcessingError simulates processing errors
func (m *MockTranscriber) SimulateProcessingError(filePath string, message string) *MockTranscriber {
	return m.SetErrorForFile(filePath, fmt.Errorf("processing error: %s", message))
}

// SimulateNetworkError simulates network errors for API-based transcription
func (m *MockTranscriber) SimulateNetworkError(filePath string) *MockTranscriber {
	return m.SetErrorForFile(filePath, fmt.Errorf("network error: connection timeout"))
}

// SimulateQuotaExceededError simulates quota exceeded errors
func (m *MockTranscriber) SimulateQuotaExceededError(filePath string) *MockTranscriber {
	return m.SetErrorForFile(filePath, fmt.Errorf("quota exceeded: API rate limit reached"))
}

// State Inspection Methods

// GetCallCount returns the total number of calls made
func (m *MockTranscriber) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CallCount
}

// GetCallHistory returns the complete call history
func (m *MockTranscriber) GetCallHistory() []TranscriptionCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history := make([]TranscriptionCall, len(m.CallHistory))
	copy(history, m.CallHistory)
	return history
}

// GetLastCall returns the last transcription call
func (m *MockTranscriber) GetLastCall() *TranscriptionCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.CallHistory) == 0 {
		return nil
	}
	return &m.CallHistory[len(m.CallHistory)-1]
}

// WasCalledWith checks if the transcriber was called with a specific file path
func (m *MockTranscriber) WasCalledWith(filePath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, call := range m.CallHistory {
		if call.InputFilePath == filePath {
			return true
		}
	}
	return false
}

// GetCallsForFile returns all calls made for a specific file
func (m *MockTranscriber) GetCallsForFile(filePath string) []TranscriptionCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var calls []TranscriptionCall
	for _, call := range m.CallHistory {
		if call.InputFilePath == filePath {
			calls = append(calls, call)
		}
	}
	return calls
}

// GetAverageLatency returns the average processing latency
func (m *MockTranscriber) GetAverageLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.CallHistory) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, call := range m.CallHistory {
		total += call.Duration
	}
	return total / time.Duration(len(m.CallHistory))
}

// Reset Methods

// Reset clears all state and returns to default configuration
func (m *MockTranscriber) Reset() *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.CallCount = 0
	m.CallHistory = make([]TranscriptionCall, 0)
	m.ErrorMap = make(map[string]error)
	m.ResponseMap = make(map[string]string)
	m.LatencyMap = make(map[string]time.Duration)
	m.DefaultError = nil
	m.DefaultResponse = "This is a mock transcription result."
	m.DefaultLatency = 10 * time.Millisecond
	m.EnableRealistic = true
	m.EnableCallTracking = true
	
	return m
}

// ClearHistory clears call history but keeps configuration
func (m *MockTranscriber) ClearHistory() *MockTranscriber {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCount = 0
	m.CallHistory = make([]TranscriptionCall, 0)
	return m
}

// Test Helper Methods

// ExpectTranscriptCall sets up an expectation for a specific transcript call
func (m *MockTranscriber) ExpectTranscriptCall(filePath string, response string, err error) *MockTranscriber {
	m.On("Transcript", filePath).Return(response, err)
	return m
}

// ExpectTranscriptCallOnce sets up a one-time expectation
func (m *MockTranscriber) ExpectTranscriptCallOnce(filePath string, response string, err error) *MockTranscriber {
	m.On("Transcript", filePath).Return(response, err).Once()
	return m
}

// ExpectTranscriptCallTimes sets up an expectation for specific number of calls
func (m *MockTranscriber) ExpectTranscriptCallTimes(filePath string, response string, err error, times int) *MockTranscriber {
	m.On("Transcript", filePath).Return(response, err).Times(times)
	return m
}

// Private helper methods

// generateRealisticResponse generates a realistic transcription response based on file path
func (m *MockTranscriber) generateRealisticResponse(filePath string) string {
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	
	// Generate response based on filename patterns
	if strings.Contains(nameWithoutExt, "jfk") {
		return "And so, my fellow Americans, ask not what your country can do for you, ask what you can do for your country."
	}
	
	if strings.Contains(nameWithoutExt, "test") {
		return "This is a test audio file for transcription testing purposes."
	}
	
	if strings.Contains(nameWithoutExt, "empty") || strings.Contains(nameWithoutExt, "silence") {
		return ""
	}
	
	if strings.Contains(nameWithoutExt, "chinese") || strings.Contains(nameWithoutExt, "中文") {
		return "这是一个中文语音转文字的测试文件。"
	}
	
	if strings.Contains(nameWithoutExt, "long") {
		return "This is a longer transcription response that simulates the output from a longer audio file. It contains multiple sentences and should be used for testing scenarios that involve longer text processing and analysis."
	}
	
	if strings.Contains(nameWithoutExt, "podcast") {
		return "Welcome to our podcast. Today we're discussing the latest developments in artificial intelligence and machine learning."
	}
	
	// Default response with file information
	return fmt.Sprintf("Mock transcription result for file: %s. This is a realistic response generated for testing purposes.", filename)
}

// Interface compliance check
var _ api.Transcriber = (*MockTranscriber)(nil)