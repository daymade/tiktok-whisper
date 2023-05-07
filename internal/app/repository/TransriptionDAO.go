package repository

import (
	"tiktok-whisper/internal/app/model"
	"time"
)

type TranscriptionDAO interface {
	Close() error

	GetAllByUser(userNickname string) ([]model.Transcription, error)

	CheckIfFileProcessed(fileName string) (int, error)

	RecordToDB(user, inputDir, fileName, mp3FileName string, audioDuration int, transcription string,
		lastConversionTime time.Time, hasError int, errorMessage string)
}
