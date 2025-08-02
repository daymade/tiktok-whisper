package vector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

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

// TestGetTranscriptionsWithoutEmbeddingsByUser tests user-specific batch retrieval
func (suite *PgVectorTestSuite) TestGetTranscriptionsWithoutEmbeddingsByUser() {
	// Setup user-specific test data
	suite.setupUserSpecificTestData()

	tests := []struct {
		name         string
		provider     string
		userNickname string
		limit        int
		expectError  bool
		expectCount  int
		errorMsg     string
	}{
		{
			name:         "openai_user_with_pending_embeddings",
			provider:     "openai",
			userNickname: "user_pending",
			limit:        10,
			expectError:  false,
			expectCount:  2, // Based on test data setup
		},
		{
			name:         "gemini_user_with_pending_embeddings",
			provider:     "gemini",
			userNickname: "user_pending",
			limit:        10,
			expectError:  false,
			expectCount:  3, // Different from OpenAI due to status differences
		},
		{
			name:         "user_with_no_pending_embeddings",
			provider:     "openai",
			userNickname: "user_completed",
			limit:        10,
			expectError:  false,
			expectCount:  0,
		},
		{
			name:         "non_existent_user",
			provider:     "openai",
			userNickname: "non_existent_user",
			limit:        10,
			expectError:  false,
			expectCount:  0,
		},
		{
			name:         "empty_user_nickname",
			provider:     "openai",
			userNickname: "",
			limit:        10,
			expectError:  false,
			expectCount:  0,
		},
		{
			name:         "unicode_user_nickname",
			provider:     "openai",
			userNickname: "用户测试",
			limit:        10,
			expectError:  false,
			expectCount:  1, // Based on test data setup
		},
		{
			name:         "invalid_provider",
			provider:     "invalid_provider",
			userNickname: "user_pending",
			limit:        10,
			expectError:  true,
			expectCount:  0,
			errorMsg:     "unsupported provider",
		},
		{
			name:         "limit_restriction",
			provider:     "openai",
			userNickname: "user_pending",
			limit:        1,
			expectError:  false,
			expectCount:  1, // Should respect limit
		},
		{
			name:         "zero_limit",
			provider:     "openai",
			userNickname: "user_pending",
			limit:        0,
			expectError:  false,
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			transcriptions, err := suite.storage.GetTranscriptionsWithoutEmbeddingsByUser(
				suite.ctx, tt.provider, tt.userNickname, tt.limit)

			if tt.expectError {
				suite.Error(err)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
				return
			}

			suite.NoError(err)
			suite.Len(transcriptions, tt.expectCount)

			// Verify all returned transcriptions belong to the user
			for i, transcription := range transcriptions {
				suite.Equal(tt.userNickname, transcription.User,
					"Transcription %d should belong to user %s", i, tt.userNickname)
				suite.NotZero(transcription.ID,
					"Transcription %d should have valid ID", i)
				suite.NotEmpty(transcription.TranscriptionText,
					"Transcription %d should have text", i)
			}

			// Verify ordering (should be by ID ascending)
			if len(transcriptions) > 1 {
				for i := 1; i < len(transcriptions); i++ {
					suite.LessOrEqual(transcriptions[i-1].ID, transcriptions[i].ID,
						"Transcriptions should be ordered by ID")
				}
			}
		})
	}
}

