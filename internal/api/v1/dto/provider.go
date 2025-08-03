package dto

import (
	"time"

	"tiktok-whisper/internal/app/api/provider"
)

// ProviderResponse represents a provider in API responses
type ProviderResponse struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Type              string                 `json:"type"`
	Available         bool                   `json:"available"`
	HealthStatus      string                 `json:"health_status"`
	SupportedFormats  []string               `json:"supported_formats"`
	RequiresAPIKey    bool                   `json:"requires_api_key"`
	IsDefault         bool                   `json:"is_default"`
	Priority          int                    `json:"priority"`
	Configuration     map[string]interface{} `json:"configuration,omitempty"`
	Capabilities      ProviderCapabilities   `json:"capabilities"`
	LastHealthCheck   *time.Time             `json:"last_health_check,omitempty"`
}

// ProviderCapabilities represents provider capabilities
type ProviderCapabilities struct {
	SupportsStreaming bool     `json:"supports_streaming"`
	SupportsLanguages []string `json:"supports_languages"`
	SupportsModels    []string `json:"supports_models"`
	MaxFileSizeMB     int      `json:"max_file_size_mb"`
	MaxDurationSec    int      `json:"max_duration_sec"`
}

// ProviderStatusResponse represents provider status information
type ProviderStatusResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	ResponseTime  int64                  `json:"response_time_ms"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
	CheckedAt     time.Time              `json:"checked_at"`
}

// ProviderStatsResponse represents provider usage statistics
type ProviderStatsResponse struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	TotalRequests        int64     `json:"total_requests"`
	SuccessfulRequests   int64     `json:"successful_requests"`
	FailedRequests       int64     `json:"failed_requests"`
	AverageResponseTime  float64   `json:"average_response_time_ms"`
	TotalProcessingTime  int64     `json:"total_processing_time_ms"`
	TotalAudioDuration   float64   `json:"total_audio_duration_sec"`
	SuccessRate          float64   `json:"success_rate"`
	LastUsed             *time.Time `json:"last_used,omitempty"`
	PeriodStart          time.Time `json:"period_start"`
	PeriodEnd            time.Time `json:"period_end"`
}

// TestProviderRequest represents a request to test a provider
type TestProviderRequest struct {
	TestFile string                 `json:"test_file,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// TestProviderResponse represents the result of testing a provider
type TestProviderResponse struct {
	Success        bool                   `json:"success"`
	ResponseTime   int64                  `json:"response_time_ms"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	TestResult     map[string]interface{} `json:"test_result,omitempty"`
	TestedAt       time.Time              `json:"tested_at"`
}

// ToProviderResponse converts provider info to response DTO
func ToProviderResponse(info provider.ProviderInfo, healthStatus string, isDefault bool, priority int) ProviderResponse {
	// Convert audio formats to strings
	formats := make([]string, len(info.SupportedFormats))
	for i, f := range info.SupportedFormats {
		formats[i] = string(f)
	}

	// Determine availability based on health status
	available := healthStatus == "healthy"
	
	// Generate description based on provider type
	description := info.DisplayName
	if info.Type == provider.ProviderTypeLocal {
		description += " (local)"
	} else if info.Type == provider.ProviderTypeRemote {
		description += " (remote API)"
	}

	return ProviderResponse{
		ID:               info.Name,
		Name:             info.DisplayName,
		Description:      description,
		Type:             string(info.Type),
		Available:        available,
		HealthStatus:     healthStatus,
		SupportedFormats: formats,
		RequiresAPIKey:   info.RequiresAPIKey,
		IsDefault:        isDefault,
		Priority:         priority,
		Capabilities: ProviderCapabilities{
			SupportsStreaming: info.SupportsStreaming,
			SupportsLanguages: info.SupportedLanguages,
			SupportsModels:    []string{info.DefaultModel}, // Use default model as available model
			MaxFileSizeMB:     info.MaxFileSizeMB,
			MaxDurationSec:    info.MaxDurationSec,
		},
	}
}