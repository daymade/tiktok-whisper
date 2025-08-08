package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"tiktok-whisper/internal/api/middleware"
	v1routes "tiktok-whisper/internal/api/v1/routes"
	"tiktok-whisper/internal/api/v1/services"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/repository"
	_ "tiktok-whisper/docs" // Generated swagger docs
)

// Config represents API server configuration
type Config struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Environment  string
}

// Server represents the API server
type Server struct {
	config     Config
	router     *gin.Engine
	httpServer *http.Server
	logger     *slog.Logger
}

// NewServer creates a new API server
func NewServer(
	config Config,
	orchestrator provider.TranscriptionOrchestrator,
	repository repository.TranscriptionDAOV2,
	providerRegistry provider.ProviderRegistry,
	logger *slog.Logger,
) *Server {
	// Set Gin mode based on environment
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create router
	router := gin.New()

	// Apply global middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogging(logger))
	router.Use(middleware.ErrorHandler(logger))
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// Create services
	transcriptionService := services.NewTranscriptionService(orchestrator, repository)
	providerService := services.NewProviderService(providerRegistry)
	
	// Create storage service (use MinIO if available, otherwise mock)
	var storageService services.StorageService
	if os.Getenv("ENABLE_MINIO") == "true" || os.Getenv("MINIO_ENDPOINT") != "" {
		minioService, err := services.NewMinioStorageService()
		if err != nil {
			logger.Warn("Failed to create MinIO storage service, using mock", zap.Error(err))
			storageService = services.NewMockStorageService()
		} else {
			storageService = minioService
			logger.Info("MinIO storage service initialized")
		}
	} else {
		storageService = services.NewMockStorageService()
		logger.Info("Using mock storage service")
	}
	
	serviceContainer := &v1routes.ServiceContainer{
		TranscriptionService: transcriptionService,
		ProviderService:      providerService,
		EmbeddingService:     services.NewEmbeddingService(repository),
		StatsService:         services.NewStatsService(repository),
		ExportService:        services.NewExportService(repository),
		WhisperJobService:    services.NewWhisperJobService(repository, transcriptionService, providerService),
		StorageService:       storageService,
		// TODO: Initialize other services (Download, Config)
	}

	// Register API routes
	api := router.Group("/api")
	{
		// V1 routes
		v1 := api.Group("/v1")
		v1routes.RegisterRoutes(v1, serviceContainer)
		
		// Whisper compatibility routes (for frontend)
		whisper := api.Group("/whisper")
		v1routes.RegisterRoutes(whisper, serviceContainer)
	}

	// Swagger documentation routes
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	
	// API documentation info endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "TikTok Whisper API",
			"version": "1.0",
			"documentation": "/swagger/index.html",
			"endpoints": gin.H{
				"health":        "/health",
				"transcriptions": "/api/v1/transcriptions",
				"providers":     "/api/v1/providers",
			},
		})
	})

	// Create HTTP server
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return &Server{
		config:     config,
		router:     router,
		httpServer: httpServer,
		logger:     logger,
	}
}

// Start starts the API server
func (s *Server) Start() error {
	s.logger.Info("Starting API server",
		"host", s.config.Host,
		"port", s.config.Port,
		"environment", s.config.Environment,
	)

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	s.logger.Info("API server started successfully",
		"address", s.httpServer.Addr,
	)

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down API server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", "error", err)
		return err
	}

	s.logger.Info("API server shutdown complete")
	return nil
}

// Router returns the Gin router (useful for testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}