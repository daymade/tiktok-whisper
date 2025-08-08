package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/api/v1/services"
)

// WhisperJobHandler handles whisper job-related HTTP requests
type WhisperJobHandler struct {
	jobService          services.WhisperJobService
	providerService     services.ProviderService
	transcriptionService services.TranscriptionService
	storageService      services.StorageService
}

// NewWhisperJobHandler creates a new whisper job handler
func NewWhisperJobHandler(
	jobService services.WhisperJobService,
	providerService services.ProviderService, 
	transcriptionService services.TranscriptionService,
	storageService services.StorageService,
) *WhisperJobHandler {
	return &WhisperJobHandler{
		jobService:          jobService,
		providerService:     providerService,
		transcriptionService: transcriptionService,
		storageService:      storageService,
	}
}

// CreateJob creates a new whisper transcription job
// @Summary Create a new transcription job
// @Description Creates a new asynchronous transcription job
// @Tags WhisperJobs
// @Accept json
// @Produce json
// @Param request body dto.CreateWhisperJobRequest true "Job creation request"
// @Success 200 {object} dto.WhisperJobResponse
// @Router /api/whisper/jobs [post]
func (h *WhisperJobHandler) CreateJob(c *gin.Context) {
	var req dto.CreateWhisperJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Get user ID from context (would come from auth middleware)
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	// Create job
	job, err := h.jobService.CreateJob(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    500,
			Message: "Failed to create job: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code: 0,
		Data: job,
		Message: "Job created successfully",
	})
}

// GetJob retrieves a specific job by ID
// @Summary Get job details
// @Description Get details of a specific transcription job
// @Tags WhisperJobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} dto.WhisperJobResponse
// @Router /api/whisper/jobs/{id} [get]
func (h *WhisperJobHandler) GetJob(c *gin.Context) {
	jobID := c.Param("id")
	
	job, err := h.jobService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Code:    404,
			Message: "Job not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code: 0,
		Data: job,
	})
}

// ListJobs lists all jobs for the current user
// @Summary List user's jobs
// @Description List all transcription jobs for the authenticated user
// @Tags WhisperJobs
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param status query string false "Filter by status"
// @Success 200 {object} dto.WhisperJobListResponse
// @Router /api/whisper/jobs [get]
func (h *WhisperJobHandler) ListJobs(c *gin.Context) {
	// Get user ID from context
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

	jobs, total, err := h.jobService.ListJobs(c.Request.Context(), userID, page, limit, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    500,
			Message: "Failed to list jobs: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code: 0,
		Data: map[string]interface{}{
			"jobs":  jobs,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// DeleteJob cancels/deletes a job
// @Summary Delete a job
// @Description Cancel or delete a transcription job
// @Tags WhisperJobs
// @Param id path string true "Job ID"
// @Success 200 {object} dto.SuccessResponse
// @Router /api/whisper/jobs/{id} [delete]
func (h *WhisperJobHandler) DeleteJob(c *gin.Context) {
	jobID := c.Param("id")
	
	if err := h.jobService.DeleteJob(c.Request.Context(), jobID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    500,
			Message: "Failed to delete job: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code:    0,
		Message: "Job deleted successfully",
	})
}

// GetProviders lists available providers in frontend format
// @Summary List providers
// @Description List all available transcription providers
// @Tags WhisperProviders
// @Produce json
// @Success 200 {object} dto.ProviderListResponse
// @Router /api/whisper/providers [get]
func (h *WhisperJobHandler) GetProviders(c *gin.Context) {
	providers, err := h.providerService.ListProviders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    500,
			Message: "Failed to list providers: " + err.Error(),
		})
		return
	}

	// Convert to frontend format
	frontendProviders := make([]map[string]interface{}, 0)
	for _, p := range providers {
		frontendProviders = append(frontendProviders, map[string]interface{}{
			"id":                p.ID,
			"name":              p.Name,
			"type":              p.Type,
			"available":         p.Available,
			"description":       p.Description,
			"is_default":        p.IsDefault,
			"requires_api_key":  p.RequiresAPIKey,
			"supported_formats": p.SupportedFormats,
			"capabilities": map[string]interface{}{
				"max_file_size_mb":      p.Capabilities.MaxFileSizeMB,
				"max_duration_sec":      p.Capabilities.MaxDurationSec,
				"supports_streaming":    p.Capabilities.SupportsStreaming,
				"supports_languages":    p.Capabilities.SupportsLanguages,
				"supports_models":       p.Capabilities.SupportsModels,
			},
			"health_status":     p.HealthStatus,
			"last_health_check": time.Now().Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code: 0,
		Data: frontendProviders,
	})
}

