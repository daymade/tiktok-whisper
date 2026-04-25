package temporal

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/temporal/activities"
	"tiktok-whisper/internal/app/temporal/workflows"
)

var (
	temporalHost string
	taskQueue    string
	workerMode   string
)

// TemporalCmd represents the temporal command
var TemporalCmd = &cobra.Command{
	Use:   "temporal",
	Short: "Temporal workflow management commands",
	Long: `Commands for managing Temporal workflows and workers for distributed transcription.
	
Available subcommands:
  worker - Start a temporal worker
  status - Check temporal connection status`,
}

// workerCmd starts a temporal worker
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start a temporal worker",
	Long: `Start a temporal worker to process transcription workflows.

Modes:
  simple - Basic transcription without provider framework
  full   - Full transcription with provider framework and orchestration

Example:
  v2t temporal worker --mode full --host localhost:7233`,
	Run: runWorker,
}

// statusCmd checks temporal connection
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check temporal connection status",
	Run:   runStatus,
}

func init() {
	// Add subcommands
	TemporalCmd.AddCommand(workerCmd)
	TemporalCmd.AddCommand(statusCmd)
	
	// Worker flags
	workerCmd.Flags().StringVar(&temporalHost, "host", "localhost:7233", "Temporal server host:port")
	workerCmd.Flags().StringVar(&taskQueue, "queue", "v2t-transcription-queue", "Task queue name")
	workerCmd.Flags().StringVar(&workerMode, "mode", "full", "Worker mode: simple or full")
	
	// Status flags
	statusCmd.Flags().StringVar(&temporalHost, "host", "localhost:7233", "Temporal server host:port")
}

func runWorker(cmd *cobra.Command, args []string) {
	// Create logger
	config := zap.NewProductionConfig()
	if os.Getenv("ENV") == "development" {
		config = zap.NewDevelopmentConfig()
	}
	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
		Logger:   temporalClientLogger{logger},
	})
	if err != nil {
		logger.Fatal("Failed to create temporal client", zap.Error(err))
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: 10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register workflows
	w.RegisterWorkflow(workflows.SingleFileTranscriptionWorkflow)
	w.RegisterWorkflow(workflows.BatchTranscriptionWorkflow)
	w.RegisterWorkflow(workflows.TranscriptionWithFallbackWorkflow)
	
	logger.Info("Registered workflows",
		zap.String("queue", taskQueue),
		zap.String("mode", workerMode),
	)

	// Register activities based on mode
	switch workerMode {
	case "simple":
		// Simple mode - basic activities only
		simpleActivities := &activities.SimpleTranscribeActivities{}
		w.RegisterActivity(simpleActivities)
		logger.Info("Registered simple transcription activities")
		
	case "full":
		// Full mode - complete provider framework
		providerRegistry, err := initializeProviderRegistry(logger)
		if err != nil {
			logger.Fatal("Failed to initialize provider registry", zap.Error(err))
		}
		
		// Register full activities
		transcribeActivities := activities.NewTranscribeActivities(providerRegistry)
		storageActivities, err := activities.NewStorageActivities(
			os.Getenv("MINIO_ENDPOINT"),
			os.Getenv("MINIO_ACCESS_KEY"),
			os.Getenv("MINIO_SECRET_KEY"),
			os.Getenv("MINIO_BUCKET"),
		)
		if err != nil {
			logger.Warn("Failed to create storage activities", zap.Error(err))
		}
		
		w.RegisterActivity(transcribeActivities)
		if storageActivities != nil {
			w.RegisterActivity(storageActivities)
		}
		
		logger.Info("Registered full transcription activities with provider framework")
		
	default:
		logger.Fatal("Invalid mode", zap.String("mode", workerMode))
	}

	// Start worker in background
	go func() {
		err := w.Run(worker.InterruptCh())
		if err != nil {
			logger.Fatal("Worker failed", zap.Error(err))
		}
	}()

	logger.Info("Worker started successfully",
		zap.String("host", temporalHost),
		zap.String("queue", taskQueue),
		zap.String("mode", workerMode),
	)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down worker...")
	w.Stop()
	logger.Info("Worker stopped")
}

func runStatus(cmd *cobra.Command, args []string) {
	// Create temporal client to test connection
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		fmt.Printf("❌ Failed to connect to Temporal at %s: %v\n", temporalHost, err)
		os.Exit(1)
	}
	defer c.Close()

	// Test connection by describing namespace
	ctx := cmd.Context()
	_, err = c.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{
		Namespace: "default",
	})
	
	if err != nil {
		fmt.Printf("❌ Temporal connection test failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully connected to Temporal at %s\n", temporalHost)
	fmt.Printf("   Namespace: default\n")
	fmt.Printf("   Status: healthy\n")
}

// initializeProviderRegistry creates and configures the provider registry
func initializeProviderRegistry(logger *zap.Logger) (provider.ProviderRegistry, error) {
	reg := provider.NewProviderRegistry()
	factory := provider.NewProviderFactory()

	// Load provider configuration
	configPath := os.Getenv("PROVIDER_CONFIG_PATH")
	if configPath == "" {
		configPath = os.ExpandEnv("$HOME/.tiktok-whisper/providers.yaml")
	}

	// Try to load config file
	if _, err := os.Stat(configPath); err == nil {
		// TODO: Load from config file
		logger.Info("Loading provider configuration", zap.String("path", configPath))
	}

	// Create default whisper_cpp provider
	whisperConfig := map[string]interface{}{
		"binary_path": os.Getenv("WHISPER_BINARY_PATH"),
		"model_path":  os.Getenv("WHISPER_MODEL_PATH"),
		"language":    "auto",
	}

	if whisperConfig["binary_path"] == "" {
		whisperConfig["binary_path"] = "/usr/local/bin/whisper"
	}
	if whisperConfig["model_path"] == "" {
		whisperConfig["model_path"] = "/models/ggml-base.bin"
	}

	whisperProvider, err := factory.CreateProvider("whisper_cpp", whisperConfig)
	if err != nil {
		logger.Warn("Failed to create whisper_cpp provider", zap.Error(err))
	} else {
		if err := reg.RegisterProvider("whisper_cpp", whisperProvider); err != nil {
			logger.Warn("Failed to register whisper_cpp provider", zap.Error(err))
		} else {
			logger.Info("Registered whisper_cpp provider")
		}
	}

	// Create OpenAI provider if API key is available
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		openaiConfig := map[string]interface{}{
			"api_key": apiKey,
			"model":   "whisper-1",
		}
		
		openaiProvider, err := factory.CreateProvider("openai", openaiConfig)
		if err != nil {
			logger.Warn("Failed to create OpenAI provider", zap.Error(err))
		} else {
			if err := reg.RegisterProvider("openai", openaiProvider); err != nil {
				logger.Warn("Failed to register OpenAI provider", zap.Error(err))
			} else {
				logger.Info("Registered OpenAI provider")
			}
		}
	}

	// Set default provider
	if err := reg.SetDefaultProvider("whisper_cpp"); err != nil {
		logger.Warn("Failed to set default provider", zap.Error(err))
	}

	return reg, nil
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