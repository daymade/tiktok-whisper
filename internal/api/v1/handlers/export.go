package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/api/v1/services"
)

// ExportHandler handles export-related HTTP requests
type ExportHandler struct {
	service services.ExportService
}

// NewExportHandler creates a new export handler
func NewExportHandler(service services.ExportService) *ExportHandler {
	return &ExportHandler{
		service: service,
	}
}

// Export handles GET /api/v1/export
func (h *ExportHandler) Export(c *gin.Context) {
	var req dto.ExportRequest
	
	// Parse query parameters
	req.Format = c.Query("format")
	if req.Format == "" {
		req.Format = "csv" // Default format
	}
	
	req.User = c.Query("user")
	req.StartDate = c.Query("startDate")
	req.EndDate = c.Query("endDate")
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// Validate format
	if req.Format != "csv" && req.Format != "json" && req.Format != "xlsx" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid format. Must be csv, json, or xlsx",
		})
		return
	}

	// Set appropriate content type
	var contentType string
	var filename string
	switch req.Format {
	case "csv":
		contentType = "text/csv"
		filename = "transcriptions.csv"
	case "json":
		contentType = "application/json"
		filename = "transcriptions.json"
	case "xlsx":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		filename = "transcriptions.xlsx"
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Export directly to response writer
	err := h.service.ExportTranscriptions(c.Request.Context(), req, c.Writer)
	if err != nil {
		// Since we may have already started writing, we can't change the status code
		// Log the error instead
		c.Writer.WriteString(fmt.Sprintf("\nError: %v", err))
		return
	}
}