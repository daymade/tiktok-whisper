package services

import (
	"context"

	"tiktok-whisper/internal/api/v1/dto"
)

// TranscriptionService defines the interface for transcription operations
type TranscriptionService interface {
	CreateTranscription(ctx context.Context, req *dto.CreateTranscriptionRequest) (*dto.TranscriptionResponse, error)
	GetTranscription(ctx context.Context, id int) (*dto.TranscriptionResponse, error)
	ListTranscriptions(ctx context.Context, query dto.ListTranscriptionsQuery) (*dto.PaginatedTranscriptionsResponse, error)
	DeleteTranscription(ctx context.Context, id int) error
}

// ProviderService defines the interface for provider operations
type ProviderService interface {
	ListProviders(ctx context.Context) ([]dto.ProviderResponse, error)
	GetProvider(ctx context.Context, id string) (*dto.ProviderResponse, error)
	GetProviderStatus(ctx context.Context, id string) (*dto.ProviderStatusResponse, error)
	GetProviderStats(ctx context.Context, id string) (*dto.ProviderStatsResponse, error)
	TestProvider(ctx context.Context, id string, req *dto.TestProviderRequest) (*dto.TestProviderResponse, error)
}

// DownloadService defines the interface for download operations
type DownloadService interface {
	CreateDownload(ctx context.Context, req interface{}) (interface{}, error)
	GetDownload(ctx context.Context, id string) (interface{}, error)
	ListDownloads(ctx context.Context, query interface{}) (interface{}, error)
}

// EmbeddingService defines the interface for embedding operations
type EmbeddingService interface {
	CreateEmbedding(ctx context.Context, req interface{}) (interface{}, error)
	GetEmbedding(ctx context.Context, id string) (interface{}, error)
	SearchEmbeddings(ctx context.Context, req interface{}) (interface{}, error)
}

// ExportService defines the interface for export operations
type ExportService interface {
	CreateExport(ctx context.Context, req interface{}) (interface{}, error)
	GetExport(ctx context.Context, id string) (interface{}, error)
}

// ConfigService defines the interface for configuration operations
type ConfigService interface {
	GetConfig(ctx context.Context) (interface{}, error)
	UpdateConfig(ctx context.Context, req interface{}) (interface{}, error)
}