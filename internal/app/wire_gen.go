// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package app

import (
	"log"
	"path/filepath"
	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/api/openai"
	"tiktok-whisper/internal/app/api/openai/whisper"
	"tiktok-whisper/internal/app/api/whisper_cpp"
	"tiktok-whisper/internal/app/converter"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/util/files"
)

// Injectors from wire.go:

func InitializeConverter() *converter.Converter {
	transcriber := provideLocalTranscriber()
	transcriptionDAO := provideTranscriptionDAO()
	converterConverter := converter.NewConverter(transcriber, transcriptionDAO)
	return converterConverter
}

// wire.go:

// provideRemoteTranscriber with openai's remote service conversion, must set environment variable OPENAI_API_KEY
func provideRemoteTranscriber() api.Transcriber {
	return whisper.NewRemoteTranscriber(openai.GetClient())
}

// provideLocalTranscriber with native whisper.cpp conversion, you need to compile whisper.cpp/main executable by yourself
func provideLocalTranscriber() api.Transcriber {
	binaryPath := "/Volumes/SSD2T/workspace/cpp/whisper.cpp/main"
	modelPath := "/Volumes/SSD2T/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"
	return whisper_cpp.NewLocalTranscriber(binaryPath, modelPath)
}

func provideTranscriptionDAO() repository.TranscriptionDAO {
	projectRoot, err := files.GetProjectRoot()
	if err != nil {
		log.Fatalf("Failed to get project root: %v\n", err)
	}

	dbPath := filepath.Join(projectRoot, "data/transcription.db")
	return sqlite.NewSQLiteDB(dbPath)
}
