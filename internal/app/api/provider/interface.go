package provider

import (
	"context"
)

// TranscriptionProvider defines the enhanced interface for transcription providers
// Following SOLID principles:
// - Single Responsibility: Each method has a single, well-defined purpose
// - Open/Closed: New providers can be added without modifying existing code
// - Liskov Substitution: All implementations should be interchangeable
// - Interface Segregation: Focused interface with clear contract
// - Dependency Inversion: Depends on abstractions, not concretions
type TranscriptionProvider interface {
	// Core transcription functionality (backward compatible with existing Transcriber)
	Transcript(inputFilePath string) (string, error)
	
	// Enhanced transcription with full options and context support
	TranscriptWithOptions(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error)
	
	// Provider metadata and capabilities
	GetProviderInfo() ProviderInfo
	
	// Configuration validation and health checks
	ValidateConfiguration() error
	
	// Health check to verify provider is available and functioning
	HealthCheck(ctx context.Context) error
}

// ConfigurableProvider defines providers that support runtime configuration
type ConfigurableProvider interface {
	TranscriptionProvider
	
	// Update configuration at runtime
	UpdateConfiguration(config map[string]interface{}) error
	
	// Get current configuration
	GetConfiguration() map[string]interface{}
}

// StreamingProvider defines providers that support streaming transcription
type StreamingProvider interface {
	TranscriptionProvider
	
	// Start streaming transcription
	StartStream(ctx context.Context, config StreamConfig) (StreamHandle, error)
}

// StreamConfig represents configuration for streaming transcription
type StreamConfig struct {
	Language       string                 `json:"language,omitempty"`
	Model          string                 `json:"model,omitempty"`
	SampleRate     int                    `json:"sample_rate,omitempty"`
	Channels       int                    `json:"channels,omitempty"`
	BufferSize     int                    `json:"buffer_size,omitempty"`
	Options        map[string]interface{} `json:"options,omitempty"`
}

// StreamHandle represents an active streaming transcription session
type StreamHandle interface {
	// Send audio data for transcription
	SendAudio(data []byte) error
	
	// Receive transcription results
	ReceiveResult() (*TranscriptionResponse, error)
	
	// Close the stream
	Close() error
}

// ProviderFactory creates provider instances based on configuration
type ProviderFactory interface {
	// Create a provider instance
	CreateProvider(providerType string, config map[string]interface{}) (TranscriptionProvider, error)
	
	// List available provider types
	GetAvailableProviders() []string
	
	// Get provider information without creating an instance
	GetProviderInfo(providerType string) (ProviderInfo, error)
}

// ProviderRegistry manages multiple transcription providers
type ProviderRegistry interface {
	// Register a provider
	RegisterProvider(name string, provider TranscriptionProvider) error
	
	// Get a provider by name
	GetProvider(name string) (TranscriptionProvider, error)
	
	// List all registered providers
	ListProviders() []string
	
	// Get default provider
	GetDefaultProvider() (TranscriptionProvider, error)
	
	// Set default provider
	SetDefaultProvider(name string) error
	
	// Health check all providers
	HealthCheckAll(ctx context.Context) map[string]error
}

// TranscriptionOrchestrator manages intelligent routing and fallback
type TranscriptionOrchestrator interface {
	// Transcribe with automatic provider selection
	Transcribe(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error)
	
	// Transcribe with specific provider, fallback to others if needed
	TranscribeWithProvider(ctx context.Context, providerName string, request *TranscriptionRequest) (*TranscriptionResponse, error)
	
	// Get provider recommendations for a given request
	RecommendProvider(request *TranscriptionRequest) ([]string, error)
	
	// Get orchestrator statistics
	GetStats() OrchestratorStats
}

// OrchestratorStats provides statistics about transcription operations
type OrchestratorStats struct {
	TotalRequests        int64            `json:"total_requests"`
	SuccessfulRequests   int64            `json:"successful_requests"`
	FailedRequests       int64            `json:"failed_requests"`
	ProviderUsage        map[string]int64 `json:"provider_usage"`
	AverageLatencyMs     float64          `json:"average_latency_ms"`
	ErrorsByProvider     map[string]int64 `json:"errors_by_provider"`
	LastHealthCheck      map[string]bool  `json:"last_health_check"`
}

// ProviderMetrics provides performance and usage metrics for providers
type ProviderMetrics interface {
	// Record a successful transcription
	RecordSuccess(provider string, latencyMs int64, audioLengthSec float64)
	
	// Record a failed transcription
	RecordFailure(provider string, errorType string)
	
	// Get metrics for a provider
	GetProviderMetrics(provider string) ProviderStats
	
	// Get overall metrics
	GetOverallMetrics() OverallStats
}

// ProviderStats contains statistics for a specific provider
type ProviderStats struct {
	Provider             string  `json:"provider"`
	TotalRequests        int64   `json:"total_requests"`
	SuccessfulRequests   int64   `json:"successful_requests"`
	FailedRequests       int64   `json:"failed_requests"`
	SuccessRate          float64 `json:"success_rate"`
	AverageLatencyMs     float64 `json:"average_latency_ms"`
	TotalAudioProcessed  float64 `json:"total_audio_processed_sec"`
	LastUsed             int64   `json:"last_used_timestamp"`
	IsHealthy            bool    `json:"is_healthy"`
	ErrorBreakdown       map[string]int64 `json:"error_breakdown"`
}

// OverallStats contains overall transcription statistics
type OverallStats struct {
	TotalProviders       int                `json:"total_providers"`
	ActiveProviders      int                `json:"active_providers"`
	TotalRequests        int64              `json:"total_requests"`
	SuccessfulRequests   int64              `json:"successful_requests"`
	OverallSuccessRate   float64            `json:"overall_success_rate"`
	FastestProvider      string             `json:"fastest_provider"`
	MostReliableProvider string             `json:"most_reliable_provider"`
	ProviderStats        map[string]ProviderStats `json:"provider_stats"`
}