package provider

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
)

// MockProvider is a mock implementation for testing
// TDD Cycle 2: Deterministic implementation based on text hashing
type MockProvider struct {
	dimension int
}

// NewMockProvider creates a new mock provider with specified dimension
func NewMockProvider(dimension int) *MockProvider {
	return &MockProvider{dimension: dimension}
}

// GenerateEmbedding generates deterministic embeddings based on SHA256 hash
func (m *MockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Validate input
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty text provided")
	}

	// Generate deterministic embedding using SHA256 hash
	hash := sha256.Sum256([]byte(text))
	embedding := make([]float32, m.dimension)
	
	// Convert hash bytes to float32 values in range [-1, 1]
	for i := 0; i < m.dimension; i++ {
		byteIndex := i % len(hash)
		// Convert byte (0-255) to float32 in range [-1, 1]
		embedding[i] = (float32(hash[byteIndex]) / 255.0) * 2 - 1
	}
	
	return embedding, nil
}

// GetProviderInfo returns mock provider information
func (m *MockProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:      "mock",
		Model:     "mock-model",
		Dimension: m.dimension,
	}
}