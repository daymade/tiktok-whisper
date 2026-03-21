package custom_http

import (
	"tiktok-whisper/internal/app/api/provider"
)

func init() {
	provider.RegisterProvider("custom_http", createCustomHTTPProvider)
}

func createCustomHTTPProvider(config map[string]interface{}) (provider.TranscriptionProvider, error) {
	// The config passed from the factory is already the settings map
	// No need to extract nested settings
	return NewCustomHTTPProvider(config)
}