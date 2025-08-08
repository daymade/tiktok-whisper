package provider

import (
	"context"
	"time"
	"tiktok-whisper/internal/app/common"
)

// AudioFormat is re-exported from common package for backward compatibility
type AudioFormat = common.AudioFormat

// Re-export audio format constants
const (
	FormatWAV   = common.FormatWAV
	FormatMP3   = common.FormatMP3
	FormatM4A   = common.FormatM4A
	FormatFLAC  = common.FormatFLAC
	FormatOGG   = common.FormatOGG
	FormatAMR   = common.FormatAMR
	FormatWEBM  = common.FormatWEBM
)

// ProviderType is re-exported from common package for backward compatibility
type ProviderType = common.ProviderType

// Re-export provider type constants
const (
	ProviderTypeLocal  = common.ProviderTypeLocal
	ProviderTypeRemote = common.ProviderTypeRemote
	ProviderTypeHybrid = common.ProviderTypeHybrid
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

// ProviderInfo is re-exported from common package for backward compatibility
type ProviderInfo = common.ProviderInfo

// ConfigInfo represents configuration validation information
type ConfigInfo struct {
	Required []string               `json:"required"` // Required configuration fields
	Optional []string               `json:"optional"` // Optional configuration fields
	Schema   map[string]interface{} `json:"schema"`   // JSON schema for validation
}

// ProviderHealthStatus represents the health status of a provider
type ProviderHealthStatus struct {
	ProviderName string    `json:"provider_name"`
	IsHealthy    bool      `json:"is_healthy"`
	LastChecked  time.Time `json:"last_checked"`
	LastError    string    `json:"last_error,omitempty"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
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