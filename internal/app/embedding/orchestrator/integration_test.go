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
	"tiktok-whisper/internal/app/embedding/provider"
	"tiktok-whisper/internal/app/storage/vector"
	"tiktok-whisper/internal/app/testutil"
)

// =============================================================================
// INTEGRATION TESTS - Full System Testing
// =============================================================================

// TestEmbeddingOrchestrator_FullWorkflow tests the complete embedding workflow
func TestEmbeddingOrchestrator_FullWorkflow(t *testing.T) {
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
	batchProcessor := NewBatchProcessor(orchestrator, mockStorage, mockLogger)

	// Create test transcriptions
	transcriptions := []*vector.Transcription{
		{ID: 1, TranscriptionText: "First test transcription", User: "user1"},
		{ID: 2, TranscriptionText: "Second test transcription", User: "user1"},
		{ID: 3, TranscriptionText: "Third test transcription", User: "user1"},
	}

	// Setup mocks for full workflow
	openaiEmbedding := make([]float32, 1536)
	geminiEmbedding := make([]float32, 768)

	for _, transcription := range transcriptions {
		mockOpenAI.On("GenerateEmbedding", mock.Anything, transcription.TranscriptionText).Return(openaiEmbedding, nil)
		mockGemini.On("GenerateEmbedding", mock.Anything, transcription.TranscriptionText).Return(geminiEmbedding, nil)
		mockStorage.On("StoreDualEmbeddings", mock.Anything, transcription.ID, openaiEmbedding, geminiEmbedding).Return(nil)
	}

	mockStorage.On("GetTranscriptionsWithoutEmbeddings", mock.Anything, "openai", 0).Return(transcriptions, nil)
	mockLogger.SetEnabled(true)

	// Act
	err := batchProcessor.ProcessAllTranscriptions(context.Background(), []string{"openai", "gemini"}, 2)

	// Assert
	assert.NoError(t, err)

	// Verify all transcriptions were processed
	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)

	// Verify logging
	infoLogs := mockLogger.GetLogsByLevel(testutil.LogLevelInfo)
	assert.Greater(t, len(infoLogs), 0, "Should have logged processing information")
}

// TestEmbeddingOrchestrator_NetworkFailureRecovery tests network failure scenarios
func TestEmbeddingOrchestrator_NetworkFailureRecovery(t *testing.T) {
	tests := []struct {
		name           string
		openaiErrors   []error
		geminiErrors   []error
		expectedResult bool
		description    string
	}{
		{
			name:           "Intermittent OpenAI failures",
			openaiErrors:   []error{errors.New("network timeout"), nil, errors.New("rate limit"), nil},
			geminiErrors:   []error{nil, nil, nil, nil},
			expectedResult: false, // Should fail due to OpenAI errors
			description:    "OpenAI has intermittent failures while Gemini is stable",
		},
		{
			name:           "All providers eventually succeed",
			openaiErrors:   []error{nil, nil, nil, nil},
			geminiErrors:   []error{nil, nil, nil, nil},
			expectedResult: true,
			description:    "All providers work correctly",
		},
		{
			name:           "Persistent failures",
			openaiErrors:   []error{errors.New("service unavailable"), errors.New("service unavailable")},
			geminiErrors:   []error{errors.New("quota exceeded"), errors.New("quota exceeded")},
			expectedResult: false,
			description:    "Both providers fail consistently",
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

			// Setup mocks with error patterns
			openaiEmbedding := make([]float32, 1536)
			geminiEmbedding := make([]float32, 768)

			for i, openaiErr := range tt.openaiErrors {
				text := fmt.Sprintf("test text %d", i+1)
				if openaiErr != nil {
					mockOpenAI.On("GenerateEmbedding", mock.Anything, text).Return([]float32(nil), openaiErr)
				} else {
					mockOpenAI.On("GenerateEmbedding", mock.Anything, text).Return(openaiEmbedding, nil)
				}
			}

			for i, geminiErr := range tt.geminiErrors {
				text := fmt.Sprintf("test text %d", i+1)
				if geminiErr != nil {
					mockGemini.On("GenerateEmbedding", mock.Anything, text).Return([]float32(nil), geminiErr)
				} else {
					mockGemini.On("GenerateEmbedding", mock.Anything, text).Return(geminiEmbedding, nil)
				}
			}

			// Setup storage for successful cases
			if tt.expectedResult {
				for i := range tt.openaiErrors {
					if tt.openaiErrors[i] == nil && tt.geminiErrors[i] == nil {
						mockStorage.On("StoreDualEmbeddings", mock.Anything, i+1, openaiEmbedding, geminiEmbedding).Return(nil)
					}
				}
			}

			// Setup error logging
			mockLogger.SetEnabled(true)

			// Act & Assert
			for i := range tt.openaiErrors {
				text := fmt.Sprintf("test text %d", i+1)
				err := orchestrator.ProcessTranscription(context.Background(), i+1, text)

				if tt.expectedResult && tt.openaiErrors[i] == nil && tt.geminiErrors[i] == nil {
					assert.NoError(t, err, "Should succeed when both providers work")
				} else if tt.openaiErrors[i] != nil || tt.geminiErrors[i] != nil {
					assert.Error(t, err, "Should fail when any provider fails")
				}
			}

			// Verify error logging occurred for failures
			if !tt.expectedResult {
				errorLogs := mockLogger.GetLogsByLevel(testutil.LogLevelError)
				assert.Greater(t, len(errorLogs), 0, "Should have logged errors for failures")
			}
		})
	}
}

