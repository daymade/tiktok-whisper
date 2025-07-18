package vector

import (
	"context"
	"time"
)

// VectorStorage defines the interface for vector storage operations
// Following Single Responsibility Principle - focused on vector storage only
type VectorStorage interface {
	// Single embedding operations
	StoreEmbedding(ctx context.Context, transcriptionID int, provider string, embedding []float32) error
	GetEmbedding(ctx context.Context, transcriptionID int, provider string) ([]float32, error)
	
	// Dual embedding operations
	StoreDualEmbeddings(ctx context.Context, transcriptionID int, openaiEmbedding, geminiEmbedding []float32) error
	GetDualEmbeddings(ctx context.Context, transcriptionID int) (*DualEmbedding, error)
	
	// Batch operations
	GetTranscriptionsWithoutEmbeddings(ctx context.Context, provider string, limit int) ([]*Transcription, error)
	
	// Lifecycle
	Close() error
}

// DualEmbedding represents both OpenAI and Gemini embeddings
type DualEmbedding struct {
	OpenAI []float32
	Gemini []float32
}

// Transcription represents a transcription record
type Transcription struct {
	ID                int
	User              string
	Mp3FileName       string
	TranscriptionText string
	CreatedAt         time.Time
}

// EmbeddingStatus represents the status of embedding generation
type EmbeddingStatus struct {
	TranscriptionID int
	Provider        string
	Status          string // 'pending', 'processing', 'completed', 'failed'
	CreatedAt       time.Time
	Error           string
}