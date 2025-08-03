//go:build integration
// +build integration

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

// PgVectorIntegrationTestSuite groups all PostgreSQL vector storage integration tests
type PgVectorIntegrationTestSuite struct {
	suite.Suite
	db      *sql.DB
	storage *PgVectorStorage
	ctx     context.Context
}

// SetupSuite runs once before all tests
func (suite *PgVectorIntegrationTestSuite) SetupSuite() {
	// Get PostgreSQL connection from environment
	pgURL := os.Getenv("POSTGRES_TEST_URL")
	if pgURL == "" {
		suite.T().Skip("Skipping PostgreSQL integration tests: POSTGRES_TEST_URL not set")
	}

	// Setup test database
	db, err := sql.Open("postgres", pgURL)
	require.NoError(suite.T(), err)
	
	err = db.Ping()
	require.NoError(suite.T(), err)
	
	suite.db = db
	suite.storage = NewPgVectorStorage(db)
	suite.ctx = context.Background()
	
	// Create schema
	suite.createTestSchema(db)
}

// TearDownSuite runs once after all tests
func (suite *PgVectorIntegrationTestSuite) TearDownSuite() {
	if suite.storage != nil {
		suite.storage.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *PgVectorIntegrationTestSuite) SetupTest() {
	// Clean data before each test
	suite.cleanTestData()
	suite.seedTestData()
}

// createTestSchema creates necessary tables and extensions for testing
func (suite *PgVectorIntegrationTestSuite) createTestSchema(db *sql.DB) {
	// Create pgvector extension
	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	require.NoError(suite.T(), err)
	
	// Create test table with vector columns
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS transcriptions (
			id SERIAL PRIMARY KEY,
			transcription TEXT NOT NULL,
			user_nickname VARCHAR(255) NOT NULL,
			embedding_openai vector(1536),
			embedding_gemini vector(768),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(suite.T(), err)
	
	// Create indexes for vector similarity search
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_embedding_openai ON transcriptions USING ivfflat (embedding_openai vector_cosine_ops)`)
	require.NoError(suite.T(), err)
	
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_embedding_gemini ON transcriptions USING ivfflat (embedding_gemini vector_cosine_ops)`)
	require.NoError(suite.T(), err)
}

// cleanTestData removes all test data
func (suite *PgVectorIntegrationTestSuite) cleanTestData() {
	_, err := suite.db.Exec("TRUNCATE TABLE transcriptions RESTART IDENTITY CASCADE")
	require.NoError(suite.T(), err)
}

// seedTestData adds initial test data
func (suite *PgVectorIntegrationTestSuite) seedTestData() {
	// Add test transcriptions
	testData := []struct {
		id           int
		text         string
		userNickname string
	}{
		{1, "Hello world", "test_user_1"},
		{2, "Golang is awesome", "test_user_1"},
		{3, "Testing vectors", "test_user_2"},
	}
	
	for _, data := range testData {
		_, err := suite.db.Exec(
			"INSERT INTO transcriptions (id, transcription, user_nickname) VALUES ($1, $2, $3)",
			data.id, data.text, data.userNickname,
		)
		require.NoError(suite.T(), err)
	}
}

// Move all the actual test methods here from pgvector_test.go
// For example:

func (suite *PgVectorIntegrationTestSuite) TestStoreAndRetrieveSingleEmbedding() {
	// Test implementation...
}

func (suite *PgVectorIntegrationTestSuite) TestStoreDualEmbeddings() {
	// Test implementation...
}

// ... other integration tests ...

// TestPgVectorIntegrationSuite runs the test suite
func TestPgVectorIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PgVectorIntegrationTestSuite))
}