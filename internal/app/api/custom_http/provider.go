package custom_http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/common"
)

// CustomHTTPProvider implements a generic HTTP-based transcription provider
type CustomHTTPProvider struct {
	common.BaseProvider
	endpoint    string
	apiKey      string
	headers     map[string]string
	method      string
	fieldName   string
	extraParams map[string]string
	client      *http.Client
}

// NewCustomHTTPProvider creates a new custom HTTP provider
func NewCustomHTTPProvider(settings map[string]interface{}) (*CustomHTTPProvider, error) {
	endpoint, ok := settings["endpoint"].(string)
	if !ok || endpoint == "" {
		return nil, fmt.Errorf("custom_http provider requires 'endpoint' setting")
	}

	// Optional settings
	apiKey, _ := settings["api_key"].(string)
	method, _ := settings["method"].(string)
	if method == "" {
		method = "POST"
	}

	fieldName, _ := settings["field_name"].(string)
	if fieldName == "" {
		fieldName = "file"
	}

	// Parse headers
	headers := make(map[string]string)
	if h, ok := settings["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if str, ok := v.(string); ok {
				headers[k] = str
			}
		}
	}

	// Parse extra parameters
	extraParams := make(map[string]string)
	if p, ok := settings["extra_params"].(map[string]interface{}); ok {
		for k, v := range p {
			if str, ok := v.(string); ok {
				extraParams[k] = str
			}
		}
	}

	// Create HTTP client with timeout
	timeout := 5 * time.Minute
	if t, ok := settings["timeout"].(int); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	// Create base provider
	baseProvider := common.NewBaseProvider(
		"custom_http",
		"Custom HTTP Whisper Service",
		provider.ProviderTypeRemote,
		"1.0.0",
	)

	// Set specific attributes for Custom HTTP provider
	baseProvider.SupportedFormats = []provider.AudioFormat{
		provider.FormatWAV,
		provider.FormatMP3,
		provider.FormatM4A,
		provider.FormatFLAC,
	}
	baseProvider.SupportedLanguages = []string{} // Depends on underlying service
	baseProvider.MaxFileSizeMB = 0 // Depends on underlying service
	baseProvider.MaxDurationSec = 0 // Depends on underlying service
	baseProvider.SupportsTimestamps = false // Generic implementation
	baseProvider.SupportsWordLevel = false // Generic implementation
	baseProvider.SupportsConfidence = false // Generic implementation
	baseProvider.SupportsLanguageDetection = false // Depends on service
	baseProvider.SupportsStreaming = false
	baseProvider.RequiresInternet = true
	baseProvider.RequiresAPIKey = apiKey != "" // Based on configuration
	baseProvider.RequiresBinary = false
	baseProvider.DefaultModel = "" // Depends on service
	baseProvider.AvailableModels = []string{} // Depends on service

	return &CustomHTTPProvider{
		BaseProvider: baseProvider,
		endpoint:    endpoint,
		apiKey:      apiKey,
		headers:     headers,
		method:      method,
		fieldName:   fieldName,
		extraParams: extraParams,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Transcript implements the TranscriptionProvider interface
func (p *CustomHTTPProvider) Transcript(inputFilePath string) (string, error) {
	// Open the file
	file, err := os.Open(inputFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add file
	part, err := writer.CreateFormFile(p.fieldName, inputFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Add extra parameters
	for k, v := range p.extraParams {
		if err := writer.WriteField(k, v); err != nil {
			return "", fmt.Errorf("failed to write field %s: %w", k, err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest(p.method, p.endpoint, &requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}

	// Add API key if provided
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned error: %d - %s", resp.StatusCode, string(body))
	}

	// Try to parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// If not JSON, return as plain text
		return string(body), nil
	}

	// Look for common text fields
	if text, ok := result["text"].(string); ok {
		return text, nil
	}
	if text, ok := result["transcription"].(string); ok {
		return text, nil
	}
	if text, ok := result["result"].(string); ok {
		return text, nil
	}

	// Return full JSON if no text field found
	return string(body), nil
}

// TranscriptWithOptions implements the enhanced interface
func (p *CustomHTTPProvider) TranscriptWithOptions(ctx context.Context, request *provider.TranscriptionRequest) (*provider.TranscriptionResponse, error) {
	// For now, just use the simple transcript method
	text, err := p.Transcript(request.InputFilePath)
	if err != nil {
		return nil, err
	}

	return &provider.TranscriptionResponse{
		Text: text,
	}, nil
}

// GetProviderInfo method is now inherited from BaseProvider with additional metadata
func (p *CustomHTTPProvider) GetProviderInfo() provider.ProviderInfo {
	info := p.BaseProvider.GetProviderInfo()

	// Add Custom HTTP-specific metadata
	info.ConfigSchema = map[string]interface{}{
		"endpoint": map[string]string{
			"type":        "string",
			"description": "HTTP endpoint URL for the transcription service",
			"required":    "true",
		},
		"api_key": map[string]string{
			"type":        "string",
			"description": "API key for authentication (optional)",
			"required":    "false",
		},
		"method": map[string]string{
			"type":        "string",
			"description": "HTTP method to use",
			"default":     "POST",
		},
		"field_name": map[string]string{
			"type":        "string",
			"description": "Form field name for the audio file",
			"default":     "file",
		},
		"timeout": map[string]string{
			"type":        "number",
			"description": "Request timeout in seconds",
			"default":     "300",
		},
		"headers": map[string]string{
			"type":        "object",
			"description": "Custom HTTP headers",
			"required":    "false",
		},
		"extra_params": map[string]string{
			"type":        "object",
			"description": "Extra form parameters to send",
			"required":    "false",
		},
	}

	return info
}

// ValidateConfiguration checks if the provider configuration is valid
func (p *CustomHTTPProvider) ValidateConfiguration() error {
	if p.endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	return nil
}

// HealthCheck verifies the provider is working
func (p *CustomHTTPProvider) HealthCheck(ctx context.Context) error {
	// Simple ping to the endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	// Accept any 2xx or 404 (endpoint might not support GET)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 || resp.StatusCode == 404 {
		return nil
	}

	return fmt.Errorf("health check returned status %d", resp.StatusCode)
}