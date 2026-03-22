package whisper_server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/common"
)

// WhisperServerProvider implements transcription via HTTP to a whisper-server instance
type WhisperServerProvider struct {
	common.BaseProvider
	config WhisperServerConfig
	client *http.Client
}

// WhisperServerConfig represents configuration for whisper-server HTTP API
type WhisperServerConfig struct {
	BaseURL         string            `yaml:"base_url"`         // Base URL of whisper-server (e.g., "http://192.0.2.20:8080")
	InferencePath   string            `yaml:"inference_path"`   // Inference endpoint path (default: "/inference")
	LoadPath        string            `yaml:"load_path"`        // Model loading endpoint path (default: "/load")
	Timeout         time.Duration     `yaml:"timeout"`          // Request timeout
	Language        string            `yaml:"language"`         // Default language code
	ResponseFormat  string            `yaml:"response_format"`  // Default response format (json, text, srt, vtt, verbose_json)
	Temperature     float64           `yaml:"temperature"`      // Decoding temperature (0.0-1.0)
	Translate       bool              `yaml:"translate"`        // Translate to English
	NoTimestamps    bool              `yaml:"no_timestamps"`    // Disable timestamps
	WordThreshold   float64           `yaml:"word_threshold"`   // Word-level timestamp threshold
	MaxLength       int               `yaml:"max_length"`       // Maximum segment length
	CustomHeaders   map[string]string `yaml:"custom_headers"`   // Custom HTTP headers
	InsecureSkipTLS bool              `yaml:"insecure_skip_tls"` // Skip TLS verification
}

// WhisperServerResponse represents the response from whisper-server
type WhisperServerResponse struct {
	Text                        string                   `json:"text,omitempty"`
	Task                        string                   `json:"task,omitempty"`
	Language                    string                   `json:"language,omitempty"`
	Duration                    float64                  `json:"duration,omitempty"`
	Segments                    []WhisperServerSegment   `json:"segments,omitempty"`
	DetectedLanguage            string                   `json:"detected_language,omitempty"`
	DetectedLanguageProbability float64                  `json:"detected_language_probability,omitempty"`
}

// WhisperServerSegment represents a segment in verbose response
type WhisperServerSegment struct {
	ID          int                    `json:"id"`
	Text        string                 `json:"text"`
	Start       float64                `json:"start"`
	End         float64                `json:"end"`
	Tokens      []int                  `json:"tokens,omitempty"`
	Words       []WhisperServerWord    `json:"words,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	AvgLogprob  float64                `json:"avg_logprob,omitempty"`
	NoSpeechProb float64               `json:"no_speech_prob,omitempty"`
}

// WhisperServerWord represents a word in segment
type WhisperServerWord struct {
	Word        string  `json:"word"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Probability float64 `json:"probability,omitempty"`
}

