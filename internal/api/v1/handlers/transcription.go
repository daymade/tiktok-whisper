package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tiktok-whisper/internal/api/errors"
	"tiktok-whisper/internal/api/middleware"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/api/v1/services"
)

// TranscriptionHandler handles transcription-related API endpoints
type TranscriptionHandler struct {
	service services.TranscriptionService
}

// NewTranscriptionHandler creates a new transcription handler
func NewTranscriptionHandler(service services.TranscriptionService) *TranscriptionHandler {
	return &TranscriptionHandler{
		service: service,
	}
}

// Create handles POST /api/v1/transcriptions
// Creates a new transcription job
//
// @Summary Create a new transcription job
// @Description Creates a new audio transcription job with specified provider and options
// @Tags transcriptions
// @Accept json
// @Produce json
// @Param transcription body dto.CreateTranscriptionRequest true "Transcription creation data"
// @Success 201 {object} dto.TranscriptionResponse "Transcription created successfully"
// @Failure 400 {object} errors.APIError "Bad request - invalid input data"
// @Failure 422 {object} errors.APIError "Validation error"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /transcriptions [post]
func (h *TranscriptionHandler) Create(c *gin.Context) {
	var req dto.CreateTranscriptionRequest
	
	// Validate request
	if err := middleware.ValidateRequest(c, &req); err != nil {
		middleware.HandleError(c, err)
		return
	}

	// Create transcription
	response, err := h.service.CreateTranscription(c.Request.Context(), &req)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Get handles GET /api/v1/transcriptions/:id
// Retrieves a specific transcription by ID
//
// @Summary Get transcription by ID
// @Description Retrieves detailed information about a specific transcription job
// @Tags transcriptions
// @Accept json
// @Produce json
// @Param id path int true "Transcription ID" format(int32) minimum(1)
// @Success 200 {object} dto.TranscriptionResponse "Transcription details"
// @Failure 400 {object} errors.APIError "Bad request - invalid ID"
// @Failure 404 {object} errors.APIError "Transcription not found"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /transcriptions/{id} [get]
func (h *TranscriptionHandler) Get(c *gin.Context) {
	// Parse transcription ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		middleware.HandleError(c, errors.NewBadRequestError("Invalid transcription ID"))
		return
	}

	// Get transcription
	response, err := h.service.GetTranscription(c.Request.Context(), id)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// List handles GET /api/v1/transcriptions
// Lists transcriptions with pagination and filtering
//
// @Summary List transcriptions with pagination
// @Description Retrieves a paginated list of transcriptions with optional filtering by user, status, and provider
// @Tags transcriptions
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Param user_id query string false "Filter by user ID"
// @Param status query string false "Filter by status" Enums(pending,processing,completed,failed)
// @Param provider query string false "Filter by provider" Enums(whisper_cpp,openai/whisper,elevenlabs)
// @Param order_by query string false "Sort field" default(created_at) Enums(created_at,updated_at,duration,file_size)
// @Param order query string false "Sort order" default(desc) Enums(asc,desc)
// @Success 200 {object} dto.PaginatedTranscriptionsResponse "List of transcriptions with pagination"
// @Failure 400 {object} errors.APIError "Bad request - invalid query parameters"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Header 200 {string} X-Total-Count "Total number of transcriptions"
// @Router /transcriptions [get]
func (h *TranscriptionHandler) List(c *gin.Context) {
	var query dto.ListTranscriptionsQuery
	
	// Validate query parameters
	if err := middleware.ValidateQuery(c, &query); err != nil {
		middleware.HandleError(c, err)
		return
	}

	// List transcriptions
	response, err := h.service.ListTranscriptions(c.Request.Context(), query)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	// Set total count header
	c.Header("X-Total-Count", strconv.Itoa(response.Pagination.Total))
	
	c.JSON(http.StatusOK, response)
}

// Upload handles POST /api/v1/transcriptions/upload
// Uploads an audio file for transcription
//
// @Summary Upload audio file for transcription
// @Description Upload an audio file and create a transcription job
// @Tags transcriptions
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Audio file to transcribe"
// @Param provider formData string false "Provider to use for transcription"
// @Param language formData string false "Language code"
// @Success 200 {object} dto.TranscriptionResponse "Upload successful and transcription started"
// @Failure 400 {object} errors.APIError "Bad request - invalid file"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /transcriptions/upload [post]
func (h *TranscriptionHandler) Upload(c *gin.Context) {
	// Get file from request
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		middleware.HandleError(c, errors.NewBadRequestError("No file uploaded"))
		return
	}
	defer file.Close()

	// Get optional parameters
	provider := c.PostForm("provider")
	language := c.PostForm("language")
	
	// TODO: Save file to temporary storage
	// For now, create a transcription request with the file info
	req := &dto.CreateTranscriptionRequest{
		Provider: provider,
		Language: language,
		Options: map[string]interface{}{
			"filename": header.Filename,
			"size":     header.Size,
		},
	}
	
	// Create transcription
	response, err := h.service.CreateTranscription(c.Request.Context(), req)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("Failed to create transcription"))
		return
	}
	
	c.JSON(http.StatusOK, response)
}

// Delete handles DELETE /api/v1/transcriptions/:id
// Deletes a transcription by ID
//
// @Summary Delete a transcription
// @Description Soft deletes a transcription job by its ID. The transcription data is marked as deleted but not permanently removed.
// @Tags transcriptions
// @Accept json
// @Produce json
// @Param id path int true "Transcription ID" format(int32) minimum(1)
// @Success 204 "Transcription deleted successfully"
// @Failure 400 {object} errors.APIError "Bad request - invalid ID"
// @Failure 404 {object} errors.APIError "Transcription not found"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /transcriptions/{id} [delete]
func (h *TranscriptionHandler) Delete(c *gin.Context) {
	// Parse transcription ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		middleware.HandleError(c, errors.NewBadRequestError("Invalid transcription ID"))
		return
	}

	// Delete transcription
	if err := h.service.DeleteTranscription(c.Request.Context(), id); err != nil {
		middleware.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}