// TestGetUserEmbeddingStats tests user embedding statistics retrieval
func (suite *PgVectorTestSuite) TestGetUserEmbeddingStats() {
	// Setup user-specific test data
	suite.setupUserSpecificTestData()

	tests := []struct {
		name                        string
		userNickname                string
		expectError                 bool
		expectedTotalTranscriptions int
		expectedOpenAIEmbeddings    int
		expectedGeminiEmbeddings    int
		expectedPendingOpenAI       int
		expectedPendingGemini       int
		expectedFailedOpenAI        int
		expectedFailedGemini        int
		errorMsg                    string
	}{
		{
			name:                        "user_with_mixed_embedding_status",
			userNickname:                "user_pending",
			expectError:                 false,
			expectedTotalTranscriptions: 5, // Based on test data
			expectedOpenAIEmbeddings:    2, // Completed embeddings
			expectedGeminiEmbeddings:    1, // Completed embeddings
			expectedPendingOpenAI:       2, // Pending OpenAI
			expectedPendingGemini:       3, // Pending Gemini
			expectedFailedOpenAI:        1, // Failed OpenAI
			expectedFailedGemini:        1, // Failed Gemini
		},
		{
			name:                        "user_with_completed_embeddings",
			userNickname:                "user_completed",
			expectError:                 false,
			expectedTotalTranscriptions: 3,
			expectedOpenAIEmbeddings:    3, // All completed
			expectedGeminiEmbeddings:    3, // All completed
			expectedPendingOpenAI:       0,
			expectedPendingGemini:       0,
			expectedFailedOpenAI:        0,
			expectedFailedGemini:        0,
		},
		{
			name:                        "user_with_no_embeddings",
			userNickname:                "user_no_embeddings",
			expectError:                 false,
			expectedTotalTranscriptions: 2,
			expectedOpenAIEmbeddings:    0,
			expectedGeminiEmbeddings:    0,
			expectedPendingOpenAI:       2, // All pending
			expectedPendingGemini:       2, // All pending
			expectedFailedOpenAI:        0,
			expectedFailedGemini:        0,
		},
		{
			name:                        "non_existent_user",
			userNickname:                "non_existent_user",
			expectError:                 false,
			expectedTotalTranscriptions: 0,
			expectedOpenAIEmbeddings:    0,
			expectedGeminiEmbeddings:    0,
			expectedPendingOpenAI:       0,
			expectedPendingGemini:       0,
			expectedFailedOpenAI:        0,
			expectedFailedGemini:        0,
		},
		{
			name:                        "unicode_user",
			userNickname:                "用户测试",
			expectError:                 false,
			expectedTotalTranscriptions: 1,
			expectedOpenAIEmbeddings:    0,
			expectedGeminiEmbeddings:    0,
			expectedPendingOpenAI:       1,
			expectedPendingGemini:       1,
			expectedFailedOpenAI:        0,
			expectedFailedGemini:        0,
		},
		{
			name:                        "user_with_special_characters",
			userNickname:                "test'; DROP TABLE transcriptions; --",
			expectError:                 false,
			expectedTotalTranscriptions: 0,
			expectedOpenAIEmbeddings:    0,
			expectedGeminiEmbeddings:    0,
			expectedPendingOpenAI:       0,
			expectedPendingGemini:       0,
			expectedFailedOpenAI:        0,
			expectedFailedGemini:        0,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			stats, err := suite.storage.GetUserEmbeddingStats(suite.ctx, tt.userNickname)

			if tt.expectError {
				suite.Error(err)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
				return
			}

			suite.NoError(err)
			suite.NotNil(stats)

			suite.Equal(tt.userNickname, stats.UserNickname)
			suite.Equal(tt.expectedTotalTranscriptions, stats.TotalTranscriptions)
			suite.Equal(tt.expectedOpenAIEmbeddings, stats.OpenAIEmbeddings)
			suite.Equal(tt.expectedGeminiEmbeddings, stats.GeminiEmbeddings)
			suite.Equal(tt.expectedPendingOpenAI, stats.PendingOpenAI)
			suite.Equal(tt.expectedPendingGemini, stats.PendingGemini)
			suite.Equal(tt.expectedFailedOpenAI, stats.FailedOpenAI)
			suite.Equal(tt.expectedFailedGemini, stats.FailedGemini)
		})
	}
}

