package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/app/storage/vector"
	"tiktok-whisper/internal/app/testutil"
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

// =============================================================================
// COMPREHENSIVE BATCH PROCESSOR TEST SUITE - Enhanced Coverage
// =============================================================================

// TestBatchProcessor_LargeBatchProcessing tests processing of large batches
func TestBatchProcessor_LargeBatchProcessing(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create large batch of transcriptions
	transcriptions := make([]*vector.Transcription, 100)
	for i := 0; i < 100; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Transcription %d content", i+1),
			User:              "test_user",
		}
	}

	// Setup mocks for all transcriptions
	for i := 0; i < 100; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil)
	}
	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 10)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, result.Processed)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Errors)

	// Verify progress logging occurred
	infoLogs := mockLogger.GetLogsByLevel(testutil.LogLevelInfo)
	assert.Greater(t, len(infoLogs), 0, "Should have logged progress")

	mockOrchestrator.AssertExpectations(t)
}

// TestBatchProcessor_ConcurrentProcessing tests concurrent processing within batches
func TestBatchProcessor_ConcurrentProcessing(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Track concurrent execution
	var activeCount int32
	var maxConcurrent int32
	var mu sync.Mutex

	// Create transcriptions
	transcriptions := make([]*vector.Transcription, 5)
	for i := 0; i < 5; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Transcription %d", i+1),
			User:              "test_user",
		}
	}

	// Setup mocks with delay to observe concurrency
	for i := 0; i < 5; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			mu.Lock()
			activeCount++
			if activeCount > maxConcurrent {
				maxConcurrent = activeCount
			}
			mu.Unlock()

			time.Sleep(50 * time.Millisecond) // Simulate processing time

			mu.Lock()
			activeCount--
			mu.Unlock()
		})
	}
	mockLogger.SetEnabled(true)

	// Act
	start := time.Now()
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 5)
	duration := time.Since(start)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 5, result.Processed)
	assert.Equal(t, 0, result.Failed)

	// Verify concurrent processing (should be faster than sequential)
	assert.Less(t, duration, 200*time.Millisecond, "Should process concurrently")

	// Verify maximum concurrency was achieved
	mu.Lock()
	assert.Greater(t, maxConcurrent, int32(1), "Should have concurrent processing")
	mu.Unlock()

	mockOrchestrator.AssertExpectations(t)
}

// TestBatchProcessor_PartialFailures tests handling of partial failures in batches
func TestBatchProcessor_PartialFailures(t *testing.T) {
	tests := []struct {
		name              string
		totalTranscriptions int
		failureIndices    []int
		expectedProcessed int
		expectedFailed    int
	}{
		{
			name:              "Single failure in middle",
			totalTranscriptions: 5,
			failureIndices:    []int{2},
			expectedProcessed: 4,
			expectedFailed:    1,
		},
		{
			name:              "Multiple failures",
			totalTranscriptions: 10,
			failureIndices:    []int{1, 3, 7},
			expectedProcessed: 7,
			expectedFailed:    3,
		},
		{
			name:              "All failures",
			totalTranscriptions: 3,
			failureIndices:    []int{0, 1, 2},
			expectedProcessed: 0,
			expectedFailed:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockOrchestrator := new(MockEmbeddingOrchestrator)
			mockStorage := new(MockVectorStorage)
			mockLogger := testutil.NewMockLogger()

			processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

			// Create transcriptions
			transcriptions := make([]*vector.Transcription, tt.totalTranscriptions)
			for i := 0; i < tt.totalTranscriptions; i++ {
				transcriptions[i] = &vector.Transcription{
					ID:                i + 1,
					TranscriptionText: fmt.Sprintf("Transcription %d", i+1),
					User:              "test_user",
				}
			}

			// Setup mocks
			failureSet := make(map[int]bool)
			for _, idx := range tt.failureIndices {
				failureSet[idx] = true
			}

			for i := 0; i < tt.totalTranscriptions; i++ {
				if failureSet[i] {
					mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(errors.New("processing failed"))
				} else {
					mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil)
				}
			}
			mockLogger.SetEnabled(true)

			// Act
			result, err := processor.ProcessBatch(context.Background(), transcriptions, 5)

			// Assert
			assert.NoError(t, err) // Batch processing should continue despite failures
			assert.Equal(t, tt.expectedProcessed, result.Processed)
			assert.Equal(t, tt.expectedFailed, result.Failed)
			assert.Len(t, result.Errors, tt.expectedFailed)

			mockOrchestrator.AssertExpectations(t)
		})
	}
}

