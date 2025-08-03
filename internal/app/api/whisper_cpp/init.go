package whisper_cpp

import (
	"fmt"
	"tiktok-whisper/internal/app/api/provider"
)

func init() {
	// Register whisper_cpp provider with the factory
	provider.RegisterProvider("whisper_cpp", createWhisperCppProvider)
}

// createWhisperCppProvider creates a whisper.cpp provider from configuration
func createWhisperCppProvider(config map[string]interface{}) (provider.TranscriptionProvider, error) {
	// Extract settings from config
	settings, ok := config["settings"].(map[string]interface{})
	if !ok {
		settings = config // Use entire config as settings if not nested
	}
	
	// Get binary path - required
	binaryPath, ok := settings["binary_path"].(string)
	if !ok || binaryPath == "" {
		return nil, fmt.Errorf("whisper_cpp provider requires 'binary_path' setting")
	}
	
	// Get model path - required
	modelPath, ok := settings["model_path"].(string)
	if !ok || modelPath == "" {
		return nil, fmt.Errorf("whisper_cpp provider requires 'model_path' setting")
	}
	
	// Create local provider config from settings
	providerConfig := LocalProviderConfig{
		BinaryPath: binaryPath,
		ModelPath:  modelPath,
	}
	
	// Extract optional settings for the enhanced provider
	if language, ok := settings["language"].(string); ok {
		providerConfig.Language = language
	}
	
	if prompt, ok := settings["prompt"].(string); ok {
		providerConfig.Prompt = prompt
	}
	
	if outputFormat, ok := settings["output_format"].(string); ok {
		providerConfig.OutputFormat = outputFormat
	}
	
	if maxConcurrent, ok := settings["max_concurrent"].(float64); ok {
		providerConfig.MaxConcurrent = int(maxConcurrent)
	} else if maxConcurrent, ok := settings["max_concurrent"].(int); ok {
		providerConfig.MaxConcurrent = maxConcurrent
	}
	
	if tempDir, ok := settings["temp_dir"].(string); ok {
		providerConfig.TempDir = tempDir
	}
	
	// Return enhanced version
	return NewEnhancedLocalTranscriber(providerConfig), nil
}