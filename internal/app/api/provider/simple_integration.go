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
	// Load configuration
	configPath := filepath.Join(os.Getenv("HOME"), ".tiktok-whisper", "providers.yaml")
	configManager := NewConfigManager(configPath)
	
	config, err := configManager.LoadConfig()
	if err != nil {
		log.Printf("Failed to load provider configuration: %v", err)
		return nil
	}

	// Get the default provider
	defaultProviderName := config.DefaultProvider
	if defaultProviderName == "" {
		defaultProviderName = "whisper_cpp"
	}

	providerConfig, exists := config.Providers[defaultProviderName]
	if !exists {
		log.Printf("Default provider '%s' not found in configuration", defaultProviderName)
		return nil
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
		log.Printf("Failed to create provider '%s': %v", defaultProviderName, err)
		return nil
	}

	log.Printf("Using provider: %s (%s)", defaultProviderName, providerConfig.Type)

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