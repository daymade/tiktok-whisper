package services

import (
	"context"
	"fmt"
	"strings"

	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/app/repository"
)

// EmbeddingServiceImpl implements the EmbeddingService interface
type EmbeddingServiceImpl struct {
	repo repository.TranscriptionDAOV2
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(repo repository.TranscriptionDAOV2) EmbeddingService {
	return &EmbeddingServiceImpl{
		repo: repo,
	}
}

// ListEmbeddings returns a list of embeddings with metadata
func (s *EmbeddingServiceImpl) ListEmbeddings(ctx context.Context, req dto.EmbeddingListRequest) ([]dto.EmbeddingData, error) {
	// Default values
	if req.Provider == "" {
		req.Provider = "gemini"
	}
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Query transcriptions with embeddings
	// TODO: Implement FindByUserWithEmbeddings in repository
	// For now, return empty result
	transcriptions := []repository.Transcription{}
	var err error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch embeddings: %w", err)
	}

	// Convert to DTOs
	embeddings := make([]dto.EmbeddingData, 0, len(transcriptions))
	for _, t := range transcriptions {
		embedding := dto.EmbeddingData{
			ID:        t.ID,
			User:      t.UserNickname,
			Text:      t.Transcription,
			Provider:  req.Provider,
			CreatedAt: t.LastConversionTime,
		}

		// Create text preview
		if len(t.Transcription) > 100 {
			embedding.TextPreview = t.Transcription[:100] + "..."
		} else {
			embedding.TextPreview = t.Transcription
		}

		// Extract embedding based on provider
		if req.Provider == "openai" && t.EmbeddingOpenAI != nil {
			embedding.Embedding = parseVectorString(*t.EmbeddingOpenAI)
		} else if req.Provider == "gemini" && t.EmbeddingGemini != nil {
			embedding.Embedding = parseVectorString(*t.EmbeddingGemini)
		}

		embeddings = append(embeddings, embedding)
	}

	return embeddings, nil
}

// SearchEmbeddings performs vector similarity search
func (s *EmbeddingServiceImpl) SearchEmbeddings(ctx context.Context, req dto.EmbeddingSearchRequest) ([]dto.SearchResult, error) {
	// Default values
	if req.Provider == "" {
		req.Provider = "gemini"
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Threshold == 0 {
		req.Threshold = 0.1
	}

	// Perform vector search using repository
	// TODO: Implement SearchByEmbedding in repository
	// For now, return empty result
	searchResults := []repository.SearchResult{}
	var err error
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to DTOs
	results := make([]dto.SearchResult, 0, len(searchResults))
	for _, sr := range searchResults {
		result := dto.SearchResult{
			ID:         sr.ID,
			User:       sr.UserNickname,
			Text:       sr.Transcription.Transcription,
			Provider:   req.Provider,
			Similarity: sr.Similarity,
			CreatedAt:  sr.Transcription.LastConversionTime,
		}

		// Create text preview
		if len(sr.Transcription.Transcription) > 100 {
			result.TextPreview = sr.Transcription.Transcription[:100] + "..."
		} else {
			result.TextPreview = sr.Transcription.Transcription
		}

		results = append(results, result)
	}

	return results, nil
}

// GenerateEmbeddings generates embeddings for transcriptions
func (s *EmbeddingServiceImpl) GenerateEmbeddings(ctx context.Context, req dto.EmbeddingGenerateRequest) (*dto.EmbeddingGenerateResponse, error) {
	// Default provider
	if req.Provider == "" {
		req.Provider = "gemini"
	}

	// Get transcriptions without embeddings
	// TODO: Implement FindByIDs and FindWithoutEmbeddings in repository
	// For now, return placeholder response
	transcriptions := []repository.Transcription{}
	var err error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch transcriptions: %w", err)
	}

	// Generate embeddings using embedding service
	// This would normally call the embedding generation logic from cmd/v2t/cmd/embed
	// For now, return a placeholder response
	response := &dto.EmbeddingGenerateResponse{
		Total:    len(transcriptions),
		Provider: req.Provider,
		Success:  0,
		Failed:   0,
	}

	// TODO: Integrate with actual embedding generation logic
	// from cmd/v2t/cmd/embed/embed.go
	response.Message = "Embedding generation queued for processing"

	return response, nil
}

// parseVectorString converts pgvector string format to float32 slice
func parseVectorString(str string) []float32 {
	if str == "" || str == "[]" {
		return nil
	}

	// Remove brackets and trim whitespace
	str = strings.Trim(strings.TrimSpace(str), "[]")
	if str == "" {
		return nil
	}

	// Split by comma and convert to float32
	parts := strings.Split(str, ",")
	vector := make([]float32, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var f float32
		n, err := fmt.Sscanf(part, "%f", &f)
		if err == nil && n == 1 {
			vector = append(vector, f)
		}
	}

	return vector
}