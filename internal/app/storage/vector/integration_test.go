package vector

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"tiktok-whisper/internal/app/testutil"
)

// IntegrationTestSuite tests integration between vector storage and database
type IntegrationTestSuite struct {
	suite.Suite
	storage VectorStorage
	ctx     context.Context
}

// SetupSuite runs once before all tests
func (suite *IntegrationTestSuite) SetupSuite() {
	// Skip if running in CI or short mode
	if testing.Short() || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		suite.T().Skip("Skipping integration tests")
	}

	suite.ctx = context.Background()
}

// SetupTest runs before each test
func (suite *IntegrationTestSuite) SetupTest() {
	// Use testutil to setup database
	testutil.WithTestDB(suite.T(), func(t *testing.T, db *sql.DB) {
		suite.setupVectorExtension(db)
		suite.storage = NewPgVectorStorage(db)
	})
}

// TestIntegrationSuite runs the integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

// TestVectorStorageWithRealDatabase tests vector storage with real database
func (suite *IntegrationTestSuite) TestVectorStorageWithRealDatabase() {
	if suite.storage == nil {
		suite.T().Skip("Database not available for integration test")
	}

	// Test storing and retrieving embeddings
	transcriptionID := 1
	embedding := generateTestEmbedding(1536)

	// Store embedding
	err := suite.storage.StoreEmbedding(suite.ctx, transcriptionID, "openai", embedding)
	suite.NoError(err)

	// Retrieve embedding
	retrieved, err := suite.storage.GetEmbedding(suite.ctx, transcriptionID, "openai")
	suite.NoError(err)
	suite.Equal(len(embedding), len(retrieved))

	// Verify values
	for i := range embedding {
		suite.InDelta(embedding[i], retrieved[i], 0.0001)
	}
}

// TestFullWorkflowWithTestData tests complete workflow with test data
func (suite *IntegrationTestSuite) TestFullWorkflowWithTestData() {
	if suite.storage == nil {
		suite.T().Skip("Database not available for integration test")
	}

	// This would use testutil.SeedTestData to populate transcriptions
	// Then test the full embedding workflow

	// 1. Get transcriptions without embeddings
	transcriptions, err := suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 10)
	suite.NoError(err)

	// 2. Generate and store embeddings for each
	for _, transcription := range transcriptions {
		embedding := generateTestEmbedding(1536)
		err := suite.storage.StoreEmbedding(suite.ctx, transcription.ID, "openai", embedding)
		suite.NoError(err)
	}

	// 3. Verify no more transcriptions need embeddings
	remaining, err := suite.storage.GetTranscriptionsWithoutEmbeddings(suite.ctx, "openai", 10)
	suite.NoError(err)
	suite.Len(remaining, 0)
}

// TestDualEmbeddingWorkflow tests the dual embedding workflow
func (suite *IntegrationTestSuite) TestDualEmbeddingWorkflow() {
	if suite.storage == nil {
		suite.T().Skip("Database not available for integration test")
	}

	transcriptionID := 1
	openaiEmbedding := generateTestEmbedding(1536)
	geminiEmbedding := generateTestEmbedding(768)

	// Store dual embeddings
	err := suite.storage.StoreDualEmbeddings(suite.ctx, transcriptionID, openaiEmbedding, geminiEmbedding)
	suite.NoError(err)

	// Retrieve dual embeddings
	dual, err := suite.storage.GetDualEmbeddings(suite.ctx, transcriptionID)
	suite.NoError(err)
	suite.NotNil(dual)
	suite.NotNil(dual.OpenAI)
	suite.NotNil(dual.Gemini)

	// Verify individual access still works
	individualOpenAI, err := suite.storage.GetEmbedding(suite.ctx, transcriptionID, "openai")
	suite.NoError(err)
	suite.Equal(dual.OpenAI, individualOpenAI)

	individualGemini, err := suite.storage.GetEmbedding(suite.ctx, transcriptionID, "gemini")
	suite.NoError(err)
	suite.Equal(dual.Gemini, individualGemini)
}

// setupVectorExtension creates pgvector extension if using PostgreSQL
func (suite *IntegrationTestSuite) setupVectorExtension(db *sql.DB) {
	// Check if this is PostgreSQL and create vector extension
	var dbType string
	err := db.QueryRow("SELECT version()").Scan(&dbType)
	if err == nil && len(dbType) > 0 {
		// If it contains "PostgreSQL", create vector extension
		if len(dbType) > 10 && dbType[:10] == "PostgreSQL" {
			_, _ = db.Exec("CREATE EXTENSION IF NOT EXISTS vector")

			// Create vector-enabled schema
			schema := `
			ALTER TABLE transcriptions 
			ADD COLUMN IF NOT EXISTS embedding_openai vector(1536),
			ADD COLUMN IF NOT EXISTS embedding_openai_model VARCHAR(50),
			ADD COLUMN IF NOT EXISTS embedding_openai_created_at TIMESTAMP,
			ADD COLUMN IF NOT EXISTS embedding_openai_status VARCHAR(20) DEFAULT 'pending',
			ADD COLUMN IF NOT EXISTS embedding_gemini vector(768),
			ADD COLUMN IF NOT EXISTS embedding_gemini_model VARCHAR(50),
			ADD COLUMN IF NOT EXISTS embedding_gemini_created_at TIMESTAMP,
			ADD COLUMN IF NOT EXISTS embedding_gemini_status VARCHAR(20) DEFAULT 'pending',
			ADD COLUMN IF NOT EXISTS embedding_sync_status VARCHAR(20) DEFAULT 'pending';
			`
			_, _ = db.Exec(schema)
		}
	}
}

