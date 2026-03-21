package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	log.Printf("[DEBUG Upload] Extracted parameters - Provider: '%s', Language: '%s'", provider, language)

	// Debug: check all form values
	log.Printf("[DEBUG Upload] All form values: %+v", c.Request.Form)
	log.Printf("[DEBUG Upload] All post form values: %+v", c.Request.PostForm)

	// Get user ID from context (set by auth middleware)
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous" // Default for testing
	}

	log.Printf("[DEBUG Upload] Starting upload for user: %s, file: %s, size: %d", userID, header.Filename, header.Size)

	// Get DATA_PATH from environment for file storage
	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		log.Printf("[ERROR Upload] DATA_PATH not set")
		middleware.HandleError(c, errors.NewInternalError("DATA_PATH environment variable not set"))
		return
	}

	log.Printf("[DEBUG Upload] Using DATA_PATH: %s", dataPath)

	// Create uploads directory if it doesn't exist
	uploadsDir := filepath.Join(dataPath, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Printf("[ERROR Upload] Failed to create uploads directory: %v", err)
		middleware.HandleError(c, errors.NewInternalError("Failed to create uploads directory"))
		return
	}

	log.Printf("[DEBUG Upload] Created uploads directory: %s", uploadsDir)

	// Generate unique filename to avoid conflicts
	fileExt := filepath.Ext(header.Filename)
	uniqueID := uuid.New().String()
	savedFileName := uniqueID + fileExt
	savedFilePath := filepath.Join(uploadsDir, savedFileName)

	log.Printf("[DEBUG Upload] Generated file path: %s", savedFilePath)

	// Create destination file
	dest, err := os.Create(savedFilePath)
	if err != nil {
		log.Printf("[ERROR Upload] Failed to create destination file: %v", err)
		middleware.HandleError(c, errors.NewInternalError("Failed to create destination file"))
		return
	}

	log.Printf("[DEBUG Upload] Created destination file, starting copy...")

	// Copy uploaded file content to destination
	bytesWritten, err := io.Copy(dest, file)
	if err != nil {
		log.Printf("[ERROR Upload] Failed to copy file content: %v", err)
		// Clean up the partial file on error
		os.Remove(savedFilePath)
		middleware.HandleError(c, errors.NewInternalError("Failed to save uploaded file"))
		return
	}

	log.Printf("[DEBUG Upload] Copied %d bytes to file", bytesWritten)

	// Explicitly close the file to ensure all data is written
	if err := dest.Close(); err != nil {
		log.Printf("[ERROR Upload] Failed to close file: %v", err)
		os.Remove(savedFilePath)
		middleware.HandleError(c, errors.NewInternalError("Failed to finalize uploaded file"))
		return
	}

	log.Printf("[DEBUG Upload] File saved successfully: %s", savedFilePath)

	// Create transcription request with saved file path
	req := &dto.CreateTranscriptionRequest{
		FilePath: savedFilePath,
		Provider: provider,
		Language: language,
		UserID:   userID,
		Options: map[string]interface{}{
			"original_filename": header.Filename,
			"file_size":        header.Size,
			"saved_filename":   savedFileName,
		},
	}

	log.Printf("[DEBUG Upload] Creating transcription request - FilePath: %s, Provider: %s, UserID: %s", savedFilePath, provider, userID)

	// Create transcription
	response, err := h.service.CreateTranscription(c.Request.Context(), req)
	if err != nil {
		log.Printf("[ERROR Upload] CreateTranscription failed: %v", err)
		middleware.HandleError(c, errors.NewInternalError("Failed to create transcription"))
		return
	}

	log.Printf("[DEBUG Upload] Transcription created successfully with ID: %d", response.ID)
	
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