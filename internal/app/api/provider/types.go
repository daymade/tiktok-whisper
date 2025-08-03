package provider

import (
	"context"
	"time"
)

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

// TranscriptionRequest represents a transcription request with all possible options
type TranscriptionRequest struct {
	// Core fields
	InputFilePath string `json:"input_file_path"`
	
	// Language and model options
	Language    string `json:"language,omitempty"`    // "zh", "en", "auto", etc.
	Model       string `json:"model,omitempty"`       // Provider-specific model ID
	
	// Quality and processing options
	Temperature float32 `json:"temperature,omitempty"` // For some providers (0.0-1.0)
	Prompt      string  `json:"prompt,omitempty"`      // Context prompt for better accuracy
	
	// Output format options
	ResponseFormat string `json:"response_format,omitempty"` // "text", "json", "verbose_json", "srt", "vtt"
	TimestampGranularities []string `json:"timestamp_granularities,omitempty"` // "word", "segment"
	
	// Provider-specific options
	ProviderOptions map[string]interface{} `json:"provider_options,omitempty"`
	
	// Context for cancellation and timeouts
	Context context.Context `json:"-"`
}

// TranscriptionResponse represents the response from a transcription provider
type TranscriptionResponse struct {
	// Core result
	Text string `json:"text"`
	
	// Metadata
	Language   string        `json:"language,omitempty"`
	Duration   time.Duration `json:"duration,omitempty"`
	Confidence float32       `json:"confidence,omitempty"`
	
	// Timing information (if supported)
	Segments []TranscriptionSegment `json:"segments,omitempty"`
	Words    []TranscriptionWord    `json:"words,omitempty"`
	
	// Provider-specific metadata
	ProviderMetadata map[string]interface{} `json:"provider_metadata,omitempty"`
	
	// Processing info
	ProcessingTime time.Duration `json:"processing_time,omitempty"`
	ModelUsed      string        `json:"model_used,omitempty"`
}

// TranscriptionSegment represents a time-segmented piece of transcription
type TranscriptionSegment struct {
	ID               int       `json:"id"`
	Text             string    `json:"text"`
	Start            float64   `json:"start"`            // Start time in seconds
	End              float64   `json:"end"`              // End time in seconds
	AvgLogprob       float64   `json:"avg_logprob,omitempty"`
	CompressionRatio float64   `json:"compression_ratio,omitempty"`
	NoSpeechProb     float64   `json:"no_speech_prob,omitempty"`
	Temperature      float64   `json:"temperature,omitempty"`
	Words            []TranscriptionWord `json:"words,omitempty"`
}

// TranscriptionWord represents a single word with timing information
type TranscriptionWord struct {
	Word        string  `json:"word"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Probability float64 `json:"probability,omitempty"`
}

// ProviderInfo contains metadata about a transcription provider
type ProviderInfo struct {
	// Basic info
	Name        string       `json:"name"`        // Provider name (e.g., "whisper_cpp", "openai", "elevenlabs")
	DisplayName string       `json:"display_name"` // Human-readable name
	Type        ProviderType `json:"type"`        // local, remote, hybrid
	Version     string       `json:"version,omitempty"`
	
	// Capabilities
	SupportedFormats   []AudioFormat `json:"supported_formats"`
	SupportedLanguages []string      `json:"supported_languages,omitempty"` // Empty means all languages
	MaxFileSizeMB      int           `json:"max_file_size_mb,omitempty"`     // 0 means no limit
	MaxDurationSec     int           `json:"max_duration_sec,omitempty"`     // 0 means no limit
	
	// Features
	SupportsTimestamps   bool `json:"supports_timestamps"`
	SupportsWordLevel    bool `json:"supports_word_level"`
	SupportsConfidence   bool `json:"supports_confidence"`
	SupportsLanguageDetection bool `json:"supports_language_detection"`
	SupportsStreaming    bool `json:"supports_streaming"`
	
	// Requirements
	RequiresInternet bool `json:"requires_internet"`
	RequiresAPIKey   bool `json:"requires_api_key"`
	RequiresBinary   bool `json:"requires_binary"`
	
	// Configuration
	DefaultModel     string                 `json:"default_model,omitempty"`
	AvailableModels  []string              `json:"available_models,omitempty"`
	ConfigSchema     map[string]interface{} `json:"config_schema,omitempty"`
	
	// Performance characteristics
	TypicalLatencyMs int `json:"typical_latency_ms,omitempty"` // Typical processing time per minute of audio
	CostPerMinute    string `json:"cost_per_minute,omitempty"`  // Cost information if applicable
}

// ConfigInfo represents configuration validation information
type ConfigInfo struct {
	Required []string               `json:"required"` // Required configuration fields
	Optional []string               `json:"optional"` // Optional configuration fields
	Schema   map[string]interface{} `json:"schema"`   // JSON schema for validation
}

// TranscriptionError represents provider-specific errors
type TranscriptionError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Provider    string `json:"provider"`
	Retryable   bool   `json:"retryable"`
	Suggestions []string `json:"suggestions,omitempty"`
}

func (e *TranscriptionError) Error() string {
	return e.Message
}

// Validation helpers

// IsValidAudioFormat checks if the given format is supported
func IsValidAudioFormat(format string) bool {
	switch AudioFormat(format) {
	case FormatWAV, FormatMP3, FormatM4A, FormatFLAC, FormatOGG, FormatAMR, FormatWEBM:
		return true
	default:
		return false
	}
}

// GetAudioFormatFromFilename extracts audio format from filename
func GetAudioFormatFromFilename(filename string) AudioFormat {
	// Simple extension-based detection
	if len(filename) < 4 {
		return ""
	}
	
	ext := filename[len(filename)-4:]
	switch ext {
	case ".wav":
		return FormatWAV
	case ".mp3":
		return FormatMP3
	case ".m4a":
		return FormatM4A
	case "flac":
		return FormatFLAC
	case ".ogg":
		return FormatOGG
	case ".amr":
		return FormatAMR
	case "webm":
		return FormatWEBM
	default:
		return ""
	}
}