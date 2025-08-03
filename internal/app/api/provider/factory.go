package provider

import (
	"fmt"
)

// DefaultProviderFactory implements ProviderFactory interface
type DefaultProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *DefaultProviderFactory {
	return &DefaultProviderFactory{}
}

// CreateProvider creates a provider instance based on type and configuration
func (f *DefaultProviderFactory) CreateProvider(providerType string, config map[string]interface{}) (TranscriptionProvider, error) {
	switch providerType {
	case "whisper_cpp":
		return f.createWhisperCppProvider(config)
	case "openai":
		return f.createOpenAIProvider(config)
	case "elevenlabs":
		return f.createElevenLabsProvider(config)
	case "ssh_whisper":
		return f.createSSHWhisperProvider(config)
	case "whisper_server":
		return f.createWhisperServerProvider(config)
	case "custom_http":
		return f.createCustomHTTPProvider(config)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// GetAvailableProviders returns a list of registered provider types
func (f *DefaultProviderFactory) GetAvailableProviders() []string {
	// Return only registered providers
	return ListRegisteredProviders()
}

// GetProviderInfo returns provider information without creating an instance
func (f *DefaultProviderFactory) GetProviderInfo(providerType string) (ProviderInfo, error) {
	switch providerType {
	case "whisper_cpp":
		return f.getWhisperCppInfo(), nil
	case "openai":
		return f.getOpenAIInfo(), nil
	case "elevenlabs":
		return f.getElevenLabsInfo(), nil
	case "ssh_whisper":
		return f.getSSHWhisperInfo(), nil
	case "whisper_server":
		return f.getWhisperServerInfo(), nil
	case "custom_http":
		return f.getCustomHTTPInfo(), nil
	default:
		return ProviderInfo{}, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// createWhisperCppProvider creates a whisper.cpp provider
func (f *DefaultProviderFactory) createWhisperCppProvider(config map[string]interface{}) (TranscriptionProvider, error) {
	creator, err := GetProviderCreator("whisper_cpp")
	if err != nil {
		return nil, fmt.Errorf("whisper_cpp provider not registered: %w", err)
	}
	return creator(config)
}

// createOpenAIProvider creates an OpenAI provider
func (f *DefaultProviderFactory) createOpenAIProvider(config map[string]interface{}) (TranscriptionProvider, error) {
	creator, err := GetProviderCreator("openai")
	if err != nil {
		return nil, fmt.Errorf("openai provider not registered: %w", err)
	}
	return creator(config)
}

// createElevenLabsProvider creates an ElevenLabs provider
func (f *DefaultProviderFactory) createElevenLabsProvider(config map[string]interface{}) (TranscriptionProvider, error) {
	creator, err := GetProviderCreator("elevenlabs")
	if err != nil {
		return nil, fmt.Errorf("elevenlabs provider not registered: %w", err)
	}
	return creator(config)
}

// createSSHWhisperProvider creates an SSH whisper provider
func (f *DefaultProviderFactory) createSSHWhisperProvider(config map[string]interface{}) (TranscriptionProvider, error) {
	creator, err := GetProviderCreator("ssh_whisper")
	if err != nil {
		return nil, fmt.Errorf("ssh_whisper provider not registered: %w", err)
	}
	return creator(config)
}

// createWhisperServerProvider creates a whisper-server HTTP provider
func (f *DefaultProviderFactory) createWhisperServerProvider(config map[string]interface{}) (TranscriptionProvider, error) {
	creator, err := GetProviderCreator("whisper_server")
	if err != nil {
		return nil, fmt.Errorf("whisper_server provider not registered: %w", err)
	}
	return creator(config)
}

// createCustomHTTPProvider creates a custom HTTP provider
func (f *DefaultProviderFactory) createCustomHTTPProvider(config map[string]interface{}) (TranscriptionProvider, error) {
	creator, err := GetProviderCreator("custom_http")
	if err != nil {
		return nil, fmt.Errorf("custom_http provider not registered: %w", err)
	}
	return creator(config)
}

// Provider info methods

func (f *DefaultProviderFactory) getWhisperCppInfo() ProviderInfo {
	return ProviderInfo{
		Name:        "whisper_cpp",
		DisplayName: "Whisper.cpp (Local)",
		Type:        ProviderTypeLocal,
		Version:     "1.0.0",
		SupportedFormats: []AudioFormat{
			FormatWAV,
			FormatMP3,
			FormatM4A,
			FormatFLAC,
		},
		SupportedLanguages: []string{
			"zh", "en", "ja", "ko", "es", "fr", "de", "it", "pt", "ru",
			"ar", "tr", "pl", "nl", "sv", "da", "no", "fi", "hu", "cs",
		},
		MaxFileSizeMB:             0, // No specific limit
		MaxDurationSec:            0, // No specific limit
		SupportsTimestamps:        true,
		SupportsWordLevel:         false,
		SupportsConfidence:        false,
		SupportsLanguageDetection: true,
		SupportsStreaming:         false,
		RequiresInternet:          false,
		RequiresAPIKey:            false,
		RequiresBinary:            true,
		DefaultModel:              "ggml-large-v2.bin",
		AvailableModels: []string{
			"ggml-tiny.bin", "ggml-base.bin", "ggml-small.bin",
			"ggml-medium.bin", "ggml-large-v1.bin", "ggml-large-v2.bin", "ggml-large-v3.bin",
		},
		TypicalLatencyMs: 5000,
		ConfigSchema: map[string]interface{}{
			"binary_path": map[string]string{
				"type":        "string",
				"description": "Path to whisper.cpp binary",
				"required":    "true",
			},
			"model_path": map[string]string{
				"type":        "string",
				"description": "Path to whisper model file",
				"required":    "true",
			},
		},
	}
}

func (f *DefaultProviderFactory) getOpenAIInfo() ProviderInfo {
	return ProviderInfo{
		Name:        "openai",
		DisplayName: "OpenAI Whisper API",
		Type:        ProviderTypeRemote,
		Version:     "1.0.0",
		SupportedFormats: []AudioFormat{
			FormatMP3,
			FormatM4A,
			FormatWAV,
			FormatWEBM,
		},
		SupportedLanguages:        []string{}, // All languages
		MaxFileSizeMB:             25,
		MaxDurationSec:            0,
		SupportsTimestamps:        true,
		SupportsWordLevel:         true,
		SupportsConfidence:        true,
		SupportsLanguageDetection: true,
		SupportsStreaming:         false,
		RequiresInternet:          true,
		RequiresAPIKey:            true,
		RequiresBinary:            false,
		DefaultModel:              "whisper-1",
		AvailableModels:           []string{"whisper-1"},
		TypicalLatencyMs:          2000,
		CostPerMinute:             "$0.006",
		ConfigSchema: map[string]interface{}{
			"api_key": map[string]string{
				"type":        "string",
				"description": "OpenAI API key",
				"required":    "true",
			},
		},
	}
}

func (f *DefaultProviderFactory) getElevenLabsInfo() ProviderInfo {
	return ProviderInfo{
		Name:        "elevenlabs",
		DisplayName: "ElevenLabs Speech-to-Text",
		Type:        ProviderTypeRemote,
		Version:     "1.0.0",
		SupportedFormats: []AudioFormat{
			FormatMP3,
			FormatWAV,
			FormatFLAC,
			FormatM4A,
		},
		SupportedLanguages:        []string{}, // Many languages
		MaxFileSizeMB:             25,
		MaxDurationSec:            0,
		SupportsTimestamps:        false,
		SupportsWordLevel:         true,
		SupportsConfidence:        false,
		SupportsLanguageDetection: true,
		SupportsStreaming:         false,
		RequiresInternet:          true,
		RequiresAPIKey:            true,
		RequiresBinary:            false,
		DefaultModel:              "whisper-large-v3",
		AvailableModels: []string{
			"whisper-large-v3",
			"whisper-large-v2",
		},
		TypicalLatencyMs: 3000,
		CostPerMinute:    "Variable",
		ConfigSchema: map[string]interface{}{
			"api_key": map[string]string{
				"type":        "string",
				"description": "ElevenLabs API key",
				"required":    "true",
			},
		},
	}
}

func (f *DefaultProviderFactory) getSSHWhisperInfo() ProviderInfo {
	return ProviderInfo{
		Name:        "ssh_whisper",
		DisplayName: "SSH Remote Whisper.cpp",
		Type:        ProviderTypeHybrid,
		Version:     "1.0.0",
		SupportedFormats: []AudioFormat{
			FormatWAV,
			FormatMP3,
			FormatM4A,
			FormatFLAC,
		},
		SupportedLanguages:        []string{}, // Whisper supports all languages
		MaxFileSizeMB:             1000,       // Limited by SSH transfer
		MaxDurationSec:            3600,       // 1 hour
		SupportsTimestamps:        true,
		SupportsWordLevel:         false,
		SupportsConfidence:        false,
		SupportsLanguageDetection: true,
		SupportsStreaming:         false,
		RequiresInternet:          false, // Requires SSH access
		RequiresAPIKey:            false,
		RequiresBinary:            true,  // Requires remote binary
		DefaultModel:              "models/ggml-base.en.bin",
		AvailableModels: []string{
			"models/ggml-tiny.bin",
			"models/ggml-tiny.en.bin",
			"models/ggml-base.bin",
			"models/ggml-base.en.bin",
			"models/ggml-small.bin",
			"models/ggml-small.en.bin",
			"models/ggml-medium.bin",
			"models/ggml-medium.en.bin",
			"models/ggml-large-v1.bin",
			"models/ggml-large-v2.bin",
			"models/ggml-large-v3.bin",
		},
		TypicalLatencyMs: 5000, // Includes file transfer time
		CostPerMinute:    "Free (SSH + Compute)",
		ConfigSchema: map[string]interface{}{
			"host": map[string]string{
				"type":        "string",
				"description": "SSH host (user@hostname)",
				"required":    "true",
			},
			"remote_dir": map[string]string{
				"type":        "string",
				"description": "Remote whisper.cpp directory",
				"required":    "true",
			},
			"binary_path": map[string]string{
				"type":        "string",
				"description": "Remote binary path (relative to remote_dir)",
				"default":     "./build/bin/whisper-cli",
			},
			"model_path": map[string]string{
				"type":        "string",
				"description": "Remote model path (relative to remote_dir)",
				"default":     "models/ggml-base.en.bin",
			},
			"language": map[string]string{
				"type":        "string",
				"description": "Language code (auto-detect if not specified)",
			},
			"threads": map[string]string{
				"type":        "number",
				"description": "Number of threads for processing",
				"default":     "4",
			},
		},
	}
}

func (f *DefaultProviderFactory) getCustomHTTPInfo() ProviderInfo {
	return ProviderInfo{
		Name:        "custom_http",
		DisplayName: "Custom HTTP Whisper Service",
		Type:        ProviderTypeRemote,
		Version:     "1.0.0",
		SupportedFormats: []AudioFormat{
			FormatWAV,
			FormatMP3,
			FormatM4A,
			FormatFLAC,
		},
		SupportedLanguages:        []string{}, // Depends on implementation
		MaxFileSizeMB:             0,          // Depends on implementation
		MaxDurationSec:            0,          // Depends on implementation
		SupportsTimestamps:        false,      // Depends on implementation
		SupportsWordLevel:         false,      // Depends on implementation
		SupportsConfidence:        false,      // Depends on implementation
		SupportsLanguageDetection: false,      // Depends on implementation
		SupportsStreaming:         false,      // Depends on implementation
		RequiresInternet:          true,
		RequiresAPIKey:            false, // Depends on implementation
		RequiresBinary:            false,
		DefaultModel:              "",
		AvailableModels:           []string{},
		TypicalLatencyMs:          0, // Unknown
		ConfigSchema: map[string]interface{}{
			"base_url": map[string]string{
				"type":        "string",
				"description": "Base URL of the HTTP service",
				"required":    "true",
			},
			"api_key": map[string]string{
				"type":        "string",
				"description": "API key (if required)",
				"required":    "false",
			},
		},
	}
}

func (f *DefaultProviderFactory) getWhisperServerInfo() ProviderInfo {
	return ProviderInfo{
		Name:        "whisper_server",
		DisplayName: "Whisper Server (HTTP API)",
		Type:        ProviderTypeRemote,
		Version:     "1.0.0",
		SupportedFormats: []AudioFormat{
			FormatWAV,
			FormatMP3,
			FormatM4A,
			FormatFLAC,
			FormatOGG,
			FormatWEBM,
		},
		SupportedLanguages:        []string{}, // Whisper supports all languages
		MaxFileSizeMB:             100,        // Typical server limit
		MaxDurationSec:            3600,       // 1 hour
		SupportsTimestamps:        true,
		SupportsWordLevel:         true,
		SupportsConfidence:        true,
		SupportsLanguageDetection: true,
		SupportsStreaming:         false,
		RequiresInternet:          true, // Requires network access to server
		RequiresAPIKey:            false, // Basic whisper-server doesn't require auth
		RequiresBinary:            false,
		DefaultModel:              "whisper-server",
		AvailableModels: []string{
			"whisper-server", // Server manages model internally
		},
		TypicalLatencyMs: 2000, // Depends on server performance
		CostPerMinute:    "Free (Self-hosted)",
		ConfigSchema: map[string]interface{}{
			"base_url": map[string]string{
				"type":        "string",
				"description": "Base URL of whisper-server (e.g., http://192.168.1.100:8080)",
				"required":    "true",
			},
			"inference_path": map[string]string{
				"type":        "string",
				"description": "Inference endpoint path",
				"default":     "/inference",
			},
			"timeout": map[string]string{
				"type":        "number",
				"description": "Request timeout in seconds",
				"default":     "60",
			},
			"language": map[string]string{
				"type":        "string",
				"description": "Default language code (auto-detect if not specified)",
			},
			"response_format": map[string]string{
				"type":        "string",
				"description": "Response format (json, text, srt, vtt, verbose_json)",
				"default":     "json",
			},
			"temperature": map[string]string{
				"type":        "number",
				"description": "Decoding temperature (0.0-1.0)",
				"default":     "0.0",
			},
			"translate": map[string]string{
				"type":        "boolean",
				"description": "Translate to English",
				"default":     "false",
			},
			"no_timestamps": map[string]string{
				"type":        "boolean",
				"description": "Disable timestamps",
				"default":     "false",
			},
		},
	}
}

// BuildProviderFromConfig builds a provider from ProviderConfig
func BuildProviderFromConfig(name string, config ProviderConfig) (TranscriptionProvider, error) {
	// TODO: Implement provider creation without import cycles
	// This function should be implemented in a higher-level package that can import
	// both the provider interfaces and the concrete implementations
	return nil, fmt.Errorf("provider creation from config not yet implemented due to import cycle constraints")
}