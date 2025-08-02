package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tiktok-whisper/internal/app/embedding/orchestrator"
	"tiktok-whisper/internal/app/embedding/provider"
	"tiktok-whisper/internal/app/storage/vector"
	"tiktok-whisper/internal/app/testutil"
)

// TestUserSpecificEmbeddingWorkflow tests the complete user-specific embedding workflow
func TestUserSpecificEmbeddingWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test database
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Setup components
	storage := vector.NewPgVectorStorage(db)
	defer storage.Close()

	// Create mock providers
	mockOpenAI := &MockEmbeddingProvider{
		ProviderName:  "openai",
		EmbeddingSize: 1536,
	}
	mockGemini := &MockEmbeddingProvider{
		ProviderName:  "gemini",
		EmbeddingSize: 768,
	}

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	logger := testutil.NewMockLogger()
	embeddingOrchestrator := orchestrator.NewEmbeddingOrchestrator(providers, storage, logger)
	batchProcessor := orchestrator.NewBatchProcessor(embeddingOrchestrator, storage, logger)

	// Test user
	userNickname := "integration_test_user"

	t.Run("complete_user_workflow", func(t *testing.T) {
		// Step 1: Setup test transcriptions for the user
		transcriptionIDs := setupUserTranscriptions(t, db, userNickname, 5)

		// Step 2: Verify initial state - no embeddings
		stats, err := storage.GetUserEmbeddingStats(context.Background(), userNickname)
		require.NoError(t, err)
		assert.Equal(t, 5, stats.TotalTranscriptions)
		assert.Equal(t, 0, stats.OpenAIEmbeddings)
		assert.Equal(t, 0, stats.GeminiEmbeddings)
		assert.Equal(t, 5, stats.PendingOpenAI)
		assert.Equal(t, 5, stats.PendingGemini)

		// Step 3: Process embeddings for the user
		ctx := context.Background()
		err = batchProcessor.ProcessUserTranscriptions(ctx, userNickname, []string{"openai", "gemini"}, 2)
		require.NoError(t, err)

		// Step 4: Verify all embeddings were generated
		finalStats, err := storage.GetUserEmbeddingStats(ctx, userNickname)
		require.NoError(t, err)
		assert.Equal(t, 5, finalStats.TotalTranscriptions)
		assert.Equal(t, 5, finalStats.OpenAIEmbeddings)
		assert.Equal(t, 5, finalStats.GeminiEmbeddings)
		assert.Equal(t, 0, finalStats.PendingOpenAI)
		assert.Equal(t, 0, finalStats.PendingGemini)

		// Step 5: Verify embeddings are retrievable
		for _, id := range transcriptionIDs {
			dualEmbedding, err := storage.GetDualEmbeddings(ctx, id)
			require.NoError(t, err)
			assert.NotNil(t, dualEmbedding.OpenAI)
			assert.NotNil(t, dualEmbedding.Gemini)
			assert.Len(t, dualEmbedding.OpenAI, 1536)
			assert.Len(t, dualEmbedding.Gemini, 768)
		}

		// Step 6: Verify no more transcriptions need processing
		transcriptionsOpenAI, err := storage.GetTranscriptionsWithoutEmbeddingsByUser(ctx, "openai", userNickname, 10)
		require.NoError(t, err)
		assert.Empty(t, transcriptionsOpenAI)

		transcriptionsGemini, err := storage.GetTranscriptionsWithoutEmbeddingsByUser(ctx, "gemini", userNickname, 10)
		require.NoError(t, err)
		assert.Empty(t, transcriptionsGemini)
	})
}

