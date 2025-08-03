package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"tiktok-whisper/internal/app/api"
)

// SimpleProviderTranscriber provides a simple integration of the provider framework
type SimpleProviderTranscriber struct {
	provider TranscriptionProvider
	config   *ProviderConfiguration
}

// NewSimpleProviderTranscriber creates a transcriber that uses the provider framework
func NewSimpleProviderTranscriber() api.Transcriber {
	// Determine config path - priority order:
	// 1. Local providers.yaml
	// 2. Home directory ~/.tiktok-whisper/providers.yaml
	var configPath string
	if _, err := os.Stat("providers.yaml"); err == nil {
		configPath = "providers.yaml"
	} else {
		configPath = filepath.Join(os.Getenv("HOME"), ".tiktok-whisper", "providers.yaml")
	}
	
	configManager := NewConfigManager(configPath)
	config, err := configManager.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load provider configuration from %s: %v", configPath, err)
	}

	// Get the provider name - priority order:
	// 1. Runtime provider override
	// 2. Default from config
	// 3. Fallback to whisper_cpp
	runtimeCfg := GetRuntimeConfig()
	var providerName string
	if runtimeCfg != nil && runtimeCfg.ProviderName != "" {
		providerName = runtimeCfg.ProviderName
	} else {
		providerName = config.DefaultProvider
		if providerName == "" {
			providerName = "whisper_cpp"
		}
	}

	providerConfig, exists := config.Providers[providerName]
	if !exists {
		log.Fatalf("Provider '%s' not found in configuration", providerName)
	}

	// Create provider factory
	factory := NewProviderFactory()
	
	// Convert ProviderConfig to map for factory
	configMap := map[string]interface{}{
		"type":     providerConfig.Type,
		"settings": providerConfig.Settings,
		"auth":     providerConfig.Auth,
	}

	// Create the provider
	provider, err := factory.CreateProvider(providerConfig.Type, configMap)
	if err != nil {
		log.Fatalf("Failed to create provider '%s': %v", providerName, err)
	}

	log.Printf("Using provider: %s (%s)", providerName, providerConfig.Type)

	return &SimpleProviderTranscriber{
		provider: provider,
		config:   config,
	}
}

// Transcript implements the Transcriber interface
func (t *SimpleProviderTranscriber) Transcript(inputFilePath string) (string, error) {
	if t.provider == nil {
		return "", fmt.Errorf("provider not initialized")
	}

	// Create context with timeout
	timeout := 5 * time.Minute
	if t.config != nil && t.config.Global.GlobalTimeoutSec > 0 {
		timeout = time.Duration(t.config.Global.GlobalTimeoutSec) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create transcription request
	request := &TranscriptionRequest{
		InputFilePath: inputFilePath,
	}

	// Get provider settings
	var language string
	var prompt string
	if providerConfig, exists := t.config.Providers[t.config.DefaultProvider]; exists {
		if lang, ok := providerConfig.Settings["language"].(string); ok {
			language = lang
		}
		if p, ok := providerConfig.Settings["prompt"].(string); ok {
			prompt = p
		}
	}

	if language != "" {
		request.Language = language
	}
	if prompt != "" {
		request.Prompt = prompt
	}

	// Execute transcription
	response, err := t.provider.TranscriptWithOptions(ctx, request)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}

	return response.Text, nil
}