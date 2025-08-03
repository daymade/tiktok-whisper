package dto

import (
	"time"

	"tiktok-whisper/internal/api/errors"
	"tiktok-whisper/internal/app/model"
)

// CreateTranscriptionRequest represents the request to create a transcription
type CreateTranscriptionRequest struct {
	FilePath     string                 `json:"file_path" binding:"required"`
	Provider     string                 `json:"provider,omitempty"`
	Language     string                 `json:"language,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
	UserID       string                 `json:"user_id,omitempty"`
	OutputFormat string                 `json:"output_format,omitempty" binding:"omitempty,oneof=text json srt vtt"`
}

// Validate performs domain-specific validation
func (r *CreateTranscriptionRequest) Validate() error {
	validationErrors := make(map[string]string)

	if r.FilePath == "" {
		validationErrors["file_path"] = "file path is required"
	}

	// Validate provider if specified
	if r.Provider != "" {
		validProviders := []string{"whisper_cpp", "openai/whisper", "elevenlabs", "ssh_whisper", "whisper_server", "custom_http"}
		valid := false
		for _, p := range validProviders {
			if r.Provider == p {
				valid = true
				break
			}
		}
		if !valid {
			validationErrors["provider"] = "invalid provider specified"
		}
	}

	if len(validationErrors) > 0 {
		return errors.NewValidationError("Invalid transcription request", validationErrors)
	}

	return nil
}

// TranscriptionResponse represents a transcription in API responses
type TranscriptionResponse struct {
	ID                 int                    `json:"id"`
	UserID             string                 `json:"user_id"`
	FilePath           string                 `json:"file_path"`
	Status             string                 `json:"status"`
	Provider           string                 `json:"provider"`
	Language           string                 `json:"language,omitempty"`
	Model              string                 `json:"model,omitempty"`
	Duration           float64                `json:"duration,omitempty"`
	Transcription      string                 `json:"transcription,omitempty"`
	Segments           []SegmentResponse      `json:"segments,omitempty"`
	Error              string                 `json:"error,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
	ProcessingTimeMs   int64                  `json:"processing_time_ms,omitempty"`
	FileSize           int64                  `json:"file_size,omitempty"`
	FileHash           string                 `json:"file_hash,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// SegmentResponse represents a transcription segment
type SegmentResponse struct {
	ID         int           `json:"id"`
	Start      float64       `json:"start"`
	End        float64       `json:"end"`
	Text       string        `json:"text"`
	Confidence float64       `json:"confidence,omitempty"`
	Words      []WordResponse `json:"words,omitempty"`
}

// WordResponse represents a word in a segment
type WordResponse struct {
	Word       string  `json:"word"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence,omitempty"`
}

// ListTranscriptionsQuery represents query parameters for listing transcriptions
type ListTranscriptionsQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	Limit    int    `form:"limit,default=20" binding:"min=1,max=100"`
	UserID   string `form:"user_id"`
	Status   string `form:"status" binding:"omitempty,oneof=pending processing completed failed"`
	Provider string `form:"provider"`
	OrderBy  string `form:"order_by,default=created_at" binding:"omitempty,oneof=created_at updated_at duration file_size"`
	Order    string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
}

// PaginatedTranscriptionsResponse represents a paginated list of transcriptions
type PaginatedTranscriptionsResponse struct {
	Transcriptions []TranscriptionResponse `json:"transcriptions"`
	Pagination     PaginationResponse      `json:"pagination"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page      int  `json:"page"`
	Limit     int  `json:"limit"`
	Total     int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext   bool `json:"has_next"`
	HasPrev   bool `json:"has_prev"`
}

// ToTranscriptionResponse converts a model to response DTO
func ToTranscriptionResponse(t *model.TranscriptionFull) TranscriptionResponse {
	resp := TranscriptionResponse{
		ID:               t.ID,
		UserID:           t.User,
		FilePath:         t.FileName,
		Status:           DetermineStatus(t),
		Provider:         t.ProviderType,
		Language:         t.Language,
		Model:            t.ModelName,
		Duration:         float64(t.AudioDuration), // Convert int to float64
		Transcription:    t.Transcription,
		Error:            t.ErrorMessage,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
		FileSize:         t.FileSize,
		FileHash:         t.FileHash,
	}

	// Add completed time if available
	if !t.LastConversionTime.IsZero() {
		resp.CompletedAt = &t.LastConversionTime
		resp.ProcessingTimeMs = t.LastConversionTime.Sub(t.CreatedAt).Milliseconds()
	}

	return resp
}

// DetermineStatus determines the transcription status based on the model
func DetermineStatus(t *model.TranscriptionFull) string {
	if t.HasError == 1 && t.ErrorMessage != "" {
		return "failed"
	}
	if t.Transcription != "" {
		return "completed"
	}
	if !t.LastConversionTime.IsZero() {
		return "processing"
	}
	return "pending"
}