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
	provider     TranscriptionProvider
	config       *ProviderConfiguration
	providerName string
}

// NewSimpleProviderTranscriber creates a transcriber that uses the provider framework
func NewSimpleProviderTranscriber() api.Transcriber {
	localConfig := "providers.yaml"
	homeConfig := filepath.Join(os.Getenv("HOME"), ".tiktok-whisper", "providers.yaml")
	configPath, warning, err := resolveProviderConfigPath(localConfig, homeConfig)
	if warning != "" {
		log.Printf("WARNING: %s", warning)
	}
	if err != nil {
		log.Fatal(err)
	}

	absConfigPath, _ := filepath.Abs(configPath)
	log.Printf("Loading provider config: %s", absConfigPath)

	configManager := NewConfigManager(configPath)
	config, err := configManager.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load provider configuration from %s: %v", absConfigPath, err)
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
			providerName = ProviderNameWhisperCpp
		}
	}

	providerConfig, exists := config.Providers[providerName]
	if !exists {
		log.Fatalf("Provider '%s' not found in configuration", providerName)
	}

	// Create provider factory
	factory := NewProviderFactory()
	
	authMap := providerConfig.Auth.ToMap()
	if authMap == nil {
		authMap = make(map[string]interface{})
	}
	configMap := map[string]interface{}{
		"type":     providerConfig.Type,
		"settings": providerConfig.Settings,
		"auth":     authMap,
	}

	// Create the provider
	provider, err := factory.CreateProvider(providerConfig.Type, configMap)
	if err != nil {
		log.Fatalf("Failed to create provider '%s': %v", providerName, err)
	}

	log.Printf("Using provider: %s (%s)", providerName, providerConfig.Type)

	return &SimpleProviderTranscriber{
		provider:     provider,
		config:       config,
		providerName: providerName,
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
	if providerConfig, exists := t.config.Providers[t.providerName]; exists {
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

func resolveProviderConfigPath(localConfig, homeConfig string) (string, string, error) {
	_, localErr := os.Stat(localConfig)
	_, homeErr := os.Stat(homeConfig)
	localExists := localErr == nil
	homeExists := homeErr == nil

	switch {
	case homeExists && localExists:
		absLocal, _ := filepath.Abs(localConfig)
		return homeConfig, fmt.Sprintf("both provider configs exist; preferring %s and ignoring %s", homeConfig, absLocal), nil
	case homeExists:
		return homeConfig, "", nil
	case localExists:
		return localConfig, "", nil
	default:
		return "", "", fmt.Errorf("no providers.yaml found. Expected at %s or %s", homeConfig, localConfig)
	}
}
