package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DirectWhisperServerProvider implements a direct whisper-server provider without import cycles
type DirectWhisperServerProvider struct {
	baseURL        string
	responseFormat string
	language       string
	prompt         string
	client         *http.Client
}

// NewDirectWhisperServerProvider creates a new whisper server provider
func NewDirectWhisperServerProvider(settings map[string]interface{}) TranscriptionProvider {
	// Extract settings
	baseURL, _ := settings["base_url"].(string)
	if baseURL == "" {
		return nil
	}

	responseFormat, _ := settings["response_format"].(string)
	if responseFormat == "" {
		responseFormat = "text"
	}

	language, _ := settings["language"].(string)
	prompt, _ := settings["prompt"].(string)

	return &DirectWhisperServerProvider{
		baseURL:        baseURL,
		responseFormat: responseFormat,
		language:       language,
		prompt:         prompt,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Transcript implements the simple transcription interface
func (p *DirectWhisperServerProvider) Transcript(inputFilePath string) (string, error) {
	ctx := context.Background()
	request := &TranscriptionRequest{
		InputFilePath: inputFilePath,
		Language:      p.language,
		Prompt:        p.prompt,
	}
	
	response, err := p.TranscriptWithOptions(ctx, request)
	if err != nil {
		return "", err
	}
	
	return response.Text, nil
}

// TranscriptWithOptions implements the enhanced transcription interface
func (p *DirectWhisperServerProvider) TranscriptWithOptions(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error) {
	// Open the audio file
	file, err := os.Open(request.InputFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(request.InputFilePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Add other fields
	if p.responseFormat != "" {
		writer.WriteField("response_format", p.responseFormat)
	}
	
	if request.Language != "" {
		writer.WriteField("language", request.Language)
	} else if p.language != "" {
		writer.WriteField("language", p.language)
	}
	
	if request.Prompt != "" {
		writer.WriteField("prompt", request.Prompt)
	} else if p.prompt != "" {
		writer.WriteField("prompt", p.prompt)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/inference", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Return response based on format
	if p.responseFormat == "text" {
		return &TranscriptionResponse{
			Text: string(responseBody),
		}, nil
	}

	// For JSON formats, just return the raw text for now
	// In a real implementation, we would parse the JSON
	return &TranscriptionResponse{
		Text: string(responseBody),
	}, nil
}

// GetProviderInfo returns provider information
func (p *DirectWhisperServerProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:               "whisper_server",
		DisplayName:        "Whisper Server (HTTP API)",
		Type:               ProviderTypeRemote,
		Version:            "1.0.0",
		SupportsTimestamps: true,
		SupportsWordLevel:  false,
		RequiresAPIKey:     false,
		RequiresInternet:   true,
		MaxFileSizeMB:      100,
		TypicalLatencyMs:   2000,
		SupportedFormats: []AudioFormat{
			FormatWAV, FormatMP3, FormatM4A, FormatFLAC, FormatOGG, FormatWEBM,
		},
	}
}

// ValidateConfiguration validates the provider configuration
func (p *DirectWhisperServerProvider) ValidateConfiguration() error {
	if p.baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	return nil
}

// HealthCheck performs a health check
func (p *DirectWhisperServerProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/", nil)
	if err != nil {
		return err
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}
	
	return nil
}

// Initialize performs any necessary initialization
func (p *DirectWhisperServerProvider) Initialize(ctx context.Context) error {
	return p.HealthCheck(ctx)
}

// Close performs cleanup
func (p *DirectWhisperServerProvider) Close() error {
	return nil
}