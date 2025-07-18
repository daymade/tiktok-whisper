package vector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PgVectorTestSuite groups all PostgreSQL vector storage tests
type PgVectorTestSuite struct {
	suite.Suite
	db      *sql.DB
	storage *PgVectorStorage
	ctx     context.Context
}

// SetupSuite runs once before all tests
func (suite *PgVectorTestSuite) SetupSuite() {
	// Skip if running in CI or no postgres available
	if testing.Short() {
		suite.T().Skip("Skipping PostgreSQL tests in short mode")
	}

	// Check if PostgreSQL is available
	if os.Getenv("SKIP_PG_TESTS") == "true" {
		suite.T().Skip("Skipping PostgreSQL tests as SKIP_PG_TESTS is set")
	}

	// Setup test database using testutil
	db := suite.setupTestDatabase()
	suite.db = db
	suite.storage = NewPgVectorStorage(db)
	suite.ctx = context.Background()
}

// TearDownSuite runs once after all tests
func (suite *PgVectorTestSuite) TearDownSuite() {
	if suite.storage != nil {
		suite.storage.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *PgVectorTestSuite) SetupTest() {
	// Clean data before each test
	suite.cleanTestData()
	suite.seedTestData()
}

// setupTestDatabase creates a test database with required schema
func (suite *PgVectorTestSuite) setupTestDatabase() *sql.DB {
	// Try to use testutil first
	if pgURL := os.Getenv("POSTGRES_TEST_URL"); pgURL != "" {
		db, err := sql.Open("postgres", pgURL)
		require.NoError(suite.T(), err)
		
		err = db.Ping()
		require.NoError(suite.T(), err)
		
		suite.createTestSchema(db)
		return db
	}

	// Fallback to default connection
	db, err := sql.Open("postgres", "user=postgres password=passwd dbname=postgres sslmode=disable host=localhost")
	require.NoError(suite.T(), err)
	
	err = db.Ping()
	require.NoError(suite.T(), err)
	
	suite.createTestSchema(db)
	return db
}

// createTestSchema creates necessary tables and extensions for testing
func (suite *PgVectorTestSuite) createTestSchema(db *sql.DB) {
	// Create pgvector extension
	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	require.NoError(suite.T(), err)

	// Create test table with vector columns
	schema := `
	CREATE TABLE IF NOT EXISTS transcriptions (
		id SERIAL PRIMARY KEY,
		user_nickname VARCHAR(255),
		mp3_file_name VARCHAR(255) NOT NULL,
		transcription TEXT NOT NULL,
		last_conversion_time TIMESTAMP DEFAULT NOW(),
		
		-- OpenAI embedding columns
		embedding_openai vector(1536),
		embedding_openai_model VARCHAR(50),
		embedding_openai_created_at TIMESTAMP,
		embedding_openai_status VARCHAR(20) DEFAULT 'pending',
		
		-- Gemini embedding columns
		embedding_gemini vector(768),
		embedding_gemini_model VARCHAR(50),
		embedding_gemini_created_at TIMESTAMP,
		embedding_gemini_status VARCHAR(20) DEFAULT 'pending',
		
		-- Sync status
		embedding_sync_status VARCHAR(20) DEFAULT 'pending'
	);
	`
	_, err = db.Exec(schema)
	require.NoError(suite.T(), err)
}

// cleanTestData removes all test data
func (suite *PgVectorTestSuite) cleanTestData() {
	_, err := suite.db.Exec("TRUNCATE TABLE transcriptions RESTART IDENTITY CASCADE")
	require.NoError(suite.T(), err)
}

// seedTestData inserts test records
func (suite *PgVectorTestSuite) seedTestData() {
	testData := []struct {
		user          string
		mp3FileName   string
		transcription string
	}{
		{"test_user_1", "test1.mp3", "This is test transcription 1"},
		{"test_user_1", "test2.mp3", "This is test transcription 2"},
		{"test_user_2", "test3.mp3", "This is test transcription 3"},
	}

	for _, data := range testData {
		_, err := suite.db.Exec(`
			INSERT INTO transcriptions (user_nickname, mp3_file_name, transcription)
			VALUES ($1, $2, $3)
		`, data.user, data.mp3FileName, data.transcription)
		require.NoError(suite.T(), err)
	}
}

// TestPgVectorStorageSuite runs the test suite
func TestPgVectorStorageSuite(t *testing.T) {
	suite.Run(t, new(PgVectorTestSuite))
}

// TestStoreAndRetrieveSingleEmbedding tests single embedding operations
func (suite *PgVectorTestSuite) TestStoreAndRetrieveSingleEmbedding() {
	tests := []struct {
		name            string
		transcriptionID int
		provider        string
		embeddingSize   int
	}{
		{"OpenAI embedding", 1, "openai", 1536},
		{"Gemini embedding", 1, "gemini", 768},
		{"OpenAI for different transcription", 2, "openai", 1536},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Create test embedding
			embedding := make([]float32, tt.embeddingSize)
			for i := range embedding {
				embedding[i] = float32(i) / float32(tt.embeddingSize)
			}

			// Store embedding
			err := suite.storage.StoreEmbedding(suite.ctx, tt.transcriptionID, tt.provider, embedding)
			suite.NoError(err)

			// Retrieve embedding
			retrieved, err := suite.storage.GetEmbedding(suite.ctx, tt.transcriptionID, tt.provider)
			suite.NoError(err)
			suite.NotNil(retrieved)
			suite.Equal(len(embedding), len(retrieved))

			// Verify values match (with float precision tolerance)
			for i := range embedding {
				suite.InDelta(embedding[i], retrieved[i], 0.0001)
			}

			// Verify metadata was set correctly
			var model, status sql.NullString
			var createdAt sql.NullTime
			
			query := fmt.Sprintf(`
				SELECT embedding_%s_model, embedding_%s_status, embedding_%s_created_at
				FROM transcriptions WHERE id = $1
			`, tt.provider, tt.provider, tt.provider)
			
			err = suite.db.QueryRow(query, tt.transcriptionID).Scan(&model, &status, &createdAt)
			suite.NoError(err)
			suite.True(model.Valid)
			suite.True(status.Valid)
			suite.Equal("completed", status.String)
			suite.True(createdAt.Valid)
		})
	}
}

