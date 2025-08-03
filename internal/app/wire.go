//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
	"log"
	"os"
	"path/filepath"
	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/api/openai"
	"tiktok-whisper/internal/app/api/openai/whisper"
	"tiktok-whisper/internal/app/api/provider"
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
	// Get paths from environment variables - no defaults
	binaryPath := os.Getenv("WHISPER_CPP_BINARY")
	if binaryPath == "" {
		log.Fatal("WHISPER_CPP_BINARY environment variable must be set")
	}
	
	modelPath := os.Getenv("WHISPER_CPP_MODEL")
	if modelPath == "" {
		log.Fatal("WHISPER_CPP_MODEL environment variable must be set")
	}
	
	return whisper_cpp.NewLocalTranscriber(binaryPath, modelPath)
}

// provideEnhancedTranscriber provides the provider framework-based transcriber
func provideEnhancedTranscriber() api.Transcriber {
	// Use the provider framework - fail fast if it doesn't work
	transcriber := provider.NewSimpleProviderTranscriber()
	if transcriber == nil {
		log.Fatal("Provider framework initialization failed - check your configuration")
	}
	return transcriber
}

func provideTranscriptionDAO() repository.TranscriptionDAO {
	projectRoot, err := files.GetProjectRoot()
	if err != nil {
		log.Fatalf("Failed to get project root: %v\n", err)
	}

	dbPath := filepath.Join(projectRoot, "data/transcription.db")
	return sqlite.NewSQLiteDB(dbPath)
}

// provideTranscriptionDAOV2 provides the enhanced DAO with new fields support
func provideTranscriptionDAOV2() repository.TranscriptionDAOV2 {
	projectRoot, err := files.GetProjectRoot()
	if err != nil {
		log.Fatalf("Failed to get project root: %v\n", err)
	}

	dbPath := filepath.Join(projectRoot, "data/transcription.db")
	db := sqlite.NewSQLiteDB(dbPath)
	
	// SQLiteDB already implements TranscriptionDAOV2
	return db
}

func InitializeConverter() *converter.Converter {
	wire.Build(converter.NewConverter, provideEnhancedTranscriber, provideTranscriptionDAO)
	return &converter.Converter{}
}

func InitializeProgressAwareConverter(config converter.ProgressConfig) *converter.ProgressAwareConverter {
	wire.Build(converter.NewConverter, converter.NewProgressAwareConverter, provideEnhancedTranscriber, provideTranscriptionDAO)
	return &converter.ProgressAwareConverter{}
}

// InitializeConverterCompat creates a backward-compatible converter that uses V2 DAO
func InitializeConverterCompat() *converter.Converter {
	// This allows existing code to work with the enhanced database
	wire.Build(
		converter.NewConverter,
		provideEnhancedTranscriber,
		provideTranscriptionDAOV2,
		// Wire will handle the interface compatibility
		wire.Bind(new(repository.TranscriptionDAO), new(repository.TranscriptionDAOV2)),
	)
	return &converter.Converter{}
}