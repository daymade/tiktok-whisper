package orchestrator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/app/embedding/provider"
	"tiktok-whisper/internal/app/storage/vector"
	"tiktok-whisper/internal/app/testutil"
)

// MockEmbeddingProvider for testing
type MockEmbeddingProvider struct {
	mock.Mock
}

func (m *MockEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float32), args.Error(1)
}

func (m *MockEmbeddingProvider) GetProviderInfo() provider.ProviderInfo {
	args := m.Called()
	return args.Get(0).(provider.ProviderInfo)
}

// MockVectorStorage for testing
type MockVectorStorage struct {
	mock.Mock
}

func (m *MockVectorStorage) StoreEmbedding(ctx context.Context, transcriptionID int, provider string, embedding []float32) error {
	args := m.Called(ctx, transcriptionID, provider, embedding)
	return args.Error(0)
}

func (m *MockVectorStorage) GetEmbedding(ctx context.Context, transcriptionID int, provider string) ([]float32, error) {
	args := m.Called(ctx, transcriptionID, provider)
	return args.Get(0).([]float32), args.Error(1)
}

func (m *MockVectorStorage) StoreDualEmbeddings(ctx context.Context, transcriptionID int, openaiEmbedding, geminiEmbedding []float32) error {
	args := m.Called(ctx, transcriptionID, openaiEmbedding, geminiEmbedding)
	return args.Error(0)
}

func (m *MockVectorStorage) GetDualEmbeddings(ctx context.Context, transcriptionID int) (*vector.DualEmbedding, error) {
	args := m.Called(ctx, transcriptionID)
	return args.Get(0).(*vector.DualEmbedding), args.Error(1)
}

func (m *MockVectorStorage) GetTranscriptionsWithoutEmbeddings(ctx context.Context, provider string, limit int) ([]*vector.Transcription, error) {
	args := m.Called(ctx, provider, limit)
	return args.Get(0).([]*vector.Transcription), args.Error(1)
}

func (m *MockVectorStorage) GetTranscriptionsWithoutEmbeddingsByUser(ctx context.Context, provider string, user string, limit int) ([]*vector.Transcription, error) {
	args := m.Called(ctx, provider, user, limit)
	return args.Get(0).([]*vector.Transcription), args.Error(1)
}

func (m *MockVectorStorage) GetUserEmbeddingStats(ctx context.Context, userNickname string) (*vector.UserEmbeddingStats, error) {
	args := m.Called(ctx, userNickname)
	return args.Get(0).(*vector.UserEmbeddingStats), args.Error(1)
}