// TestEmbeddingOrchestrator_DatabaseConnectivityIssues tests database connectivity scenarios
func TestEmbeddingOrchestrator_DatabaseConnectivityIssues(t *testing.T) {
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

	// Setup storage failures
	storageErrors := []error{
		errors.New("connection pool exhausted"),
		errors.New("database timeout"),
		errors.New("deadlock detected"),
		errors.New("disk full"),
	}

	for _, storageErr := range storageErrors {
		mockStorage.On("StoreDualEmbeddings", mock.Anything, mock.AnythingOfType("int"), openaiEmbedding, geminiEmbedding).Return(storageErr).Once()
	}

	mockLogger.SetEnabled(true)

	// Act & Assert
	for i, expectedErr := range storageErrors {
		err := orchestrator.ProcessTranscription(context.Background(), i+1, "test text")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to store dual embeddings")
		assert.Contains(t, err.Error(), expectedErr.Error())
	}

	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// TestBatchProcessor_LargeScaleProcessing tests processing at scale
func TestBatchProcessor_LargeScaleProcessing(t *testing.T) {
	// Skip in short mode as this is a longer-running test
	if testing.Short() {
		t.Skip("Skipping large scale test in short mode")
	}

	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create large number of transcriptions
	const totalTranscriptions = 1000
	transcriptions := make([]*vector.Transcription, totalTranscriptions)
	for i := 0; i < totalTranscriptions; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Large scale transcription %d content", i+1),
			User:              "stress_test_user",
		}
	}

	// Setup mocks for all transcriptions
	for i := 0; i < totalTranscriptions; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil)
	}
	mockLogger.SetEnabled(true)

	// Act
	start := time.Now()
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 50)
	duration := time.Since(start)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, totalTranscriptions, result.Processed)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Errors)

	// Performance assertion - should complete within reasonable time
	assert.Less(t, duration, 30*time.Second, "Large scale processing should complete efficiently")

	// Verify progress logging
	infoLogs := mockLogger.GetLogsByLevel(testutil.LogLevelInfo)
	assert.Greater(t, len(infoLogs), 10, "Should have logged progress multiple times")

	mockOrchestrator.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_MemoryPressure tests behavior under memory pressure
func TestEmbeddingOrchestrator_MemoryPressure(t *testing.T) {
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

	// Create large embeddings to simulate memory pressure
	largeOpenAIEmbedding := make([]float32, 10000) // Larger than typical
	largeGeminiEmbedding := make([]float32, 10000) // Larger than typical

	// Fill with data to ensure memory allocation
	for i := range largeOpenAIEmbedding {
		largeOpenAIEmbedding[i] = float32(i) * 0.001
	}
	for i := range largeGeminiEmbedding {
		largeGeminiEmbedding[i] = float32(i) * 0.002
	}

	// Setup mocks
	mockOpenAI.On("GenerateEmbedding", mock.Anything, "large text").Return(largeOpenAIEmbedding, nil)
	mockGemini.On("GenerateEmbedding", mock.Anything, "large text").Return(largeGeminiEmbedding, nil)
	mockStorage.On("StoreDualEmbeddings", mock.Anything, 1, largeOpenAIEmbedding, largeGeminiEmbedding).Return(nil)
	mockLogger.SetEnabled(true)

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "large text")

	// Assert
	assert.NoError(t, err)
	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)

	// Verify successful completion with large embeddings
	successLogs := mockLogger.GetLogsByLevel(testutil.LogLevelInfo)
	found := false
	for _, log := range successLogs {
		if log.Message == "Successfully processed dual embeddings" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should have logged successful processing")
}

