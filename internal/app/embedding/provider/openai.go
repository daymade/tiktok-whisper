package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements EmbeddingProvider using OpenAI API
type OpenAIProvider struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewOpenAIProvider creates a new OpenAI embedding provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	client := openai.NewClient(apiKey)
	return &OpenAIProvider{
		client: client,
		model:  openai.AdaEmbeddingV2, // text-embedding-ada-002
	}
}

// GenerateEmbedding generates an embedding using OpenAI API
func (o *OpenAIProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Validate input
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty text provided")
	}

	// Create embedding request
	request := openai.EmbeddingRequest{
		Model: o.model,
		Input: []string{text},
	}

	// Call OpenAI API
	response, err := o.client.CreateEmbeddings(ctx, request)
	if err != nil {
		return nil, err
	}

	// Validate response
	if len(response.Data) == 0 {
		return nil, errors.New("no embedding data returned from OpenAI")
	}

	return response.Data[0].Embedding, nil
}

// GetProviderInfo returns information about the OpenAI provider
func (o *OpenAIProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:      "openai",
		Model:     "text-embedding-ada-002",
		Dimension: 1536,
	}
}
