package whisper_server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"tiktok-whisper/internal/app/api/provider"
)

// Mock HTTP server for testing
func createMockWhisperServer(t *testing.T, responses map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/inference":
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			// Parse multipart form
			err := r.ParseMultipartForm(10 << 20) // 10MB max
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Failed to parse form"))
				return
			}

			// Check if file was uploaded
			file, _, err := r.FormFile("file")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("No file uploaded"))
				return
			}
			file.Close()

			// Get response format
			responseFormat := r.FormValue("response_format")
			if responseFormat == "" {
				responseFormat = "json"
			}

			// Return mock response based on format
			switch responseFormat {
			case "json":
				response := WhisperServerResponse{
					Text:     "This is a test transcription.",
					Task:     "transcribe",
					Language: "english",
					Duration: 5.2,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)

			case "verbose_json":
				response := WhisperServerResponse{
					Text:     "This is a test transcription.",
					Task:     "transcribe",
					Language: "english",
					Duration: 5.2,
					Segments: []WhisperServerSegment{
						{
							ID:    0,
							Text:  "This is a test transcription.",
							Start: 0.0,
							End:   5.2,
							Words: []WhisperServerWord{
								{Word: "This", Start: 0.0, End: 0.5, Probability: 0.99},
								{Word: "is", Start: 0.5, End: 0.8, Probability: 0.98},
								{Word: "a", Start: 0.8, End: 1.0, Probability: 0.97},
								{Word: "test", Start: 1.0, End: 1.5, Probability: 0.99},
								{Word: "transcription", Start: 1.5, End: 2.8, Probability: 0.95},
							},
							Temperature: 0.0,
							AvgLogprob:  -0.1,
							NoSpeechProb: 0.01,
						},
					},
					DetectedLanguage:            "en",
					DetectedLanguageProbability: 0.95,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)

			case "text":
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("This is a test transcription."))

			case "srt":
				w.Header().Set("Content-Type", "text/plain")
				srt := `1
00:00:00,000 --> 00:00:05,200
This is a test transcription.
`
				w.Write([]byte(srt))

			case "vtt":
				w.Header().Set("Content-Type", "text/plain")
				vtt := `WEBVTT

00:00:00.000 --> 00:00:05.200
This is a test transcription.
`
				w.Write([]byte(vtt))

			default:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid response format"))
			}

		case "/load":
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": "Model loaded successfully",
			})

		case "/":
			// Health check endpoint
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Whisper server is running"))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Test helper to create temporary audio file
func createTestAudioFile(t *testing.T) string {
	tempDir := t.TempDir()
	audioFile := filepath.Join(tempDir, "test_audio.wav")
	
	// Create a dummy WAV file (minimal header + data)
	content := []byte("RIFF\x24\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00\x40\x1f\x00\x00\x80\x3e\x00\x00\x02\x00\x10\x00data\x00\x00\x00\x00")
	err := os.WriteFile(audioFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}
	
	return audioFile
}

