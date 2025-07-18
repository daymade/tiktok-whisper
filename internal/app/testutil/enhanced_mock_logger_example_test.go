package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrchestratorInterface demonstrates how to use EnhancedMockLogger with orchestrator interfaces
type MockOrchestratorInterface struct {
	mock.Mock
}

func (m *MockOrchestratorInterface) ProcessTranscription(ctx context.Context, transcriptionID int, text string) error {
	args := m.Called(ctx, transcriptionID, text)
	return args.Error(0)
}

// ExampleTestEmbeddingOrchestrator demonstrates using EnhancedMockLogger with orchestrator testing
func ExampleTestEmbeddingOrchestrator() {
	// This example shows how to use EnhancedMockLogger to test embedding orchestrator behavior
	logger := NewEnhancedMockLogger().WithMockingEnabled(true)
	
	// Set up expectations for successful processing
	logger.ExpectInfo("Processing transcription", "transcriptionID", 123, "provider", "openai")
	logger.ExpectInfo("Successfully processed embedding", "transcriptionID", 123, "provider", "openai")
	
	// Your orchestrator code would log these messages
	logger.Info("Processing transcription", "transcriptionID", 123, "provider", "openai")
	logger.Info("Successfully processed embedding", "transcriptionID", 123, "provider", "openai")
	
	// Verify expectations were met
	logger.AssertExpectations(nil) // would pass *testing.T in real usage
	
	// Verify no errors occurred
	if logger.HasError() {
		panic("Unexpected errors occurred")
	}
	
	// Verify structured data was captured
	if !logger.HasMessageWithField("transcriptionID", 123) {
		panic("Expected transcriptionID 123 to be logged")
	}
	
	// Output:
}

