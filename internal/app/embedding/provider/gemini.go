package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// GeminiProvider implements EmbeddingProvider using Google Gemini API
type GeminiProvider struct {
	apiKey string
	model  string
}

// NewGeminiProvider creates a new Gemini embedding provider
func NewGeminiProvider(apiKey string) *GeminiProvider {
	return &GeminiProvider{
		apiKey: apiKey,
		model:  "gemini-embedding-001",
	}
}

// GenerateEmbedding generates an embedding using Gemini API
func (g *GeminiProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Validate input
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty text provided")
	}

	// If API key is empty, fall back to mock implementation for testing
	if g.apiKey == "" {
		return g.generateMockEmbedding(text), nil
	}

	// Create Gemini client using new SDK with API key authentication
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Prepare content for embedding
	contents := []*genai.Content{
		genai.NewContentFromText(text, genai.RoleUser),
	}

	// Set output dimensionality to 768
	outputDim := int32(768)

	// Generate embedding with specified output dimensionality
	result, err := client.Models.EmbedContent(ctx, g.model, contents, &genai.EmbedContentConfig{
		TaskType:             "RETRIEVAL_DOCUMENT",
		OutputDimensionality: &outputDim,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if result == nil || len(result.Embeddings) == 0 || len(result.Embeddings[0].Values) == 0 {
		return nil, errors.New("received empty embedding from Gemini API")
	}

	// Convert to float32 slice
	embedding := make([]float32, len(result.Embeddings[0].Values))
	for i, val := range result.Embeddings[0].Values {
		embedding[i] = float32(val)
	}

	return embedding, nil
}

// generateMockEmbedding creates a deterministic mock embedding for testing
func (g *GeminiProvider) generateMockEmbedding(text string) []float32 {
	embedding := make([]float32, 768)

	// Use a simple hash-like function based on character values
	hash := 0
	for i, char := range text {
		hash = hash*31 + int(char) + i
	}

	for i := range embedding {
		// Create different values based on position and text hash
		value := (hash + i*7) % 256
		embedding[i] = float32(value) / 256.0
	}

	return embedding
}

// GetProviderInfo returns information about the Gemini provider
func (g *GeminiProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:      "gemini",
		Model:     "gemini-embedding-001",
		Dimension: 768, // Using OutputDimensionality parameter to get 768 dimensions
	}
}