func TestNewWhisperServerProvider(t *testing.T) {
	tests := []struct {
		name   string
		config WhisperServerConfig
		want   WhisperServerConfig
	}{
		{
			name: "minimal config with defaults",
			config: WhisperServerConfig{
				BaseURL: "http://localhost:8080",
			},
			want: WhisperServerConfig{
				BaseURL:        "http://localhost:8080",
				InferencePath:  "/inference",
				LoadPath:       "/load",
				Timeout:        60 * time.Second,
				ResponseFormat: "json",
				CustomHeaders:  map[string]string{},
			},
		},
		{
			name: "full config",
			config: WhisperServerConfig{
				BaseURL:         "http://192.168.1.100:8080",
				InferencePath:   "/custom/inference",
				LoadPath:        "/custom/load",
				Timeout:         30 * time.Second,
				Language:        "zh",
				ResponseFormat:  "verbose_json",
				Temperature:     0.5,
				Translate:       true,
				NoTimestamps:    true,
				WordThreshold:   0.1,
				MaxLength:       1000,
				CustomHeaders:   map[string]string{"Authorization": "Bearer token"},
				InsecureSkipTLS: true,
			},
			want: WhisperServerConfig{
				BaseURL:         "http://192.168.1.100:8080",
				InferencePath:   "/custom/inference",
				LoadPath:        "/custom/load",
				Timeout:         30 * time.Second,
				Language:        "zh",
				ResponseFormat:  "verbose_json",
				Temperature:     0.5,
				Translate:       true,
				NoTimestamps:    true,
				WordThreshold:   0.1,
				MaxLength:       1000,
				CustomHeaders:   map[string]string{"Authorization": "Bearer token"},
				InsecureSkipTLS: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewWhisperServerProvider(tt.config)
			
			if provider.config.BaseURL != tt.want.BaseURL {
				t.Errorf("BaseURL = %v, want %v", provider.config.BaseURL, tt.want.BaseURL)
			}
			if provider.config.InferencePath != tt.want.InferencePath {
				t.Errorf("InferencePath = %v, want %v", provider.config.InferencePath, tt.want.InferencePath)
			}
			if provider.config.LoadPath != tt.want.LoadPath {
				t.Errorf("LoadPath = %v, want %v", provider.config.LoadPath, tt.want.LoadPath)
			}
			if provider.config.Timeout != tt.want.Timeout {
				t.Errorf("Timeout = %v, want %v", provider.config.Timeout, tt.want.Timeout)
			}
			if provider.config.ResponseFormat != tt.want.ResponseFormat {
				t.Errorf("ResponseFormat = %v, want %v", provider.config.ResponseFormat, tt.want.ResponseFormat)
			}
		})
	}
}

func TestNewWhisperServerProviderFromSettings(t *testing.T) {
	tests := []struct {
		name        string
		settings    map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid minimal settings",
			settings: map[string]interface{}{
				"base_url": "http://localhost:8080",
			},
			expectError: false,
		},
		{
			name: "valid full settings",
			settings: map[string]interface{}{
				"base_url":         "http://192.168.1.100:8080",
				"inference_path":   "/custom/inference",
				"load_path":        "/custom/load",
				"timeout":          float64(30),
				"language":         "zh",
				"response_format":  "verbose_json",
				"temperature":      0.5,
				"translate":        true,
				"no_timestamps":    true,
				"word_threshold":   0.1,
				"max_length":       float64(1000),
				"insecure_skip_tls": true,
				"custom_headers": map[string]interface{}{
					"Authorization": "Bearer token",
					"User-Agent":    "test-client",
				},
			},
			expectError: false,
		},
		{
			name:        "missing base_url",
			settings:    map[string]interface{}{},
			expectError: true,
			errorMsg:    "base_url is required",
		},
		{
			name: "invalid base_url type",
			settings: map[string]interface{}{
				"base_url": 123,
			},
			expectError: true,
			errorMsg:    "base_url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewWhisperServerProviderFromSettings(tt.settings)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if provider == nil {
					t.Errorf("Expected provider but got nil")
				}
			}
		})
	}
}

