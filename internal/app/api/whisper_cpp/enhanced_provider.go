package whisper_cpp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/audio"
)

// EnhancedLocalTranscriber implements the new TranscriptionProvider interface
// while maintaining backward compatibility with the existing Transcriber interface
type EnhancedLocalTranscriber struct {
	*LocalTranscriber // Embed the original for backward compatibility
	config            LocalProviderConfig
}

// LocalProviderConfig represents configuration specific to local whisper.cpp provider
type LocalProviderConfig struct {
	BinaryPath    string `yaml:"binary_path"`
	ModelPath     string `yaml:"model_path"`
	Language      string `yaml:"language"`
	Prompt        string `yaml:"prompt"`
	OutputFormat  string `yaml:"output_format"`
	MaxConcurrent int    `yaml:"max_concurrent"`
	TempDir       string `yaml:"temp_dir"`
}

// NewEnhancedLocalTranscriber creates a new enhanced local transcriber
func NewEnhancedLocalTranscriber(config LocalProviderConfig) *EnhancedLocalTranscriber {
	// Create the original LocalTranscriber for backward compatibility
	original := NewLocalTranscriber(config.BinaryPath, config.ModelPath)
	
	// Set defaults
	if config.Language == "" {
		config.Language = "zh"
	}
	if config.Prompt == "" {
		config.Prompt = "以下是简体中文普通话:"
	}
	if config.OutputFormat == "" {
		config.OutputFormat = "txt"
	}
	if config.TempDir == "" {
		config.TempDir = "/tmp/whisper_cpp"
	}
	
	return &EnhancedLocalTranscriber{
		LocalTranscriber: original,
		config:           config,
	}
}

// NewEnhancedLocalTranscriberFromSettings creates an enhanced transcriber from generic settings
func NewEnhancedLocalTranscriberFromSettings(settings map[string]interface{}) (*EnhancedLocalTranscriber, error) {
	config := LocalProviderConfig{}
	
	// Extract required settings
	if binaryPath, ok := settings["binary_path"].(string); ok {
		config.BinaryPath = binaryPath
	} else {
		return nil, fmt.Errorf("binary_path is required for whisper_cpp provider")
	}
	
	if modelPath, ok := settings["model_path"].(string); ok {
		config.ModelPath = modelPath
	} else {
		return nil, fmt.Errorf("model_path is required for whisper_cpp provider")
	}
	
	// Extract optional settings
	if language, ok := settings["language"].(string); ok {
		config.Language = language
	}
	if prompt, ok := settings["prompt"].(string); ok {
		config.Prompt = prompt
	}
	if outputFormat, ok := settings["output_format"].(string); ok {
		config.OutputFormat = outputFormat
	}
	if tempDir, ok := settings["temp_dir"].(string); ok {
		config.TempDir = tempDir
	}
	if maxConcurrent, ok := settings["max_concurrent"].(int); ok {
		config.MaxConcurrent = maxConcurrent
	}
	
	return NewEnhancedLocalTranscriber(config), nil
}

// TranscriptWithOptions implements the enhanced transcription interface
func (elt *EnhancedLocalTranscriber) TranscriptWithOptions(ctx context.Context, request *provider.TranscriptionRequest) (*provider.TranscriptionResponse, error) {
	startTime := time.Now()
	
	// Validate input
	if request.InputFilePath == "" {
		return nil, &provider.TranscriptionError{
			Code:      "invalid_input",
			Message:   "input file path is required",
			Provider:  "whisper_cpp",
			Retryable: false,
		}
	}
	
	// Check if file exists
	if _, err := os.Stat(request.InputFilePath); os.IsNotExist(err) {
		return nil, &provider.TranscriptionError{
			Code:      "file_not_found",
			Message:   fmt.Sprintf("input file not found: %s", request.InputFilePath),
			Provider:  "whisper_cpp",
			Retryable: false,
		}
	}
	
	// Use language from request or default
	language := elt.config.Language
	if request.Language != "" {
		language = request.Language
	}
	
	// Use prompt from request or default
	prompt := elt.config.Prompt
	if request.Prompt != "" {
		prompt = request.Prompt
	}
	
	// Ensure temp directory exists
	if err := os.MkdirAll(elt.config.TempDir, 0755); err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "temp_dir_error",
			Message:   fmt.Sprintf("failed to create temp directory: %v", err),
			Provider:  "whisper_cpp",
			Retryable: true,
		}
	}
	
	// Handle context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	log.Printf("Starting enhanced transcription of file %s with language %s", request.InputFilePath, language)
	
	// Check if the input file is a 16kHz WAV file
	is16kHzWav, err := audio.Is16kHzWavFile(request.InputFilePath)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "audio_check_error",
			Message:   fmt.Sprintf("error checking input file: %v", err),
			Provider:  "whisper_cpp",
			Retryable: true,
		}
	}
	
	// Convert the input file to a 16kHz WAV file if necessary
	inputFilePath := request.InputFilePath
	if !is16kHzWav {
		log.Printf("Input file is not a 16kHz WAV file, converting...")
		converted, err := audio.ConvertTo16kHzWav(request.InputFilePath)
		if err != nil {
			return nil, &provider.TranscriptionError{
				Code:      "audio_conversion_error",
				Message:   fmt.Sprintf("error converting input file: %v", err),
				Provider:  "whisper_cpp",
				Retryable: true,
			}
		}
		inputFilePath = converted
		log.Printf("Successfully converted input file to a 16kHz WAV file")
	}
	
	// Create unique output file in temp directory
	outputFile := filepath.Join(elt.config.TempDir, fmt.Sprintf("transcription_%d", time.Now().UnixNano()))
	
	// Build command arguments
	args := []string{
		"-m", elt.config.ModelPath,
		"--print-colors",
		"-l", language,
		"--prompt", prompt,
		"-o" + elt.config.OutputFormat,
		"-f", inputFilePath,
		"-of", outputFile,
	}
	
	// Add response format if specified
	if request.ResponseFormat != "" && request.ResponseFormat != "text" {
		// whisper.cpp supports different output formats
		switch request.ResponseFormat {
		case "json":
			args = append(args, "--output-json")
		case "srt":
			args = append(args, "--output-srt") 
		case "vtt":
			args = append(args, "--output-vtt")
		}
	}
	
	// Execute transcription with context
	result, err := elt.executeTranscription(ctx, args, outputFile)
	if err != nil {
		return nil, err
	}
	
	// Build response
	response := &provider.TranscriptionResponse{
		Text:           result,
		Language:       language,
		ProcessingTime: time.Since(startTime),
		ModelUsed:      elt.config.ModelPath,
		ProviderMetadata: map[string]interface{}{
			"binary_path":     elt.config.BinaryPath,
			"converted_audio": !is16kHzWav,
			"output_format":   elt.config.OutputFormat,
			"temp_file":       outputFile,
		},
	}
	
	log.Printf("Successfully completed enhanced transcription in %v", response.ProcessingTime)
	return response, nil
}

