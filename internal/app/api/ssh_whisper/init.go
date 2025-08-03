package ssh_whisper

import (
	"fmt"
	"tiktok-whisper/internal/app/api/provider"
)

func init() {
	provider.RegisterProvider("ssh_whisper", createSSHWhisperProvider)
}

func createSSHWhisperProvider(config map[string]interface{}) (provider.TranscriptionProvider, error) {
	// Extract settings
	settings, ok := config["settings"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("ssh_whisper provider requires 'settings' configuration")
	}

	// Use the existing NewSSHWhisperProviderFromSettings function
	return NewSSHWhisperProviderFromSettings(settings)
}