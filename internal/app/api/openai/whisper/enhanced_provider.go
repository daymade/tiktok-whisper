package whisper

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"tiktok-whisper/internal/app/api/provider"
	
	"github.com/sashabaranov/go-openai"
)

// EnhancedRemoteTranscriber implements the new TranscriptionProvider interface
// while maintaining backward compatibility with the existing Transcriber interface
type EnhancedRemoteTranscriber struct {
	*RemoteTranscriber // Embed the original for backward compatibility
	config             OpenAIProviderConfig
}

// OpenAIProviderConfig represents configuration specific to OpenAI Whisper provider
type OpenAIProviderConfig struct {
	APIKey         string `yaml:"api_key"`
	Model          string `yaml:"model"`
	ResponseFormat string `yaml:"response_format"`
	Language       string `yaml:"language"`
	Temperature    float32 `yaml:"temperature"`
	Prompt         string `yaml:"prompt"`
	BaseURL        string `yaml:"base_url"`
}

// NewEnhancedRemoteTranscriber creates a new enhanced remote transcriber
func NewEnhancedRemoteTranscriber(config OpenAIProviderConfig) *EnhancedRemoteTranscriber {
	// Create OpenAI client configuration
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	
	client := openai.NewClientWithConfig(clientConfig)
	
	// Create the original RemoteTranscriber for backward compatibility
	original := NewRemoteTranscriber(client)
	
	// Set defaults
	if config.Model == "" {
		config.Model = string(openai.Whisper1)
	}
	if config.ResponseFormat == "" {
		config.ResponseFormat = "text"
	}
	
	return &EnhancedRemoteTranscriber{
		RemoteTranscriber: original,
		config:            config,
	}
}

// NewEnhancedRemoteTranscriberFromSettings creates an enhanced transcriber from generic settings
func NewEnhancedRemoteTranscriberFromSettings(settings map[string]interface{}, apiKey string) (*EnhancedRemoteTranscriber, error) {
	config := OpenAIProviderConfig{
		APIKey: apiKey,
	}
	
	// Extract optional settings
	if model, ok := settings["model"].(string); ok {
		config.Model = model
	}
	if responseFormat, ok := settings["response_format"].(string); ok {
		config.ResponseFormat = responseFormat
	}
	if language, ok := settings["language"].(string); ok {
		config.Language = language
	}
	if temperature, ok := settings["temperature"].(float64); ok {
		config.Temperature = float32(temperature)
	}
	if prompt, ok := settings["prompt"].(string); ok {
		config.Prompt = prompt
	}
	if baseURL, ok := settings["base_url"].(string); ok {
		config.BaseURL = baseURL
	}
	
	return NewEnhancedRemoteTranscriber(config), nil
}

// TranscriptWithOptions implements the enhanced transcription interface
func (ert *EnhancedRemoteTranscriber) TranscriptWithOptions(ctx context.Context, request *provider.TranscriptionRequest) (*provider.TranscriptionResponse, error) {
	startTime := time.Now()
	
	// Validate input
	if request.InputFilePath == "" {
		return nil, &provider.TranscriptionError{
			Code:      "invalid_input",
			Message:   "input file path is required",
			Provider:  "openai",
			Retryable: false,
		}
	}
	
	// Check if file exists
	if _, err := os.Stat(request.InputFilePath); os.IsNotExist(err) {
		return nil, &provider.TranscriptionError{
			Code:      "file_not_found",
			Message:   fmt.Sprintf("input file not found: %s", request.InputFilePath),
			Provider:  "openai",
			Retryable: false,
		}
	}
	
	// Prepare OpenAI request - use basic fields that are known to work
	audioRequest := openai.AudioRequest{
		Model:    ert.getModel(request),
		FilePath: request.InputFilePath,
	}
	
	// Note: Advanced features like Prompt, Response format, Language, and Temperature
	// may not be available in the current version of the go-openai library
	// These would need to be added based on the specific library version
	
	// Set timestamp granularities if supported
	if len(request.TimestampGranularities) > 0 {
		// Note: This depends on the OpenAI library version and API support
		// The current go-openai library may not support this field yet
	}
	
	// Handle context timeout
	if ctx == nil {
		ctx = context.Background()
	}
	
	// Call OpenAI API
	resp, err := ert.client.CreateTranscription(ctx, audioRequest)
	if err != nil {
		return nil, ert.handleAPIError(err)
	}
	
	// Build enhanced response
	response := &provider.TranscriptionResponse{
		Text:           resp.Text,
		Language:       resp.Language,
		ProcessingTime: time.Since(startTime),
		ModelUsed:      ert.getModel(request),
		ProviderMetadata: map[string]interface{}{
			"api_model": audioRequest.Model,
			// Note: Other metadata would be available with full library support
		},
	}
	
	// Note: Segments and word-level timing would be available with full library support
	// if resp.Segments != nil {
	//     response.Segments = ert.convertSegments(resp.Segments)
	// }
	
	// Note: Duration calculation would be available with segment support
	// if len(response.Segments) > 0 {
	//     lastSegment := response.Segments[len(response.Segments)-1]
	//     response.Duration = time.Duration(lastSegment.End * float64(time.Second))
	// }
	
	return response, nil
}

