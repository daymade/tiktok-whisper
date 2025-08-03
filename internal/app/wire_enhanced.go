//go:build wireinject
// +build wireinject

package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/api/elevenlabs"
	"tiktok-whisper/internal/app/api/openai"
	"tiktok-whisper/internal/app/api/openai/whisper"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/api/whisper_cpp"
	"tiktok-whisper/internal/app/converter"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/util/files"

	"github.com/google/wire"
)

// Enhanced provider functions

// provideProviderConfig loads the provider configuration
func provideProviderConfig() (*provider.ProviderConfiguration, error) {
	configPath := provider.GetDefaultConfigPath()
	configManager := provider.NewConfigManager(configPath)
	return configManager.LoadConfig()
}

// provideConfigManager creates a configuration manager
func provideConfigManager() *provider.ConfigManager {
	configPath := provider.GetDefaultConfigPath()
	return provider.NewConfigManager(configPath)
}

// provideProviderMetrics creates provider metrics
func provideProviderMetrics() provider.ProviderMetrics {
	return provider.NewProviderMetrics()
}

// provideProviderRegistry creates and configures the provider registry
func provideProviderRegistry(config *provider.ProviderConfiguration) (provider.ProviderRegistry, error) {
	registry := provider.NewProviderRegistry()
	
	// Register providers based on configuration
	for name, providerConfig := range config.Providers {
		if !providerConfig.Enabled {
			continue
		}
		
		var transcriptionProvider provider.TranscriptionProvider
		var err error
		
		switch providerConfig.Type {
		case "whisper_cpp":
			transcriptionProvider, err = createWhisperCppProvider(providerConfig)
		case "openai":
			transcriptionProvider, err = createOpenAIProvider(providerConfig)
		case "elevenlabs":
			transcriptionProvider, err = createElevenLabsProvider(providerConfig)
		default:
			log.Printf("Unknown provider type: %s, skipping", providerConfig.Type)
			continue
		}
		
		if err != nil {
			log.Printf("Failed to create provider %s: %v", name, err)
			continue
		}
		
		if err := registry.RegisterProvider(name, transcriptionProvider); err != nil {
			log.Printf("Failed to register provider %s: %v", name, err)
			continue
		}
		
		log.Printf("Registered provider: %s (%s)", name, providerConfig.Type)
	}
	
	// Set default provider
	if config.DefaultProvider != "" {
		if err := registry.SetDefaultProvider(config.DefaultProvider); err != nil {
			log.Printf("Warning: Failed to set default provider %s: %v", config.DefaultProvider, err)
		}
	}
	
	return registry, nil
}

// createWhisperCppProvider creates a whisper.cpp provider from configuration
func createWhisperCppProvider(config provider.ProviderConfig) (provider.TranscriptionProvider, error) {
	return whisper_cpp.NewEnhancedLocalTranscriberFromSettings(config.Settings)
}

// createOpenAIProvider creates an OpenAI provider from configuration  
func createOpenAIProvider(config provider.ProviderConfig) (provider.TranscriptionProvider, error) {
	apiKey := config.Auth.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	
	return whisper.NewEnhancedRemoteTranscriberFromSettings(config.Settings, apiKey)
}

// createElevenLabsProvider creates an ElevenLabs provider from configuration
func createElevenLabsProvider(config provider.ProviderConfig) (provider.TranscriptionProvider, error) {
	apiKey := config.Auth.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ELEVENLABS_API_KEY")
	}
	
	if apiKey == "" {
		return nil, fmt.Errorf("ElevenLabs API key is required")
	}
	
	return elevenlabs.NewElevenLabsSTTProviderFromSettings(config.Settings, apiKey)
}

// provideTranscriptionOrchestrator creates the orchestrator
func provideTranscriptionOrchestrator(
	registry provider.ProviderRegistry,
	metrics provider.ProviderMetrics,
	config *provider.ProviderConfiguration,
) provider.TranscriptionOrchestrator {
	return provider.NewTranscriptionOrchestrator(registry, metrics, config.Orchestrator)
}

