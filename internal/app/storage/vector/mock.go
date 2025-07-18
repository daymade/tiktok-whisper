package vector

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// MockVectorStorage is a mock implementation for testing
// TDD Cycle 3: GREEN - Minimal implementation to make tests pass
type MockVectorStorage struct {
	embeddings     map[string][]float32
	transcriptions map[int]*Transcription
	mu             sync.RWMutex
}

// NewMockVectorStorage creates a new mock vector storage
func NewMockVectorStorage() *MockVectorStorage {
	return &MockVectorStorage{
		embeddings:     make(map[string][]float32),
		transcriptions: make(map[int]*Transcription),
	}
}

// StoreEmbedding stores an embedding in memory
func (s *MockVectorStorage) StoreEmbedding(ctx context.Context, transcriptionID int, provider string, embedding []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := fmt.Sprintf("%d-%s", transcriptionID, provider)
	s.embeddings[key] = embedding
	return nil
}

// GetEmbedding retrieves an embedding from memory
func (s *MockVectorStorage) GetEmbedding(ctx context.Context, transcriptionID int, provider string) ([]float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	key := fmt.Sprintf("%d-%s", transcriptionID, provider)
	embedding, exists := s.embeddings[key]
	if !exists {
		return nil, errors.New("embedding not found")
	}
	return embedding, nil
}

// StoreDualEmbeddings stores both OpenAI and Gemini embeddings
func (s *MockVectorStorage) StoreDualEmbeddings(ctx context.Context, transcriptionID int, openaiEmbedding, geminiEmbedding []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	openaiKey := fmt.Sprintf("%d-openai", transcriptionID)
	geminiKey := fmt.Sprintf("%d-gemini", transcriptionID)
	
	s.embeddings[openaiKey] = openaiEmbedding
	s.embeddings[geminiKey] = geminiEmbedding
	return nil
}

// GetDualEmbeddings retrieves both embeddings
func (s *MockVectorStorage) GetDualEmbeddings(ctx context.Context, transcriptionID int) (*DualEmbedding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	openaiKey := fmt.Sprintf("%d-openai", transcriptionID)
	geminiKey := fmt.Sprintf("%d-gemini", transcriptionID)
	
	openaiEmbedding, openaiExists := s.embeddings[openaiKey]
	geminiEmbedding, geminiExists := s.embeddings[geminiKey]
	
	if !openaiExists && !geminiExists {
		return nil, errors.New("no embeddings found")
	}
	
	return &DualEmbedding{
		OpenAI: openaiEmbedding,
		Gemini: geminiEmbedding,
	}, nil
}

// GetTranscriptionsWithoutEmbeddings returns transcriptions that don't have embeddings for the specified provider
func (s *MockVectorStorage) GetTranscriptionsWithoutEmbeddings(ctx context.Context, provider string, limit int) ([]*Transcription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*Transcription
	count := 0
	
	for id, transcription := range s.transcriptions {
		if count >= limit {
			break
		}
		
		key := fmt.Sprintf("%d-%s", id, provider)
		if _, exists := s.embeddings[key]; !exists {
			result = append(result, transcription)
			count++
		}
	}
	
	return result, nil
}

// AddMockTranscription adds a mock transcription for testing
func (s *MockVectorStorage) AddMockTranscription(id int, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.transcriptions[id] = &Transcription{
		ID:                id,
		User:              "test_user",
		Mp3FileName:       fmt.Sprintf("test_%d.mp3", id),
		TranscriptionText: text,
		CreatedAt:         time.Now(),
	}
}

// Close closes the mock storage (no-op for mock)
func (s *MockVectorStorage) Close() error {
	return nil
}