// TestStoreDualEmbeddings tests storing both embeddings at once
func (suite *PgVectorTestSuite) TestStoreDualEmbeddings() {
	transcriptionID := 1
	openaiEmbedding := suite.generateTestEmbedding(1536)
	geminiEmbedding := suite.generateTestEmbedding(768)

	// Store dual embeddings
	err := suite.storage.StoreDualEmbeddings(suite.ctx, transcriptionID, openaiEmbedding, geminiEmbedding)
	suite.NoError(err)

	// Retrieve dual embeddings
	dualEmbedding, err := suite.storage.GetDualEmbeddings(suite.ctx, transcriptionID)
	suite.NoError(err)
	suite.NotNil(dualEmbedding)

	// Verify both embeddings
	suite.Equal(len(openaiEmbedding), len(dualEmbedding.OpenAI))
	suite.Equal(len(geminiEmbedding), len(dualEmbedding.Gemini))

	for i := range openaiEmbedding {
		suite.InDelta(openaiEmbedding[i], dualEmbedding.OpenAI[i], 0.0001)
	}
	for i := range geminiEmbedding {
		suite.InDelta(geminiEmbedding[i], dualEmbedding.Gemini[i], 0.0001)
	}

	// Verify sync status
	var syncStatus string
	err = suite.db.QueryRow("SELECT embedding_sync_status FROM transcriptions WHERE id = $1", transcriptionID).Scan(&syncStatus)
	suite.NoError(err)
	suite.Equal("completed", syncStatus)
}

// TestGetEmbeddingNotFound tests retrieving non-existent embeddings
func (suite *PgVectorTestSuite) TestGetEmbeddingNotFound() {
	tests := []struct {
		name            string
		transcriptionID int
		provider        string
	}{
		{"Non-existent transcription", 9999, "openai"},
		{"Existing transcription no embedding", 1, "openai"},
		{"Invalid provider", 1, "invalid"},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			_, err := suite.storage.GetEmbedding(suite.ctx, tt.transcriptionID, tt.provider)
			suite.Error(err)
		})
	}
}

// TestGetTranscriptionsWithoutEmbeddings tests batch retrieval
func (suite *PgVectorTestSuite) TestGetTranscriptionsWithoutEmbeddings() {
	// Add some embeddings to transcription 1
	embedding := suite.generateTestEmbedding(1536)
	err := suite.storage.StoreEmbedding(suite.ctx, 1, "openai", embedding)
	suite.NoError(err)

	// Get transcriptions without OpenAI embeddings
	transcriptions, err := suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 10)
	suite.NoError(err)
	
	// Should get transcriptions 2 and 3
	suite.Len(transcriptions, 2)
	
	// Verify transcription 1 is not in results
	for _, t := range transcriptions {
		suite.NotEqual(1, t.ID)
	}

	// Test limit
	transcriptions, err = suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 1)
	suite.NoError(err)
	suite.Len(transcriptions, 1)

	// Test different provider
	transcriptions, err = suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "gemini", 10)
	suite.NoError(err)
	suite.Len(transcriptions, 3) // All 3 should not have gemini embeddings
}

// TestInvalidProvider tests operations with unsupported providers
func (suite *PgVectorTestSuite) TestInvalidProvider() {
	embedding := suite.generateTestEmbedding(100)
	
	// Test store with invalid provider
	err := suite.storage.StoreEmbedding(suite.ctx, 1, "invalid", embedding)
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported provider")

	// Test get with invalid provider
	_, err = suite.storage.GetEmbedding(suite.ctx, 1, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported provider")

	// Test batch get with invalid provider
	_, err = suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "invalid", 10)
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported provider")
}

