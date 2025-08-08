package dto

import (
	"time"
)

// CreateWhisperJobRequest represents a request to create a new whisper job
type CreateWhisperJobRequest struct {
	FileName     string                 `json:"fileName" binding:"required"`
	FileURL      string                 `json:"fileUrl" binding:"required"`
	FileSize     int64                  `json:"fileSize"`
	AudioDuration int                   `json:"audioDuration"`
	Provider     string                 `json:"provider"`
	Language     string                 `json:"language"`
	OutputFormat string                 `json:"outputFormat"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// WhisperJobResponse represents a whisper job
type WhisperJobResponse struct {
	ID                string                 `json:"id"`
	UserID            string                 `json:"userId"`
	WhisperJobID      *int                   `json:"whisperJobId,omitempty"`
	Status            string                 `json:"status"`
	FileName          string                 `json:"fileName,omitempty"`
	FileURL           string                 `json:"fileUrl,omitempty"`
	FileSize          int64                  `json:"fileSize,omitempty"`
	AudioDuration     int                    `json:"audioDuration,omitempty"`
	CreditCost        int                    `json:"creditCost,omitempty"`
	ProviderID        string                 `json:"providerId,omitempty"`
	ProviderName      string                 `json:"providerName,omitempty"`
	Language          string                 `json:"language,omitempty"`
	OutputFormat      string                 `json:"outputFormat,omitempty"`
	TranscriptionText string                 `json:"transcriptionText,omitempty"`
	TranscriptionURL  string                 `json:"transcriptionUrl,omitempty"`
	EmbeddingsCount   int                    `json:"embeddingsCount,omitempty"`
	Error             string                 `json:"error,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
	StartedAt         *time.Time             `json:"startedAt,omitempty"`
	CompletedAt       *time.Time             `json:"completedAt,omitempty"`
}

// WhisperJobListResponse represents a list of whisper jobs
type WhisperJobListResponse struct {
	Jobs  []WhisperJobResponse `json:"jobs"`
	Total int                  `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

// WhisperJobStatus represents job status
type WhisperJobStatus string

const (
	JobStatusPending    WhisperJobStatus = "pending"
	JobStatusProcessing WhisperJobStatus = "processing"
	JobStatusCompleted  WhisperJobStatus = "completed"
	JobStatusFailed     WhisperJobStatus = "failed"
	JobStatusCancelled  WhisperJobStatus = "cancelled"
)

// UserStatsResponse represents user statistics
type UserStatsResponse struct {
	UserID              string    `json:"userId"`
	TotalJobs           int       `json:"totalJobs"`
	CompletedJobs       int       `json:"completedJobs"`
	FailedJobs          int       `json:"failedJobs"`
	TotalCreditsUsed    int       `json:"totalCreditsUsed"`
	TotalAudioMinutes   int       `json:"totalAudioMinutes"`
	TotalTranscriptions int       `json:"totalTranscriptions"`
	LastJobAt           time.Time `json:"lastJobAt,omitempty"`
	ProviderUsage       map[string]int `json:"providerUsage"`
}

// UploadResponse represents file upload response
type UploadResponse struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	FileURL  string `json:"file_url"`
}

// PresignedURLResponse represents presigned URL response
type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
	ExpiresAt string `json:"expires_at"`
}

// PricingResponse represents pricing information
type PricingResponse struct {
	CreditsPerMinute int                    `json:"credits_per_minute"`
	MinimumCredits   int                    `json:"minimum_credits"`
	Providers        map[string]interface{} `json:"providers"`
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}