// NewWhisperServerProvider creates a new whisper-server HTTP provider
func NewWhisperServerProvider(config WhisperServerConfig) *WhisperServerProvider {
	// Set defaults
	if config.InferencePath == "" {
		config.InferencePath = "/inference"
	}
	if config.LoadPath == "" {
		config.LoadPath = "/load"
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.ResponseFormat == "" {
		config.ResponseFormat = "json"
	}
	if config.CustomHeaders == nil {
		config.CustomHeaders = make(map[string]string)
	}

	// Create base provider
	baseProvider := common.NewBaseProvider(
		provider.ProviderNameWhisperServer,
		"Whisper Server (HTTP API)",
		provider.ProviderTypeRemote,
		"1.0.0",
	)
	
	// Set specific attributes for whisper server provider
	baseProvider.SupportedFormats = []provider.AudioFormat{
		provider.FormatWAV,
		provider.FormatMP3,
		provider.FormatM4A,
		provider.FormatFLAC,
		provider.FormatOGG,
		provider.FormatWEBM,
	}
	baseProvider.MaxFileSizeMB = 100
	baseProvider.MaxDurationSec = 3600
	baseProvider.SupportsTimestamps = true
	baseProvider.SupportsWordLevel = true
	baseProvider.SupportsConfidence = true
	baseProvider.SupportsLanguageDetection = true
	baseProvider.SupportsStreaming = false
	baseProvider.RequiresInternet = true
	baseProvider.RequiresAPIKey = false
	baseProvider.RequiresBinary = false
	baseProvider.DefaultModel = "whisper-server"
	baseProvider.AvailableModels = []string{"whisper-server"}

	// Create HTTP client
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Configure TLS if needed
	if config.InsecureSkipTLS {
		// Add TLS skip logic here if needed
	}

	return &WhisperServerProvider{
		BaseProvider: baseProvider,
		config:       config,
		client:       client,
	}
}

// NewWhisperServerProviderFromSettings creates provider from generic settings
func NewWhisperServerProviderFromSettings(settings map[string]interface{}) (*WhisperServerProvider, error) {
	config := WhisperServerConfig{}

	// Extract required settings
	if baseURL, ok := settings["base_url"].(string); ok {
		config.BaseURL = baseURL
	} else {
		return nil, fmt.Errorf("base_url is required")
	}

	// Extract optional settings
	if inferencePath, ok := settings["inference_path"].(string); ok {
		config.InferencePath = inferencePath
	}
	if loadPath, ok := settings["load_path"].(string); ok {
		config.LoadPath = loadPath
	}
	if timeout, ok := settings["timeout"].(float64); ok {
		config.Timeout = time.Duration(timeout) * time.Second
	}
	if language, ok := settings["language"].(string); ok {
		config.Language = language
	}
	if responseFormat, ok := settings["response_format"].(string); ok {
		config.ResponseFormat = responseFormat
	}
	if temperature, ok := settings["temperature"].(float64); ok {
		config.Temperature = temperature
	}
	if translate, ok := settings["translate"].(bool); ok {
		config.Translate = translate
	}
	if noTimestamps, ok := settings["no_timestamps"].(bool); ok {
		config.NoTimestamps = noTimestamps
	}
	if wordThreshold, ok := settings["word_threshold"].(float64); ok {
		config.WordThreshold = wordThreshold
	}
	if maxLength, ok := settings["max_length"].(float64); ok {
		config.MaxLength = int(maxLength)
	}
	if insecureSkipTLS, ok := settings["insecure_skip_tls"].(bool); ok {
		config.InsecureSkipTLS = insecureSkipTLS
	}

	// Extract custom headers
	if headers, ok := settings["custom_headers"].(map[string]interface{}); ok {
		config.CustomHeaders = make(map[string]string)
		for k, v := range headers {
			if str, ok := v.(string); ok {
				config.CustomHeaders[k] = str
			}
		}
	}

	return NewWhisperServerProvider(config), nil
}

// Transcript implements the basic transcription interface for backward compatibility
func (wsp *WhisperServerProvider) Transcript(inputFilePath string) (string, error) {
	ctx := context.Background()
	request := &provider.TranscriptionRequest{
		InputFilePath: inputFilePath,
	}

	response, err := wsp.TranscriptWithOptions(ctx, request)
	if err != nil {
		return "", err
	}

	return response.Text, nil
}

// TranscriptWithOptions implements the enhanced transcription interface
func (wsp *WhisperServerProvider) TranscriptWithOptions(ctx context.Context, request *provider.TranscriptionRequest) (*provider.TranscriptionResponse, error) {
	startTime := time.Now()

	// Validate input
	if request.InputFilePath == "" {
		return nil, &provider.TranscriptionError{
			Code:      "invalid_input",
			Message:   "input file path is required",
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: false,
		}
	}

	// Check if local file exists
	if _, err := os.Stat(request.InputFilePath); os.IsNotExist(err) {
		return nil, &provider.TranscriptionError{
			Code:      "file_not_found",
			Message:   fmt.Sprintf("input file not found: %s", request.InputFilePath),
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: false,
		}
	}

	// Create multipart form
	body, contentType, err := wsp.createMultipartForm(request)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "form_creation_failed",
			Message:   fmt.Sprintf("failed to create multipart form: %v", err),
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: false,
		}
	}

	// Create HTTP request with dynamic timeout based on audio duration.
	// whisper.cpp processes at ~0.1x RTF on M2; allow 0.5x RTF + 60s buffer.
	requestTimeout := wsp.config.Timeout
	if request.AudioDuration > 0 {
		dynamicTimeout := time.Duration(request.AudioDuration/2)*time.Second + 60*time.Second
		if dynamicTimeout > requestTimeout {
			requestTimeout = dynamicTimeout
		}
	}
	reqCtx, reqCancel := context.WithTimeout(ctx, requestTimeout)
	defer reqCancel()

	url := wsp.config.BaseURL + wsp.config.InferencePath
	httpReq, err := http.NewRequestWithContext(reqCtx, "POST", url, body)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "request_creation_failed",
			Message:   fmt.Sprintf("failed to create HTTP request: %v", err),
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: false,
		}
	}

	// Set headers
	httpReq.Header.Set("Content-Type", contentType)
	for key, value := range wsp.config.CustomHeaders {
		httpReq.Header.Set(key, value)
	}

	// Use a client without global timeout — per-request context handles it
	reqClient := &http.Client{Transport: wsp.client.Transport}
	resp, err := reqClient.Do(httpReq)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "request_failed",
			Message:   fmt.Sprintf("HTTP request failed: %v", err),
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: true,
		}
	}
	defer resp.Body.Close()

	// Read response
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "response_read_failed",
			Message:   fmt.Sprintf("failed to read response: %v", err),
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: true,
		}
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, &provider.TranscriptionError{
			Code:      "api_error",
			Message:   fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(responseData)),
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: resp.StatusCode >= 500, // Retry on server errors
		}
	}

	// Parse response based on format
	transcriptionText, metadata, err := wsp.parseResponse(responseData, wsp.getResponseFormat(request))
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "response_parse_failed",
			Message:   fmt.Sprintf("failed to parse response: %v", err),
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: false,
		}
	}
	
	// Log detected language if available
	if detectedLang, ok := metadata["language"].(string); ok && detectedLang != "" {
		log.Printf("Whisper server detected language: %s (requested: %s) for file: %s",
			detectedLang, wsp.getLanguage(request, nil), request.InputFilePath)
	}

	if transcriptionText == "" {
		return nil, &provider.TranscriptionError{
			Code:      "empty_transcription",
			Message:   "no transcription text found in response",
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: false,
			Suggestions: []string{"Check audio file format", "Verify whisper-server is running correctly"},
		}
	}

	// Detect and reject hallucinated/nonsensical transcriptions
	if wsp.isHallucinatedTranscription(transcriptionText) {
		return nil, &provider.TranscriptionError{
			Code:      "hallucinated_transcription",
			Message:   fmt.Sprintf("transcription appears to be hallucinated or nonsensical: %s", transcriptionText),
			Provider:  provider.ProviderNameWhisperServer,
			Retryable: true,
			Suggestions: []string{
				"Audio file may be corrupted or empty",
				"Whisper model may be hallucinating",
				"Try a different provider or model",
			},
		}
	}

	// Build response
	response := &provider.TranscriptionResponse{
		Text:           transcriptionText,
		Language:       wsp.getLanguage(request, metadata),
		ProcessingTime: time.Since(startTime),
		ModelUsed:      "whisper-server", // Server doesn't expose model info
		ProviderMetadata: map[string]interface{}{
			"base_url":        wsp.config.BaseURL,
			"response_format": wsp.getResponseFormat(request),
			"temperature":     wsp.getTemperature(request),
			"http_status":     resp.StatusCode,
			"content_type":    resp.Header.Get("Content-Type"),
			"response_size":   len(responseData),
		},
	}

	// Add additional metadata if available
	if metadata != nil {
		if duration, ok := metadata["duration"].(float64); ok {
			response.ProviderMetadata["duration"] = duration
		}
		if segments, ok := metadata["segments"].([]WhisperServerSegment); ok {
			response.ProviderMetadata["segments_count"] = len(segments)
		}
	}

	return response, nil
}

