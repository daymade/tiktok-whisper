package model

import "time"

type Transcription struct {
	ID                 int
	User               string
	LastConversionTime time.Time
	Mp3FileName        string
	AudioDuration      float64
	Transcription      string
	ErrorMessage       string
}