// TestUserSpecificEmbeddingWorkflow_PartialProcessing tests scenarios with partial processing
func TestUserSpecificEmbeddingWorkflow_PartialProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test database
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Setup components
	storage := vector.NewPgVectorStorage(db)
	defer storage.Close()

	// Create mock provider that fails for specific transcriptions
	failingProvider := &MockEmbeddingProvider{
		ProviderName:  "openai",
		EmbeddingSize: 1536,
		FailForIDs:    map[int]bool{2: true, 4: true}, // Fail for transcriptions 2 and 4
	}

	providers := map[string]provider.EmbeddingProvider{
		"openai": failingProvider,
	}

	logger := testutil.NewMockLogger()
	embeddingOrchestrator := orchestrator.NewEmbeddingOrchestrator(providers, storage, logger)
	batchProcessor := orchestrator.NewBatchProcessor(embeddingOrchestrator, storage, logger)

	userNickname := "partial_test_user"

	t.Run("partial_processing_with_failures", func(t *testing.T) {
		// Step 1: Setup test transcriptions
		transcriptionIDs := setupUserTranscriptions(t, db, userNickname, 5)

		// Step 2: Process embeddings (some will fail)
		ctx := context.Background()
		err := batchProcessor.ProcessUserTranscriptions(ctx, userNickname, []string{"openai"}, 3)
		require.Error(t, err) // Should fail because some transcriptions failed

		// Step 3: Verify partial success
		stats, err := storage.GetUserEmbeddingStats(ctx, userNickname)
		require.NoError(t, err)
		assert.Equal(t, 5, stats.TotalTranscriptions)
		assert.Equal(t, 3, stats.OpenAIEmbeddings) // 3 succeeded
		assert.Equal(t, 2, stats.FailedOpenAI)     // 2 failed

		// Step 4: Verify specific embeddings exist
		for _, id := range transcriptionIDs {
			_, err := storage.GetEmbedding(ctx, id, "openai")
			if id == transcriptionIDs[1] || id == transcriptionIDs[3] { // IDs 2 and 4 (0-indexed)
				assert.Error(t, err, "Should not have embedding for failed transcription %d", id)
			} else {
				assert.NoError(t, err, "Should have embedding for successful transcription %d", id)
			}
		}

		// Step 5: Retry processing should work for failed items
		failingProvider.FailForIDs = map[int]bool{} // Remove failures
		err = batchProcessor.ProcessUserTranscriptions(ctx, userNickname, []string{"openai"}, 3)
		require.NoError(t, err)

		// Step 6: Verify all embeddings now exist
		finalStats, err := storage.GetUserEmbeddingStats(ctx, userNickname)
		require.NoError(t, err)
		assert.Equal(t, 5, finalStats.OpenAIEmbeddings)
		assert.Equal(t, 0, finalStats.FailedOpenAI)
	})
}

// TestUserSpecificEmbeddingWorkflow_MultipleUsers tests processing for multiple users
func TestUserSpecificEmbeddingWorkflow_MultipleUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test database
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Setup components
	storage := vector.NewPgVectorStorage(db)
	defer storage.Close()

	mockProvider := &MockEmbeddingProvider{
		ProviderName:  "openai",
		EmbeddingSize: 1536,
	}

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockProvider,
	}

	logger := testutil.NewMockLogger()
	embeddingOrchestrator := orchestrator.NewEmbeddingOrchestrator(providers, storage, logger)
	batchProcessor := orchestrator.NewBatchProcessor(embeddingOrchestrator, storage, logger)

	users := []string{"user_a", "user_b", "user_c"}
	transcriptionsPerUser := 3

	t.Run("multiple_users_isolated_processing", func(t *testing.T) {
		// Step 1: Setup transcriptions for multiple users
		userTranscriptions := make(map[string][]int)
		for _, user := range users {
			userTranscriptions[user] = setupUserTranscriptions(t, db, user, transcriptionsPerUser)
		}

		ctx := context.Background()

		// Step 2: Process embeddings for each user separately
		for _, user := range users {
			err := batchProcessor.ProcessUserTranscriptions(ctx, user, []string{"openai"}, 2)
			require.NoError(t, err, "Processing should succeed for user %s", user)

			// Verify only this user's transcriptions were processed
			stats, err := storage.GetUserEmbeddingStats(ctx, user)
			require.NoError(t, err)
			assert.Equal(t, transcriptionsPerUser, stats.TotalTranscriptions)
			assert.Equal(t, transcriptionsPerUser, stats.OpenAIEmbeddings)
			assert.Equal(t, 0, stats.PendingOpenAI)
		}

		// Step 3: Verify cross-user isolation
		for _, user := range users {
			transcriptions, err := storage.GetTranscriptionsWithoutEmbeddingsByUser(ctx, "openai", user, 10)
			require.NoError(t, err)
			assert.Empty(t, transcriptions, "User %s should have no pending transcriptions", user)
		}

		// Step 4: Verify embeddings exist for all users
		for user, transcriptionIDs := range userTranscriptions {
			for _, id := range transcriptionIDs {
				embedding, err := storage.GetEmbedding(ctx, id, "openai")
				require.NoError(t, err, "User %s transcription %d should have embedding", user, id)
				assert.Len(t, embedding, 1536)
			}
		}
	})
}