// Enhanced converter that uses the orchestrator
type EnhancedConverter struct {
	orchestrator provider.TranscriptionOrchestrator
	db           repository.TranscriptionDAO
}

// NewEnhancedConverter creates a new enhanced converter
func NewEnhancedConverter(
	orchestrator provider.TranscriptionOrchestrator,
	db repository.TranscriptionDAO,
) *EnhancedConverter {
	return &EnhancedConverter{
		orchestrator: orchestrator,
		db:           db,
	}
}

// Backward compatibility - implement converter interface
func (ec *EnhancedConverter) ConvertDirectoryToText(inputDir, userNickname string) {
	// Implementation would use the orchestrator for transcription
	// This is a placeholder for backward compatibility
}

// Backward compatibility wrappers for existing functions
func provideBackwardCompatibleTranscriber(registry provider.ProviderRegistry) api.Transcriber {
	// Return the default provider for backward compatibility
	defaultProvider, err := registry.GetDefaultProvider()
	if err != nil {
		log.Printf("Warning: No default provider available, falling back to local whisper.cpp")
		return provideLocalTranscriber()
	}
	
	// Create a wrapper that implements the old interface
	return &backwardCompatibilityWrapper{provider: defaultProvider}
}

// backwardCompatibilityWrapper wraps a TranscriptionProvider to implement the old Transcriber interface
type backwardCompatibilityWrapper struct {
	provider provider.TranscriptionProvider
}

func (w *backwardCompatibilityWrapper) Transcript(inputFilePath string) (string, error) {
	return w.provider.Transcript(inputFilePath)
}

// Original provider functions (for fallback)

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

// Enhanced wire injectors

// InitializeEnhancedConverter creates a new enhanced converter with provider registry
func InitializeEnhancedConverter() (*EnhancedConverter, error) {
	wire.Build(
		NewEnhancedConverter,
		provideTranscriptionOrchestrator,
		provideProviderRegistry,
		provideProviderMetrics,
		provideProviderConfig,
		provideTranscriptionDAO,
	)
	return &EnhancedConverter{}, nil
}

// InitializeProviderRegistry creates a standalone provider registry
func InitializeProviderRegistry() (provider.ProviderRegistry, error) {
	wire.Build(
		provideProviderRegistry,
		provideProviderConfig,
	)
	return nil, nil
}

// InitializeTranscriptionOrchestrator creates a standalone orchestrator
func InitializeTranscriptionOrchestrator() (provider.TranscriptionOrchestrator, error) {
	wire.Build(
		provideTranscriptionOrchestrator,
		provideProviderRegistry,
		provideProviderMetrics,
		provideProviderConfig,
	)
	return nil, nil
}

// Backward compatible injectors

// InitializeConverter maintains backward compatibility
func InitializeConverter() *converter.Converter {
	// Try to use enhanced converter first, fallback to original
	enhancedConverter, err := InitializeEnhancedConverter()
	if err != nil {
		log.Printf("Warning: Failed to initialize enhanced converter, falling back to original: %v", err)
		wire.Build(converter.NewConverter, provideLocalTranscriber, provideTranscriptionDAO)
		return &converter.Converter{}
	}
	
	// Wrap enhanced converter to maintain interface compatibility
	// This would require implementing the conversion between interfaces
	// For now, fallback to original
	wire.Build(converter.NewConverter, provideLocalTranscriber, provideTranscriptionDAO)
	return &converter.Converter{}
}

// InitializeProgressAwareConverter maintains backward compatibility
func InitializeProgressAwareConverter(config converter.ProgressConfig) *converter.ProgressAwareConverter {
	wire.Build(converter.NewConverter, converter.NewProgressAwareConverter, provideLocalTranscriber, provideTranscriptionDAO)
	return &converter.ProgressAwareConverter{}
}

// InitializeBackwardCompatibleTranscriber provides backward compatible transcriber
func InitializeBackwardCompatibleTranscriber() (api.Transcriber, error) {
	wire.Build(
		provideBackwardCompatibleTranscriber,
		provideProviderRegistry,
		provideProviderConfig,
	)
	return nil, nil
}