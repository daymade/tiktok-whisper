package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/api/v1/services"
)

// EmbeddingHandler handles embedding-related HTTP requests
type EmbeddingHandler struct {
	service services.EmbeddingService
}

// NewEmbeddingHandler creates a new embedding handler
func NewEmbeddingHandler(service services.EmbeddingService) *EmbeddingHandler {
	return &EmbeddingHandler{
		service: service,
	}
}

// List handles GET /api/v1/embeddings
func (h *EmbeddingHandler) List(c *gin.Context) {
	var req dto.EmbeddingListRequest
	
	// Parse query parameters
	req.Provider = c.Query("provider")
	req.User = c.Query("user")
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}
	
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			req.Page = page
		}
	}

	embeddings, err := h.service.ListEmbeddings(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, embeddings)
}

// Search handles GET /api/v1/embeddings/search
func (h *EmbeddingHandler) Search(c *gin.Context) {
	var req dto.EmbeddingSearchRequest
	
	// Parse query parameters
	req.Query = c.Query("q")
	req.Provider = c.Query("provider")
	
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Query parameter 'q' is required",
		})
		return
	}
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}
	
	if thresholdStr := c.Query("threshold"); thresholdStr != "" {
		if threshold, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			req.Threshold = threshold
		}
	}

	results, err := h.service.SearchEmbeddings(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, results)
}

// Generate handles POST /api/v1/embeddings/generate
func (h *EmbeddingHandler) Generate(c *gin.Context) {
	var req dto.EmbeddingGenerateRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	response, err := h.service.GenerateEmbeddings(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}