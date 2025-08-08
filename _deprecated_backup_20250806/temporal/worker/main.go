package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/api/provider/registry"
	"tiktok-whisper/internal/app/config"
	"tiktok-whisper/temporal/activities"
	"tiktok-whisper/temporal/pkg/common"
	"tiktok-whisper/temporal/workflows"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Setup logger
	logger := common.MustNewLogger(common.GetEnv("ENV", "production") == "development")
	defer logger.Sync()

	// Get Temporal configuration
	config := common.DefaultTemporalConfig()
	workerIdentity := common.GetEnv("WORKER_IDENTITY", fmt.Sprintf("v2t-worker-%s", getHostname()))
	
	// MinIO configuration
	minioEndpoint := common.GetEnv("MINIO_ENDPOINT", common.DefaultMinIOEndpoint)
	minioAccessKey := common.GetEnv("MINIO_ACCESS_KEY", common.DefaultMinIOAccessKey)
	minioSecretKey := common.GetEnv("MINIO_SECRET_KEY", common.DefaultMinIOSecretKey)
	minioBucket := common.GetEnv("MINIO_BUCKET", common.DefaultMinIOBucket)

	logger.Info("Starting v2t Temporal worker",
		zap.String("temporalHost", config.HostPort),
		zap.String("taskQueue", config.TaskQueue),
		zap.String("namespace", config.Namespace),
		zap.String("identity", workerIdentity),
	)

	// Create Temporal client
	temporalClient, err := common.NewTemporalClient(config)
	if err != nil {
		logger.Fatal("Failed to create Temporal client", zap.Error(err))
	}
	defer temporalClient.Close()

	// Initialize v2t provider registry
	providerRegistry, err := initializeProviderRegistry(logger)
	if err != nil {
		logger.Fatal("Failed to initialize provider registry", zap.Error(err))
	}

	// Create activities
	transcribeActivities := activities.NewTranscribeActivities(providerRegistry)
	
	storageActivities, err := activities.NewStorageActivities(
		minioEndpoint,
		minioAccessKey,
		minioSecretKey,
		minioBucket,
	)
	if err != nil {
		logger.Fatal("Failed to create storage activities", zap.Error(err))
	}

	// Ensure MinIO bucket exists
	ctx := context.Background()
	if err := storageActivities.EnsureBucketExists(ctx); err != nil {
		logger.Warn("Failed to ensure MinIO bucket exists", zap.Error(err))
	}

	// Create worker
	w := worker.New(temporalClient, config.TaskQueue, worker.Options{
		Identity:                 workerIdentity,
		MaxConcurrentActivityExecutionSize: 10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
		EnableLoggingInReplay:   false,
		BackgroundActivityContext: ctx,
	})

	// Register workflows
	w.RegisterWorkflow(workflows.SingleFileTranscriptionWorkflow)
	w.RegisterWorkflow(workflows.BatchTranscriptionWorkflow)
	w.RegisterWorkflow(workflows.BatchWithRetryWorkflow)
	w.RegisterWorkflow(workflows.TranscriptionWithFallbackWorkflow)
	w.RegisterWorkflow(workflows.SmartFallbackWorkflow)

	// Register activities
	w.RegisterActivity(transcribeActivities.TranscribeFile)
	w.RegisterActivity(transcribeActivities.GetProviderStatus)
	w.RegisterActivity(transcribeActivities.ListAvailableProviders)
	w.RegisterActivity(transcribeActivities.GetRecommendedProvider)

	w.RegisterActivity(storageActivities.UploadFile)
	w.RegisterActivity(storageActivities.DownloadFile)
	w.RegisterActivity(storageActivities.CleanupTempFile)
	w.RegisterActivity(storageActivities.ListFiles)
	w.RegisterActivity(storageActivities.EnsureBucketExists)

	// Initialize health status
	healthStatus := &HealthStatus{
		WorkerID:  workerIdentity,
		TaskQueue: taskQueue,
		Temporal: ConnectionStatus{
			Connected: true,
			Endpoint:  temporalHost,
		},
		MinIO: ConnectionStatus{
			Connected: true,
			Endpoint:  minioEndpoint,
		},
	}
	
	// Populate provider status
	providers := providerRegistry.ListProviders()
	for _, p := range providers {
		info := p.GetProviderInfo()
		status := ProviderStatus{
			Name:      info.Name,
			Type:      string(info.Type),
			Available: true,
		}
		
		// Check provider health
		if err := p.HealthCheck(ctx); err != nil {
			status.Available = false
			status.Error = err.Error()
		}
		
		healthStatus.Providers = append(healthStatus.Providers, status)
	}
	
	// Start health server
	healthPort := getEnv("HEALTH_PORT", ":8081")
	startHealthServer(healthPort, healthStatus)
	logger.Info("Health server started", zap.String("port", healthPort))

	// Handle shutdown gracefully
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	// Start worker in background
	go func() {
		err := w.Run(worker.InterruptCh())
		if err != nil {
			logger.Fatal("Worker failed", zap.Error(err))
		}
	}()

	logger.Info("Worker started successfully", 
		zap.String("taskQueue", taskQueue),
		zap.Int("maxConcurrentActivities", 10),
	)

	// Wait for shutdown signal
	<-shutdownCh
	logger.Info("Shutting down worker...")
	
	w.Stop()
	logger.Info("Worker stopped")
}