// createMultipartForm creates the multipart form for the API request
func (wsp *WhisperServerProvider) createMultipartForm(request *provider.TranscriptionRequest) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add audio file
	file, err := os.Open(request.InputFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	filename := filepath.Base(request.InputFilePath)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to copy file content: %v", err)
	}

	// Add parameters
	params := map[string]string{
		"response_format": wsp.getResponseFormat(request),
		"temperature":     fmt.Sprintf("%.2f", wsp.getTemperature(request)),
	}

	// Add provider if specified (for API compatibility)
	if request.Provider != "" {
		params["provider"] = request.Provider
		log.Printf("Setting provider parameter for whisper server: %s for file: %s",
			request.Provider, request.InputFilePath)
	}

	// Add language if specified
	if language := wsp.getLanguage(request, nil); language != "" {
		params["language"] = language
		log.Printf("Setting language parameter for whisper server: %s for file: %s",
			language, request.InputFilePath)
	} else {
		log.Printf("No language specified for file %s, whisper server will use default",
			request.InputFilePath)
	}

	// Add translate if enabled
	if wsp.getTranslate(request) {
		params["translate"] = "true"
	}

	// Add no_timestamps if enabled
	if wsp.getNoTimestamps(request) {
		params["no_timestamps"] = "true"
	}

	// Add word_threshold if specified
	if wordThreshold := wsp.getWordThreshold(request); wordThreshold > 0 {
		params["word_thold"] = fmt.Sprintf("%.3f", wordThreshold)
	}

	// Add max_len if specified
	if maxLength := wsp.getMaxLength(request); maxLength > 0 {
		params["max_len"] = strconv.Itoa(maxLength)
	}

	// Write all parameters to form
	for key, value := range params {
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", fmt.Errorf("failed to write field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType(), nil
}

