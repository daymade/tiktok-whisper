package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/common"
)

// ElevenLabsSTTProvider implements the TranscriptionProvider interface for ElevenLabs Speech-to-Text API
type ElevenLabsSTTProvider struct {
	common.BaseProvider
	config ElevenLabsConfig
	client *http.Client
}

// ElevenLabsConfig represents configuration for ElevenLabs STT provider
type ElevenLabsConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
	Timeout int    `yaml:"timeout_sec"`
}

// ElevenLabsResponse represents the response from ElevenLabs STT API
type ElevenLabsResponse struct {
	Text      string  `json:"text"`
	Language  string  `json:"language,omitempty"`
	Alignment []Word  `json:"alignment,omitempty"`
}

// Word represents word-level timing information from ElevenLabs
type Word struct {
	Word      string  `json:"word"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

// NewElevenLabsSTTProvider creates a new ElevenLabs STT provider
func NewElevenLabsSTTProvider(config ElevenLabsConfig) *ElevenLabsSTTProvider {
	// Set defaults
	if config.BaseURL == "" {
		config.BaseURL = "https://api.elevenlabs.io/v1"
	}
	if config.Model == "" {
		config.Model = "whisper-large-v3"
	}
	if config.Timeout == 0 {
		config.Timeout = 120 // 2 minutes default
	}
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	
	// Create base provider
	baseProvider := common.NewBaseProvider(
		"elevenlabs",
		"ElevenLabs Speech-to-Text",
		provider.ProviderTypeRemote,
		"1.0.0",
	)
	
	// Set specific attributes for ElevenLabs provider
	baseProvider.SupportedFormats = []provider.AudioFormat{
		provider.FormatMP3,
		provider.FormatWAV,
		provider.FormatFLAC,
		provider.FormatM4A,
	}
	baseProvider.SupportedLanguages = []string{} // ElevenLabs supports many languages
	baseProvider.MaxFileSizeMB = 25
	baseProvider.MaxDurationSec = 0 // No specific limit mentioned
	baseProvider.SupportsTimestamps = false // Basic implementation doesn't include segments
	baseProvider.SupportsWordLevel = true  // ElevenLabs provides word-level alignment
	baseProvider.SupportsConfidence = false // Not in basic response
	baseProvider.SupportsLanguageDetection = true
	baseProvider.SupportsStreaming = false
	baseProvider.RequiresInternet = true
	baseProvider.RequiresAPIKey = true
	baseProvider.RequiresBinary = false
	baseProvider.DefaultModel = "whisper-large-v3"
	baseProvider.AvailableModels = []string{
		"whisper-large-v3",
		"whisper-large-v2",
	}
	
	return &ElevenLabsSTTProvider{
		BaseProvider: baseProvider,
		config: config,
		client: client,
	}
}

// NewElevenLabsSTTProviderFromSettings creates a provider from generic settings
func NewElevenLabsSTTProviderFromSettings(settings map[string]interface{}, apiKey string) (*ElevenLabsSTTProvider, error) {
	config := ElevenLabsConfig{
		APIKey: apiKey,
	}
	
	// Extract optional settings
	if baseURL, ok := settings["base_url"].(string); ok {
		config.BaseURL = baseURL
	}
	if model, ok := settings["model"].(string); ok {
		config.Model = model
	}
	if timeout, ok := settings["timeout_sec"].(int); ok {
		config.Timeout = timeout
	}
	
	return NewElevenLabsSTTProvider(config), nil
}

// Transcript implements the original Transcriber interface for backward compatibility
func (el *ElevenLabsSTTProvider) Transcript(inputFilePath string) (string, error) {
	ctx := context.Background()
	request := &provider.TranscriptionRequest{
		InputFilePath: inputFilePath,
	}
	
	response, err := el.TranscriptWithOptions(ctx, request)
	if err != nil {
		return "", err
	}
	
	return response.Text, nil
}

// TranscriptWithOptions implements the enhanced transcription interface
func (el *ElevenLabsSTTProvider) TranscriptWithOptions(ctx context.Context, request *provider.TranscriptionRequest) (*provider.TranscriptionResponse, error) {
	startTime := time.Now()
	
	// Validate input
	if request.InputFilePath == "" {
		return nil, &provider.TranscriptionError{
			Code:      "invalid_input",
			Message:   "input file path is required",
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	
	// Check if file exists
	fileInfo, err := os.Stat(request.InputFilePath)
	if os.IsNotExist(err) {
		return nil, &provider.TranscriptionError{
			Code:      "file_not_found",
			Message:   fmt.Sprintf("input file not found: %s", request.InputFilePath),
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	
	// Check file size (ElevenLabs has a limit, typically 25MB like OpenAI)
	if fileInfo.Size() > 25*1024*1024 {
		return nil, &provider.TranscriptionError{
			Code:        "file_too_large",
			Message:     "file size exceeds 25MB limit",
			Provider:    "elevenlabs",
			Retryable:   false,
			Suggestions: []string{"Reduce file size", "Split into smaller chunks"},
		}
	}
	
	// Create the HTTP request
	httpReq, err := el.createHTTPRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	
	// Make the API call
	resp, err := el.client.Do(httpReq)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "network_error",
			Message:   fmt.Sprintf("failed to call ElevenLabs API: %v", err),
			Provider:  "elevenlabs",
			Retryable: true,
		}
	}
	defer resp.Body.Close()
	
	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, el.handleHTTPError(resp)
	}
	
	// Parse response
	var elevenLabsResp ElevenLabsResponse
	if err := json.NewDecoder(resp.Body).Decode(&elevenLabsResp); err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "response_parse_error",
			Message:   fmt.Sprintf("failed to parse API response: %v", err),
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	
	// Build response
	response := &provider.TranscriptionResponse{
		Text:           elevenLabsResp.Text,
		Language:       elevenLabsResp.Language,
		ProcessingTime: time.Since(startTime),
		ModelUsed:      el.getModel(request),
		ProviderMetadata: map[string]interface{}{
			"api_model":    el.getModel(request),
			"file_size":    fileInfo.Size(),
			"has_alignment": len(elevenLabsResp.Alignment) > 0,
		},
	}
	
	// Convert word-level alignment if available
	if len(elevenLabsResp.Alignment) > 0 {
		response.Words = el.convertAlignment(elevenLabsResp.Alignment)
	}
	
	return response, nil
}

// createHTTPRequest creates the HTTP request for the ElevenLabs API
func (el *ElevenLabsSTTProvider) createHTTPRequest(ctx context.Context, request *provider.TranscriptionRequest) (*http.Request, error) {
	// Open the audio file
	file, err := os.Open(request.InputFilePath)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "file_open_error",
			Message:   fmt.Sprintf("failed to open audio file: %v", err),
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	defer file.Close()
	
	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	
	// Add audio file
	part, err := writer.CreateFormFile("audio", filepath.Base(request.InputFilePath))
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "form_creation_error",
			Message:   fmt.Sprintf("failed to create form: %v", err),
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	
	if _, err := io.Copy(part, file); err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "file_copy_error",
			Message:   fmt.Sprintf("failed to copy file data: %v", err),
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	
	// Add model parameter
	if err := writer.WriteField("model", el.getModel(request)); err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "form_field_error",
			Message:   fmt.Sprintf("failed to add model field: %v", err),
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	
	// Add language if specified
	if language := el.getLanguage(request); language != "" {
		if err := writer.WriteField("language", language); err != nil {
			return nil, &provider.TranscriptionError{
				Code:      "form_field_error",
				Message:   fmt.Sprintf("failed to add language field: %v", err),
				Provider:  "elevenlabs",
				Retryable: false,
			}
		}
	}
	
	// Add response format
	responseFormat := "json" // ElevenLabs typically returns JSON with alignment
	if request.ResponseFormat != "" && request.ResponseFormat != "text" {
		responseFormat = request.ResponseFormat
	}
	if err := writer.WriteField("response_format", responseFormat); err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "form_field_error",
			Message:   fmt.Sprintf("failed to add response format field: %v", err),
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	
	writer.Close()
	
	// Create HTTP request
	url := fmt.Sprintf("%s/speech-to-text", el.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "request_creation_error",
			Message:   fmt.Sprintf("failed to create HTTP request: %v", err),
			Provider:  "elevenlabs",
			Retryable: false,
		}
	}
	
	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("xi-api-key", el.config.APIKey)
	req.Header.Set("User-Agent", "tiktok-whisper/1.0")
	
	return req, nil
}

// handleHTTPError handles HTTP error responses
func (el *ElevenLabsSTTProvider) handleHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	
	switch resp.StatusCode {
	case 401:
		return &provider.TranscriptionError{
			Code:        "authentication_failed",
			Message:     "ElevenLabs API key is invalid or missing",
			Provider:    "elevenlabs",
			Retryable:   false,
			Suggestions: []string{"Check your ELEVENLABS_API_KEY environment variable"},
		}
	case 429:
		return &provider.TranscriptionError{
			Code:        "rate_limit_exceeded",
			Message:     "ElevenLabs API rate limit exceeded",
			Provider:    "elevenlabs",
			Retryable:   true,
			Suggestions: []string{"Wait a moment and try again"},
		}
	case 413:
		return &provider.TranscriptionError{
			Code:        "file_too_large",
			Message:     "Audio file is too large",
			Provider:    "elevenlabs",
			Retryable:   false,
			Suggestions: []string{"Reduce file size", "Split into smaller chunks"},
		}
	case 400:
		return &provider.TranscriptionError{
			Code:        "invalid_request",
			Message:     fmt.Sprintf("Invalid request: %s", string(body)),
			Provider:    "elevenlabs",
			Retryable:   false,
		}
	case 500, 502, 503, 504:
		return &provider.TranscriptionError{
			Code:      "server_error",
			Message:   "ElevenLabs server error",
			Provider:  "elevenlabs",
			Retryable: true,
		}
	default:
		return &provider.TranscriptionError{
			Code:      "unknown_error",
			Message:   fmt.Sprintf("Unexpected HTTP status %d: %s", resp.StatusCode, string(body)),
			Provider:  "elevenlabs",
			Retryable: true,
		}
	}
}

// getModel determines which model to use
func (el *ElevenLabsSTTProvider) getModel(request *provider.TranscriptionRequest) string {
	if request.Model != "" {
		return request.Model
	}
	return el.config.Model
}

// getLanguage determines which language to use
func (el *ElevenLabsSTTProvider) getLanguage(request *provider.TranscriptionRequest) string {
	return request.Language // ElevenLabs auto-detects if not specified
}

// convertAlignment converts ElevenLabs alignment to provider format
func (el *ElevenLabsSTTProvider) convertAlignment(alignment []Word) []provider.TranscriptionWord {
	words := make([]provider.TranscriptionWord, len(alignment))
	
	for i, word := range alignment {
		words[i] = provider.TranscriptionWord{
			Word:  word.Word,
			Start: word.StartTime,
			End:   word.EndTime,
		}
	}
	
	return words
}

// GetProviderInfo method is now inherited from BaseProvider with additional metadata
func (el *ElevenLabsSTTProvider) GetProviderInfo() provider.ProviderInfo {
	info := el.BaseProvider.GetProviderInfo()
	
	// Add ElevenLabs-specific metadata
	info.TypicalLatencyMs = 3000 // Estimate: 3 seconds per minute
	info.CostPerMinute = "Variable" // ElevenLabs pricing varies
	info.ConfigSchema = map[string]interface{}{
		"api_key": map[string]string{
			"type":        "string",
			"description": "ElevenLabs API key",
			"required":    "true",
		},
		"model": map[string]string{
			"type":        "string",
			"description": "Model to use",
			"default":     "whisper-large-v3",
		},
		"base_url": map[string]string{
			"type":        "string",
			"description": "Base URL for ElevenLabs API",
			"default":     "https://api.elevenlabs.io/v1",
		},
		"timeout_sec": map[string]string{
			"type":        "number",
			"description": "Request timeout in seconds",
			"default":     "120",
		},
	}
	
	return info
}

// ValidateConfiguration validates the provider configuration
func (el *ElevenLabsSTTProvider) ValidateConfiguration() error {
	// Check API key
	if el.config.APIKey == "" {
		return fmt.Errorf("ElevenLabs API key is required")
	}
	
	// Validate API key format (basic check)
	if !strings.HasPrefix(el.config.APIKey, "sk_") && !strings.HasPrefix(el.config.APIKey, "el_") {
		return fmt.Errorf("ElevenLabs API key format appears to be invalid")
	}
	
	// Validate base URL
	if el.config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	
	// Validate timeout
	if el.config.Timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}
	
	return nil
}

// HealthCheck performs a health check on the provider
func (el *ElevenLabsSTTProvider) HealthCheck(ctx context.Context) error {
	// Basic configuration validation
	if err := el.ValidateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Test API connectivity with a lightweight endpoint
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}
	
	// Create a simple GET request to check API connectivity
	req, err := http.NewRequestWithContext(ctx, "GET", el.config.BaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	req.Header.Set("xi-api-key", el.config.APIKey)
	
	resp, err := el.client.Do(req)
	if err != nil {
		return fmt.Errorf("ElevenLabs API health check failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check for authentication errors
	if resp.StatusCode == 401 {
		return fmt.Errorf("ElevenLabs API authentication failed")
	}
	
	return nil
}