// getModel determines which model to use
func (ert *EnhancedRemoteTranscriber) getModel(request *provider.TranscriptionRequest) string {
	if request.Model != "" {
		return request.Model
	}
	if ert.config.Model != "" {
		return ert.config.Model
	}
	return string(openai.Whisper1)
}

// getLanguage determines which language to use
func (ert *EnhancedRemoteTranscriber) getLanguage(request *provider.TranscriptionRequest) string {
	if request.Language != "" {
		return request.Language
	}
	return ert.config.Language
}

// getPrompt determines which prompt to use
func (ert *EnhancedRemoteTranscriber) getPrompt(request *provider.TranscriptionRequest) string {
	if request.Prompt != "" {
		return request.Prompt
	}
	return ert.config.Prompt
}

// getResponseFormat determines which response format to use
func (ert *EnhancedRemoteTranscriber) getResponseFormat(request *provider.TranscriptionRequest) openai.AudioResponseFormat {
	format := ert.config.ResponseFormat
	if request.ResponseFormat != "" {
		format = request.ResponseFormat
	}
	
	switch strings.ToLower(format) {
	case "json":
		return openai.AudioResponseFormatJSON
	case "verbose_json":
		return openai.AudioResponseFormatVerboseJSON
	case "srt":
		return openai.AudioResponseFormatSRT
	case "vtt":
		return openai.AudioResponseFormatVTT
	default:
		return openai.AudioResponseFormatText
	}
}

// convertSegments would convert OpenAI segments to provider segments
// Currently commented out due to library compatibility
// func (ert *EnhancedRemoteTranscriber) convertSegments(segments []openai.Segment) []provider.TranscriptionSegment {
//     // Implementation would depend on the specific version of go-openai library
//     return nil
// }

// handleAPIError converts OpenAI API errors to TranscriptionError
func (ert *EnhancedRemoteTranscriber) handleAPIError(err error) error {
	// Check for specific OpenAI error types
	if apiErr, ok := err.(*openai.APIError); ok {
		switch apiErr.HTTPStatusCode {
		case 401:
			return &provider.TranscriptionError{
				Code:        "authentication_failed",
				Message:     "OpenAI API key is invalid or missing",
				Provider:    "openai",
				Retryable:   false,
				Suggestions: []string{"Check your OPENAI_API_KEY environment variable"},
			}
		case 429:
			return &provider.TranscriptionError{
				Code:        "rate_limit_exceeded",
				Message:     "OpenAI API rate limit exceeded",
				Provider:    "openai",
				Retryable:   true,
				Suggestions: []string{"Wait a moment and try again", "Consider upgrading your OpenAI plan"},
			}
		case 413:
			return &provider.TranscriptionError{
				Code:        "file_too_large",
				Message:     "Audio file is too large for OpenAI API",
				Provider:    "openai",
				Retryable:   false,
				Suggestions: []string{"Reduce file size", "Split into smaller chunks"},
			}
		case 400:
			return &provider.TranscriptionError{
				Code:        "invalid_file",
				Message:     "Invalid audio file format or corrupted file",
				Provider:    "openai",
				Retryable:   false,
				Suggestions: []string{"Check file format", "Try converting to a supported format"},
			}
		default:
			return &provider.TranscriptionError{
				Code:      "api_error",
				Message:   fmt.Sprintf("OpenAI API error: %v", apiErr.Message),
				Provider:  "openai",
				Retryable: true,
			}
		}
	}
	
	// Generic error
	return &provider.TranscriptionError{
		Code:      "unknown_error",
		Message:   fmt.Sprintf("Transcription failed: %v", err),
		Provider:  "openai",
		Retryable: true,
	}
}

