package cmd

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"tiktok-whisper/internal/api/server"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/sqlite"
)

// @title TikTok Whisper API
// @version 1.0
// @description RESTful API for the TikTok Whisper CLI tool providing audio transcription services
// @description
// @description This API provides 1:1 mapping of CLI commands to RESTful endpoints for:
// @description - Transcription management (convert command)
// @description - Provider management (providers command)
// @description - Download operations (download command)
// @description - Embedding operations (embed command)
// @description - Export functionality (export command)
// @description - Configuration management (config command)
// @termsOfService https://github.com/your-org/tiktok-whisper
// @contact.name TikTok Whisper API Support
// @contact.url https://github.com/your-org/tiktok-whisper/issues
// @contact.email support@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Start the RESTful API server",
	Long: `Start a RESTful API server that provides HTTP endpoints for all CLI functionality.

The API server provides a 1:1 mapping of CLI commands to RESTful endpoints:
- Transcription management (convert command)
- Provider management (providers command)  
- Download operations (download command)
- Embedding operations (embed command)
- Export functionality (export command)
- Configuration management (config command)

Example:
  v2t api --port 8080
  v2t api --host 0.0.0.0 --port 3000`,
	Run: runAPIServer,
}

var (
	apiHost         string
	apiPort         string
	apiReadTimeout  int
	apiWriteTimeout int
	apiIdleTimeout  int
	apiEnv          string
)

func init() {
	rootCmd.AddCommand(apiCmd)

	apiCmd.Flags().StringVar(&apiHost, "host", "0.0.0.0", "API server host")
	apiCmd.Flags().StringVar(&apiPort, "port", "8080", "API server port")
	apiCmd.Flags().IntVar(&apiReadTimeout, "read-timeout", 30, "Read timeout in seconds")
	apiCmd.Flags().IntVar(&apiWriteTimeout, "write-timeout", 30, "Write timeout in seconds")
	apiCmd.Flags().IntVar(&apiIdleTimeout, "idle-timeout", 120, "Idle timeout in seconds")
	apiCmd.Flags().StringVar(&apiEnv, "env", "development", "Environment (development|production)")
}

// provideTranscriptionDAOV2 provides the enhanced DAO with new fields support
func provideTranscriptionDAOV2() repository.TranscriptionDAOV2 {
	// NO FALLBACK - DATA_PATH must be explicitly set
	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		log.Fatal("DATA_PATH environment variable must be set")
	}

	dbPath := filepath.Join(dataPath, "transcription.db")
	db := sqlite.NewSQLiteDB(dbPath)
	
	// SQLiteDB already implements TranscriptionDAOV2
	return db
}

func runAPIServer(cmd *cobra.Command, args []string) {
	// Create logger
	var logger *slog.Logger
	if apiEnv == "production" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	// Initialize provider configuration
	// NO FALLBACK - PROVIDERS_CONFIG must be explicitly set
	configPath := os.Getenv("PROVIDERS_CONFIG")
	if configPath == "" {
		configPath = os.Getenv("PROVIDER_CONFIG_PATH")
	}
	if configPath == "" {
		log.Fatal("PROVIDERS_CONFIG or PROVIDER_CONFIG_PATH must be set")
	}
	
	configManager := provider.NewConfigManager(configPath)
	config, err := configManager.LoadConfig()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to load provider configuration: %v", err)
		// Create minimal configuration
		config = &provider.ProviderConfiguration{
			DefaultProvider: "openai/whisper",
			Providers:       make(map[string]provider.ProviderConfig),
		}
	}

	// Build provider registry
	registry := provider.NewProviderRegistry()
	for name, providerConfig := range config.Providers {
		if !providerConfig.Enabled {
			continue
		}
		
		providerInstance, err := provider.BuildProviderFromConfig(name, providerConfig)
		if err != nil {
			logger.Warn("Failed to create provider", "provider", name, "error", err)
			continue
		}
		
		if err := registry.RegisterProvider(name, providerInstance); err != nil {
			logger.Warn("Failed to register provider", "provider", name, "error", err)
			continue
		}
	}

	// Set default provider
	if config.DefaultProvider != "" {
		if err := registry.SetDefaultProvider(config.DefaultProvider); err != nil {
			logger.Warn("Failed to set default provider", "provider", config.DefaultProvider, "error", err)
		}
	}

	// Create orchestrator with required dependencies
	// Use default metrics implementation
	metrics := &provider.DefaultProviderMetrics{}
	orchestratorConfig := provider.OrchestratorConfig{
		MaxRetries:          3,
		RetryDelay:          time.Second,
		HealthCheckInterval: 5 * time.Minute,
		PreferLocal:         false,
	}
	orchestrator := provider.NewTranscriptionOrchestrator(registry, metrics, orchestratorConfig)
	
	// Initialize repository
	repository := provideTranscriptionDAOV2()

	// Create server config
	serverConfig := server.Config{
		Host:         apiHost,
		Port:         apiPort,
		ReadTimeout:  time.Duration(apiReadTimeout) * time.Second,
		WriteTimeout: time.Duration(apiWriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(apiIdleTimeout) * time.Second,
		Environment:  apiEnv,
	}

	// Create and start server
	srv := server.NewServer(serverConfig, orchestrator, repository, registry, logger)
	
	if err := srv.Start(); err != nil {
		log.Fatalf("❌ Failed to start API server: %v", err)
	}

	// Display startup information
	fmt.Println("🚀 RESTful API Server Started")
	fmt.Printf("📍 Address: http://%s:%s\n", apiHost, apiPort)
	fmt.Printf("🌍 Environment: %s\n", apiEnv)
	fmt.Println("\n📚 Available Endpoints:")
	fmt.Println("  - GET    /health                          Health check")
	fmt.Println("  - POST   /api/v1/transcriptions           Create transcription")
	fmt.Println("  - GET    /api/v1/transcriptions/:id       Get transcription")
	fmt.Println("  - GET    /api/v1/transcriptions           List transcriptions")
	fmt.Println("  - DELETE /api/v1/transcriptions/:id       Delete transcription")
	fmt.Println("  - GET    /api/v1/providers                List providers")
	fmt.Println("  - GET    /api/v1/providers/:id            Get provider info")
	fmt.Println("  - GET    /api/v1/providers/:id/status     Get provider status")
	fmt.Println("  - GET    /api/v1/providers/:id/stats      Get provider stats")
	fmt.Println("  - POST   /api/v1/providers/:id/test       Test provider")
	fmt.Println("\n✨ API server is ready to accept requests!")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down API server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("❌ API server forced to shutdown: %v", err)
	}

	log.Println("✅ API server stopped gracefully")
}