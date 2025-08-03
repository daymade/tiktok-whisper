package whisper_server

import (
	"fmt"
	"tiktok-whisper/internal/app/api/provider"
)

func init() {
	provider.RegisterProvider("whisper_server", createWhisperServerProvider)
}

func createWhisperServerProvider(config map[string]interface{}) (provider.TranscriptionProvider, error) {
	// Extract settings
	settings, ok := config["settings"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("whisper_server provider requires 'settings' configuration")
	}

	// Get required server URL
	serverURL, ok := settings["server_url"].(string)
	if !ok || serverURL == "" {
		return nil, fmt.Errorf("whisper_server provider requires 'server_url' setting")
	}

	// Optional settings
	model, _ := settings["model"].(string)
	if model == "" {
		model = "base"
	}

	language, _ := settings["language"].(string)
	outputFormat, _ := settings["output_format"].(string)
	if outputFormat == "" {
		outputFormat = "json"
	}

	// Optional auth token
	authToken, _ := settings["auth_token"].(string)

	// Create provider with settings
	providerSettings := map[string]interface{}{
		"server_url":    serverURL,
		"model":         model,
		"language":      language,
		"output_format": outputFormat,
	}

	if authToken != "" {
		providerSettings["auth_token"] = authToken
	}

	return NewWhisperServerProviderFromSettings(settings)
}