// TestBatchProcessor_ProcessAllTranscriptions_StorageFailure tests storage failures in ProcessAllTranscriptions
func TestBatchProcessor_ProcessAllTranscriptions_StorageFailure(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Setup storage to return error
	storageError := errors.New("database connection failed")
	mockStorage.On("GetTranscriptionsWithoutEmbeddings", mock.Anything, "openai", 0).Return(([]*vector.Transcription)(nil), storageError)

	// Act
	err := processor.ProcessAllTranscriptions(context.Background(), 10)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, storageError, err)
	mockStorage.AssertExpectations(t)
}

// TestBatchProcessor_ProcessAllTranscriptions_NoTranscriptions tests behavior with no transcriptions to process
func TestBatchProcessor_ProcessAllTranscriptions_NoTranscriptions(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Setup storage to return empty list
	emptyTranscriptions := []*vector.Transcription{}
	mockStorage.On("GetTranscriptionsWithoutEmbeddings", mock.Anything, "openai", 0).Return(emptyTranscriptions, nil)
	mockLogger.SetEnabled(true)

	// Act
	err := processor.ProcessAllTranscriptions(context.Background(), 10)

	// Assert
	assert.NoError(t, err)

	// Verify logging of no transcriptions to process
	assert.True(t, mockLogger.ContainsMessage("No transcriptions to process"))

	mockStorage.AssertExpectations(t)
}

// TestBatchProcessor_ContextCancellation tests context cancellation during batch processing
func TestBatchProcessor_ContextCancellation(t *testing.T) {
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
			TranscriptionText: fmt.Sprintf("Transcription %d", i+1),
			User:              "test_user",
		}
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Setup mocks with delay and context checking
	processedCount := 0
	for i := 0; i < 10; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			// Simulate processing time
			time.Sleep(10 * time.Millisecond)
			processedCount++
			
			// Cancel after processing a few items
			if processedCount == 3 {
				cancel()
			}
		})
	}
	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(ctx, transcriptions, 5)

	// Assert
	assert.NoError(t, err) // Should complete gracefully when context is cancelled
	assert.Less(t, result.Processed, 10, "Should have processed fewer items due to cancellation")

	mockOrchestrator.AssertExpectations(t)
}

// TestBatchProcessor_PauseResumeIntegration tests pause/resume functionality during actual processing
func TestBatchProcessor_PauseResumeIntegration(t *testing.T) {
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
			TranscriptionText: fmt.Sprintf("Transcription %d", i+1),
			User:              "test_user",
		}
	}

	// Track processing state
	var processedOrder []int
	var mu sync.Mutex

	// Setup mocks
	for i := 0; i < 5; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			id := args.Get(1).(int)
			mu.Lock()
			processedOrder = append(processedOrder, id)
			
			// Pause after processing first item
			if id == 1 {
				go func() {
					time.Sleep(10 * time.Millisecond)
					processor.PauseProcessing()
					
					// Resume after a delay
					time.Sleep(50 * time.Millisecond)
					processor.ResumeProcessing()
				}()
			}
			mu.Unlock()
			
			time.Sleep(20 * time.Millisecond) // Simulate processing time
		})
	}
	mockLogger.SetEnabled(true)

	// Act
	start := time.Now()
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 2)
	duration := time.Since(start)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 5, result.Processed)
	assert.Equal(t, 0, result.Failed)

	// Verify pause caused delay (should be longer due to pause)
	assert.Greater(t, duration, 100*time.Millisecond, "Should have been delayed by pause")

	mockOrchestrator.AssertExpectations(t)
}

