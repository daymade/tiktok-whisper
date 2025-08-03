package config

import "time"

// Provider default configuration constants
const (
	// Timeout defaults
	DefaultWhisperCppTimeout = 300 * time.Second
	DefaultOpenAITimeout     = 60 * time.Second
	DefaultElevenLabsTimeout = 120 * time.Second
	DefaultSSHTimeout        = 180 * time.Second
	DefaultHTTPTimeout       = 120 * time.Second

	// Concurrency defaults
	DefaultWhisperCppConcurrency = 2
	DefaultOpenAIConcurrency     = 5
	DefaultElevenLabsConcurrency = 3
	DefaultSSHConcurrency        = 1
	DefaultHTTPConcurrency       = 2

	// Retry defaults
	DefaultRetries      = 2
	DefaultRetryDelayMs = 1000

	// OpenAI specific
	DefaultOpenAIMaxRetries = 3

	// Network defaults
	DefaultHTTPPort = "8080"
	DefaultSSHPort  = "22"

	// Model defaults
	DefaultWhisperModel    = "ggml-large-v2.bin"
	DefaultWhisperLanguage = "zh"
	DefaultWhisperPrompt   = "以下是简体中文普通话:"
)

// ProviderDefaults holds all default configurations for providers
type ProviderDefaults struct {
	Timeout      time.Duration
	Concurrency  int
	Retries      int
	RetryDelayMs int
}

// GetProviderDefaults returns default configuration for a given provider type
func GetProviderDefaults(providerType string) ProviderDefaults {
	switch providerType {
	case "whisper_cpp":
		return ProviderDefaults{
			Timeout:      DefaultWhisperCppTimeout,
			Concurrency:  DefaultWhisperCppConcurrency,
			Retries:      DefaultRetries,
			RetryDelayMs: DefaultRetryDelayMs,
		}
	case "openai":
		return ProviderDefaults{
			Timeout:      DefaultOpenAITimeout,
			Concurrency:  DefaultOpenAIConcurrency,
			Retries:      DefaultOpenAIMaxRetries,
			RetryDelayMs: DefaultRetryDelayMs,
		}
	case "elevenlabs":
		return ProviderDefaults{
			Timeout:      DefaultElevenLabsTimeout,
			Concurrency:  DefaultElevenLabsConcurrency,
			Retries:      DefaultRetries,
			RetryDelayMs: DefaultRetryDelayMs,
		}
	case "ssh_whisper":
		return ProviderDefaults{
			Timeout:      DefaultSSHTimeout,
			Concurrency:  DefaultSSHConcurrency,
			Retries:      DefaultRetries,
			RetryDelayMs: DefaultRetryDelayMs * 2, // SSH needs longer retry delay
		}
	case "whisper_server":
		return ProviderDefaults{
			Timeout:      DefaultHTTPTimeout,
			Concurrency:  DefaultHTTPConcurrency,
			Retries:      DefaultRetries,
			RetryDelayMs: DefaultRetryDelayMs,
		}
	default:
		// Return sensible defaults for unknown providers
		return ProviderDefaults{
			Timeout:      60 * time.Second,
			Concurrency:  1,
			Retries:      2,
			RetryDelayMs: 1000,
		}
	}
}