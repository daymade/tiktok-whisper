package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/api/v1/services"
)

// StatsHandler handles statistics-related HTTP requests
type StatsHandler struct {
	service services.StatsService
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(service services.StatsService) *StatsHandler {
	return &StatsHandler{
		service: service,
	}
}

// GetSystemStats handles GET /api/v1/stats
func (h *StatsHandler) GetSystemStats(c *gin.Context) {
	stats, err := h.service.GetSystemStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserStats handles GET /api/v1/stats/users
func (h *StatsHandler) GetUserStats(c *gin.Context) {
	var req dto.StatsRequest
	
	// Parse query parameters
	req.User = c.Query("user")
	req.StartDate = c.Query("startDate")
	req.EndDate = c.Query("endDate")

	stats, err := h.service.GetUserStats(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}