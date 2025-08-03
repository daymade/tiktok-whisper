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

	// TODO: Add other routes as handlers are implemented
	// - Downloads
	// - Embeddings
	// - Exports
	// - Configuration
}

// ServiceContainer holds all services needed by handlers
type ServiceContainer struct {
	TranscriptionService services.TranscriptionService
	ProviderService      services.ProviderService
	DownloadService      services.DownloadService
	EmbeddingService     services.EmbeddingService
	ExportService        services.ExportService
	ConfigService        services.ConfigService
}