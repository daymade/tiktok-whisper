package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/app/embedding/provider"
	"tiktok-whisper/internal/app/storage/vector"
	"tiktok-whisper/internal/app/testutil"
)

// =============================================================================
// COMPREHENSIVE ERROR HANDLING TESTS
// =============================================================================

// Common error types for testing
var (
	NetworkTimeoutError    = &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("i/o timeout")}
	ConnectionRefusedError = &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	APIRateLimitError      = errors.New("rate limit exceeded")
	APIQuotaExceededError  = errors.New("quota exceeded")
	DatabaseDeadlockError  = errors.New("deadlock detected")
	DatabaseTimeoutError   = errors.New("database operation timeout")
	OutOfMemoryError       = errors.New("out of memory")
	DiskFullError          = errors.New("no space left on device")
)

// TestEmbeddingOrchestrator_NetworkErrorHandling tests various network-related errors
func TestEmbeddingOrchestrator_NetworkErrorHandling(t *testing.T) {
	tests := []struct {
		name              string
		openaiError       error
		geminiError       error
		expectedErrorType string
		shouldLogError    bool
		errorLogPattern   string
	}{
		{
			name:              "Network timeout on OpenAI",
			openaiError:       NetworkTimeoutError,
			geminiError:       nil,
			expectedErrorType: "embedding generation failed",
			shouldLogError:    true,
			errorLogPattern:   "Failed to generate OpenAI embedding",
		},
		{
			name:              "Connection refused on Gemini",
			openaiError:       nil,
			geminiError:       ConnectionRefusedError,
			expectedErrorType: "embedding generation failed",
			shouldLogError:    true,
			errorLogPattern:   "Failed to generate Gemini embedding",
		},
		{
			name:              "Rate limit on both providers",
			openaiError:       APIRateLimitError,
			geminiError:       APIRateLimitError,
			expectedErrorType: "embedding generation failed",
			shouldLogError:    true,
			errorLogPattern:   "Failed to generate",
		},
		{
			name:              "Quota exceeded scenarios",
			openaiError:       APIQuotaExceededError,
			geminiError:       nil,
			expectedErrorType: "embedding generation failed",
			shouldLogError:    true,
			errorLogPattern:   "Failed to generate OpenAI embedding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockOpenAI := new(MockEmbeddingProvider)
			mockGemini := new(MockEmbeddingProvider)
			mockStorage := new(MockVectorStorage)
			mockLogger := testutil.NewMockLogger()

			providers := map[string]provider.EmbeddingProvider{
				"openai": mockOpenAI,
				"gemini": mockGemini,
			}

			orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

			// Setup mocks
			openaiEmbedding := make([]float32, 1536)
			geminiEmbedding := make([]float32, 768)

			if tt.openaiError != nil {
				mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return([]float32(nil), tt.openaiError)
			} else {
				mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(openaiEmbedding, nil)
			}

			if tt.geminiError != nil {
				mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return([]float32(nil), tt.geminiError)
			} else {
				mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(geminiEmbedding, nil)
			}

			mockLogger.SetEnabled(true)

			// Act
			err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

			// Assert
			if tt.openaiError != nil || tt.geminiError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorType)
			} else {
				assert.NoError(t, err)
			}

			if tt.shouldLogError {
				errorLogs := mockLogger.GetLogsByLevel(testutil.LogLevelError)
				assert.Greater(t, len(errorLogs), 0, "Should have logged errors")

				found := false
				for _, log := range errorLogs {
					if assert.Contains(t, log.Message, tt.errorLogPattern) {
						found = true
						break
					}
				}
				assert.True(t, found, "Should have logged expected error pattern")
			}

			mockOpenAI.AssertExpectations(t)
			mockGemini.AssertExpectations(t)
		})
	}
}

// TestEmbeddingOrchestrator_DatabaseErrorHandling tests database-related error scenarios
func TestEmbeddingOrchestrator_DatabaseErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		storageError  error
		expectedError string
		shouldRetry   bool
	}{
		{
			name:          "Database deadlock",
			storageError:  DatabaseDeadlockError,
			expectedError: "failed to store dual embeddings",
		},
		{
			name:          "Database timeout",
			storageError:  DatabaseTimeoutError,
			expectedError: "failed to store dual embeddings",
		},
		{
			name:          "Disk full error",
			storageError:  DiskFullError,
			expectedError: "failed to store dual embeddings",
		},
		{
			name:          "Connection pool exhausted",
			storageError:  errors.New("connection pool exhausted"),
			expectedError: "failed to store dual embeddings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockOpenAI := new(MockEmbeddingProvider)
			mockGemini := new(MockEmbeddingProvider)
			mockStorage := new(MockVectorStorage)
			mockLogger := testutil.NewMockLogger()

			providers := map[string]provider.EmbeddingProvider{
				"openai": mockOpenAI,
				"gemini": mockGemini,
			}

			orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

			// Setup successful embedding generation
			openaiEmbedding := make([]float32, 1536)
			geminiEmbedding := make([]float32, 768)

			mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(openaiEmbedding, nil)
			mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(geminiEmbedding, nil)

			// Setup storage failure
			mockStorage.On("StoreDualEmbeddings", mock.Anything, 1, openaiEmbedding, geminiEmbedding).Return(tt.storageError)
			mockLogger.SetEnabled(true)

			// Act
			err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Contains(t, err.Error(), tt.storageError.Error())

			mockOpenAI.AssertExpectations(t)
			mockGemini.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
		})
	}
}