// TestEnhancedMockLoggerWithOrchestrator demonstrates real-world usage with orchestrator pattern
func TestEnhancedMockLoggerWithOrchestrator(t *testing.T) {
	// Create enhanced mock logger
	logger := NewEnhancedMockLogger().WithMockingEnabled(true)
	
	// Create mock orchestrator
	orchestrator := &MockOrchestratorInterface{}
	
	// Set up expectations
	logger.ExpectInfo("Starting batch processing", "totalTranscriptions", 100, "batchSize", 10)
	logger.ExpectInfo("Batch processing progress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	logger.ExpectInfo("Batch processing completed", "processed", 100, "failed", 0)
	
	orchestrator.On("ProcessTranscription", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	// Simulate batch processing
	logger.Info("Starting batch processing", "totalTranscriptions", 100, "batchSize", 10)
	
	// Process a single transcription (in real code, this would be in a loop)
	err := orchestrator.ProcessTranscription(context.Background(), 123, "test transcription")
	assert.NoError(t, err)
	
	// Log progress
	logger.Info("Batch processing progress", "progress", 50.0, "processed", 50, "failed", 0)
	
	// Complete processing
	logger.Info("Batch processing completed", "processed", 100, "failed", 0)
	
	// Verify expectations
	logger.AssertExpectations(t)
	orchestrator.AssertExpectations(t)
	
	// Verify structured data
	assert.True(t, logger.HasMessageWithField("totalTranscriptions", 100))
	assert.True(t, logger.HasMessageWithField("batchSize", 10))
	assert.True(t, logger.HasMessageWithField("processed", 100))
	assert.True(t, logger.HasMessageWithField("failed", 0))
	
	// Verify no errors occurred
	assert.False(t, logger.HasError())
	
	// Verify progress was logged
	progressMessages := logger.FindMessages("progress")
	assert.Len(t, progressMessages, 1)
}

// TestEnhancedMockLoggerErrorHandling demonstrates error handling patterns
func TestEnhancedMockLoggerErrorHandling(t *testing.T) {
	logger := NewEnhancedMockLogger().WithMockingEnabled(true)
	
	// Set up expectations for error scenarios
	logger.ExpectInfo("Processing transcription", "transcriptionID", 123)
	logger.ExpectError("Failed to process transcription", "transcriptionID", 123, "error", mock.AnythingOfType("string"))
	
	// Simulate processing with error
	logger.Info("Processing transcription", "transcriptionID", 123)
	logger.Error("Failed to process transcription", "transcriptionID", 123, "error", "network timeout")
	
	// Verify expectations
	logger.AssertExpectations(t)
	
	// Verify error was logged
	assert.True(t, logger.HasError())
	
	// Verify error details
	errorMessages := logger.GetErrorMessages()
	assert.Len(t, errorMessages, 1)
	assert.Contains(t, errorMessages[0].Message, "Failed to process")
	
	// Verify structured error data
	assert.True(t, logger.HasMessageWithField("transcriptionID", 123))
	assert.True(t, logger.HasMessageWithField("error", "network timeout"))
	
	// Find error messages with specific transcription ID
	errorMessagesForTranscription := logger.FindMessagesByField("transcriptionID", 123)
	errorCount := 0
	for _, msg := range errorMessagesForTranscription {
		if msg.Level == LogLevelError {
			errorCount++
		}
	}
	assert.Equal(t, 1, errorCount)
}

// TestEnhancedMockLoggerDualEmbeddingScenario demonstrates dual embedding testing
func TestEnhancedMockLoggerDualEmbeddingScenario(t *testing.T) {
	logger := NewEnhancedMockLogger().WithMockingEnabled(true)
	
	// Set up expectations for dual embedding processing
	logger.ExpectInfo("Processing dual embeddings", "transcriptionID", 123)
	logger.ExpectInfo("OpenAI embedding generated", "transcriptionID", 123, "dimensions", 1536)
	logger.ExpectInfo("Gemini embedding generated", "transcriptionID", 123, "dimensions", 768)
	logger.ExpectInfo("Successfully processed dual embeddings", "transcriptionID", 123)
	
	// Simulate dual embedding processing
	logger.Info("Processing dual embeddings", "transcriptionID", 123)
	logger.Info("OpenAI embedding generated", "transcriptionID", 123, "dimensions", 1536)
	logger.Info("Gemini embedding generated", "transcriptionID", 123, "dimensions", 768)
	logger.Info("Successfully processed dual embeddings", "transcriptionID", 123)
	
	// Verify expectations
	logger.AssertExpectations(t)
	
	// Verify no errors occurred
	assert.False(t, logger.HasError())
	
	// Verify dual embedding workflow
	messages := logger.GetEnhancedMessages()
	assert.Len(t, messages, 4)
	
	// Verify specific embedding dimensions were logged
	assert.True(t, logger.HasMessageWithField("dimensions", 1536))
	assert.True(t, logger.HasMessageWithField("dimensions", 768))
	
	// Verify all messages are for the same transcription
	transcriptionMessages := logger.FindMessagesByField("transcriptionID", 123)
	assert.Len(t, transcriptionMessages, 4)
	
	// Verify workflow progression
	assert.Contains(t, messages[0].Message, "Processing dual embeddings")
	assert.Contains(t, messages[1].Message, "OpenAI embedding generated")
	assert.Contains(t, messages[2].Message, "Gemini embedding generated")
	assert.Contains(t, messages[3].Message, "Successfully processed dual embeddings")
}

// TestEnhancedMockLoggerFlexibleMocking demonstrates flexible mocking patterns
func TestEnhancedMockLoggerFlexibleMocking(t *testing.T) {
	logger := NewEnhancedMockLogger().WithMockingEnabled(true)
	
	// Set up flexible expectations
	logger.ExpectAnyInfo().Times(3)
	// Don't set error expectations - we expect 0 errors
	
	// Log various info messages
	logger.Info("Starting process")
	logger.Info("Processing item", "id", 1)
	logger.Info("Process complete", "total", 1)
	
	// Verify expectations
	logger.AssertExpectations(t)
	
	// Verify no errors occurred
	assert.False(t, logger.HasError())
	
	// Verify message count
	assert.Equal(t, 3, len(logger.GetInfoMessages()))
	assert.Equal(t, 0, len(logger.GetErrorMessages()))
}

// TestEnhancedMockLoggerBasicUsageWithoutMocking demonstrates basic usage without testify/mock
func TestEnhancedMockLoggerBasicUsageWithoutMocking(t *testing.T) {
	// Create logger without mocking enabled (default behavior)
	logger := NewEnhancedMockLogger()
	
	// Log some messages (no expectations needed)
	logger.Info("Processing started", "batchSize", 10)
	logger.Info("Processing progress", "completed", 5, "remaining", 5)
	logger.Info("Processing completed", "total", 10)
	
	// Verify logging behavior
	assert.Equal(t, 3, len(logger.GetInfoMessages()))
	assert.False(t, logger.HasError())
	
	// Verify structured data
	assert.True(t, logger.HasMessageWithField("batchSize", 10))
	assert.True(t, logger.HasMessageWithField("total", 10))
	
	// Verify progress tracking
	progressMessages := logger.FindMessages("progress")
	assert.Len(t, progressMessages, 1)
	
	// No need to assert expectations since mocking is disabled
}