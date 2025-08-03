package whisper

import (
	"fmt"
	"tiktok-whisper/internal/app/api/provider"
)

func init() {
	// Register openai provider with the factory
	provider.RegisterProvider("openai", createOpenAIProvider)
}

// createOpenAIProvider creates an OpenAI Whisper provider from configuration
func createOpenAIProvider(config map[string]interface{}) (provider.TranscriptionProvider, error) {
	// Extract settings from config
	settings, ok := config["settings"].(map[string]interface{})
	if !ok {
		settings = make(map[string]interface{})
	}
	
	// Extract auth from config
	auth, ok := config["auth"].(map[string]interface{})
	if !ok {
		auth = make(map[string]interface{})
	}
	
	// Get API key from auth or config - required
	var apiKey string
	if key, ok := auth["api_key"].(string); ok && key != "" {
		apiKey = key
	} else if key, ok := config["api_key"].(string); ok && key != "" {
		apiKey = key
	}
	
	if apiKey == "" {
		return nil, fmt.Errorf("openai provider requires 'api_key' in auth configuration")
	}
	
	// Create provider config from settings
	providerConfig := OpenAIProviderConfig{
		APIKey: apiKey,
	}
	
	// Extract optional settings for the enhanced provider
	if model, ok := settings["model"].(string); ok {
		providerConfig.Model = model
	} else {
		providerConfig.Model = "whisper-1"
	}
	
	if language, ok := settings["language"].(string); ok {
		providerConfig.Language = language
	}
	
	if prompt, ok := settings["prompt"].(string); ok {
		providerConfig.Prompt = prompt
	}
	
	if responseFormat, ok := settings["response_format"].(string); ok {
		providerConfig.ResponseFormat = responseFormat
	} else {
		providerConfig.ResponseFormat = "text"
	}
	
	if temperature, ok := settings["temperature"].(float64); ok {
		providerConfig.Temperature = float32(temperature)
	}
	
	if baseURL, ok := settings["base_url"].(string); ok {
		providerConfig.BaseURL = baseURL
	}
	
	// Return enhanced version
	return NewEnhancedRemoteTranscriber(providerConfig), nil
}