// parseResponse parses the response based on the response format
func (wsp *WhisperServerProvider) parseResponse(data []byte, format string) (string, map[string]interface{}, error) {
	switch format {
	case "json":
		var resp WhisperServerResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", nil, fmt.Errorf("failed to parse JSON response: %v", err)
		}
		metadata := map[string]interface{}{
			"language": resp.Language,
			"duration": resp.Duration,
		}
		return resp.Text, metadata, nil

	case "verbose_json":
		var resp WhisperServerResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", nil, fmt.Errorf("failed to parse verbose JSON response: %v", err)
		}
		metadata := map[string]interface{}{
			"task":                         resp.Task,
			"language":                     resp.Language,
			"duration":                     resp.Duration,
			"segments":                     resp.Segments,
			"detected_language":            resp.DetectedLanguage,
			"detected_language_probability": resp.DetectedLanguageProbability,
		}
		return resp.Text, metadata, nil

	case "text":
		return strings.TrimSpace(string(data)), nil, nil

	case "srt", "vtt":
		// For subtitle formats, extract text from timestamp lines
		content := strings.TrimSpace(string(data))
		text := wsp.extractTextFromSubtitles(content, format)
		metadata := map[string]interface{}{
			"subtitle_format": format,
			"subtitle_content": content,
		}
		return text, metadata, nil

	default:
		return strings.TrimSpace(string(data)), nil, nil
	}
}

// extractTextFromSubtitles extracts plain text from subtitle format
func (wsp *WhisperServerProvider) extractTextFromSubtitles(content, format string) string {
	lines := strings.Split(content, "\n")
	var textLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip sequence numbers and timestamps
		if format == "srt" {
			// Skip lines that are numbers or timestamps (contain "-->")
			if strings.Contains(line, "-->") || (len(line) <= 3 && isNumeric(line)) {
				continue
			}
		} else if format == "vtt" {
			// Skip WEBVTT header and timestamps
			if line == "WEBVTT" || strings.Contains(line, "-->") {
				continue
			}
		}

		textLines = append(textLines, line)
	}

	return strings.Join(textLines, " ")
}

// isNumeric checks if string is numeric
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// Helper methods to get configuration values with request overrides
func (wsp *WhisperServerProvider) getLanguage(request *provider.TranscriptionRequest, metadata map[string]interface{}) string {
	if request != nil && request.Language != "" {
		return request.Language
	}
	if metadata != nil {
		if lang, ok := metadata["language"].(string); ok && lang != "" {
			return lang
		}
		if detectedLang, ok := metadata["detected_language"].(string); ok && detectedLang != "" {
			return detectedLang
		}
	}
	return wsp.config.Language
}

func (wsp *WhisperServerProvider) getResponseFormat(request *provider.TranscriptionRequest) string {
	// For now, always use json as it provides the most information
	// In the future, this could be configurable per request
	return "json"
}

func (wsp *WhisperServerProvider) getTemperature(request *provider.TranscriptionRequest) float64 {
	// Request-level temperature override not supported yet in provider.TranscriptionRequest
	// Could be added as a custom field in the future
	return wsp.config.Temperature
}

func (wsp *WhisperServerProvider) getTranslate(request *provider.TranscriptionRequest) bool {
	// Request-level translate override not supported yet
	return wsp.config.Translate
}

func (wsp *WhisperServerProvider) getNoTimestamps(request *provider.TranscriptionRequest) bool {
	// Request-level no_timestamps override not supported yet
	return wsp.config.NoTimestamps
}

func (wsp *WhisperServerProvider) getWordThreshold(request *provider.TranscriptionRequest) float64 {
	// Request-level word threshold override not supported yet
	return wsp.config.WordThreshold
}

func (wsp *WhisperServerProvider) getMaxLength(request *provider.TranscriptionRequest) int {
	// Request-level max length override not supported yet
	return wsp.config.MaxLength
}

// GetProviderInfo method is now inherited from BaseProvider