func (m *MockVectorStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockLogger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// TDD Cycle 6: RED - Test EmbeddingOrchestrator interface
func TestEmbeddingOrchestrator(t *testing.T) {
	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockGemini := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Setup mocks
	openaiEmbedding := make([]float32, 1536)
	geminiEmbedding := make([]float32, 768)

	mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(openaiEmbedding, nil)
	mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(geminiEmbedding, nil)
	mockStorage.On("StoreDualEmbeddings", mock.Anything, 1, openaiEmbedding, geminiEmbedding).Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

	// Assert
	assert.NoError(t, err)
	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// Test single provider processing
func TestEmbeddingOrchestratorSingleProvider(t *testing.T) {
	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Setup mocks
	embedding := make([]float32, 1536)

	mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(embedding, nil)
	mockStorage.On("StoreEmbedding", mock.Anything, 1, "openai", embedding).Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

	// Assert
	assert.NoError(t, err)
	mockOpenAI.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// Test error handling
func TestEmbeddingOrchestratorErrorHandling(t *testing.T) {
	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Setup mocks for error case
	mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return([]float32(nil), assert.AnError)
	mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

	// Assert
	assert.Error(t, err)
	mockOpenAI.AssertExpectations(t)
}

// Test embedding status retrieval
func TestEmbeddingOrchestratorGetStatus(t *testing.T) {
	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockGemini := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Setup mocks
	dualEmbedding := &vector.DualEmbedding{
		OpenAI: make([]float32, 1536),
		Gemini: make([]float32, 768),
	}

	mockStorage.On("GetDualEmbeddings", mock.Anything, 1).Return(dualEmbedding, nil)

	// Act
	status, err := orchestrator.GetEmbeddingStatus(context.Background(), 1)

	// Assert
	assert.NoError(t, err)
	assert.True(t, status.OpenAICompleted)
	assert.True(t, status.GeminiCompleted)
	mockStorage.AssertExpectations(t)
}

// =============================================================================
// COMPREHENSIVE TEST SUITE - Enhanced Coverage
// =============================================================================

// TestEmbeddingOrchestrator_DualProviderPartialFailure tests when one provider fails but the other succeeds
func TestEmbeddingOrchestrator_DualProviderPartialFailure(t *testing.T) {
	tests := []struct {
		name           string
		openaiError    error
		geminiError    error
		expectedError  bool
		errorSubstring string
	}{
		{
			name:           "OpenAI fails, Gemini succeeds",
			openaiError:    errors.New("OpenAI API error"),
			geminiError:    nil,
			expectedError:  true,
			errorSubstring: "embedding generation failed",
		},
		{
			name:           "OpenAI succeeds, Gemini fails",
			openaiError:    nil,
			geminiError:    errors.New("Gemini API error"),
			expectedError:  true,
			errorSubstring: "embedding generation failed",
		},
		{
			name:           "Both providers fail",
			openaiError:    errors.New("OpenAI API error"),
			geminiError:    errors.New("Gemini API error"),
			expectedError:  true,
			errorSubstring: "embedding generation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockOpenAI := new(MockEmbeddingProvider)
			mockGemini := new(MockEmbeddingProvider)
			mockStorage := new(MockVectorStorage)
			mockLogger := new(MockLogger)

			providers := map[string]provider.EmbeddingProvider{
				"openai": mockOpenAI,
				"gemini": mockGemini,
			}

			orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

			// Setup mocks
			openaiEmbedding := make([]float32, 1536)
			geminiEmbedding := make([]float32, 768)

			mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(openaiEmbedding, tt.openaiError)
			mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(geminiEmbedding, tt.geminiError)

			// Expect error logging for failures
			if tt.openaiError != nil {
				mockLogger.On("Error", mock.MatchedBy(func(msg string) bool {
					return msg == "Failed to generate OpenAI embedding"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			}
			if tt.geminiError != nil {
				mockLogger.On("Error", mock.MatchedBy(func(msg string) bool {
					return msg == "Failed to generate Gemini embedding"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			}

			// Act
			err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorSubstring)
			} else {
				assert.NoError(t, err)
			}

			mockOpenAI.AssertExpectations(t)
			mockGemini.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// TestEmbeddingOrchestrator_StorageFailureDualEmbeddings tests storage failures for dual embeddings
func TestEmbeddingOrchestrator_StorageFailureDualEmbeddings(t *testing.T) {
	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockGemini := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Setup mocks - both providers succeed
	openaiEmbedding := make([]float32, 1536)
	geminiEmbedding := make([]float32, 768)
	storageError := errors.New("database connection failed")

	mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(openaiEmbedding, nil)
	mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(geminiEmbedding, nil)
	mockStorage.On("StoreDualEmbeddings", mock.Anything, 1, openaiEmbedding, geminiEmbedding).Return(storageError)

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store dual embeddings")
	assert.Contains(t, err.Error(), "database connection failed")
	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_SingleProviderStorageFailure tests storage failures for single provider
func TestEmbeddingOrchestrator_SingleProviderStorageFailure(t *testing.T) {
	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := new(MockLogger)

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Setup mocks
	embedding := make([]float32, 1536)
	storageError := errors.New("storage write failed")

	mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(embedding, nil)
	mockStorage.On("StoreEmbedding", mock.Anything, 1, "openai", embedding).Return(storageError)
	mockLogger.On("Error", mock.MatchedBy(func(msg string) bool {
		return msg == "Failed to store embedding"
	}), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding generation failed")
	mockOpenAI.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_ConcurrentProcessing tests concurrent processing behavior
func TestEmbeddingOrchestrator_ConcurrentProcessing(t *testing.T) {
	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockGemini := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger() // Use the enhanced logger from testutil

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Track call timing to verify concurrency
	var openaiStartTime, geminiStartTime, openaiEndTime, geminiEndTime time.Time
	var mu sync.Mutex

	// Setup mocks with artificial delays
	mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(func(ctx context.Context, text string) ([]float32, error) {
		mu.Lock()
		openaiStartTime = time.Now()
		mu.Unlock()

		time.Sleep(50 * time.Millisecond) // Simulate processing time

		mu.Lock()
		openaiEndTime = time.Now()
		mu.Unlock()

		return make([]float32, 1536), nil
	})

	mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(func(ctx context.Context, text string) ([]float32, error) {
		mu.Lock()
		geminiStartTime = time.Now()
		mu.Unlock()

		time.Sleep(50 * time.Millisecond) // Simulate processing time

		mu.Lock()
		geminiEndTime = time.Now()
		mu.Unlock()

		return make([]float32, 768), nil
	})

	mockStorage.On("StoreDualEmbeddings", mock.Anything, 1, mock.Anything, mock.Anything).Return(nil)

	// Act
	start := time.Now()
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")
	totalDuration := time.Since(start)

	// Assert
	assert.NoError(t, err)

	// Verify concurrency: total time should be less than sequential processing
	// Sequential would be ~100ms, concurrent should be ~50-60ms
	assert.Less(t, totalDuration, 80*time.Millisecond, "Processing should be concurrent")

	// Verify both providers started processing (timing may vary in tests)
	mu.Lock()
	assert.False(t, openaiStartTime.IsZero(), "OpenAI should have started processing")
	assert.False(t, geminiStartTime.IsZero(), "Gemini should have started processing")
	assert.False(t, openaiEndTime.IsZero(), "OpenAI should have finished processing")
	assert.False(t, geminiEndTime.IsZero(), "Gemini should have finished processing")
	mu.Unlock()

	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_ContextCancellation tests context cancellation handling
func TestEmbeddingOrchestrator_ContextCancellation(t *testing.T) {
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

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Setup mocks to check context and simulate delay
	mockOpenAI.On("GenerateEmbedding", mock.MatchedBy(func(ctx context.Context) bool {
		// Cancel context during processing
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		return true
	}), "test text").Return(func(ctx context.Context, text string) ([]float32, error) {
		// Simulate work that respects context cancellation
		select {
		case <-time.After(100 * time.Millisecond):
			return make([]float32, 1536), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(func(ctx context.Context, text string) ([]float32, error) {
		// Simulate work that respects context cancellation
		select {
		case <-time.After(100 * time.Millisecond):
			return make([]float32, 768), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	// Expect error logging due to context cancellation
	mockLogger.SetEnabled(true)

	// Act
	err := orchestrator.ProcessTranscription(ctx, 1, "test text")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding generation failed")

	// Verify that error logs were generated
	errorLogs := mockLogger.GetLogsByLevel(testutil.LogLevelError)
	assert.Greater(t, len(errorLogs), 0, "Should have logged errors due to context cancellation")

	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_EmptyProviders tests behavior with no providers
func TestEmbeddingOrchestrator_EmptyProviders(t *testing.T) {
	// Arrange
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	providers := map[string]provider.EmbeddingProvider{} // Empty providers

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

	// Assert
	assert.NoError(t, err) // Should handle gracefully with no providers
}

// TestEmbeddingOrchestrator_GetStatusStorageError tests GetEmbeddingStatus with storage errors
func TestEmbeddingOrchestrator_GetStatusStorageError(t *testing.T) {
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

	// Setup mocks to return storage error
	storageError := errors.New("database unavailable")
	mockStorage.On("GetDualEmbeddings", mock.Anything, 1).Return((*vector.DualEmbedding)(nil), storageError)

	// Act
	status, err := orchestrator.GetEmbeddingStatus(context.Background(), 1)

	// Assert
	assert.NoError(t, err) // Should not propagate storage error
	assert.NotNil(t, status)
	assert.Equal(t, 1, status.TranscriptionID)
	assert.False(t, status.OpenAICompleted)
	assert.False(t, status.GeminiCompleted)
	mockStorage.AssertExpectations(t)
}

// TestEmbeddingOrchestrator_GetStatusPartialEmbeddings tests status with partial embeddings
func TestEmbeddingOrchestrator_GetStatusPartialEmbeddings(t *testing.T) {
	tests := []struct {
		name            string
		openaiEmbedding []float32
		geminiEmbedding []float32
		expectedOpenAI  bool
		expectedGemini  bool
	}{
		{
			name:            "Only OpenAI embedding exists",
			openaiEmbedding: make([]float32, 1536),
			geminiEmbedding: nil,
			expectedOpenAI:  true,
			expectedGemini:  false,
		},
		{
			name:            "Only Gemini embedding exists",
			openaiEmbedding: nil,
			geminiEmbedding: make([]float32, 768),
			expectedOpenAI:  false,
			expectedGemini:  true,
		},
		{
			name:            "No embeddings exist",
			openaiEmbedding: nil,
			geminiEmbedding: nil,
			expectedOpenAI:  false,
			expectedGemini:  false,
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
			dualEmbedding := &vector.DualEmbedding{
				OpenAI: tt.openaiEmbedding,
				Gemini: tt.geminiEmbedding,
			}

			mockStorage.On("GetDualEmbeddings", mock.Anything, 1).Return(dualEmbedding, nil)

			// Act
			status, err := orchestrator.GetEmbeddingStatus(context.Background(), 1)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, status)
			assert.Equal(t, 1, status.TranscriptionID)
			assert.Equal(t, tt.expectedOpenAI, status.OpenAICompleted)
			assert.Equal(t, tt.expectedGemini, status.GeminiCompleted)
			mockStorage.AssertExpectations(t)
		})
	}
}

// TestEmbeddingOrchestrator_ProviderCoordination tests proper coordination between providers
func TestEmbeddingOrchestrator_ProviderCoordination(t *testing.T) {
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

	// Track call order to verify coordination
	var callOrder []string
	var mu sync.Mutex

	// Setup mocks to track execution order
	mockOpenAI.On("GenerateEmbedding", mock.Anything, "test text").Return(func(ctx context.Context, text string) ([]float32, error) {
		mu.Lock()
		callOrder = append(callOrder, "openai_start")
		mu.Unlock()

		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		callOrder = append(callOrder, "openai_end")
		mu.Unlock()

		return make([]float32, 1536), nil
	})

	mockGemini.On("GenerateEmbedding", mock.Anything, "test text").Return(func(ctx context.Context, text string) ([]float32, error) {
		mu.Lock()
		callOrder = append(callOrder, "gemini_start")
		mu.Unlock()

		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		callOrder = append(callOrder, "gemini_end")
		mu.Unlock()

		return make([]float32, 768), nil
	})

	mockStorage.On("StoreDualEmbeddings", mock.Anything, 1, mock.Anything, mock.Anything).Return(func(ctx context.Context, id int, openai, gemini []float32) error {
		mu.Lock()
		callOrder = append(callOrder, "storage")
		mu.Unlock()
		return nil
	})

	// Act
	err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")

	// Assert
	assert.NoError(t, err)

	// Verify coordination: both providers should start before storage is called
	mu.Lock()
	assert.Contains(t, callOrder, "openai_start")
	assert.Contains(t, callOrder, "gemini_start")
	assert.Contains(t, callOrder, "openai_end")
	assert.Contains(t, callOrder, "gemini_end")
	assert.Contains(t, callOrder, "storage")

	// Storage should be called last
	storageIndex := -1
	for i, call := range callOrder {
		if call == "storage" {
			storageIndex = i
			break
		}
	}
	assert.Equal(t, len(callOrder)-1, storageIndex, "Storage should be called last")
	mu.Unlock()

	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// =============================================================================
// BATCH PROCESSOR TESTS - User-Specific Functionality
// =============================================================================

// TestBatchProcessor_ProcessUserTranscriptions tests user-specific batch processing
func TestBatchProcessor_ProcessUserTranscriptions(t *testing.T) {
	tests := []struct {
		name               string
		userNickname       string
		providers          []string
		batchSize          int
		mockTranscriptions []*vector.Transcription
		expectedProcessed  int
		expectedFailed     int
		setupMocks         func(*MockEmbeddingOrchestratorInterface, *MockVectorStorage, *MockLogger)
		expectError        bool
		errorSubstring     string
	}{
		{
			name:         "successful_user_processing_single_provider",
			userNickname: "test_user",
			providers:    []string{"openai"},
			batchSize:    2,
			mockTranscriptions: []*vector.Transcription{
				{ID: 1, User: "test_user", TranscriptionText: "Test 1"},
				{ID: 2, User: "test_user", TranscriptionText: "Test 2"},
			},
			expectedProcessed: 2,
			expectedFailed:    0,
			setupMocks: func(orchestrator *MockEmbeddingOrchestratorInterface, storage *MockVectorStorage, logger *MockLogger) {
				storage.On("GetTranscriptionsWithoutEmbeddingsByUser", mock.Anything, "openai", "test_user", 10000).Return([]*vector.Transcription{
					{ID: 1, User: "test_user", TranscriptionText: "Test 1"},
					{ID: 2, User: "test_user", TranscriptionText: "Test 2"},
				}, nil)

				orchestrator.On("ProcessTranscription", mock.Anything, 1, "Test 1").Return(nil)
				orchestrator.On("ProcessTranscription", mock.Anything, 2, "Test 2").Return(nil)

				logger.On("Info", mock.MatchedBy(func(msg string) bool {
					return msg == "Starting user-specific batch processing"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				logger.On("Info", mock.MatchedBy(func(msg string) bool {
					return msg == "Batch processing progress"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

				logger.On("Info", mock.MatchedBy(func(msg string) bool {
					return msg == "Completed user-specific batch processing"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			},
			expectError: false,
		},
		{
			name:         "successful_user_processing_dual_providers",
			userNickname: "test_user",
			providers:    []string{"openai", "gemini"},
			batchSize:    3,
			mockTranscriptions: []*vector.Transcription{
				{ID: 1, User: "test_user", TranscriptionText: "Test 1"},
				{ID: 2, User: "test_user", TranscriptionText: "Test 2"},
				{ID: 3, User: "test_user", TranscriptionText: "Test 3"},
			},
			expectedProcessed: 3,
			expectedFailed:    0,
			setupMocks: func(orchestrator *MockEmbeddingOrchestratorInterface, storage *MockVectorStorage, logger *MockLogger) {
				// OpenAI has 2 transcriptions
				storage.On("GetTranscriptionsWithoutEmbeddingsByUser", mock.Anything, "openai", "test_user", 10000).Return([]*vector.Transcription{
					{ID: 1, User: "test_user", TranscriptionText: "Test 1"},
					{ID: 2, User: "test_user", TranscriptionText: "Test 2"},
				}, nil)

				// Gemini has 2 transcriptions (with overlap)
				storage.On("GetTranscriptionsWithoutEmbeddingsByUser", mock.Anything, "gemini", "test_user", 10000).Return([]*vector.Transcription{
					{ID: 2, User: "test_user", TranscriptionText: "Test 2"}, // Overlaps with OpenAI
					{ID: 3, User: "test_user", TranscriptionText: "Test 3"},
				}, nil)

				// Should process each unique transcription once
				orchestrator.On("ProcessTranscription", mock.Anything, 1, "Test 1").Return(nil)
				orchestrator.On("ProcessTranscription", mock.Anything, 2, "Test 2").Return(nil)
				orchestrator.On("ProcessTranscription", mock.Anything, 3, "Test 3").Return(nil)

				logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			},
			expectError: false,
		},
		{
			name:               "user_with_no_pending_transcriptions",
			userNickname:       "user_no_pending",
			providers:          []string{"openai"},
			batchSize:          10,
			mockTranscriptions: []*vector.Transcription{},
			expectedProcessed:  0,
			expectedFailed:     0,
			setupMocks: func(orchestrator *MockEmbeddingOrchestratorInterface, storage *MockVectorStorage, logger *MockLogger) {
				storage.On("GetTranscriptionsWithoutEmbeddingsByUser", mock.Anything, "openai", "user_no_pending", 10000).Return([]*vector.Transcription{}, nil)

				logger.On("Info", mock.MatchedBy(func(msg string) bool {
					return msg == "No transcriptions to process for user"
				}), mock.Anything, mock.Anything).Return()
			},
			expectError: false,
		},
		{
			name:         "database_error_retrieving_transcriptions",
			userNickname: "test_user",
			providers:    []string{"openai"},
			batchSize:    10,
			setupMocks: func(orchestrator *MockEmbeddingOrchestratorInterface, storage *MockVectorStorage, logger *MockLogger) {
				storage.On("GetTranscriptionsWithoutEmbeddingsByUser", mock.Anything, "openai", "test_user", 10000).Return(
					[]*vector.Transcription(nil), errors.New("database connection failed"))
			},
			expectError:    true,
			errorSubstring: "database connection failed",
		},
		{
			name:         "unicode_user_nickname",
			userNickname: "用户测试",
			providers:    []string{"openai"},
			batchSize:    1,
			mockTranscriptions: []*vector.Transcription{
				{ID: 1, User: "用户测试", TranscriptionText: "Unicode test"},
			},
			expectedProcessed: 1,
			expectedFailed:    0,
			setupMocks: func(orchestrator *MockEmbeddingOrchestratorInterface, storage *MockVectorStorage, logger *MockLogger) {
				storage.On("GetTranscriptionsWithoutEmbeddingsByUser", mock.Anything, "openai", "用户测试", 10000).Return([]*vector.Transcription{
					{ID: 1, User: "用户测试", TranscriptionText: "Unicode test"},
				}, nil)

				orchestrator.On("ProcessTranscription", mock.Anything, 1, "Unicode test").Return(nil)

				logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			},
			expectError: false,
		},
		{
			name:         "empty_providers_list",
			userNickname: "test_user",
			providers:    []string{},
			batchSize:    10,
			setupMocks: func(orchestrator *MockEmbeddingOrchestratorInterface, storage *MockVectorStorage, logger *MockLogger) {
				logger.On("Info", mock.MatchedBy(func(msg string) bool {
					return msg == "No transcriptions to process for user"
				}), mock.Anything, mock.Anything).Return()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockOrchestrator := new(MockEmbeddingOrchestratorInterface)
			mockStorage := new(MockVectorStorage)
			mockLogger := new(MockLogger)

			processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

			// Setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockOrchestrator, mockStorage, mockLogger)
			}

			// Act
			err := processor.ProcessUserTranscriptions(context.Background(), tt.userNickname, tt.providers, tt.batchSize)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorSubstring != "" {
					assert.Contains(t, err.Error(), tt.errorSubstring)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify all mocks were called as expected
			mockOrchestrator.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// TestBatchProcessor_ProcessUserTranscriptions_ConcurrentProcessing tests concurrent processing within batches
func TestBatchProcessor_ProcessUserTranscriptions_ConcurrentProcessing(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestratorInterface)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	userNickname := "test_user"
	providers := []string{"openai"}
	batchSize := 3

	// Track processing timing to verify concurrency
	var processingTimes []time.Time
	var mu sync.Mutex

	// Setup mocks
	mockStorage.On("GetTranscriptionsWithoutEmbeddingsByUser", mock.Anything, "openai", userNickname, 10000).Return([]*vector.Transcription{
		{ID: 1, User: userNickname, TranscriptionText: "Test 1"},
		{ID: 2, User: userNickname, TranscriptionText: "Test 2"},
		{ID: 3, User: userNickname, TranscriptionText: "Test 3"},
	}, nil)

	// Mock ProcessTranscription to record timing
	mockOrchestrator.On("ProcessTranscription", mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, id int, text string) error {
		mu.Lock()
		processingTimes = append(processingTimes, time.Now())
		mu.Unlock()

		// Simulate some processing time
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	// Act
	start := time.Now()
	err := processor.ProcessUserTranscriptions(context.Background(), userNickname, providers, batchSize)
	totalDuration := time.Since(start)

	// Assert
	assert.NoError(t, err)

	// Verify concurrent processing: total time should be less than sequential processing
	// Sequential would be ~150ms (3 * 50ms), concurrent should be ~50-100ms depending on timing
	assert.Less(t, totalDuration, 120*time.Millisecond, "Processing should be concurrent within batch")

	// Verify all three items were processed
	mu.Lock()
	assert.Len(t, processingTimes, 3, "All three transcriptions should be processed")
	mu.Unlock()

	mockOrchestrator.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// TestBatchProcessor_ProcessUserTranscriptions_ContextCancellation tests context cancellation
func TestBatchProcessor_ProcessUserTranscriptions_ContextCancellation(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestratorInterface)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	userNickname := "test_user"
	providers := []string{"openai"}
	batchSize := 2

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Setup mocks
	mockStorage.On("GetTranscriptionsWithoutEmbeddingsByUser", mock.Anything, "openai", userNickname, 10000).Return([]*vector.Transcription{
		{ID: 1, User: userNickname, TranscriptionText: "Test 1"},
		{ID: 2, User: userNickname, TranscriptionText: "Test 2"},
	}, nil)

	// Mock ProcessTranscription to simulate work that can be cancelled
	mockOrchestrator.On("ProcessTranscription", mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, id int, text string) error {
		// Cancel context during processing
		if id == 1 {
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
		}

		// Simulate work that respects context cancellation
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	// Act
	err := processor.ProcessUserTranscriptions(ctx, userNickname, providers, batchSize)

	// Assert
	// The method itself doesn't handle context cancellation directly,
	// but the orchestrator should respect it
	// We mainly verify that the cancellation doesn't cause a panic
	assert.NoError(t, err) // ProcessUserTranscriptions doesn't propagate context cancellation errors directly

	mockOrchestrator.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// MockEmbeddingOrchestratorInterface for testing
type MockEmbeddingOrchestratorInterface struct {
	mock.Mock
}

func (m *MockEmbeddingOrchestratorInterface) ProcessTranscription(ctx context.Context, transcriptionID int, text string) error {
	args := m.Called(ctx, transcriptionID, text)
	return args.Error(0)
}

func (m *MockEmbeddingOrchestratorInterface) GetEmbeddingStatus(ctx context.Context, transcriptionID int) (*EmbeddingStatus, error) {
	args := m.Called(ctx, transcriptionID)
	return args.Get(0).(*EmbeddingStatus), args.Error(1)
}
