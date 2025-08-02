package vector

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// MockVectorTestSuite groups all mock storage tests
type MockVectorTestSuite struct {
	suite.Suite
	storage *MockVectorStorage
	ctx     context.Context
}

// SetupTest runs before each test
func (suite *MockVectorTestSuite) SetupTest() {
	suite.storage = NewMockVectorStorage()
	suite.ctx = context.Background()
}

// TestMockVectorStorageSuite runs the test suite
func TestMockVectorStorageSuite(t *testing.T) {
	suite.Run(t, new(MockVectorTestSuite))
}

// TestMockImplementsInterface verifies MockVectorStorage implements VectorStorage
func (suite *MockVectorTestSuite) TestMockImplementsInterface() {
	var _ VectorStorage = suite.storage
	// Compilation will fail if MockVectorStorage doesn't implement VectorStorage
}

// TestStoreAndRetrieveSingleEmbedding tests single embedding operations
func (suite *MockVectorTestSuite) TestStoreAndRetrieveSingleEmbedding() {
	tests := []struct {
		name            string
		transcriptionID int
		provider        string
		embedding       []float32
	}{
		{
			name:            "Small embedding",
			transcriptionID: 1,
			provider:        "openai",
			embedding:       []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		},
		{
			name:            "Large OpenAI embedding",
			transcriptionID: 2,
			provider:        "openai",
			embedding:       generateMockTestEmbedding(1536),
		},
		{
			name:            "Gemini embedding",
			transcriptionID: 3,
			provider:        "gemini",
			embedding:       generateMockTestEmbedding(768),
		},
		{
			name:            "Empty embedding",
			transcriptionID: 4,
			provider:        "openai",
			embedding:       []float32{},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Store embedding
			err := suite.storage.StoreEmbedding(suite.ctx, tt.transcriptionID, tt.provider, tt.embedding)
			suite.NoError(err)

			// Retrieve embedding
			retrieved, err := suite.storage.GetEmbedding(suite.ctx, tt.transcriptionID, tt.provider)
			suite.NoError(err)
			suite.Equal(tt.embedding, retrieved)
		})
	}
}

// TestStoreDualEmbeddings tests storing both embeddings at once
func (suite *MockVectorTestSuite) TestStoreDualEmbeddings() {
	transcriptionID := 1
	openaiEmbedding := generateMockTestEmbedding(1536)
	geminiEmbedding := generateMockTestEmbedding(768)

	// Store dual embeddings
	err := suite.storage.StoreDualEmbeddings(suite.ctx, transcriptionID, openaiEmbedding, geminiEmbedding)
	suite.NoError(err)

	// Retrieve dual embeddings
	dualEmbedding, err := suite.storage.GetDualEmbeddings(suite.ctx, transcriptionID)
	suite.NoError(err)
	suite.NotNil(dualEmbedding)
	suite.Equal(openaiEmbedding, dualEmbedding.OpenAI)
	suite.Equal(geminiEmbedding, dualEmbedding.Gemini)

	// Verify we can also retrieve individually
	individualOpenAI, err := suite.storage.GetEmbedding(suite.ctx, transcriptionID, "openai")
	suite.NoError(err)
	suite.Equal(openaiEmbedding, individualOpenAI)

	individualGemini, err := suite.storage.GetEmbedding(suite.ctx, transcriptionID, "gemini")
	suite.NoError(err)
	suite.Equal(geminiEmbedding, individualGemini)
}

// TestGetEmbeddingNotFound tests retrieving non-existent embeddings
func (suite *MockVectorTestSuite) TestGetEmbeddingNotFound() {
	// Try to get embedding that doesn't exist
	_, err := suite.storage.GetEmbedding(suite.ctx, 999, "openai")
	suite.Error(err)
	suite.Contains(err.Error(), "not found")

	// Try to get dual embeddings that don't exist
	_, err = suite.storage.GetDualEmbeddings(suite.ctx, 999)
	suite.Error(err)
	suite.Contains(err.Error(), "no embeddings found")
}