// initializeProviderRegistry initializes the v2t provider registry
func initializeProviderRegistry(logger *zap.Logger) (provider.ProviderRegistry, error) {
	// Load provider configuration
	configPath := getEnv("PROVIDER_CONFIG_PATH", "/app/config/providers.yaml")
	
	// Try to load config file, fall back to environment variables if not found
	var cfg *config.ProvidersConfig
	if _, err := os.Stat(configPath); err == nil {
		cfg, err = config.LoadProvidersConfig(configPath)
		if err != nil {
			logger.Warn("Failed to load provider config, using defaults", zap.Error(err))
			cfg = createDefaultConfig()
		}
	} else {
		logger.Info("Config file not found, using environment-based configuration")
		cfg = createDefaultConfig()
	}

	// Create registry and factory
	reg := registry.NewDefaultProviderRegistry()
	factory := provider.NewProviderFactory()
	
	// Initialize configured providers
	for name, providerConfig := range cfg.Providers {
		if !providerConfig.Enabled {
			continue
		}
		
		logger.Info("Initializing provider", 
			zap.String("name", name),
			zap.String("type", providerConfig.Type),
		)
		
		// Convert config to map for factory
		configMap := make(map[string]interface{})
		
		// Copy settings
		for k, v := range providerConfig.Settings {
			configMap[k] = v
		}
		
		// Add auth if present
		if providerConfig.Auth != nil {
			for k, v := range providerConfig.Auth {
				// Expand environment variables
				if strVal, ok := v.(string); ok && strings.HasPrefix(strVal, "${") {
					envVar := strings.TrimSuffix(strings.TrimPrefix(strVal, "${"), "}")
					configMap[k] = os.Getenv(envVar)
				} else {
					configMap[k] = v
				}
			}
		}
		
		// Create provider
		p, err := factory.CreateProvider(providerConfig.Type, configMap)
		if err != nil {
			logger.Error("Failed to create provider", 
				zap.String("name", name),
				zap.Error(err))
			continue
		}
		
		// Register provider
		if err := reg.RegisterProvider(name, p); err != nil {
			logger.Error("Failed to register provider",
				zap.String("name", name),
				zap.Error(err))
		}
	}
	
	// Create orchestrator
	orchestrator := registry.NewDefaultTranscriptionOrchestrator(reg)
	
	// Set default provider
	if cfg.DefaultProvider != "" {
		orchestrator.SetDefaultProvider(cfg.DefaultProvider)
	}

	return orchestrator, nil
}

// createDefaultConfig creates a default configuration from environment
func createDefaultConfig() *config.ProvidersConfig {
	cfg := &config.ProvidersConfig{
		DefaultProvider: "whisper_cpp",
		Providers:       make(map[string]config.ProviderConfig),
	}
	
	// Configure whisper_cpp if binary exists
	whisperPath := getEnv("WHISPER_BINARY_PATH", "/usr/local/bin/whisper")
	modelPath := getEnv("WHISPER_MODEL_PATH", "/models/ggml-large-v2.bin")
	
	if _, err := os.Stat(whisperPath); err == nil {
		cfg.Providers["whisper_cpp"] = config.ProviderConfig{
			Type:    "whisper_cpp",
			Enabled: true,
			Settings: map[string]interface{}{
				"binary_path": whisperPath,
				"model_path":  modelPath,
				"language":    getEnv("WHISPER_LANGUAGE", "en"),
				"threads":     4,
			},
		}
	}
	
	// Configure OpenAI if API key exists
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.Providers["openai"] = config.ProviderConfig{
			Type:    "openai",
			Enabled: true,
			Auth: map[string]interface{}{
				"api_key": apiKey,
			},
			Settings: map[string]interface{}{
				"model":           "whisper-1",
				"response_format": "text",
			},
		}
		
		// Use OpenAI as default if whisper_cpp not available
		if _, exists := cfg.Providers["whisper_cpp"]; !exists {
			cfg.DefaultProvider = "openai"
		}
	}
	
	// Configure ElevenLabs if API key exists
	if apiKey := os.Getenv("ELEVENLABS_API_KEY"); apiKey != "" {
		cfg.Providers["elevenlabs"] = config.ProviderConfig{
			Type:    "elevenlabs",
			Enabled: true,
			Auth: map[string]interface{}{
				"api_key": apiKey,
			},
			Settings: map[string]interface{}{
				"model": "whisper-large-v3",
			},
		}
	}
	
	return cfg
}

// Helper functions

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// temporalClientLogger adapts zap logger to Temporal's logger interface
type temporalClientLogger struct {
	*zap.Logger
}

func (l temporalClientLogger) Debug(msg string, keyvals ...interface{}) {
	l.Logger.Debug(msg, toZapFields(keyvals)...)
}

func (l temporalClientLogger) Info(msg string, keyvals ...interface{}) {
	l.Logger.Info(msg, toZapFields(keyvals)...)
}

func (l temporalClientLogger) Warn(msg string, keyvals ...interface{}) {
	l.Logger.Warn(msg, toZapFields(keyvals)...)
}

func (l temporalClientLogger) Error(msg string, keyvals ...interface{}) {
	l.Logger.Error(msg, toZapFields(keyvals)...)
}

func toZapFields(keyvals []interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key, ok := keyvals[i].(string)
			if ok {
				fields = append(fields, zap.Any(key, keyvals[i+1]))
			}
		}
	}
	return fields
}