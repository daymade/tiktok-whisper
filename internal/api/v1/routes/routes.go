package routes

import (
	"github.com/gin-gonic/gin"
	"tiktok-whisper/internal/api/middleware"
	"tiktok-whisper/internal/api/v1/handlers"
	"tiktok-whisper/internal/api/v1/services"
)

// RegisterRoutes registers all v1 API routes
func RegisterRoutes(router *gin.RouterGroup, container *ServiceContainer) {
	// Apply global middleware for v1
	router.Use(middleware.RequestID())
	
	// Transcription routes
	transcriptionHandler := handlers.NewTranscriptionHandler(container.TranscriptionService)
	transcriptions := router.Group("/transcriptions")
	{
		transcriptions.POST("", transcriptionHandler.Create)
		transcriptions.POST("/upload", transcriptionHandler.Upload)
		transcriptions.GET("/:id", transcriptionHandler.Get)
		transcriptions.GET("", transcriptionHandler.List)
		transcriptions.DELETE("/:id", transcriptionHandler.Delete)
	}

	// Provider routes
	providerHandler := handlers.NewProviderHandler(container.ProviderService)
	providers := router.Group("/providers")
	{
		providers.GET("", providerHandler.List)
		providers.GET("/:id", providerHandler.Get)
		providers.GET("/:id/status", providerHandler.GetStatus)
		providers.GET("/:id/stats", providerHandler.GetStats)
		providers.POST("/:id/test", providerHandler.Test)
	}

	// Embedding routes
	if container.EmbeddingService != nil {
		embeddingHandler := handlers.NewEmbeddingHandler(container.EmbeddingService)
		embeddings := router.Group("/embeddings")
		{
			embeddings.GET("", embeddingHandler.List)
			embeddings.GET("/search", embeddingHandler.Search)
			embeddings.POST("/generate", embeddingHandler.Generate)
		}
	}

	// Stats routes
	if container.StatsService != nil {
		statsHandler := handlers.NewStatsHandler(container.StatsService)
		stats := router.Group("/stats")
		{
			stats.GET("", statsHandler.GetSystemStats)
			stats.GET("/users", statsHandler.GetUserStats)
		}
	}

	// Export routes
	if container.ExportService != nil {
		exportHandler := handlers.NewExportHandler(container.ExportService)
		router.GET("/export", exportHandler.Export)
	}

	// Whisper job routes (frontend compatibility)
	if container.WhisperJobService != nil {
		// Use mock storage if no storage service is provided
		storageService := container.StorageService
		if storageService == nil {
			storageService = services.NewMockStorageService()
		}
		
		whisperJobHandler := handlers.NewWhisperJobHandler(
			container.WhisperJobService,
			container.ProviderService,
			container.TranscriptionService,
			storageService,
		)
		RegisterWhisperRoutes(router, whisperJobHandler)
	}

	// TODO: Add other routes as handlers are implemented
	// - Downloads
	// - Configuration
}

// ServiceContainer holds all services needed by handlers
type ServiceContainer struct {
	TranscriptionService services.TranscriptionService
	ProviderService      services.ProviderService
	DownloadService      services.DownloadService
	EmbeddingService     services.EmbeddingService
	StatsService         services.StatsService
	ExportService        services.ExportService
	ConfigService        services.ConfigService
	WhisperJobService    services.WhisperJobService
	StorageService       services.StorageService
}