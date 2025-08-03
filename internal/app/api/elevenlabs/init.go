package elevenlabs

import (
	"fmt"
	"tiktok-whisper/internal/app/api/provider"
)

func init() {
	provider.RegisterProvider("elevenlabs", createElevenLabsProvider)
}

func createElevenLabsProvider(config map[string]interface{}) (provider.TranscriptionProvider, error) {
	// Extract settings
	settings, ok := config["settings"].(map[string]interface{})
	if !ok {
		settings = make(map[string]interface{})
	}

	// Extract auth
	auth, ok := config["auth"].(map[string]interface{})
	if !ok {
		auth = make(map[string]interface{})
	}

	// Get API key
	apiKey, _ := auth["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("elevenlabs provider requires 'api_key' in auth configuration")
	}

	// Create provider with settings
	return NewElevenLabsSTTProviderFromSettings(settings, apiKey)
}