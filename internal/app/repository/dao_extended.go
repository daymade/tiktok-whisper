package repository

import (
	"context"
	"time"
)

// These are additional repository methods needed by the new API services
// They should be added to TranscriptionDAOV2 interface and implemented

// Transcription represents a transcription record for the API services
type Transcription struct {
	ID                 int
	UserNickname       string
	FileName           string
	Mp3FileName        string
	AudioDuration      float64
	Transcription      string
	LastConversionTime time.Time
	HasError           int
	ErrorMessage       string
	EmbeddingOpenAI    *string // pgvector format string
	EmbeddingGemini    *string // pgvector format string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SearchResult represents a search result with similarity
type SearchResult struct {
	Transcription
	Similarity float64
}

// RepositoryCounts represents aggregate counts
type RepositoryCounts struct {
	Total            int
	GeminiEmbeddings int
	OpenAIEmbeddings int
}

// UserStatistics represents user statistics
type UserStatistics struct {
	User             string
	TotalTranscripts int
	GeminiEmbeddings int
	OpenAIEmbeddings int
}

// Extended repository methods for new API services
type ExtendedRepository interface {
	// Embedding-related methods
	FindByUserWithEmbeddings(ctx context.Context, user string, provider string, limit int, offset int) ([]Transcription, error)
	FindWithoutEmbeddings(ctx context.Context, provider string, user string, limit int) ([]Transcription, error)
	FindByIDs(ctx context.Context, ids []int) ([]Transcription, error)
	SearchByEmbedding(ctx context.Context, query string, provider string, limit int, threshold float64) ([]SearchResult, error)
	
	// Stats-related methods
	GetCounts(ctx context.Context) (*RepositoryCounts, error)
	GetUserStats(ctx context.Context, user string, startTime, endTime *time.Time) ([]UserStatistics, error)
	
	// Export-related methods
	FindByFilters(ctx context.Context, user string, startTime, endTime *time.Time, limit, offset int) ([]Transcription, error)
}

// Stub implementations for missing methods
// These should be properly implemented in the actual repository implementations

// FindByUserWithEmbeddings returns transcriptions with embeddings for a user
func FindByUserWithEmbeddingsStub(ctx context.Context, user string, provider string, limit int, offset int) ([]Transcription, error) {
	// TODO: Implement actual database query
	return []Transcription{}, nil
}

// FindWithoutEmbeddings returns transcriptions without embeddings
func FindWithoutEmbeddingsStub(ctx context.Context, provider string, user string, limit int) ([]Transcription, error) {
	// TODO: Implement actual database query
	return []Transcription{}, nil
}

// FindByIDs returns transcriptions by IDs
func FindByIDsStub(ctx context.Context, ids []int) ([]Transcription, error) {
	// TODO: Implement actual database query
	return []Transcription{}, nil
}

// SearchByEmbedding performs vector similarity search
func SearchByEmbeddingStub(ctx context.Context, query string, provider string, limit int, threshold float64) ([]SearchResult, error) {
	// TODO: Implement actual vector search with pgvector
	return []SearchResult{}, nil
}

// GetCounts returns aggregate counts
func GetCountsStub(ctx context.Context) (*RepositoryCounts, error) {
	// TODO: Implement actual database query
	return &RepositoryCounts{}, nil
}

// GetUserStats returns user statistics
func GetUserStatsStub(ctx context.Context, user string, startTime, endTime *time.Time) ([]UserStatistics, error) {
	// TODO: Implement actual database query
	return []UserStatistics{}, nil
}

// FindByFilters returns transcriptions by filters
func FindByFiltersStub(ctx context.Context, user string, startTime, endTime *time.Time, limit, offset int) ([]Transcription, error) {
	// TODO: Implement actual database query
	return []Transcription{}, nil
}