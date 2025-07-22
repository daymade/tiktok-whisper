package handlers

import (
	"context"
	"database/sql"
	"fmt"

	"tiktok-whisper/internal/app/embedding/provider"
	"tiktok-whisper/internal/config"
)

// generateTextEmbedding generates an embedding for input text using the specified provider
func (h *APIHandler) generateTextEmbedding(ctx context.Context, text, providerName string, apiKeys *config.APIKeys) ([]float32, error) {
	var embeddingProvider provider.EmbeddingProvider

	switch providerName {
	case "openai":
		if apiKeys.OpenAI == "" {
			return nil, fmt.Errorf("OpenAI API key not available")
		}
		embeddingProvider = provider.NewOpenAIProvider(apiKeys.OpenAI)
	case "gemini":
		if apiKeys.Gemini == "" {
			return nil, fmt.Errorf("Gemini API key not available")
		}
		embeddingProvider = provider.NewGeminiProvider(apiKeys.Gemini)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}

	return embeddingProvider.GenerateEmbedding(ctx, text)
}

// performVectorSearch executes pgvector similarity search
func (h *APIHandler) performVectorSearch(ctx context.Context, targetEmbedding []float32, provider string, limit int, threshold float64) ([]SearchResult, error) {
	// Convert embedding to pgvector format
	embeddingStr := h.vectorToString(targetEmbedding)

	// Build query based on provider
	var embeddingColumn string
	switch provider {
	case "openai":
		embeddingColumn = "embedding_openai"
	case "gemini":
		embeddingColumn = "embedding_gemini"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// Use pgvector cosine distance operator <=> for similarity search
	// Note: cosine distance = 1 - cosine similarity, so we need to convert
	query := fmt.Sprintf(`
		SELECT 
			id, 
			transcription, 
			user_nickname,
			1 - (%s <=> $1) as similarity,
			created_at
		FROM transcriptions 
		WHERE %s IS NOT NULL
			AND 1 - (%s <=> $1) >= $2
		ORDER BY %s <=> $1
		LIMIT $3
	`, embeddingColumn, embeddingColumn, embeddingColumn, embeddingColumn)

	rows, err := h.db.QueryContext(ctx, query, embeddingStr, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute vector search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var createdAt sql.NullString
		err := rows.Scan(&result.ID, &result.Text, &result.User, &result.Similarity, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		// Generate text preview (first 100 characters)
		result.TextPreview = result.Text
		if len(result.TextPreview) > 100 {
			result.TextPreview = result.TextPreview[:100] + "..."
		}

		result.Provider = provider
		if createdAt.Valid {
			result.CreatedAt = createdAt.String
		}

		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate search results: %w", err)
	}

	return results, nil
}

// vectorToString converts float32 slice to pgvector string format
func (h *APIHandler) vectorToString(embedding []float32) string {
	if len(embedding) == 0 {
		return "[]"
	}

	result := "["
	for i, val := range embedding {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%.9f", val)
	}
	result += "]"

	return result
}