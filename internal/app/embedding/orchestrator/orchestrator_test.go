package orchestrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/app/embedding/provider"
	"tiktok-whisper/internal/app/storage/vector"
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