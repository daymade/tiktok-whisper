package model

import "time"

// TranscriptionFull represents the complete database schema
// This matches the actual SQLite schema in the database
type TranscriptionFull struct {
	// Core fields from database
	ID                 int        `json:"id"`
	User               string     `json:"user"`
	InputDir           string     `json:"input_dir"`
	FileName           string     `json:"file_name"`
	Mp3FileName        string     `json:"mp3_file_name"`
	AudioDuration      int        `json:"audio_duration"` // INTEGER in DB, not float64
	Transcription      string     `json:"transcription"`
	LastConversionTime time.Time  `json:"last_conversion_time"`
	HasError           int        `json:"has_error"` // 0 or 1
	ErrorMessage       string     `json:"error_message"`
	
	// New fields to be added in migration
	FileHash           string     `json:"file_hash,omitempty"`
	FileSize           int64      `json:"file_size,omitempty"`
	ProviderType       string     `json:"provider_type,omitempty"`
	Language           string     `json:"language,omitempty"`
	ModelName          string     `json:"model_name,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

// ToLegacy converts to the legacy Transcription model for backward compatibility
func (t *TranscriptionFull) ToLegacy() *Transcription {
	return &Transcription{
		ID:                 t.ID,
		User:               t.User,
		LastConversionTime: t.LastConversionTime,
		Mp3FileName:        t.Mp3FileName,
		AudioDuration:      float64(t.AudioDuration), // Convert INT to float64
		Transcription:      t.Transcription,
		ErrorMessage:       t.ErrorMessage,
	}
}

// FromLegacy creates a TranscriptionFull from legacy model
func FromLegacy(t *Transcription, inputDir, fileName string, hasError int) *TranscriptionFull {
	return &TranscriptionFull{
		ID:                 t.ID,
		User:               t.User,
		InputDir:           inputDir,
		FileName:           fileName,
		Mp3FileName:        t.Mp3FileName,
		AudioDuration:      int(t.AudioDuration), // Convert float64 to INT
		Transcription:      t.Transcription,
		LastConversionTime: t.LastConversionTime,
		HasError:           hasError,
		ErrorMessage:       t.ErrorMessage,
		CreatedAt:          t.LastConversionTime, // Use conversion time as created time
		UpdatedAt:          t.LastConversionTime,
	}
}