// TestBatchProcessor_ConcurrencyLimits tests concurrency behavior under load
func TestBatchProcessor_ConcurrencyLimits(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create transcriptions
	const numTranscriptions = 20
	transcriptions := make([]*vector.Transcription, numTranscriptions)
	for i := 0; i < numTranscriptions; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Concurrency test %d", i+1),
			User:              "concurrency_user",
		}
	}

	// Track maximum concurrent executions
	var activeCalls int32
	var maxConcurrent int32
	var mu sync.Mutex

	// Setup mocks with concurrency tracking
	for i := 0; i < numTranscriptions; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(func(ctx context.Context, id int, text string) error {
			mu.Lock()
			activeCalls++
			if activeCalls > maxConcurrent {
				maxConcurrent = activeCalls
			}
			mu.Unlock()

			// Simulate processing time
			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			activeCalls--
			mu.Unlock()

			return nil
		})
	}
	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 10)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, numTranscriptions, result.Processed)

	// Verify reasonable concurrency limits (should be limited by batch size)
	mu.Lock()
	assert.Greater(t, maxConcurrent, int32(1), "Should have concurrent processing")
	assert.LessOrEqual(t, maxConcurrent, int32(10), "Should respect batch size limits")
	mu.Unlock()

	mockOrchestrator.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_ProviderFailover tests provider failover scenarios
func TestEmbeddingOrchestrator_ProviderFailover(t *testing.T) {
	// Note: Current implementation doesn't support failover, but this test
	// documents the expected behavior and can be updated when failover is implemented

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

	// Setup one provider to fail completely
	mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return([]float32(nil), errors.New("provider unavailable"))
	mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(make([]float32, 768), nil)
	mockLogger.SetEnabled(true)

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

	// Assert - Currently expected to fail as no failover is implemented
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding generation failed")

	// Verify error logging
	errorLogs := mockLogger.GetLogsByLevel(testutil.LogLevelError)
	assert.Greater(t, len(errorLogs), 0, "Should have logged provider failures")

	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
}

// TestBatchProcessor_ResourceCleanup tests proper resource cleanup
func TestBatchProcessor_ResourceCleanup(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create transcriptions
	transcriptions := []*vector.Transcription{
		{ID: 1, TranscriptionText: "Cleanup test 1", User: "cleanup_user"},
		{ID: 2, TranscriptionText: "Cleanup test 2", User: "cleanup_user"},
	}

	// Setup mocks
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 1, "Cleanup test 1").Return(nil)
	mockOrchestrator.On("ProcessTranscription", mock.Anything, 2, "Cleanup test 2").Return(nil)
	mockLogger.SetEnabled(true)

	// Act
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 2)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Processed)

	// Verify processing state is cleaned up
	status, err := processor.GetProcessingStatus(context.Background())
	assert.NoError(t, err)
	assert.False(t, status.IsProcessing, "Should not be processing after completion")
	assert.False(t, status.IsPaused, "Should not be paused after completion")

	mockOrchestrator.AssertExpectations(t)
}

// Helper function for creating mock embeddings
func createMockEmbedding(size int, seed float32) []float32 {
	embedding := make([]float32, size)
	for i := range embedding {
		embedding[i] = seed + float32(i)*0.001
	}
	return embedding
}

// Benchmark tests for performance validation
func BenchmarkEmbeddingOrchestrator_DualProvider(b *testing.B) {
	// Setup
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
	openaiEmbedding := createMockEmbedding(1536, 0.1)
	geminiEmbedding := createMockEmbedding(768, 0.2)

	mockOpenAI.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(openaiEmbedding, nil)
	mockGemini.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(geminiEmbedding, nil)
	mockStorage.On("StoreDualEmbeddings", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockLogger.SetEnabled(false) // Disable logging for benchmark

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := orchestrator.ProcessTranscription(context.Background(), i, "benchmark text")
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkBatchProcessor_LargeBatch(b *testing.B) {
	// Setup
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create transcriptions
	transcriptions := make([]*vector.Transcription, 100)
	for i := 0; i < 100; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Benchmark transcription %d", i+1),
			User:              "benchmark_user",
		}
	}

	// Setup mocks
	for i := 0; i < 100; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil)
	}
	mockLogger.SetEnabled(false)

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessBatch(context.Background(), transcriptions, 10)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
