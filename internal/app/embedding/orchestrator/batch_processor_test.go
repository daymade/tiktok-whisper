package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/app/storage/vector"
)

// MockEmbeddingOrchestrator for testing
type MockEmbeddingOrchestrator struct {
	mock.Mock
}

func (m *MockEmbeddingOrchestrator) ProcessTranscription(ctx context.Context, transcriptionID int, text string) error {
	args := m.Called(ctx, transcriptionID, text)
	return args.Error(0)
}

func (m *MockEmbeddingOrchestrator) GetEmbeddingStatus(ctx context.Context, transcriptionID int) (*EmbeddingStatus, error) {
	args := m.Called(ctx, transcriptionID)
	return args.Get(0).(*EmbeddingStatus), args.Error(1)
}

// TDD Cycle 7: RED - Test BatchProcessor interface
func TestBatchProcessor(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)
	
	// Create test transcriptions
	transcriptions := []*vector.Transcription{
		{ID: 1, TranscriptionText: "First transcription", User: "user1"},
		{ID: 2, TranscriptionText: "Second transcription", User: "user1"},
		{ID: 3, TranscriptionText: "Third transcription", User: "user1"},
	}

	// Setup mocks
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 1, "First transcription").Return(nil)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 2, "Second transcription").Return(nil)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 3, "Third transcription").Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 2)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 3, result.Processed)
	assert.Equal(t, 0, result.Failed)
	mockOrchestrator.AssertExpectations(t)
}

// Test batch processing with errors
func TestBatchProcessorWithErrors(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)
	
	transcriptions := []*vector.Transcription{
		{ID: 1, TranscriptionText: "First transcription", User: "user1"},
		{ID: 2, TranscriptionText: "Second transcription", User: "user1"},
	}

	// Setup mocks - first succeeds, second fails
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 1, "First transcription").Return(nil)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 2, "Second transcription").Return(assert.AnError)
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 2)

	// Assert
	assert.NoError(t, err) // Batch processor should not fail completely
	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 1, result.Failed)
	assert.Len(t, result.Errors, 1)
	mockOrchestrator.AssertExpectations(t)
}

// Test processing all transcriptions
func TestBatchProcessorProcessAll(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)
	
	// Mock storage to return transcriptions without embeddings
	transcriptions := []*vector.Transcription{
		{ID: 1, TranscriptionText: "First transcription", User: "user1"},
		{ID: 2, TranscriptionText: "Second transcription", User: "user1"},
	}

	mockStorage.On("GetTranscriptionsWithoutEmbeddings", mock.Anything, "openai", mock.Anything).Return(transcriptions, nil)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 1, "First transcription").Return(nil)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 2, "Second transcription").Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	// Act
	err := processor.ProcessAllTranscriptions(context.Background(), 10)

	// Assert
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
	mockOrchestrator.AssertExpectations(t)
}

// Test processing with progress tracking
func TestBatchProcessorProgress(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)
	
	// Start processing in background
	go func() {
		transcriptions := []*vector.Transcription{
			{ID: 1, TranscriptionText: "First transcription", User: "user1"},
		}
		mockOrchestrator.On("ProcessTranscription", mock.Anything, 1, "First transcription").Return(nil)
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
		
		processor.ProcessBatch(context.Background(), transcriptions, 1)
	}()

	// Give some time for processing to start
	time.Sleep(10 * time.Millisecond)
	
	// Act
	status, err := processor.GetProcessingStatus(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, status)
}

// Test pause and resume functionality
func TestBatchProcessorPauseResume(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Act & Assert
	err := processor.PauseProcessing()
	assert.NoError(t, err)
	
	err = processor.ResumeProcessing()
	assert.NoError(t, err)
	
	err = processor.StopProcessing()
	assert.NoError(t, err)
}