func TestWhisperServerProvider_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      WhisperServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: WhisperServerConfig{
				BaseURL:        "http://localhost:8080",
				Temperature:    0.5,
				WordThreshold:  0.1,
				ResponseFormat: "json",
			},
			expectError: false,
		},
		{
			name: "missing base_url",
			config: WhisperServerConfig{
				Temperature:    0.5,
				WordThreshold:  0.1,
				ResponseFormat: "json",
			},
			expectError: true,
			errorMsg:    "base_url is required",
		},
		{
			name: "invalid URL scheme",
			config: WhisperServerConfig{
				BaseURL:        "ftp://localhost:8080",
				Temperature:    0.5,
				WordThreshold:  0.1,
				ResponseFormat: "json",
			},
			expectError: true,
			errorMsg:    "base_url must start with http:// or https://",
		},
		{
			name: "invalid temperature",
			config: WhisperServerConfig{
				BaseURL:        "http://localhost:8080",
				Temperature:    1.5,
				WordThreshold:  0.1,
				ResponseFormat: "json",
			},
			expectError: true,
			errorMsg:    "temperature must be between 0.0 and 1.0",
		},
		{
			name: "invalid word threshold",
			config: WhisperServerConfig{
				BaseURL:        "http://localhost:8080",
				Temperature:    0.5,
				WordThreshold:  1.5,
				ResponseFormat: "json",
			},
			expectError: true,
			errorMsg:    "word_threshold must be between 0.0 and 1.0",
		},
		{
			name: "invalid response format",
			config: WhisperServerConfig{
				BaseURL:        "http://localhost:8080",
				Temperature:    0.5,
				WordThreshold:  0.1,
				ResponseFormat: "invalid",
			},
			expectError: true,
			errorMsg:    "response_format must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewWhisperServerProvider(tt.config)
			err := provider.ValidateConfiguration()
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWhisperServerProvider_HealthCheck(t *testing.T) {
	// Create mock server
	server := createMockWhisperServer(t, nil)
	defer server.Close()

	tests := []struct {
		name        string
		config      WhisperServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid server",
			config: WhisperServerConfig{
				BaseURL: server.URL,
			},
			expectError: false,
		},
		{
			name: "invalid config",
			config: WhisperServerConfig{
				BaseURL: "",
			},
			expectError: true,
			errorMsg:    "configuration validation failed",
		},
		{
			name: "unreachable server",
			config: WhisperServerConfig{
				BaseURL: "http://nonexistent.example.com:12345",
				Timeout: 1 * time.Second,
			},
			expectError: true,
			errorMsg:    "server connectivity test failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewWhisperServerProvider(tt.config)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			err := provider.HealthCheck(ctx)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWhisperServerProvider_Transcript(t *testing.T) {
	// Create mock server
	server := createMockWhisperServer(t, nil)
	defer server.Close()

	// Create test audio file
	audioFile := createTestAudioFile(t)

	config := WhisperServerConfig{
		BaseURL: server.URL,
	}
	provider := NewWhisperServerProvider(config)

	// Test basic transcription
	result, err := provider.Transcript(audioFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "This is a test transcription."
	if result != expected {
		t.Errorf("Expected result %q, got %q", expected, result)
	}
}

func TestWhisperServerProvider_TranscriptWithOptions(t *testing.T) {
	// Create mock server
	server := createMockWhisperServer(t, nil)
	defer server.Close()

	// Create test audio file
	audioFile := createTestAudioFile(t)

	config := WhisperServerConfig{
		BaseURL:  server.URL,
		Language: "en",
	}
	provider := NewWhisperServerProvider(config)

	tests := []struct {
		name        string
		request     *provider.TranscriptionRequest
		expectError bool
		errorCode   string
	}{
		{
			name: "valid request",
			request: &provider.TranscriptionRequest{
				InputFilePath: audioFile,
			},
			expectError: false,
		},
		{
			name: "empty input path",
			request: &provider.TranscriptionRequest{
				InputFilePath: "",
			},
			expectError: true,
			errorCode:   "invalid_input",
		},
		{
			name: "non-existent file",
			request: &provider.TranscriptionRequest{
				InputFilePath: "/non/existent/file.wav",
			},
			expectError: true,
			errorCode:   "file_not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			response, err := provider.TranscriptWithOptions(ctx, tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else {
					// Check if it's a TranscriptionError with expected code
					if transcErr, ok := err.(*provider.TranscriptionError); ok {
						if transcErr.Code != tt.errorCode {
							t.Errorf("Expected error code %q, got %q", tt.errorCode, transcErr.Code)
						}
					} else {
						t.Errorf("Expected TranscriptionError but got %T: %v", err, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if response == nil {
					t.Errorf("Expected response but got nil")
				} else {
					if response.Text == "" {
						t.Errorf("Expected non-empty transcription text")
					}
					if response.ProcessingTime <= 0 {
						t.Errorf("Expected positive processing time, got %v", response.ProcessingTime)
					}
				}
			}
		})
	}
}

func TestWhisperServerProvider_parseResponse(t *testing.T) {
	provider := NewWhisperServerProvider(WhisperServerConfig{})

	tests := []struct {
		name           string
		data           []byte
		format         string
		expectedText   string
		expectedError  bool
	}{
		{
			name:   "json format",
			data:   []byte(`{"text": "Hello world", "language": "english"}`),
			format: "json",
			expectedText: "Hello world",
			expectedError: false,
		},
		{
			name:   "text format",
			data:   []byte("Hello world"),
			format: "text",
			expectedText: "Hello world",
			expectedError: false,
		},
		{
			name:   "srt format",
			data:   []byte("1\n00:00:00,000 --> 00:00:05,000\nHello world\n\n"),
			format: "srt",
			expectedText: "Hello world",
			expectedError: false,
		},
		{
			name:   "vtt format",
			data:   []byte("WEBVTT\n\n00:00:00.000 --> 00:00:05.000\nHello world\n"),
			format: "vtt",
			expectedText: "Hello world",
			expectedError: false,
		},
		{
			name:   "invalid json",
			data:   []byte(`{"text": "incomplete`),
			format: "json",
			expectedText: "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, metadata, err := provider.parseResponse(tt.data, tt.format)
			
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if text != tt.expectedText {
					t.Errorf("Expected text %q, got %q", tt.expectedText, text)
				}
				if tt.format == "json" && metadata == nil {
					t.Errorf("Expected metadata for JSON format but got nil")
				}
			}
		})
	}
}

func TestWhisperServerProvider_extractTextFromSubtitles(t *testing.T) {
	provider := NewWhisperServerProvider(WhisperServerConfig{})

	tests := []struct {
		name     string
		content  string
		format   string
		expected string
	}{
		{
			name: "srt format",
			content: `1
00:00:00,000 --> 00:00:05,000
Hello world

2
00:00:05,000 --> 00:00:10,000
This is a test
`,
			format:   "srt",
			expected: "Hello world This is a test",
		},
		{
			name: "vtt format",
			content: `WEBVTT

00:00:00.000 --> 00:00:05.000
Hello world

00:00:05.000 --> 00:00:10.000
This is a test
`,
			format:   "vtt",
			expected: "Hello world This is a test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.extractTextFromSubtitles(tt.content, tt.format)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestWhisperServerProvider_LoadModel(t *testing.T) {
	// Create mock server
	server := createMockWhisperServer(t, nil)
	defer server.Close()

	config := WhisperServerConfig{
		BaseURL: server.URL,
	}
	provider := NewWhisperServerProvider(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := provider.LoadModel(ctx, "models/ggml-large-v3.bin")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestWhisperServerProvider_GetProviderInfo(t *testing.T) {
	provider := NewWhisperServerProvider(WhisperServerConfig{})
	info := provider.GetProviderInfo()

	if info.Name != "whisper_server" {
		t.Errorf("Expected name 'whisper_server', got %q", info.Name)
	}
	if info.Type != provider.ProviderTypeRemote {
		t.Errorf("Expected type Remote, got %v", info.Type)
	}
	if !info.RequiresInternet {
		t.Errorf("Expected RequiresInternet to be true")
	}
	if info.RequiresAPIKey {
		t.Errorf("Expected RequiresAPIKey to be false")
	}
	if len(info.SupportedFormats) == 0 {
		t.Errorf("Expected supported formats but got none")
	}
}

// Benchmark test for transcription performance
func BenchmarkWhisperServerProvider_Transcript(b *testing.B) {
	// Create mock server
	server := createMockWhisperServer(b, nil)
	defer server.Close()

	// Create test audio file
	tempDir := b.TempDir()
	audioFile := filepath.Join(tempDir, "benchmark_audio.wav")
	content := []byte("RIFF\x24\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00\x40\x1f\x00\x00\x80\x3e\x00\x00\x02\x00\x10\x00data\x00\x00\x00\x00")
	err := os.WriteFile(audioFile, content, 0644)
	if err != nil {
		b.Fatalf("Failed to create test audio file: %v", err)
	}

	config := WhisperServerConfig{
		BaseURL: server.URL,
	}
	provider := NewWhisperServerProvider(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.Transcript(audioFile)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}