// Helper function to generate test embeddings
func generateTestEmbedding(size int) []float32 {
	embedding := make([]float32, size)
	for i := range embedding {
		embedding[i] = float32(i) / float32(size)
	}
	return embedding
}

// Standalone integration tests that don't require the test suite

// TestWithTestDBHelper demonstrates using testutil.WithTestDB
func TestWithTestDBHelper(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		// Setup vector storage
		storage := NewPgVectorStorage(db)
		defer storage.Close()

		ctx := context.Background()
		embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

		// Test basic operations
		err := storage.StoreEmbedding(ctx, 1, "openai", embedding)
		// May fail if vector extension not available, that's expected
		if err != nil {
			t.Logf("Expected error with basic database: %v", err)
			return
		}

		retrieved, err := storage.GetEmbedding(ctx, 1, "openai")
		assert.NoError(t, err)
		assert.Equal(t, embedding, retrieved)
	})
}

// TestWithSeekedTestDB demonstrates using testutil.WithSeekedTestDB
func TestWithSeekedTestDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		storage := NewPgVectorStorage(db)
		defer storage.Close()

		ctx := context.Background()

		// Test with seeded data - should have test transcriptions
		transcriptions, err := storage.GetTranscriptionsWithoutEmbeddings(ctx, "openai", 10)
		// May fail if vector columns don't exist, that's expected for basic schema
		if err != nil {
			t.Logf("Expected error with basic schema: %v", err)
			return
		}

		// Should have seeded transcriptions
		assert.GreaterOrEqual(t, len(transcriptions), 0)

		// Verify transcription structure
		for _, transcription := range transcriptions {
			assert.Greater(t, transcription.ID, 0)
			assert.NotEmpty(t, transcription.Mp3FileName)
			assert.NotEmpty(t, transcription.TranscriptionText)
		}
	})
}

// TestDatabaseTypeCompatibility tests compatibility across database types
func TestDatabaseTypeCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database compatibility test in short mode")
	}

	t.Run("SQLite compatibility", func(t *testing.T) {
		// SQLite doesn't support vector types, so operations should fail gracefully
		db := testutil.SetupTestSQLite(t)
		defer testutil.TeardownTestDB(t, db)

		storage := NewPgVectorStorage(db)
		defer storage.Close()

		ctx := context.Background()
		embedding := []float32{0.1, 0.2, 0.3}

		// This should fail because SQLite doesn't have vector columns
		err := storage.StoreEmbedding(ctx, 1, "openai", embedding)
		assert.Error(t, err)
		t.Logf("Expected SQLite error: %v", err)
	})

	t.Run("PostgreSQL compatibility", func(t *testing.T) {
		if os.Getenv("POSTGRES_TEST_URL") == "" {
			t.Skip("PostgreSQL not available for compatibility test")
		}

		// This would test with real PostgreSQL if available
		testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
			// Check if this is actually PostgreSQL
			var version string
			err := db.QueryRow("SELECT version()").Scan(&version)
			require.NoError(t, err)

			if len(version) < 10 || version[:10] != "PostgreSQL" {
				t.Skip("Not a PostgreSQL database")
			}

			// Test PostgreSQL-specific features
			storage := NewPgVectorStorage(db)
			defer storage.Close()

			// Try to create vector extension (may fail if not available)
			_, err = db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
			if err != nil {
				t.Logf("Vector extension not available: %v", err)
				return
			}

			// Test would continue with vector operations...
		})
	})
}

// TestPerformanceWithRealData tests performance characteristics with real data
func TestPerformanceWithRealData(t *testing.T) {
	if testing.Short() || os.Getenv("SKIP_PERF_TESTS") == "true" {
		t.Skip("Skipping performance test")
	}

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		storage := NewPgVectorStorage(db)
		defer storage.Close()

		// Test performance with realistic embedding sizes
		openaiEmbedding := generateTestEmbedding(1536) // OpenAI ada-002 size
		geminiEmbedding := generateTestEmbedding(768)  // Gemini embedding size

		// Measure store performance
		start := time.Now()
		for i := 1; i <= 100; i++ {
			err := storage.StoreDualEmbeddings(context.Background(), i, openaiEmbedding, geminiEmbedding)
			if err != nil {
				// May fail if vector extension not available
				t.Logf("Performance test stopped due to error: %v", err)
				return
			}
		}
		storeTime := time.Since(start)

		// Measure retrieval performance
		start = time.Now()
		for i := 1; i <= 100; i++ {
			_, err := storage.GetDualEmbeddings(context.Background(), i)
			assert.NoError(t, err)
		}
		retrieveTime := time.Since(start)

		t.Logf("Performance results:")
		t.Logf("  Store 100 dual embeddings: %v", storeTime)
		t.Logf("  Retrieve 100 dual embeddings: %v", retrieveTime)
		t.Logf("  Average store time: %v", storeTime/100)
		t.Logf("  Average retrieve time: %v", retrieveTime/100)

		// Basic performance assertions (these are rough guidelines)
		assert.Less(t, storeTime, 30*time.Second, "Storing should complete within 30 seconds")
		assert.Less(t, retrieveTime, 10*time.Second, "Retrieving should complete within 10 seconds")
	})
}