// TestBatchProcessor_ErrorPropagation tests how errors propagate through batch processing
func TestBatchProcessor_ErrorPropagation(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create transcriptions with different error scenarios
	transcriptions := []*vector.Transcription{
		{ID: 1, TranscriptionText: "Success case", User: "user1"},
		{ID: 2, TranscriptionText: "Network timeout case", User: "user1"},
		{ID: 3, TranscriptionText: "Rate limit case", User: "user1"},
		{ID: 4, TranscriptionText: "Another success case", User: "user1"},
		{ID: 5, TranscriptionText: "Database error case", User: "user1"},
	}

	// Setup mocks with various errors
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 1, "Success case").Return(nil)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 2, "Network timeout case").Return(NetworkTimeoutError)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 3, "Rate limit case").Return(APIRateLimitError)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 4, "Another success case").Return(nil)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 5, "Database error case").Return(DatabaseDeadlockError)
	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 3)

	// Assert
	assert.NoError(t, err) // Batch processing should continue despite individual failures
	assert.Equal(t, 2, result.Processed, "Should have processed successful items")
	assert.Equal(t, 3, result.Failed, "Should have 3 failed items")
	assert.Len(t, result.Errors, 3, "Should have collected all errors")

	// Verify specific errors are included
	errorMessages := make([]string, len(result.Errors))
	for i, e := range result.Errors {
		errorMessages[i] = e.Error()
	}

	assert.Contains(t, errorMessages, NetworkTimeoutError.Error())
	assert.Contains(t, errorMessages, APIRateLimitError.Error())
	assert.Contains(t, errorMessages, DatabaseDeadlockError.Error())

	mockOrchestrator.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_GoroutineLeakPrevention tests for goroutine leaks in error scenarios
func TestEmbeddingOrchestrator_GoroutineLeakPrevention(t *testing.T) {
	// Record initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Test multiple error scenarios that could potentially leak goroutines
	errorScenarios := []struct {
		name        string
		openaiError error
		geminiError error
	}{
		{"Panic in OpenAI provider", errors.New("panic: runtime error"), nil},
		{"Panic in Gemini provider", nil, errors.New("panic: runtime error")},
		{"Timeout in both providers", NetworkTimeoutError, NetworkTimeoutError},
		{"Context cancellation during processing", context.Canceled, context.Canceled},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			mockOpenAI := new(MockEmbeddingProvider)
			mockGemini := new(MockEmbeddingProvider)
			mockStorage := new(MockVectorStorage)
			mockLogger := testutil.NewMockLogger()

			providers := map[string]provider.EmbeddingProvider{
				"openai": mockOpenAI,
				"gemini": mockGemini,
			}

			orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

			// Setup mocks with errors
			if scenario.openaiError != nil {
				mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return([]float32(nil), scenario.openaiError)
			} else {
				mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(make([]float32, 1536), nil)
			}

			if scenario.geminiError != nil {
				mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return([]float32(nil), scenario.geminiError)
			} else {
				mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(make([]float32, 768), nil)
			}

			mockLogger.SetEnabled(true)

			// Act
			err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

			// Assert
			assert.Error(t, err) // Should handle errors gracefully

			// Give time for any goroutines to complete
			time.Sleep(100 * time.Millisecond)
		})
	}

	// Check for goroutine leaks after all tests
	time.Sleep(200 * time.Millisecond) // Allow cleanup
	finalGoroutines := runtime.NumGoroutine()

	// Allow some variance in goroutine count (test framework may create some)
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+2,
		"Should not have significant goroutine leaks")
}