// TestBatchProcessor_ProgressTracking tests progress tracking accuracy
func TestBatchProcessor_ProgressTracking(t *testing.T) {
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
			TranscriptionText: fmt.Sprintf("Transcription %d", i+1),
			User:              "test_user",
		}
	}

	// Track progress changes
	var progressSnapshots []float64
	var mu sync.Mutex

	// Setup mocks with progress monitoring
	for i := 0; i < 10; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			// Capture progress during processing
			go func() {
				time.Sleep(5 * time.Millisecond)
				status, _ := processor.GetProcessingStatus(context.Background())
				mu.Lock()
				progressSnapshots = append(progressSnapshots, status.Progress)
				mu.Unlock()
			}()
			
			time.Sleep(10 * time.Millisecond)
		})
	}
	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 3)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 10, result.Processed)

	// Verify progress tracking
	mu.Lock()
	assert.Greater(t, len(progressSnapshots), 0, "Should have captured progress snapshots")

	// Progress should generally increase (allowing for some race conditions in testing)
	if len(progressSnapshots) > 1 {
		finalProgress := progressSnapshots[len(progressSnapshots)-1]
		assert.GreaterOrEqual(t, finalProgress, 0.0, "Progress should be non-negative")
		assert.LessOrEqual(t, finalProgress, 100.0, "Progress should not exceed 100%")
	}
	mu.Unlock()

	mockOrchestrator.AssertExpectations(t)
}

// TestBatchProcessor_StopProcessing tests stop functionality during processing
func TestBatchProcessor_StopProcessing(t *testing.T) {
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
			TranscriptionText: fmt.Sprintf("Transcription %d", i+1),
			User:              "test_user",
		}
	}

	// Setup mocks - only some will be called due to stop
	processedCount := 0
	for i := 0; i < 10; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			processedCount++
			
			// Stop processing after a few items
			if processedCount == 3 {
				go func() {
					time.Sleep(5 * time.Millisecond)
					processor.StopProcessing()
				}()
			}
			
			time.Sleep(20 * time.Millisecond)
		}).Maybe() // Use Maybe() since not all calls will happen due to stop
	}
	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 5)

	// Assert
	assert.NoError(t, err) // Should complete gracefully when stopped
	assert.Less(t, result.Processed, 10, "Should have processed fewer items due to stop")
	assert.Greater(t, result.Processed, 0, "Should have processed some items before stopping")

	// Don't assert exact expectations since stop interrupts processing
	// mockOrchestrator.AssertExpectations(t)
}

// TestBatchProcessor_BatchSizeHandling tests different batch sizes
func TestBatchProcessor_BatchSizeHandling(t *testing.T) {
	tests := []struct {
		name          string
		totalItems    int
		batchSize     int
		expectedBatches int
	}{
		{
			name:          "Exact batch division",
			totalItems:    10,
			batchSize:     5,
			expectedBatches: 2,
		},
		{
			name:          "Partial last batch",
			totalItems:    7,
			batchSize:     3,
			expectedBatches: 3,
		},
		{
			name:          "Single item batches",
			totalItems:    5,
			batchSize:     1,
			expectedBatches: 5,
		},
		{
			name:          "Batch size larger than total",
			totalItems:    3,
			batchSize:     10,
			expectedBatches: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockOrchestrator := new(MockEmbeddingOrchestrator)
			mockStorage := new(MockVectorStorage)
			mockLogger := testutil.NewMockLogger()

			processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

			// Create transcriptions
			transcriptions := make([]*vector.Transcription, tt.totalItems)
			for i := 0; i < tt.totalItems; i++ {
				transcriptions[i] = &vector.Transcription{
					ID:                i + 1,
					TranscriptionText: fmt.Sprintf("Transcription %d", i+1),
					User:              "test_user",
				}
			}

			// Setup mocks
			for i := 0; i < tt.totalItems; i++ {
				mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil)
			}
			mockLogger.SetEnabled(true)

			// Act
			result, err := processor.ProcessBatch(context.Background(), transcriptions, tt.batchSize)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.totalItems, result.Processed)
			assert.Equal(t, 0, result.Failed)

			// Verify batch count by checking final status
			status, err := processor.GetProcessingStatus(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBatches, status.TotalBatches)

			mockOrchestrator.AssertExpectations(t)
		})
	}
}