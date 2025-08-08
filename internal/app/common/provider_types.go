package common

// AudioFormat defines supported audio formats
type AudioFormat string

const (
	FormatWAV   AudioFormat = "wav"
	FormatMP3   AudioFormat = "mp3"
	FormatM4A   AudioFormat = "m4a"
	FormatFLAC  AudioFormat = "flac"
	FormatOGG   AudioFormat = "ogg"
	FormatAMR   AudioFormat = "amr"
	FormatWEBM  AudioFormat = "webm"
)

// ProviderType defines the type of transcription provider
type ProviderType string

const (
	ProviderTypeLocal  ProviderType = "local"
	ProviderTypeRemote ProviderType = "remote"
	ProviderTypeHybrid ProviderType = "hybrid"
)

// ProviderInfo contains metadata about a transcription provider
type ProviderInfo struct {
	// Basic info
	Name        string       `json:"name"`         // Provider name (e.g., "whisper_cpp", "openai", "elevenlabs")
	DisplayName string       `json:"display_name"` // Human-readable name
	Type        ProviderType `json:"type"`         // local, remote, hybrid
	Version     string       `json:"version,omitempty"`
	
	// Capabilities
	SupportedFormats   []AudioFormat `json:"supported_formats"`
	SupportedLanguages []string      `json:"supported_languages,omitempty"` // Empty means all languages
	MaxFileSizeMB      int           `json:"max_file_size_mb,omitempty"`    // 0 means no limit
	MaxDurationSec     int           `json:"max_duration_sec,omitempty"`    // 0 means no limit
	
	// Features
	SupportsTimestamps        bool `json:"supports_timestamps"`
	SupportsWordLevel         bool `json:"supports_word_level"`
	SupportsConfidence        bool `json:"supports_confidence"`
	SupportsLanguageDetection bool `json:"supports_language_detection"`
	SupportsStreaming         bool `json:"supports_streaming"`
	
	// Requirements
	RequiresInternet bool `json:"requires_internet"`
	RequiresAPIKey   bool `json:"requires_api_key"`
	RequiresBinary   bool `json:"requires_binary"`
	
	// Configuration
	DefaultModel     string                 `json:"default_model,omitempty"`
	AvailableModels  []string               `json:"available_models,omitempty"`
	ConfigSchema     map[string]interface{} `json:"config_schema,omitempty"`
	
	// Performance characteristics
	TypicalLatencyMs int    `json:"typical_latency_ms,omitempty"` // Typical processing time per minute of audio
	CostPerMinute    string `json:"cost_per_minute,omitempty"`    // Cost information if applicable
}