// TestUserSpecificEmbeddingWorkflow_UnicodeUsers tests Unicode user handling
func TestUserSpecificEmbeddingWorkflow_UnicodeUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test database
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Setup components
	storage := vector.NewPgVectorStorage(db)
	defer storage.Close()

	mockProvider := &MockEmbeddingProvider{
		ProviderName:  "openai",
		EmbeddingSize: 1536,
	}

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockProvider,
	}

	logger := testutil.NewMockLogger()
	embeddingOrchestrator := orchestrator.NewEmbeddingOrchestrator(providers, storage, logger)
	batchProcessor := orchestrator.NewBatchProcessor(embeddingOrchestrator, storage, logger)

	unicodeUsers := []string{
		"用户测试",
		"Пользователь",
		"ユーザーテスト",
		"المستخدم",
	}

	t.Run("unicode_users_processing", func(t *testing.T) {
		ctx := context.Background()

		for _, user := range unicodeUsers {
			// Setup transcriptions for Unicode user
			transcriptionIDs := setupUserTranscriptions(t, db, user, 2)

			// Process embeddings
			err := batchProcessor.ProcessUserTranscriptions(ctx, user, []string{"openai"}, 2)
			require.NoError(t, err, "Processing should succeed for Unicode user %s", user)

			// Verify processing
			stats, err := storage.GetUserEmbeddingStats(ctx, user)
			require.NoError(t, err)
			assert.Equal(t, 2, stats.TotalTranscriptions)
			assert.Equal(t, 2, stats.OpenAIEmbeddings)
			assert.Equal(t, 0, stats.PendingOpenAI)

			// Verify embeddings exist
			for _, id := range transcriptionIDs {
				embedding, err := storage.GetEmbedding(ctx, id, "openai")
				require.NoError(t, err, "Unicode user %s transcription %d should have embedding", user, id)
				assert.Len(t, embedding, 1536)
			}
		}
	})
}

// Helper functions

// setupIntegrationTestDB creates and configures a test database
func setupIntegrationTestDB(t *testing.T) *sql.DB {
	// Use environment variable or default connection
	connectionString := "postgres://postgres:postgres@localhost/postgres?sslmode=disable"
	if testURL := os.Getenv("POSTGRES_TEST_URL"); testURL != "" {
		connectionString = testURL
	}

	db, err := sql.Open("postgres", connectionString)
	require.NoError(t, err)

	err = db.Ping()
	require.NoError(t, err)

	// Create schema if needed
	createIntegrationTestSchema(t, db)

	// Clean existing data
	cleanIntegrationTestData(t, db)

	return db
}

// createIntegrationTestSchema creates the required database schema
func createIntegrationTestSchema(t *testing.T, db *sql.DB) {
	// Create pgvector extension
	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	require.NoError(t, err)

	// Create transcriptions table with embedding columns
	schema := `
	CREATE TABLE IF NOT EXISTS transcriptions (
		id SERIAL PRIMARY KEY,
		user_nickname VARCHAR(255),
		mp3_file_name VARCHAR(255) NOT NULL,
		transcription TEXT NOT NULL,
		last_conversion_time TIMESTAMP DEFAULT NOW(),
		has_error INTEGER DEFAULT 0,
		error_message TEXT DEFAULT '',
		
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
	require.NoError(t, err)
}

// cleanIntegrationTestData removes all test data
func cleanIntegrationTestData(t *testing.T, db *sql.DB) {
	_, err := db.Exec("TRUNCATE TABLE transcriptions RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// setupUserTranscriptions creates test transcriptions for a user
func setupUserTranscriptions(t *testing.T, db *sql.DB, userNickname string, count int) []int {
	var transcriptionIDs []int

	for i := 0; i < count; i++ {
		query := `
			INSERT INTO transcriptions (user_nickname, mp3_file_name, transcription, last_conversion_time)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`

		var id int
		err := db.QueryRow(query,
			userNickname,
			fmt.Sprintf("test_file_%d.mp3", i+1),
			fmt.Sprintf("Test transcription %d for user %s", i+1, userNickname),
			time.Now(),
		).Scan(&id)
		require.NoError(t, err)

		transcriptionIDs = append(transcriptionIDs, id)
	}

	return transcriptionIDs
}

// MockEmbeddingProvider for integration testing
type MockEmbeddingProvider struct {
	ProviderName  string
	EmbeddingSize int
	FailForIDs    map[int]bool
}

func (m *MockEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Extract transcription ID from context or text for failure simulation
	// This is a simplified approach - in practice you'd need to pass the ID through context

	// Check if we should fail for certain patterns
	if m.FailForIDs != nil {
		for failID := range m.FailForIDs {
			if fmt.Sprintf("Test transcription %d", failID) == text[:len(fmt.Sprintf("Test transcription %d", failID))] {
				return nil, fmt.Errorf("simulated failure for transcription %d", failID)
			}
		}
	}

	// Generate a simple test embedding
	embedding := make([]float32, m.EmbeddingSize)
	for i := range embedding {
		embedding[i] = float32(i) / float32(m.EmbeddingSize)
	}
	return embedding, nil
}

func (m *MockEmbeddingProvider) GetProviderInfo() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:      m.ProviderName,
		Model:     "mock-model",
		Dimension: m.EmbeddingSize,
	}
}