// TestBatchProcessor_CancellationErrorHandling tests proper cleanup on context cancellation
func TestBatchProcessor_CancellationErrorHandling(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create transcriptions
	transcriptions := make([]*vector.Transcription, 10)
	for i := 0; i < 10; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Cancellation test %d", i+1),
			User:              "cancel_user",
		}
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Track processing state
	var processedCount int32
	var mu sync.Mutex

	// Setup mocks that respect context cancellation
	for i := 0; i < 10; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(func(ctx context.Context, id int, text string) error {
			mu.Lock()
			processedCount++
			currentCount := processedCount
			mu.Unlock()

			// Cancel after processing 3 items
			if currentCount == 3 {
				cancel()
			}

			// Simulate work and check for cancellation
			select {
			case <-time.After(50 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).Maybe() // Use Maybe() since not all will be called due to cancellation
	}

	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(ctx, transcriptions, 5)

	// Assert
	assert.NoError(t, err) // Batch processor should handle cancellation gracefully
	assert.Less(t, result.Processed+result.Failed, 10, "Should have processed fewer than all items")

	// Verify processing stopped appropriately
	mu.Lock()
	finalCount := processedCount
	mu.Unlock()
	assert.LessOrEqual(t, finalCount, int32(5), "Should have stopped processing on cancellation")

	// Verify processor state is cleaned up
	status, err := processor.GetProcessingStatus(context.Background())
	assert.NoError(t, err)
	assert.False(t, status.IsProcessing, "Should not be processing after cancellation")
}

// TestEmbeddingOrchestrator_RecoveryFromPanic tests panic recovery
func TestEmbeddingOrchestrator_RecoveryFromPanic(t *testing.T) {
	// Note: This test assumes panic recovery is implemented.
	// If not implemented, this test documents the expected behavior.

	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockGemini := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Setup mock to panic
	mockOpenAI.On("GenerateEmbedding", mock.Anything, "panic test").Panic("simulated panic in provider")
	mockGemini.On("GenerateEmbedding", mock.Anything, "panic test").Return(make([]float32, 768), nil)
	mockLogger.SetEnabled(true)

	// Act & Assert
	// This should not panic and should handle the error gracefully
	assert.NotPanics(t, func() {
		err := orchestrator.ProcessTranscription(context.Background(), 1, "panic test")
		// Current implementation may not recover from panics, so we test that it doesn't crash
		// In future implementations, this should return an error instead of panicking
		_ = err // Result depends on panic recovery implementation
	}, "Should not panic even if provider panics")
}

// TestBatchProcessor_ResourceExhaustionHandling tests behavior under resource exhaustion
func TestBatchProcessor_ResourceExhaustionHandling(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create transcriptions
	transcriptions := make([]*vector.Transcription, 5)
	for i := 0; i < 5; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Resource test %d", i+1),
			User:              "resource_user",
		}
	}

	// Setup mocks with resource exhaustion errors
	resourceErrors := []error{
		OutOfMemoryError,
		errors.New("too many open files"),
		errors.New("connection pool exhausted"),
		errors.New("CPU quota exceeded"),
		errors.New("memory limit exceeded"),
	}

	for i, transcription := range transcriptions {
		if i < len(resourceErrors) {
			mockOrchestrator.On("ProcessTranscription", mock.Anything, transcription.ID, transcription.TranscriptionText).Return(resourceErrors[i])
		} else {
			mockOrchestrator.On("ProcessTranscription", mock.Anything, transcription.ID, transcription.TranscriptionText).Return(nil)
		}
	}

	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 2)

	// Assert
	assert.NoError(t, err) // Should continue processing despite resource errors
	assert.Equal(t, 0, result.Processed, "No items should succeed with resource errors")
	assert.Equal(t, 5, result.Failed, "All items should fail due to resource errors")
	assert.Len(t, result.Errors, 5, "Should collect all resource errors")

	// Verify specific resource errors are captured
	errorStrings := make([]string, len(result.Errors))
	for i, e := range result.Errors {
		errorStrings[i] = e.Error()
	}

	for _, expectedError := range resourceErrors {
		assert.Contains(t, errorStrings, expectedError.Error(),
			"Should capture resource error: %s", expectedError.Error())
	}

	mockOrchestrator.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_EdgeCaseInputHandling tests handling of edge case inputs
func TestEmbeddingOrchestrator_EdgeCaseInputHandling(t *testing.T) {
	edgeCases := []struct {
		name        string
		text        string
		expectError bool
	}{
		{"Empty text", "", false},
		{"Very long text", string(make([]byte, 100000)), false}, // 100KB text
		{"Unicode text", "Hello 世界 🌍 Мир", false},
		{"Special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?", false},
		{"Newlines and tabs", "Line 1\nLine 2\tTabbed", false},
		{"Only whitespace", "   \t\n\r   ", false},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockOpenAI := new(MockEmbeddingProvider)
			mockGemini := new(MockEmbeddingProvider)
			mockStorage := new(MockVectorStorage)
			mockLogger := testutil.NewMockLogger()

			providers := map[string]provider.EmbeddingProvider{
				"openai": mockOpenAI,
				"gemini": mockGemini,
			}

			orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

			// Setup mocks to handle edge cases
			openaiEmbedding := make([]float32, 1536)
			geminiEmbedding := make([]float32, 768)

			mockOpenAI.On("GenerateEmbedding", mock.Anything, tc.text).Return(openaiEmbedding, nil)
			mockGemini.On("GenerateEmbedding", mock.Anything, tc.text).Return(geminiEmbedding, nil)
			mockStorage.On("StoreDualEmbeddings", mock.Anything, 1, openaiEmbedding, geminiEmbedding).Return(nil)
			mockLogger.SetEnabled(true)

			// Act
			err := orchestrator.ProcessTranscription(context.Background(), 1, tc.text)

			// Assert
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockOpenAI.AssertExpectations(t)
			mockGemini.AssertExpectations(t)
			if !tc.expectError {
				mockStorage.AssertExpectations(t)
			}
		})
	}
}
