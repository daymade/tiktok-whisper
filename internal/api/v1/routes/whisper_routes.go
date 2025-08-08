package routes

import (
	"github.com/gin-gonic/gin"
	"tiktok-whisper/internal/api/v1/handlers"
)

// RegisterWhisperRoutes registers all whisper-related routes
func RegisterWhisperRoutes(router *gin.RouterGroup, handler *handlers.WhisperJobHandler) {
	whisper := router.Group("/whisper")
	{
		// Job management
		jobs := whisper.Group("/jobs")
		{
			jobs.POST("", handler.CreateJob)
			jobs.GET("", handler.ListJobs)
			jobs.GET("/:id", handler.GetJob)
			jobs.DELETE("/:id", handler.DeleteJob)
		}
		
		// Provider management
		whisper.GET("/providers", handler.GetProviders)
		whisper.GET("/pricing", handler.GetPricing)
		
		// File upload
		upload := whisper.Group("/upload")
		{
			upload.POST("", handler.UploadFile)
			upload.GET("", handler.GetUploadURL)
		}
		
		// Statistics
		whisper.GET("/stats", handler.GetStats)
		
		// Proxy handler for external v2t backend
		// This should be registered last as a catch-all
		proxyHandler := handlers.NewWhisperProxyHandler()
		whisper.GET("/health", proxyHandler.HealthCheck)
		whisper.Any("/*path", proxyHandler.ProxyRequest)
	}
}