// TestGetTranscriptionsWithoutEmbeddings tests batch retrieval
func (suite *MockVectorTestSuite) TestGetTranscriptionsWithoutEmbeddings() {
	// Add test transcriptions
	for i := 1; i <= 5; i++ {
		suite.storage.AddMockTranscription(i, "Test transcription "+string(rune(i+'0')))
	}

	// Store embeddings for some transcriptions
	embedding := []float32{0.1, 0.2, 0.3}
	suite.storage.StoreEmbedding(suite.ctx, 1, "openai", embedding)
	suite.storage.StoreEmbedding(suite.ctx, 3, "openai", embedding)

	// Get transcriptions without OpenAI embeddings
	transcriptions, err := suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 10)
	suite.NoError(err)
	suite.Len(transcriptions, 3) // Should return transcriptions 2, 4, 5

	// Verify correct transcriptions returned
	idsWithoutEmbeddings := []int{2, 4, 5}
	for i, t := range transcriptions {
		suite.Contains(idsWithoutEmbeddings, t.ID)
		suite.Equal("test_user", t.User)
		suite.NotEmpty(t.Mp3FileName)
		suite.NotEmpty(t.TranscriptionText)
		suite.False(t.CreatedAt.IsZero())
		_ = i // avoid unused variable
	}

	// Test with limit
	transcriptions, err = suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 2)
	suite.NoError(err)
	suite.Len(transcriptions, 2)

	// Test different provider
	transcriptions, err = suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "gemini", 10)
	suite.NoError(err)
	suite.Len(transcriptions, 5) // All should not have gemini embeddings
}

// TestConcurrentAccess tests thread-safe operations
func (suite *MockVectorTestSuite) TestConcurrentAccess() {
	var wg sync.WaitGroup
	numGoroutines := 100
	numTranscriptions := 10

	// Add test transcriptions
	for i := 1; i <= numTranscriptions; i++ {
		suite.storage.AddMockTranscription(i, "Concurrent test transcription")
	}

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			transcriptionID := (id % numTranscriptions) + 1
			provider := "openai"
			if id%2 == 0 {
				provider = "gemini"
			}
			embedding := []float32{float32(id), float32(id + 1), float32(id + 2)}

			err := suite.storage.StoreEmbedding(suite.ctx, transcriptionID, provider, embedding)
			assert.NoError(suite.T(), err)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			transcriptionID := (id % numTranscriptions) + 1
			provider := "openai"
			if id%2 == 0 {
				provider = "gemini"
			}

			// Try to read - may or may not exist yet
			_, _ = suite.storage.GetEmbedding(suite.ctx, transcriptionID, provider)
		}(i)
	}

	// Concurrent batch reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 5)
			assert.NoError(suite.T(), err)
		}()
	}

	wg.Wait()

	// Verify storage is still consistent
	transcriptions, err := suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 100)
	suite.NoError(err)
	suite.NotNil(transcriptions)
}

// TestOverwriteEmbedding tests updating existing embeddings
func (suite *MockVectorTestSuite) TestOverwriteEmbedding() {
	transcriptionID := 1
	provider := "openai"
	originalEmbedding := []float32{0.1, 0.2, 0.3}
	updatedEmbedding := []float32{0.4, 0.5, 0.6}

	// Store original
	err := suite.storage.StoreEmbedding(suite.ctx, transcriptionID, provider, originalEmbedding)
	suite.NoError(err)

	// Verify original stored
	retrieved, err := suite.storage.GetEmbedding(suite.ctx, transcriptionID, provider)
	suite.NoError(err)
	suite.Equal(originalEmbedding, retrieved)

	// Overwrite with new embedding
	err = suite.storage.StoreEmbedding(suite.ctx, transcriptionID, provider, updatedEmbedding)
	suite.NoError(err)

	// Verify updated
	retrieved, err = suite.storage.GetEmbedding(suite.ctx, transcriptionID, provider)
	suite.NoError(err)
	suite.Equal(updatedEmbedding, retrieved)
}

// TestPartialDualEmbeddings tests retrieving partial dual embeddings
func (suite *MockVectorTestSuite) TestPartialDualEmbeddings() {
	transcriptionID := 1
	openaiEmbedding := []float32{0.1, 0.2, 0.3}

	// Store only OpenAI embedding
	err := suite.storage.StoreEmbedding(suite.ctx, transcriptionID, "openai", openaiEmbedding)
	suite.NoError(err)

	// Retrieve dual embeddings
	dualEmbedding, err := suite.storage.GetDualEmbeddings(suite.ctx, transcriptionID)
	suite.NoError(err)
	suite.NotNil(dualEmbedding)
	suite.Equal(openaiEmbedding, dualEmbedding.OpenAI)
	suite.Nil(dualEmbedding.Gemini)

	// Add Gemini embedding
	geminiEmbedding := []float32{0.4, 0.5}
	err = suite.storage.StoreEmbedding(suite.ctx, transcriptionID, "gemini", geminiEmbedding)
	suite.NoError(err)

	// Retrieve again
	dualEmbedding, err = suite.storage.GetDualEmbeddings(suite.ctx, transcriptionID)
	suite.NoError(err)
	suite.NotNil(dualEmbedding)
	suite.Equal(openaiEmbedding, dualEmbedding.OpenAI)
	suite.Equal(geminiEmbedding, dualEmbedding.Gemini)
}

