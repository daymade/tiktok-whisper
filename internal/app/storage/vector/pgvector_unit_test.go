package vector

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPgVectorStorage_GetEmbedding_Unit tests GetEmbedding with mocked database
func TestPgVectorStorage_GetEmbedding_Unit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()

	tests := []struct {
		name         string
		transcriptID int
		provider     string
		setupMock    func()
		expectedErr  bool
		expectedLen  int
	}{
		{
			name:         "successful_openai_retrieval",
			transcriptID: 1,
			provider:     "openai",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"embedding_openai"}).
					AddRow("[0.1,0.2,0.3]")
				mock.ExpectQuery("SELECT embedding_openai FROM transcriptions WHERE id = \\$1 AND embedding_openai IS NOT NULL").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedErr: false,
			expectedLen: 3,
		},
		{
			name:         "successful_gemini_retrieval",
			transcriptID: 2,
			provider:     "gemini",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"embedding_gemini"}).
					AddRow("[0.4,0.5,0.6]")
				mock.ExpectQuery("SELECT embedding_gemini FROM transcriptions WHERE id = \\$1 AND embedding_gemini IS NOT NULL").
					WithArgs(2).
					WillReturnRows(rows)
			},
			expectedErr: false,
			expectedLen: 3,
		},
		{
			name:         "embedding_not_found",
			transcriptID: 3,
			provider:     "openai",
			setupMock: func() {
				mock.ExpectQuery("SELECT embedding_openai FROM transcriptions WHERE id = \\$1 AND embedding_openai IS NOT NULL").
					WithArgs(3).
					WillReturnError(sql.ErrNoRows)
			},
			expectedErr: true,
			expectedLen: 0,
		},
		{
			name:         "invalid_provider",
			transcriptID: 1,
			provider:     "invalid",
			setupMock:    func() {},
			expectedErr:  true,
			expectedLen:  0,
		},
		{
			name:         "database_error",
			transcriptID: 1,
			provider:     "openai",
			setupMock: func() {
				mock.ExpectQuery("SELECT embedding_openai FROM transcriptions WHERE id = \\$1 AND embedding_openai IS NOT NULL").
					WithArgs(1).
					WillReturnError(errors.New("database connection lost"))
			},
			expectedErr: true,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			embedding, err := storage.GetEmbedding(ctx, tt.transcriptID, tt.provider)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, embedding)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, embedding)
				assert.Len(t, embedding, tt.expectedLen)
			}

			// Ensure all expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