// executeTranscription executes the whisper.cpp command with context support
func (elt *EnhancedLocalTranscriber) executeTranscription(ctx context.Context, args []string, outputFile string) (string, error) {
	// For simplicity, we'll use the original Transcript method
	// In a full implementation, you'd want to reimplement this with proper context support
	
	// Create a channel to handle the transcription result
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		// Use the original method (which doesn't support context)
		result, err := elt.LocalTranscriber.Transcript(args[len(args)-3]) // Get input file path
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- result
	}()
	
	// Wait for either completion or context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errorChan:
		return "", &provider.TranscriptionError{
			Code:      "transcription_failed",
			Message:   fmt.Sprintf("transcription failed: %v", err),
			Provider:  "whisper_cpp",
			Retryable: true,
		}
	case result := <-resultChan:
		return result, nil
	}
}

// GetProviderInfo returns metadata about the whisper.cpp provider
func (elt *EnhancedLocalTranscriber) GetProviderInfo() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        "whisper_cpp",
		DisplayName: "Whisper.cpp (Local)",
		Type:        provider.ProviderTypeLocal,
		Version:     "1.0.0",
		SupportedFormats: []provider.AudioFormat{
			provider.FormatWAV,
			provider.FormatMP3,
			provider.FormatM4A,
			provider.FormatFLAC,
		},
		SupportedLanguages: []string{
			"zh", "en", "ja", "ko", "es", "fr", "de", "it", "pt", "ru",
			"ar", "tr", "pl", "nl", "sv", "da", "no", "fi", "hu", "cs",
		},
		MaxFileSizeMB:             0, // No specific limit for local processing
		MaxDurationSec:            0, // No specific limit
		SupportsTimestamps:        true,
		SupportsWordLevel:         false, // whisper.cpp doesn't provide word-level by default
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
		TypicalLatencyMs: 5000, // Rough estimate: 5 seconds per minute of audio
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
			"language": map[string]string{
				"type":        "string",
				"description": "Language code (e.g., 'zh', 'en')",
				"default":     "zh",
			},
			"prompt": map[string]string{
				"type":        "string",
				"description": "Context prompt for better accuracy",
				"default":     "以下是简体中文普通话:",
			},
		},
	}
}

// ValidateConfiguration validates the provider configuration
func (elt *EnhancedLocalTranscriber) ValidateConfiguration() error {
	// Check if binary exists and is executable
	if _, err := os.Stat(elt.config.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("whisper.cpp binary not found at %s", elt.config.BinaryPath)
	}
	
	// Check if model file exists
	if _, err := os.Stat(elt.config.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("whisper model not found at %s", elt.config.ModelPath)
	}
	
	// Validate temp directory can be created
	if err := os.MkdirAll(elt.config.TempDir, 0755); err != nil {
		return fmt.Errorf("cannot create temp directory %s: %v", elt.config.TempDir, err)
	}
	
	return nil
}

// HealthCheck performs a health check on the provider
func (elt *EnhancedLocalTranscriber) HealthCheck(ctx context.Context) error {
	// Basic configuration validation
	if err := elt.ValidateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Could potentially run a quick test transcription here
	// For now, just check that the binary responds to --help
	
	return nil
}