// TestMockStorageStateIsolation tests that operations don't interfere with each other
func (suite *MockVectorTestSuite) TestMockStorageStateIsolation() {
	// Create two separate storage instances
	storage1 := NewMockVectorStorage()
	storage2 := NewMockVectorStorage()

	// Add transcription and embedding to storage1
	storage1.AddMockTranscription(1, "Storage 1 transcription")
	embedding1 := []float32{1.0, 2.0, 3.0}
	err := storage1.StoreEmbedding(suite.ctx, 1, "openai", embedding1)
	suite.NoError(err)

	// Add different transcription and embedding to storage2
	storage2.AddMockTranscription(1, "Storage 2 transcription")
	embedding2 := []float32{4.0, 5.0, 6.0}
	err = storage2.StoreEmbedding(suite.ctx, 1, "openai", embedding2)
	suite.NoError(err)

	// Verify storage1 has its own data
	retrieved1, err := storage1.GetEmbedding(suite.ctx, 1, "openai")
	suite.NoError(err)
	suite.Equal(embedding1, retrieved1)

	// Verify storage2 has its own data
	retrieved2, err := storage2.GetEmbedding(suite.ctx, 1, "openai")
	suite.NoError(err)
	suite.Equal(embedding2, retrieved2)

	// Verify transcriptions are different
	transcriptions1, err := storage1.GetTranscriptionsWithoutEmbeddings(suite.ctx, "gemini", 10)
	suite.NoError(err)
	suite.Len(transcriptions1, 1)
	suite.Equal("Storage 1 transcription", transcriptions1[0].TranscriptionText)

	transcriptions2, err := storage2.GetTranscriptionsWithoutEmbeddings(suite.ctx, "gemini", 10)
	suite.NoError(err)
	suite.Len(transcriptions2, 1)
	suite.Equal("Storage 2 transcription", transcriptions2[0].TranscriptionText)
}

// TestCloseOperation tests the Close method
func (suite *MockVectorTestSuite) TestCloseOperation() {
	err := suite.storage.Close()
	suite.NoError(err)

	// Should still be able to use after close (it's a mock)
	err = suite.storage.StoreEmbedding(suite.ctx, 1, "openai", []float32{0.1})
	suite.NoError(err)
}

// Helper function to generate test embeddings for interface tests
func generateMockTestEmbedding(size int) []float32 {
	embedding := make([]float32, size)
	for i := range embedding {
		embedding[i] = float32(i) / float32(size)
	}
	return embedding
}

// Additional standalone tests for interface compliance

// TestVectorStorageInterface ensures all implementations satisfy the interface
func TestVectorStorageInterface(t *testing.T) {
	tests := []struct {
		name    string
		storage VectorStorage
	}{
		{"MockVectorStorage", NewMockVectorStorage()},
		// PgVectorStorage would be tested in integration tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Test all interface methods
			embedding := []float32{0.1, 0.2, 0.3}

			// StoreEmbedding
			err := tt.storage.StoreEmbedding(ctx, 1, "openai", embedding)
			assert.NoError(t, err)

			// GetEmbedding
			_, err = tt.storage.GetEmbedding(ctx, 1, "openai")
			// Error is ok if not found
			_ = err

			// StoreDualEmbeddings
			err = tt.storage.StoreDualEmbeddings(ctx, 1, embedding, embedding)
			assert.NoError(t, err)

			// GetDualEmbeddings
			_, err = tt.storage.GetDualEmbeddings(ctx, 1)
			_ = err

			// GetTranscriptionsWithoutEmbeddings
			_, err = tt.storage.GetTranscriptionsWithoutEmbeddings(ctx, "openai", 10)
			_ = err

			// Close
			err = tt.storage.Close()
			assert.NoError(t, err)
		})
	}
}
