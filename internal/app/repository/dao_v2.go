package repository

import (
	"tiktok-whisper/internal/app/model"
	"time"
)

// TranscriptionDAOV2 extends the original DAO interface with support for new fields
type TranscriptionDAOV2 interface {
	TranscriptionDAO // Embed original interface for backward compatibility
	
	// New methods for enhanced functionality
	RecordToDBV2(transcription *model.TranscriptionFull) error
	GetAllByUserV2(userNickname string) ([]model.TranscriptionFull, error)
	GetByFileHash(fileHash string) (*model.TranscriptionFull, error)
	GetByProvider(providerType string, limit int) ([]model.TranscriptionFull, error)
	UpdateFileMetadata(id int, fileHash string, fileSize int64) error
	SoftDelete(id int) error
	GetActiveTranscriptions(limit int) ([]model.TranscriptionFull, error)
	GetTranscriptionByID(id int) (*model.TranscriptionFull, error)
}

// RecordToDBParams contains all parameters for recording a transcription
type RecordToDBParams struct {
	User               string
	InputDir           string
	FileName           string
	Mp3FileName        string
	AudioDuration      int
	Transcription      string
	LastConversionTime time.Time
	HasError           int
	ErrorMessage       string
	FileHash           string
	FileSize           int64
	ProviderType       string
	Language           string
	ModelName          string
}