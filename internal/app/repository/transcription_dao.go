package repository

import (
	"tiktok-whisper/internal/app/model"
	"time"
)

// RecordInput holds all parameters for recording a transcription to the database.
// Using a struct eliminates the 10-parameter positional sprawl and prevents
// silent string-argument swaps.
type RecordInput struct {
	User               string
	InputDir           string
	FileName           string
	Mp3FileName        string
	AudioDuration      int
	Transcription      string
	LastConversionTime time.Time
	HasError           int
	ErrorMessage       string
	ProviderType       string
}

type TranscriptionDAO interface {
	Close() error

	GetAllByUser(userNickname string) ([]model.Transcription, error)

	CheckIfFileProcessed(fileName string) (int, error)

	DeleteByID(id int) error

	RecordToDB(input RecordInput)
}
