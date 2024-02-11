//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
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

func InitializeConverter() *converter.Converter {
	wire.Build(converter.NewConverter, provideLocalTranscriber, provideTranscriptionDAO)
	return &converter.Converter{}
}