// ValidateConfiguration validates the provider configuration
func (wsp *WhisperServerProvider) ValidateConfiguration() error {
	// Check required fields
	if wsp.config.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	// Validate URL format
	if !strings.HasPrefix(wsp.config.BaseURL, "http://") && !strings.HasPrefix(wsp.config.BaseURL, "https://") {
		return fmt.Errorf("base_url must start with http:// or https://")
	}

	// Validate temperature range
	if wsp.config.Temperature < 0.0 || wsp.config.Temperature > 1.0 {
		return fmt.Errorf("temperature must be between 0.0 and 1.0")
	}

	// Validate word threshold range
	if wsp.config.WordThreshold < 0.0 || wsp.config.WordThreshold > 1.0 {
		return fmt.Errorf("word_threshold must be between 0.0 and 1.0")
	}

	// Validate response format
	validFormats := []string{"json", "text", "srt", "vtt", "verbose_json"}
	if wsp.config.ResponseFormat != "" {
		valid := false
		for _, format := range validFormats {
			if wsp.config.ResponseFormat == format {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("response_format must be one of: %s", strings.Join(validFormats, ", "))
		}
	}

	return nil
}

// HealthCheck performs a health check on the provider
func (wsp *WhisperServerProvider) HealthCheck(ctx context.Context) error {
	// Basic configuration validation
	if err := wsp.ValidateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Test server connectivity
	req, err := http.NewRequestWithContext(ctx, "GET", wsp.config.BaseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Set custom headers
	for key, value := range wsp.config.CustomHeaders {
		req.Header.Set(key, value)
	}

	resp, err := wsp.client.Do(req)
	if err != nil {
		return fmt.Errorf("server connectivity test failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if server is responding (any 2xx, 404, or 503 is fine)
	// 503 might be returned due to proxy issues but the server is actually running
	if resp.StatusCode >= 500 && resp.StatusCode != 503 {
		return fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	return nil
}

// isHallucinatedTranscription detects if a transcription is likely hallucinated
func (wsp *WhisperServerProvider) isHallucinatedTranscription(text string) bool {
	// FAIL FAST: Immediately reject known hallucination patterns
	lowerText := strings.ToLower(text)
	
	// Critical patterns that should never appear in real transcriptions
	criticalPatterns := []string{
		"speaking in foreign language",
		"(speaking in foreign language)",
	}
	
	for _, pattern := range criticalPatterns {
		if strings.Contains(lowerText, pattern) {
			return true // Fail fast - this is definitely hallucinated
		}
	}
	
	// Detect common hallucination patterns
	// 1. Excessive repetition of the same phrase
	// Filter empty lines first — trailing \n from Whisper creates empty splits
	var nonEmptyLines []string
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			nonEmptyLines = append(nonEmptyLines, trimmed)
		}
	}
	if len(nonEmptyLines) >= 3 {
		phraseCount := make(map[string]int)
		for _, line := range nonEmptyLines {
			phraseCount[strings.ToLower(line)]++
		}

		// If any phrase repeats more than 30% of non-empty lines, it's likely hallucinated
		maxRepetition := 0
		for _, count := range phraseCount {
			if count > maxRepetition {
				maxRepetition = count
			}
		}
		if float64(maxRepetition) > float64(len(nonEmptyLines))*0.3 {
			return true
		}
	}
	
	// 2. Known hallucination patterns
	hallucinationPatterns := []string{
		"i'm a good person",
		"i'm not a good person",
		"thank you for watching",
		"please subscribe",
		"like and subscribe",
		"please like and subscribe",
		"don't forget to subscribe",
	}
	
	for _, pattern := range hallucinationPatterns {
		if strings.Contains(lowerText, pattern) {
			return true // Fail fast on ANY hallucination pattern
		}
	}
	
	// 3. Check for nonsensical repetitive content
	words := strings.Fields(text)
	if len(words) > 20 {
		wordCount := make(map[string]int)
		for _, word := range words {
			normalized := strings.ToLower(strings.Trim(word, ".,!?;:"))
			if len(normalized) > 2 { // Ignore small words
				wordCount[normalized]++
			}
		}
		
		// If any word appears more than 20% of the time, it's suspicious
		for _, count := range wordCount {
			if float64(count) > float64(len(words))*0.2 {
				return true
			}
		}
	}
	
	return false
}

// LoadModel loads a new model on the remote server (if supported)
func (wsp *WhisperServerProvider) LoadModel(ctx context.Context, modelPath string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add model parameter
	if err := writer.WriteField("model", modelPath); err != nil {
		return fmt.Errorf("failed to write model field: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %v", err)
	}

	// Create request
	url := wsp.config.BaseURL + wsp.config.LoadPath
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create load model request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range wsp.config.CustomHeaders {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := wsp.client.Do(req)
	if err != nil {
		return fmt.Errorf("load model request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("load model failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}