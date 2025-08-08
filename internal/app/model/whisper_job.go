package model

import (
	"time"
)

// WhisperJob represents a transcription job
type WhisperJob struct {
	ID                string                 `json:"id" db:"id"`
	UserID            string                 `json:"user_id" db:"user_id"`
	WhisperJobID      *int                   `json:"whisper_job_id" db:"whisper_job_id"`
	Status            string                 `json:"status" db:"status"`
	FileName          string                 `json:"file_name" db:"file_name"`
	FileURL           string                 `json:"file_url" db:"file_url"`
	FileSize          int64                  `json:"file_size" db:"file_size"`
	AudioDuration     int                    `json:"audio_duration" db:"audio_duration"`
	Provider          string                 `json:"provider" db:"provider"`
	Language          string                 `json:"language" db:"language"`
	OutputFormat      string                 `json:"output_format" db:"output_format"`
	TranscriptionText string                 `json:"transcription_text" db:"transcription_text"`
	TranscriptionURL  string                 `json:"transcription_url" db:"transcription_url"`
	CreditCost        int                    `json:"credit_cost" db:"credit_cost"`
	CreditRefunded    bool                   `json:"credit_refunded" db:"credit_refunded"`
	Error             string                 `json:"error" db:"error"`
	RetryCount        int                    `json:"retry_count" db:"retry_count"`
	Metadata          map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
	StartedAt         *time.Time             `json:"started_at" db:"started_at"`
	CompletedAt       *time.Time             `json:"completed_at" db:"completed_at"`
}

// TableName returns the table name for WhisperJob
func (WhisperJob) TableName() string {
	return "whisper_jobs"
}