// TestPgVectorStorage_StoreEmbedding_Unit tests StoreEmbedding with mocked database
func TestPgVectorStorage_StoreEmbedding_Unit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()

	tests := []struct {
		name         string
		transcriptID int
		provider     string
		embedding    []float32
		setupMock    func()
		expectedErr  bool
	}{
		{
			name:         "successful_openai_store",
			transcriptID: 1,
			provider:     "openai",
			embedding:    []float32{0.1, 0.2, 0.3},
			setupMock: func() {
				mock.ExpectExec("UPDATE transcriptions SET embedding_openai = \\$1, embedding_openai_model = \\$2, embedding_openai_created_at = now\\(\\), embedding_openai_status = 'completed' WHERE id = \\$3").
					WithArgs(sqlmock.AnyArg(), "text-embedding-ada-002", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: false,
		},
		{
			name:         "successful_gemini_store",
			transcriptID: 2,
			provider:     "gemini",
			embedding:    []float32{0.4, 0.5, 0.6},
			setupMock: func() {
				mock.ExpectExec("UPDATE transcriptions SET embedding_gemini = \\$1, embedding_gemini_model = \\$2, embedding_gemini_created_at = now\\(\\), embedding_gemini_status = 'completed' WHERE id = \\$3").
					WithArgs(sqlmock.AnyArg(), "models/embedding-001", 2).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: false,
		},
		{
			name:         "invalid_provider",
			transcriptID: 1,
			provider:     "invalid",
			embedding:    []float32{0.1, 0.2, 0.3},
			setupMock:    func() {},
			expectedErr:  true,
		},
		{
			name:         "database_error",
			transcriptID: 1,
			provider:     "openai",
			embedding:    []float32{0.1, 0.2, 0.3},
			setupMock: func() {
				mock.ExpectExec("UPDATE transcriptions SET embedding_openai = \\$1, embedding_openai_model = \\$2, embedding_openai_created_at = now\\(\\), embedding_openai_status = 'completed' WHERE id = \\$3").
					WithArgs(sqlmock.AnyArg(), "text-embedding-ada-002", 1).
					WillReturnError(errors.New("database write failed"))
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := storage.StoreEmbedding(ctx, tt.transcriptID, tt.provider, tt.embedding)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Ensure all expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

// TestPgVectorStorage_GetTranscriptionsWithoutEmbeddings_Unit tests with mocked database
func TestPgVectorStorage_GetTranscriptionsWithoutEmbeddings_Unit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()

	tests := []struct {
		name          string
		provider      string
		limit         int
		setupMock     func()
		expectedCount int
		expectedErr   bool
	}{
		{
			name:     "openai_missing_embeddings",
			provider: "openai",
			limit:    10,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"id", "user_nickname", "mp3_file_name", "transcription", "last_conversion_time"}).
					AddRow(1, "user1", "file1.mp3", "Hello world", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)).
					AddRow(2, "user2", "file2.mp3", "Test transcription", time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC))
				mock.ExpectQuery("SELECT id, user_nickname, mp3_file_name, transcription, last_conversion_time FROM transcriptions WHERE embedding_openai IS NULL AND embedding_openai_status = 'pending' ORDER BY id LIMIT \\$1").
					WithArgs(10).
					WillReturnRows(rows)
			},
			expectedCount: 2,
			expectedErr:   false,
		},
		{
			name:     "gemini_missing_embeddings",
			provider: "gemini",
			limit:    5,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"id", "user_nickname", "mp3_file_name", "transcription", "last_conversion_time"}).
					AddRow(3, "user3", "file3.mp3", "Another test", time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))
				mock.ExpectQuery("SELECT id, user_nickname, mp3_file_name, transcription, last_conversion_time FROM transcriptions WHERE embedding_gemini IS NULL AND embedding_gemini_status = 'pending' ORDER BY id LIMIT \\$1").
					WithArgs(5).
					WillReturnRows(rows)
			},
			expectedCount: 1,
			expectedErr:   false,
		},
		{
			name:     "no_missing_embeddings",
			provider: "openai",
			limit:    10,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"id", "user_nickname", "mp3_file_name", "transcription", "last_conversion_time"})
				mock.ExpectQuery("SELECT id, user_nickname, mp3_file_name, transcription, last_conversion_time FROM transcriptions WHERE embedding_openai IS NULL AND embedding_openai_status = 'pending' ORDER BY id LIMIT \\$1").
					WithArgs(10).
					WillReturnRows(rows)
			},
			expectedCount: 0,
			expectedErr:   false,
		},
		{
			name:     "database_error",
			provider: "openai",
			limit:    10,
			setupMock: func() {
				mock.ExpectQuery("SELECT id, user_nickname, mp3_file_name, transcription, last_conversion_time FROM transcriptions WHERE embedding_openai IS NULL AND embedding_openai_status = 'pending' ORDER BY id LIMIT \\$1").
					WithArgs(10).
					WillReturnError(errors.New("query failed"))
			},
			expectedCount: 0,
			expectedErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			transcriptions, err := storage.GetTranscriptionsWithoutEmbeddings(ctx, tt.provider, tt.limit)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, transcriptions)
			} else {
				assert.NoError(t, err)
				assert.Len(t, transcriptions, tt.expectedCount)
			}

			// Ensure all expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

// TestPgVectorStorage_StoreDualEmbeddings_Unit tests StoreDualEmbeddings with mocked database
func TestPgVectorStorage_StoreDualEmbeddings_Unit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewPgVectorStorage(db)
	defer storage.Close()

	ctx := context.Background()

	tests := []struct {
		name             string
		transcriptID     int
		openaiEmbedding  []float32
		geminiEmbedding  []float32
		setupMock        func()
		expectedErr      bool
	}{
		{
			name:            "successful_dual_store",
			transcriptID:    1,
			openaiEmbedding: []float32{0.1, 0.2, 0.3},
			geminiEmbedding: []float32{0.4, 0.5, 0.6},
			setupMock: func() {
				mock.ExpectExec("UPDATE transcriptions SET embedding_openai = \\$1, embedding_openai_model = 'text-embedding-ada-002', embedding_openai_created_at = now\\(\\), embedding_openai_status = 'completed', embedding_gemini = \\$2, embedding_gemini_model = 'models/embedding-001', embedding_gemini_created_at = now\\(\\), embedding_gemini_status = 'completed', embedding_sync_status = 'completed' WHERE id = \\$3").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: false,
		},
		{
			name:            "database_error",
			transcriptID:    2,
			openaiEmbedding: []float32{0.1, 0.2, 0.3},
			geminiEmbedding: []float32{0.4, 0.5, 0.6},
			setupMock: func() {
				mock.ExpectExec("UPDATE transcriptions SET embedding_openai = \\$1, embedding_openai_model = 'text-embedding-ada-002', embedding_openai_created_at = now\\(\\), embedding_openai_status = 'completed', embedding_gemini = \\$2, embedding_gemini_model = 'models/embedding-001', embedding_gemini_created_at = now\\(\\), embedding_gemini_status = 'completed', embedding_sync_status = 'completed' WHERE id = \\$3").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 2).
					WillReturnError(errors.New("update failed"))
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := storage.StoreDualEmbeddings(ctx, tt.transcriptID, tt.openaiEmbedding, tt.geminiEmbedding)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Ensure all expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}