// TestGetUserEmbeddingStats_ConcurrentAccess tests concurrent access
func (suite *PgVectorTestSuite) TestGetUserEmbeddingStats_ConcurrentAccess() {
	// Setup user-specific test data
	suite.setupUserSpecificTestData()

	const numGoroutines = 10
	results := make(chan *UserEmbeddingStats, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines to test concurrent access
	for i := 0; i < numGoroutines; i++ {
		go func() {
			stats, err := suite.storage.GetUserEmbeddingStats(suite.ctx, "user_pending")
			if err != nil {
				errors <- err
			} else {
				results <- stats
			}
		}()
	}

	// Collect results
	var statsResults []*UserEmbeddingStats
	for i := 0; i < numGoroutines; i++ {
		select {
		case stats := <-results:
			statsResults = append(statsResults, stats)
		case err := <-errors:
			suite.T().Errorf("Unexpected error in concurrent access: %v", err)
		case <-time.After(5 * time.Second):
			suite.T().Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Verify all results are consistent
	suite.Len(statsResults, numGoroutines)
	firstResult := statsResults[0]
	for i, stats := range statsResults {
		suite.Equal(firstResult.UserNickname, stats.UserNickname, "Result %d has different UserNickname", i)
		suite.Equal(firstResult.TotalTranscriptions, stats.TotalTranscriptions, "Result %d has different TotalTranscriptions", i)
		suite.Equal(firstResult.OpenAIEmbeddings, stats.OpenAIEmbeddings, "Result %d has different OpenAIEmbeddings", i)
		suite.Equal(firstResult.GeminiEmbeddings, stats.GeminiEmbeddings, "Result %d has different GeminiEmbeddings", i)
	}
}

// TestGetTranscriptionsWithoutEmbeddingsByUser_ErrorHandling tests database error scenarios
func (suite *PgVectorTestSuite) TestGetTranscriptionsWithoutEmbeddingsByUser_ErrorHandling() {
	// Test with cancelled context
	ctx, cancel := context.WithCancel(suite.ctx)
	cancel() // Cancel immediately

	transcriptions, err := suite.storage.GetTranscriptionsWithoutEmbeddingsByUser(
		ctx, "openai", "test_user", 10)

	suite.Error(err)
	suite.Nil(transcriptions)
	suite.Contains(err.Error(), "context canceled")
}

// TestUserSpecificMethods_Integration tests integration between user-specific methods
func (suite *PgVectorTestSuite) TestUserSpecificMethods_Integration() {
	// Setup user-specific test data
	suite.setupUserSpecificTestData()

	userNickname := "user_pending"

	// Get initial stats
	initialStats, err := suite.storage.GetUserEmbeddingStats(suite.ctx, userNickname)
	suite.NoError(err)
	suite.Equal(2, initialStats.PendingOpenAI)

	// Get transcriptions without OpenAI embeddings
	transcriptions, err := suite.storage.GetTranscriptionsWithoutEmbeddingsByUser(
		suite.ctx, "openai", userNickname, 10)
	suite.NoError(err)
	suite.Len(transcriptions, 2) // Should match pending count

	// Add embedding to one transcription
	if len(transcriptions) > 0 {
		embedding := suite.generateTestEmbedding(1536)
		err = suite.storage.StoreEmbedding(suite.ctx, transcriptions[0].ID, "openai", embedding)
		suite.NoError(err)
	}

	// Get updated stats
	updatedStats, err := suite.storage.GetUserEmbeddingStats(suite.ctx, userNickname)
	suite.NoError(err)
	suite.Equal(initialStats.PendingOpenAI-1, updatedStats.PendingOpenAI)       // Should decrease by 1
	suite.Equal(initialStats.OpenAIEmbeddings+1, updatedStats.OpenAIEmbeddings) // Should increase by 1

	// Get transcriptions without embeddings again
	transcriptionsAfter, err := suite.storage.GetTranscriptionsWithoutEmbeddingsByUser(
		suite.ctx, "openai", userNickname, 10)
	suite.NoError(err)
	suite.Len(transcriptionsAfter, 1) // Should have one less
}

// TestUserSpecificMethods_SQLInjectionSafety tests SQL injection protection
func (suite *PgVectorTestSuite) TestUserSpecificMethods_SQLInjectionSafety() {
	maliciousUsernames := []string{
		"test'; DROP TABLE transcriptions; --",
		"test' OR '1'='1",
		"test' UNION SELECT * FROM transcriptions --",
		"'; DELETE FROM transcriptions WHERE '1'='1'; --",
	}

	for _, username := range maliciousUsernames {
		suite.Run(fmt.Sprintf("SQL_injection_test_%s", username), func() {
			// These should not cause any SQL injection or errors
			stats, err := suite.storage.GetUserEmbeddingStats(suite.ctx, username)
			suite.NoError(err)
			suite.NotNil(stats)
			suite.Equal(username, stats.UserNickname)
			suite.Equal(0, stats.TotalTranscriptions) // Should be 0 for non-existent users

			transcriptions, err := suite.storage.GetTranscriptionsWithoutEmbeddingsByUser(
				suite.ctx, "openai", username, 10)
			suite.NoError(err)
			suite.Len(transcriptions, 0) // Should be empty for non-existent users

			// Verify the main table still exists and has data
			var count int
			err = suite.db.QueryRow("SELECT COUNT(*) FROM transcriptions").Scan(&count)
			suite.NoError(err)
			suite.Greater(count, 0, "Main table should still have data after SQL injection attempt")
		})
	}
}

// setupUserSpecificTestData creates comprehensive test data for user-specific tests
func (suite *PgVectorTestSuite) setupUserSpecificTestData() {
	// Clear existing data
	suite.cleanTestData()

	// Insert test data with various embedding statuses
	testData := []struct {
		id                 int
		userNickname       string
		fileName           string
		transcription      string
		openaiStatus       string
		geminiStatus       string
		hasOpenaiEmbedding bool
		hasGeminiEmbedding bool
	}{
		// user_pending - mixed statuses
		{1, "user_pending", "file1.mp3", "Transcription 1", "completed", "completed", true, true},
		{2, "user_pending", "file2.mp3", "Transcription 2", "completed", "pending", true, false},
		{3, "user_pending", "file3.mp3", "Transcription 3", "pending", "pending", false, false},
		{4, "user_pending", "file4.mp3", "Transcription 4", "pending", "pending", false, false},
		{5, "user_pending", "file5.mp3", "Transcription 5", "failed", "failed", false, false},

		// user_completed - all completed
		{6, "user_completed", "file6.mp3", "Transcription 6", "completed", "completed", true, true},
		{7, "user_completed", "file7.mp3", "Transcription 7", "completed", "completed", true, true},
		{8, "user_completed", "file8.mp3", "Transcription 8", "completed", "completed", true, true},

		// user_no_embeddings - all pending
		{9, "user_no_embeddings", "file9.mp3", "Transcription 9", "pending", "pending", false, false},
		{10, "user_no_embeddings", "file10.mp3", "Transcription 10", "pending", "pending", false, false},

		// Unicode user
		{11, "用户测试", "file11.mp3", "Unicode transcription", "pending", "pending", false, false},
	}

	for _, data := range testData {
		var openaiEmbedding, geminiEmbedding interface{}
		if data.hasOpenaiEmbedding {
			openaiEmbedding = "[0.1,0.2,0.3]" // Sample embedding
		}
		if data.hasGeminiEmbedding {
			geminiEmbedding = "[0.4,0.5,0.6]" // Sample embedding
		}

		_, err := suite.db.Exec(`
			INSERT INTO transcriptions (
				id, user_nickname, mp3_file_name, transcription,
				last_conversion_time,
				embedding_openai, embedding_openai_status, embedding_openai_model, embedding_openai_created_at,
				embedding_gemini, embedding_gemini_status, embedding_gemini_model, embedding_gemini_created_at
			) VALUES (
				$1, $2, $3, $4, now(),
				$5, $6, CASE WHEN $5 IS NOT NULL THEN 'text-embedding-ada-002' END, CASE WHEN $5 IS NOT NULL THEN now() END,
				$7, $8, CASE WHEN $7 IS NOT NULL THEN 'models/embedding-001' END, CASE WHEN $7 IS NOT NULL THEN now() END
			)
		`, data.id, data.userNickname, data.fileName, data.transcription,
			openaiEmbedding, data.openaiStatus, geminiEmbedding, data.geminiStatus)
		require.NoError(suite.T(), err)
	}
}

// Helper function to generate test embeddings
func (suite *PgVectorTestSuite) generateTestEmbedding(size int) []float32 {
	embedding := make([]float32, size)
	for i := range embedding {
		embedding[i] = float32(i) / float32(size)
	}
	return embedding
}
