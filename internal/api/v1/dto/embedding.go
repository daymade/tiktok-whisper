package dto

import "time"

// EmbeddingData represents an embedding with metadata
type EmbeddingData struct {
	ID          int       `json:"id"`
	User        string    `json:"user"`
	Text        string    `json:"text"`
	TextPreview string    `json:"textPreview"`
	Provider    string    `json:"provider"`
	Embedding   []float32 `json:"embedding,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

// EmbeddingListRequest represents parameters for listing embeddings
type EmbeddingListRequest struct {
	Provider string `form:"provider" json:"provider"`
	Limit    int    `form:"limit" json:"limit"`
	Page     int    `form:"page" json:"page"`
	User     string `form:"user" json:"user"`
}

// EmbeddingSearchRequest represents a semantic search request
type EmbeddingSearchRequest struct {
	Query     string  `form:"q" json:"q" binding:"required"`
	Provider  string  `form:"provider" json:"provider"`
	Limit     int     `form:"limit" json:"limit"`
	Threshold float64 `form:"threshold" json:"threshold"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	ID          int       `json:"id"`
	User        string    `json:"user"`
	Text        string    `json:"text"`
	TextPreview string    `json:"textPreview"`
	Provider    string    `json:"provider"`
	Similarity  float64   `json:"similarity"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

// EmbeddingGenerateRequest represents a request to generate embeddings
type EmbeddingGenerateRequest struct {
	Provider string   `json:"provider"`
	User     string   `json:"user"`
	IDs      []int    `json:"ids,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

// EmbeddingGenerateResponse represents the response for embedding generation
type EmbeddingGenerateResponse struct {
	Success   int    `json:"success"`
	Failed    int    `json:"failed"`
	Total     int    `json:"total"`
	Provider  string `json:"provider"`
	Message   string `json:"message,omitempty"`
}