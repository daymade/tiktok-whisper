package vector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PgVectorStorage implements VectorStorage using PostgreSQL with pgvector extension
type PgVectorStorage struct {
	db *sql.DB
}

// NewPgVectorStorage creates a new PostgreSQL vector storage instance
func NewPgVectorStorage(db *sql.DB) *PgVectorStorage {
	return &PgVectorStorage{db: db}
}

// StoreEmbedding stores a single embedding in the database
func (s *PgVectorStorage) StoreEmbedding(ctx context.Context, transcriptionID int, provider string, embedding []float32) error {
	vectorStr := vectorToString(embedding)
	
	var query string
	var model string
	
	switch provider {
	case "openai":
		query = `
			UPDATE transcriptions 
			SET embedding_openai = $1,
				embedding_openai_model = $2,
				embedding_openai_created_at = now(),
				embedding_openai_status = 'completed'
			WHERE id = $3
		`
		model = "text-embedding-ada-002"
	case "gemini":
		query = `
			UPDATE transcriptions 
			SET embedding_gemini = $1,
				embedding_gemini_model = $2,
				embedding_gemini_created_at = now(),
				embedding_gemini_status = 'completed'
			WHERE id = $3
		`
		model = "models/embedding-001"
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	
	_, err := s.db.ExecContext(ctx, query, vectorStr, model, transcriptionID)
	if err != nil {
		return fmt.Errorf("failed to store %s embedding: %w", provider, err)
	}
	
	return nil
}

// GetEmbedding retrieves a single embedding from the database
func (s *PgVectorStorage) GetEmbedding(ctx context.Context, transcriptionID int, provider string) ([]float32, error) {
	var query string
	var vectorStr string
	
	switch provider {
	case "openai":
		query = `SELECT embedding_openai FROM transcriptions WHERE id = $1 AND embedding_openai IS NOT NULL`
	case "gemini":
		query = `SELECT embedding_gemini FROM transcriptions WHERE id = $1 AND embedding_gemini IS NOT NULL`
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	err := s.db.QueryRowContext(ctx, query, transcriptionID).Scan(&vectorStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("embedding not found")
		}
		return nil, fmt.Errorf("failed to get %s embedding: %w", provider, err)
	}
	
	return stringToVector(vectorStr), nil
}

// StoreDualEmbeddings stores both OpenAI and Gemini embeddings
func (s *PgVectorStorage) StoreDualEmbeddings(ctx context.Context, transcriptionID int, openaiEmbedding, geminiEmbedding []float32) error {
	openaiStr := vectorToString(openaiEmbedding)
	geminiStr := vectorToString(geminiEmbedding)
	
	query := `
		UPDATE transcriptions 
		SET embedding_openai = $1,
			embedding_openai_model = 'text-embedding-ada-002',
			embedding_openai_created_at = now(),
			embedding_openai_status = 'completed',
			embedding_gemini = $2,
			embedding_gemini_model = 'models/embedding-001',
			embedding_gemini_created_at = now(),
			embedding_gemini_status = 'completed',
			embedding_sync_status = 'completed'
		WHERE id = $3
	`
	
	_, err := s.db.ExecContext(ctx, query, openaiStr, geminiStr, transcriptionID)
	if err != nil {
		return fmt.Errorf("failed to store dual embeddings: %w", err)
	}
	
	return nil
}

// GetDualEmbeddings retrieves both embeddings
func (s *PgVectorStorage) GetDualEmbeddings(ctx context.Context, transcriptionID int) (*DualEmbedding, error) {
	query := `
		SELECT embedding_openai, embedding_gemini 
		FROM transcriptions 
		WHERE id = $1
	`
	
	var openaiStr, geminiStr sql.NullString
	err := s.db.QueryRowContext(ctx, query, transcriptionID).Scan(&openaiStr, &geminiStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("no embeddings found")
		}
		return nil, fmt.Errorf("failed to get dual embeddings: %w", err)
	}
	
	result := &DualEmbedding{}
	if openaiStr.Valid {
		result.OpenAI = stringToVector(openaiStr.String)
	}
	if geminiStr.Valid {
		result.Gemini = stringToVector(geminiStr.String)
	}
	
	return result, nil
}

// GetTranscriptionsWithoutEmbeddings returns transcriptions that don't have embeddings for the specified provider
func (s *PgVectorStorage) GetTranscriptionsWithoutEmbeddings(ctx context.Context, provider string, limit int) ([]*Transcription, error) {
	var query string
	
	switch provider {
	case "openai":
		query = `
			SELECT id, user_nickname, mp3_file_name, transcription, last_conversion_time
			FROM transcriptions 
			WHERE embedding_openai IS NULL 
			AND embedding_openai_status = 'pending'
			ORDER BY id 
			LIMIT $1
		`
	case "gemini":
		query = `
			SELECT id, user_nickname, mp3_file_name, transcription, last_conversion_time
			FROM transcriptions 
			WHERE embedding_gemini IS NULL 
			AND embedding_gemini_status = 'pending'
			ORDER BY id 
			LIMIT $1
		`
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcriptions without embeddings: %w", err)
	}
	defer rows.Close()
	
	var transcriptions []*Transcription
	for rows.Next() {
		var t Transcription
		var userNickname sql.NullString
		var createdAt time.Time
		
		err := rows.Scan(&t.ID, &userNickname, &t.Mp3FileName, &t.TranscriptionText, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transcription: %w", err)
		}
		
		if userNickname.Valid {
			t.User = userNickname.String
		}
		t.CreatedAt = createdAt
		
		transcriptions = append(transcriptions, &t)
	}
	
	return transcriptions, nil
}

// Close closes the database connection
func (s *PgVectorStorage) Close() error {
	return s.db.Close()
}

// vectorToString converts a float32 slice to pgvector string format
func vectorToString(vector []float32) string {
	if len(vector) == 0 {
		return "[]"
	}
	
	parts := make([]string, len(vector))
	for i, v := range vector {
		parts[i] = fmt.Sprintf("%.6f", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// stringToVector converts pgvector string format to float32 slice
func stringToVector(str string) []float32 {
	if str == "" || str == "[]" {
		return nil
	}
	
	// Remove brackets
	str = strings.Trim(str, "[]")
	if str == "" {
		return nil
	}
	
	// Split by comma and convert to float32
	parts := strings.Split(str, ",")
	vector := make([]float32, len(parts))
	
	for i, part := range parts {
		var f float32
		fmt.Sscanf(strings.TrimSpace(part), "%f", &f)
		vector[i] = f
	}
	
	return vector
}