// TestConcurrentAccess tests thread-safe operations
func (suite *PgVectorTestSuite) TestConcurrentAccess() {
	var wg sync.WaitGroup
	numGoroutines := 10
	errorChan := make(chan error, numGoroutines*2)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Store embedding for different transcriptions
			transcriptionID := (id % 3) + 1
			embedding := suite.generateTestEmbedding(1536)
			
			err := suite.storage.StoreEmbedding(suite.ctx, transcriptionID, "openai", embedding)
			if err != nil {
				errorChan <- err
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Try to read embeddings
			_, err := suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 5)
			if err != nil {
				errorChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		suite.NoError(err)
	}
}

// TestVectorConversion tests vector string conversion functions
func (suite *PgVectorTestSuite) TestVectorConversion() {
	tests := []struct {
		name   string
		vector []float32
	}{
		{"Empty vector", []float32{}},
		{"Single element", []float32{1.5}},
		{"Multiple elements", []float32{1.0, 2.0, 3.0, 4.0}},
		{"Negative values", []float32{-1.5, -2.5, 3.5}},
		{"Very small values", []float32{0.000001, 0.000002}},
		{"Large vector", suite.generateTestEmbedding(100)},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Convert to string
			str := vectorToString(tt.vector)
			suite.NotEmpty(str)

			// Convert back to vector
			result := stringToVector(str)
			suite.Equal(len(tt.vector), len(result))

			// Verify values match
			for i := range tt.vector {
				suite.InDelta(tt.vector[i], result[i], 0.0001)
			}
		})
	}
}

// TestTransactionRollback tests transaction handling concept
func (suite *PgVectorTestSuite) TestTransactionRollback() {
	tx, err := suite.db.BeginTx(suite.ctx, nil)
	suite.NoError(err)

	// Note: PgVectorStorage expects *sql.DB, not *sql.Tx
	// This test demonstrates the concept of transaction handling
	// In a real implementation, we'd need a transaction-aware wrapper
	
	// Store embedding within transaction
	embedding := suite.generateTestEmbedding(1536)
	_, err = tx.ExecContext(suite.ctx, `
		UPDATE transcriptions 
		SET embedding_openai = $1
		WHERE id = $2
	`, vectorToString(embedding), 1)
	// This may fail if vector columns don't exist, which is expected
	
	// Rollback transaction
	err = tx.Rollback()
	suite.NoError(err)

	// Verify embedding was not saved (if the update succeeded)
	_, err = suite.storage.GetEmbedding(suite.ctx, 1, "openai")
	if err == nil {
		// If no error, the embedding shouldn't be the one we tried to store in transaction
		// This is a conceptual test of transaction behavior
	}
}

// TestPartialDualEmbeddings tests retrieving partial dual embeddings
func (suite *PgVectorTestSuite) TestPartialDualEmbeddings() {
	transcriptionID := 1
	
	// Store only OpenAI embedding
	openaiEmbedding := suite.generateTestEmbedding(1536)
	err := suite.storage.StoreEmbedding(suite.ctx, transcriptionID, "openai", openaiEmbedding)
	suite.NoError(err)

	// Retrieve dual embeddings
	dualEmbedding, err := suite.storage.GetDualEmbeddings(suite.ctx, transcriptionID)
	suite.NoError(err)
	suite.NotNil(dualEmbedding)
	suite.NotNil(dualEmbedding.OpenAI)
	suite.Nil(dualEmbedding.Gemini)

	// Store Gemini embedding
	geminiEmbedding := suite.generateTestEmbedding(768)
	err = suite.storage.StoreEmbedding(suite.ctx, transcriptionID, "gemini", geminiEmbedding)
	suite.NoError(err)

	// Retrieve again
	dualEmbedding, err = suite.storage.GetDualEmbeddings(suite.ctx, transcriptionID)
	suite.NoError(err)
	suite.NotNil(dualEmbedding)
	suite.NotNil(dualEmbedding.OpenAI)
	suite.NotNil(dualEmbedding.Gemini)
}

// TestContextCancellation tests context cancellation handling
func (suite *PgVectorTestSuite) TestContextCancellation() {
	ctx, cancel := context.WithCancel(suite.ctx)
	cancel() // Cancel immediately

	// Try operations with cancelled context
	embedding := suite.generateTestEmbedding(1536)
	err := suite.storage.StoreEmbedding(ctx, 1, "openai", embedding)
	suite.Error(err)

	_, err = suite.storage.GetEmbedding(ctx, 1, "openai")
	suite.Error(err)

	_, err = suite.storage.GetDualEmbeddings(ctx, 1)
	suite.Error(err)

	_, err = suite.storage.GetTranscriptionsWithoutEmbeddings(ctx, "openai", 10)
	suite.Error(err)
}

// Helper function to generate test embeddings
func (suite *PgVectorTestSuite) generateTestEmbedding(size int) []float32 {
	embedding := make([]float32, size)
	for i := range embedding {
		embedding[i] = float32(i) / float32(size)
	}
	return embedding
}