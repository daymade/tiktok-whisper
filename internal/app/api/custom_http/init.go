package custom_http

import (
	"tiktok-whisper/internal/app/api/provider"
)

func init() {
	provider.RegisterProvider("custom_http", createCustomHTTPProvider)
}

func createCustomHTTPProvider(config map[string]interface{}) (provider.TranscriptionProvider, error) {
	// Extract settings
	settings, ok := config["settings"].(map[string]interface{})
	if !ok {
		settings = make(map[string]interface{})
	}

	// Create provider with settings
	return NewCustomHTTPProvider(settings)
}