// GetPricing returns pricing information
// @Summary Get pricing
// @Description Get pricing information for transcription services
// @Tags WhisperProviders
// @Produce json
// @Success 200 {object} dto.PricingResponse
// @Router /api/whisper/pricing [get]
func (h *WhisperJobHandler) GetPricing(c *gin.Context) {
	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code: 0,
		Data: map[string]interface{}{
			"credits_per_minute": 10,
			"minimum_credits":    5,
			"providers": map[string]interface{}{
				"whisper_cpp": map[string]interface{}{
					"name":               "Local Whisper",
					"credits_per_minute": 5,
					"description":        "Free local processing",
				},
				"openai": map[string]interface{}{
					"name":               "OpenAI Whisper",
					"credits_per_minute": 10,
					"description":        "$0.006 per minute",
				},
				"elevenlabs": map[string]interface{}{
					"name":               "ElevenLabs",
					"credits_per_minute": 15,
					"description":        "Premium quality",
				},
			},
		},
	})
}

// GetStats returns user statistics
// @Summary Get user stats
// @Description Get transcription statistics for the current user
// @Tags WhisperStats
// @Produce json
// @Success 200 {object} dto.StatsResponse
// @Router /api/whisper/stats [get]
func (h *WhisperJobHandler) GetStats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	stats, err := h.jobService.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    500,
			Message: "Failed to get stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code: 0,
		Data: stats,
	})
}

// UploadFile handles file upload
// @Summary Upload audio file
// @Description Upload an audio file for transcription
// @Tags WhisperUpload
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Audio file"
// @Success 200 {object} dto.UploadResponse
// @Router /api/whisper/upload [post]
func (h *WhisperJobHandler) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    400,
			Message: "No file uploaded",
		})
		return
	}
	defer file.Close()

	// Get user ID from context (TODO: extract from auth middleware)
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous" // Default for testing
	}

	// Upload file to storage
	uploadResult, err := h.storageService.UploadFile(c.Request.Context(), file, header, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    500,
			Message: "Failed to upload file: " + err.Error(),
		})
		return
	}

	// Return response matching frontend expectations
	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code: 0,
		Data: map[string]interface{}{
			"file": map[string]interface{}{
				"url":  uploadResult.URL,
				"key":  uploadResult.Key,
				"name": uploadResult.Name,
				"size": uploadResult.Size,
				"audioDuration": uploadResult.AudioDuration,
			},
		},
		Message: "File uploaded successfully",
	})
}

// GetUploadURL generates a presigned upload URL
// @Summary Get presigned upload URL
// @Description Get a presigned URL for direct file upload
// @Tags WhisperUpload
// @Produce json
// @Success 200 {object} dto.PresignedURLResponse
// @Router /api/whisper/upload [get]
func (h *WhisperJobHandler) GetUploadURL(c *gin.Context) {
	// Get filename from query parameter
	filename := c.Query("filename")
	if filename == "" {
		filename = "audio.wav"
	}

	// Get user ID from context (TODO: extract from auth middleware)
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous" // Default for testing
	}

	// Generate presigned URL for upload
	presignedResult, err := h.storageService.GeneratePresignedURL(c.Request.Context(), "PUT", userID, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    500,
			Message: "Failed to generate upload URL: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Code: 0,
		Data: map[string]interface{}{
			"upload_url": presignedResult.URL,
			"file_id":    presignedResult.Key,
			"expires_at": presignedResult.ExpiresAt.Format(time.RFC3339),
			"method":     presignedResult.Method,
		},
	})
}