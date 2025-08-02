package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"tiktok-whisper/internal/config"

	_ "github.com/lib/pq"
)

// APIHandler handles API requests
type APIHandler struct {
	db *sql.DB
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(db *sql.DB) *APIHandler {
	return &APIHandler{db: db}
}

// EmbeddingData represents an embedding with metadata
type EmbeddingData struct {
	ID          int       `json:"id"`
	User        string    `json:"user"`
	Text        string    `json:"text"`
	TextPreview string    `json:"textPreview"`
	Provider    string    `json:"provider"`
	Embedding   []float32 `json:"embedding"`
	CreatedAt   string    `json:"createdAt,omitempty"`
}

// UserStats represents user statistics
type UserStats struct {
	User             string `json:"user"`
	TotalTranscripts int    `json:"totalTranscripts"`
	GeminiEmbeddings int    `json:"geminiEmbeddings"`
	OpenAIEmbeddings int    `json:"openaiEmbeddings"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	ID          int     `json:"id"`
	User        string  `json:"user"`
	Text        string  `json:"text"`
	TextPreview string  `json:"textPreview"`
	Provider    string  `json:"provider"`
	Similarity  float64 `json:"similarity"`
	CreatedAt   string  `json:"createdAt,omitempty"`
}

// SystemStats represents system-wide statistics
type SystemStats struct {
	TotalTranscripts  int         `json:"totalTranscripts"`
	GeminiEmbeddings  int         `json:"geminiEmbeddings"`
	OpenAIEmbeddings  int         `json:"openaiEmbeddings"`
	PendingProcessing int         `json:"pendingProcessing"`
	TopUsers          []UserStats `json:"topUsers"`
}

// ClusterData represents a cluster of embeddings
type ClusterData struct {
	ID         int     `json:"id"`
	CenterX    float64 `json:"centerX"`
	CenterY    float64 `json:"centerY"`
	CenterZ    float64 `json:"centerZ,omitempty"`
	Size       int     `json:"size"`
	Label      string  `json:"label"`
	Color      string  `json:"color"`
	Embeddings []int   `json:"embeddings"`
}

// GetEmbeddings returns all embeddings with metadata
func (h *APIHandler) GetEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "gemini" // Default to Gemini
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	ctx := context.Background()
	embeddings, err := h.getEmbeddingsFromDB(ctx, provider, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get embeddings: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(embeddings)
}

// SearchEmbeddings performs vector similarity search
func (h *APIHandler) SearchEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	query := r.URL.Query().Get("q")
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "gemini"
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default limit
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	thresholdStr := r.URL.Query().Get("threshold")
	threshold := 0.1 // default threshold
	if thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil && parsed >= 0 {
			threshold = parsed
		}
	}

	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Validate provider
	if provider != "openai" && provider != "gemini" {
		http.Error(w, "Provider must be either 'openai' or 'gemini'", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Load API keys for embedding generation
	apiKeys, err := config.GetAPIKeys()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load API keys: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate embedding for the search query
	targetEmbedding, err := h.generateTextEmbedding(ctx, query, provider, apiKeys)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate embedding: %v", err), http.StatusInternalServerError)
		return
	}

	// Perform vector similarity search
	results, err := h.performVectorSearch(ctx, targetEmbedding, provider, limit, threshold)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(results)
}

// GetClusters returns pre-computed clusters
func (h *APIHandler) GetClusters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return cluster data for 3D visualization
	// Note: Actual clustering is performed client-side in JavaScript using K-means++ algorithm
	clusters := []ClusterData{
		{
			ID:         1,
			CenterX:    0.0,
			CenterY:    0.0,
			CenterZ:    0.0,
			Size:       25,
			Label:      "抖音创业",
			Color:      "#ff6b6b",
			Embeddings: []int{1, 2, 3, 4, 5},
		},
		{
			ID:         2,
			CenterX:    1.0,
			CenterY:    1.0,
			CenterZ:    0.5,
			Size:       15,
			Label:      "营销策略",
			Color:      "#4ecdc4",
			Embeddings: []int{6, 7, 8, 9, 10},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(clusters)
}

// GetUsers returns user statistics
func (h *APIHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	users, err := h.getUsersFromDB(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get users: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(users)
}

// GetStats returns system statistics
func (h *APIHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	stats, err := h.getStatsFromDB(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(stats)
}

// getEmbeddingsFromDB retrieves embeddings from database
func (h *APIHandler) getEmbeddingsFromDB(ctx context.Context, provider string, limit int) ([]EmbeddingData, error) {
	// Query PostgreSQL with pgvector - get real embeddings
	var query string
	var embeddingColumn string

	switch provider {
	case "openai":
		embeddingColumn = "embedding_openai"
	case "gemini":
		embeddingColumn = "embedding_gemini"
	default:
		embeddingColumn = "embedding_gemini" // Default to Gemini
	}

	query = fmt.Sprintf(`
		SELECT 
			id, 
			COALESCE(user_nickname, 'Unknown') as user,
			transcription,
			last_conversion_time::text as created_at,
			%s as embedding
		FROM transcriptions 
		WHERE %s IS NOT NULL 
			AND transcription IS NOT NULL 
			AND transcription != ''
		ORDER BY id 
		LIMIT $1
	`, embeddingColumn, embeddingColumn)

	rows, err := h.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var embeddings []EmbeddingData
	for rows.Next() {
		var data EmbeddingData
		var fullText string
		var embeddingStr string

		err := rows.Scan(&data.ID, &data.User, &fullText, &data.CreatedAt, &embeddingStr)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Create text preview (first 100 characters)
		data.TextPreview = fullText
		if len(fullText) > 100 {
			data.TextPreview = fullText[:100] + "..."
		}
		data.Text = fullText
		data.Provider = provider

		// Parse pgvector format to float32 slice
		data.Embedding = h.parseVectorString(embeddingStr)

		embeddings = append(embeddings, data)
	}

	return embeddings, nil
}

// generateMockEmbedding creates a realistic mock embedding for demonstration (fallback)
func generateMockEmbedding(text string, provider string, id int) []float32 {
	// This function is kept as fallback for when real embeddings are not available
	var dimensions int
	switch provider {
	case "openai":
		dimensions = 1536
	case "gemini":
		dimensions = 768
	default:
		dimensions = 768
	}

	// Create a simple normalized random vector
	embedding := make([]float32, dimensions)
	for i := range embedding {
		embedding[i] = float32((float64(id*13+i*7) / 1000.0) - 0.5) // Deterministic "random"
	}

	// Normalize
	var norm float32
	for _, val := range embedding {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}

// getUsersFromDB retrieves user statistics from database
func (h *APIHandler) getUsersFromDB(ctx context.Context) ([]UserStats, error) {
	query := `
		SELECT 
			COALESCE(user_nickname, 'Unknown') as user,
			COUNT(*) as total_transcripts,
			COUNT(embedding_gemini) as gemini_embeddings,
			COUNT(embedding_openai) as openai_embeddings
		FROM transcriptions 
		WHERE transcription IS NOT NULL AND transcription != ''
		GROUP BY user_nickname
		ORDER BY COUNT(*) DESC
		LIMIT 20
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserStats
	for rows.Next() {
		var user UserStats
		err := rows.Scan(&user.User, &user.TotalTranscripts, &user.GeminiEmbeddings, &user.OpenAIEmbeddings)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

// getStatsFromDB retrieves system statistics from database
func (h *APIHandler) getStatsFromDB(ctx context.Context) (*SystemStats, error) {
	stats := &SystemStats{}

	// Get real counts from pgvector database
	err := h.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(embedding_gemini) as gemini_embeddings,
			COUNT(embedding_openai) as openai_embeddings,
			COUNT(CASE WHEN embedding_gemini IS NULL AND embedding_openai IS NULL THEN 1 END) as pending
		FROM transcriptions
		WHERE transcription IS NOT NULL AND transcription != ''
	`).Scan(&stats.TotalTranscripts, &stats.GeminiEmbeddings, &stats.OpenAIEmbeddings, &stats.PendingProcessing)
	if err != nil {
		return nil, err
	}

	// Get top users
	topUsers, err := h.getUsersFromDB(ctx)
	if err != nil {
		return nil, err
	}
	stats.TopUsers = topUsers

	return stats, nil
}

// parseVectorString converts pgvector string format to float32 slice
func (h *APIHandler) parseVectorString(str string) []float32 {
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

// Legacy function for compatibility
func stringToVector(str string) []float32 {
	if str == "" || str == "[]" {
		return nil
	}

	// Remove brackets
	str = strings.Trim(str, "[]")
	if str == "" {
		return nil
	}

	// Split by comma and convert to float32
	parts := strings.Split(str, ",")
	vector := make([]float32, len(parts))

	for i, part := range parts {
		var f float32
		fmt.Sscanf(strings.TrimSpace(part), "%f", &f)
		vector[i] = f
	}

	return vector
}
