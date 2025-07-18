package provider

import "context"

// EmbeddingProvider defines the interface for all embedding providers
// Following Interface Segregation Principle - keep it focused
type EmbeddingProvider interface {
	// GenerateEmbedding generates an embedding vector for the given text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	
	// GetProviderInfo returns metadata about the provider
	GetProviderInfo() ProviderInfo
}

// ProviderInfo contains metadata about an embedding provider
type ProviderInfo struct {
	Name      string // Provider name (e.g., "openai", "gemini")
	Model     string // Model identifier (e.g., "text-embedding-ada-002")
	Dimension int    // Embedding dimension (e.g., 1536 for OpenAI, 768 for Gemini)
}