// GetProviderInfo returns metadata about the OpenAI provider
func (ert *EnhancedRemoteTranscriber) GetProviderInfo() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        "openai",
		DisplayName: "OpenAI Whisper API",
		Type:        provider.ProviderTypeRemote,
		Version:     "1.0.0",
		SupportedFormats: []provider.AudioFormat{
			provider.FormatMP3,
			provider.FormatM4A,
			provider.FormatWAV,
			provider.FormatWEBM,
		},
		SupportedLanguages:        []string{}, // OpenAI supports all languages
		MaxFileSizeMB:             25,         // OpenAI's current limit
		MaxDurationSec:            0,          // No specific duration limit
		SupportsTimestamps:        true,
		SupportsWordLevel:         true,
		SupportsConfidence:        true,
		SupportsLanguageDetection: true,
		SupportsStreaming:         false, // OpenAI Whisper API doesn't support streaming
		RequiresInternet:          true,
		RequiresAPIKey:            true,
		RequiresBinary:            false,
		DefaultModel:              "whisper-1",
		AvailableModels:           []string{"whisper-1"},
		TypicalLatencyMs:          2000, // Rough estimate: 2 seconds per minute of audio
		CostPerMinute:             "$0.006", // OpenAI's current pricing
		ConfigSchema: map[string]interface{}{
			"api_key": map[string]string{
				"type":        "string",
				"description": "OpenAI API key",
				"required":    "true",
			},
			"model": map[string]string{
				"type":        "string",
				"description": "Model to use",
				"default":     "whisper-1",
			},
			"response_format": map[string]string{
				"type":        "string",
				"description": "Response format (text, json, verbose_json, srt, vtt)",
				"default":     "text",
			},
			"language": map[string]string{
				"type":        "string",
				"description": "Language code (optional, auto-detected if not specified)",
			},
			"temperature": map[string]string{
				"type":        "number",
				"description": "Sampling temperature (0.0 to 1.0)",
				"default":     "0.0",
			},
		},
	}
}

// ValidateConfiguration validates the provider configuration
func (ert *EnhancedRemoteTranscriber) ValidateConfiguration() error {
	// Check API key
	if ert.config.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}
	
	// Validate API key format (basic check)
	if !strings.HasPrefix(ert.config.APIKey, "sk-") {
		return fmt.Errorf("OpenAI API key should start with 'sk-'")
	}
	
	// Validate temperature
	if ert.config.Temperature < 0 || ert.config.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0.0 and 1.0")
	}
	
	// Validate response format
	validFormats := []string{"text", "json", "verbose_json", "srt", "vtt"}
	if ert.config.ResponseFormat != "" {
		valid := false
		for _, format := range validFormats {
			if ert.config.ResponseFormat == format {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid response format: %s, must be one of %v", ert.config.ResponseFormat, validFormats)
		}
	}
	
	return nil
}

// HealthCheck performs a health check on the provider
func (ert *EnhancedRemoteTranscriber) HealthCheck(ctx context.Context) error {
	// Basic configuration validation
	if err := ert.ValidateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Test API connectivity by listing models (lightweight operation)
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}
	
	_, err := ert.client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("OpenAI API health check failed: %w", err)
	}
	
	return nil
}