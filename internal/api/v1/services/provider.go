package services

import (
	"context"
	"time"

	"tiktok-whisper/internal/api/errors"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/app/api/provider"
)

// ProviderServiceImpl implements ProviderService
type ProviderServiceImpl struct {
	registry provider.ProviderRegistry
}

// NewProviderService creates a new provider service
func NewProviderService(registry provider.ProviderRegistry) ProviderService {
	return &ProviderServiceImpl{
		registry: registry,
	}
}

// ListProviders lists all available providers
func (s *ProviderServiceImpl) ListProviders(ctx context.Context) ([]dto.ProviderResponse, error) {
	providerNames := s.registry.ListProviders()
	
	// Get default provider name
	defaultProviderName := ""
	defaultProvider, err := s.registry.GetDefaultProvider()
	if err == nil && defaultProvider != nil {
		defaultInfo := defaultProvider.GetProviderInfo()
		defaultProviderName = defaultInfo.Name
	}
	
	responses := make([]dto.ProviderResponse, 0, len(providerNames))
	for i, name := range providerNames {
		// Get provider instance
		provider, err := s.registry.GetProvider(name)
		if err != nil {
			continue
		}

		info := provider.GetProviderInfo()
		
		// Check health status
		healthStatus := "unknown"
		if err := provider.HealthCheck(ctx); err == nil {
			healthStatus = "healthy"
		} else {
			healthStatus = "unhealthy"
		}

		// Determine if this is the default provider
		isDefault := defaultProviderName == name
		priority := len(providerNames) - i // Higher priority for providers listed first

		responses = append(responses, dto.ToProviderResponse(info, healthStatus, isDefault, priority))
	}

	return responses, nil
}

// GetProvider gets detailed information about a specific provider
func (s *ProviderServiceImpl) GetProvider(ctx context.Context, id string) (*dto.ProviderResponse, error) {
	provider, err := s.registry.GetProvider(id)
	if err != nil {
		return nil, errors.NewNotFoundError("provider")
	}

	info := provider.GetProviderInfo()
	
	// Check health status
	healthStatus := "unknown"
	if err := provider.HealthCheck(ctx); err == nil {
		healthStatus = "healthy"
	} else {
		healthStatus = "unhealthy"
	}

	// Get default provider to check if this is default
	defaultProviderName := ""
	defaultProvider, err := s.registry.GetDefaultProvider()
	if err == nil && defaultProvider != nil {
		defaultInfo := defaultProvider.GetProviderInfo()
		defaultProviderName = defaultInfo.Name
	}
	
	isDefault := defaultProviderName == id
	priority := 1 // Default priority

	resp := dto.ToProviderResponse(info, healthStatus, isDefault, priority)
	return &resp, nil
}

// GetProviderStatus gets the health status of a provider
func (s *ProviderServiceImpl) GetProviderStatus(ctx context.Context, id string) (*dto.ProviderStatusResponse, error) {
	provider, err := s.registry.GetProvider(id)
	if err != nil {
		return nil, errors.NewNotFoundError("provider")
	}

	info := provider.GetProviderInfo()
	
	// Perform health check with timing
	start := time.Now()
	healthErr := provider.HealthCheck(ctx)
	responseTime := time.Since(start).Milliseconds()

	status := "healthy"
	errorMessage := ""
	if healthErr != nil {
		status = "unhealthy"
		errorMessage = healthErr.Error()
	}

	return &dto.ProviderStatusResponse{
		ID:           id,
		Name:         info.DisplayName,
		Status:       status,
		ResponseTime: responseTime,
		ErrorMessage: errorMessage,
		CheckedAt:    time.Now(),
	}, nil
}

// GetProviderStats gets usage statistics for a provider
func (s *ProviderServiceImpl) GetProviderStats(ctx context.Context, id string) (*dto.ProviderStatsResponse, error) {
	provider, err := s.registry.GetProvider(id)
	if err != nil {
		return nil, errors.NewNotFoundError("provider")
	}

	info := provider.GetProviderInfo()
	
	// Get metrics from provider if it supports metrics
	// TODO: Implement metrics provider interface when available
	_ = provider
	hasMetrics := false
	if !hasMetrics {
		// Return empty stats if provider doesn't support metrics
		return &dto.ProviderStatsResponse{
			ID:                 id,
			Name:               info.DisplayName,
			TotalRequests:      0,
			SuccessfulRequests: 0,
			FailedRequests:     0,
			PeriodStart:        time.Now().Add(-24 * time.Hour),
			PeriodEnd:          time.Now(),
		}, nil
	}

	// Return empty stats for now
	// TODO: Implement metrics collection when provider interface supports it
	return &dto.ProviderStatsResponse{
		ID:                  id,
		Name:                info.DisplayName,
		TotalRequests:       0,
		SuccessfulRequests:  0,
		FailedRequests:      0,
		AverageResponseTime: 0,
		TotalProcessingTime: 0,
		TotalAudioDuration:  0,
		SuccessRate:         0,
		PeriodStart:         time.Now().Add(-24 * time.Hour),
		PeriodEnd:           time.Now(),
	}, nil
}

// TestProvider tests a provider with optional test file
func (s *ProviderServiceImpl) TestProvider(ctx context.Context, id string, req *dto.TestProviderRequest) (*dto.TestProviderResponse, error) {
	providerInstance, err := s.registry.GetProvider(id)
	if err != nil {
		return nil, errors.NewNotFoundError("provider")
	}

	// Use default test file if not provided
	testFile := req.TestFile
	if testFile == "" {
		// TODO: Use a default test audio file from the project
		testFile = "/tmp/test_audio.wav"
	}

	// Create transcription request
	transcriptionReq := &provider.TranscriptionRequest{
		InputFilePath:   testFile,
		ProviderOptions: req.Options,
	}

	// Perform test with timing
	start := time.Now()
	result, err := providerInstance.TranscriptWithOptions(ctx, transcriptionReq)
	responseTime := time.Since(start).Milliseconds()

	if err != nil {
		return &dto.TestProviderResponse{
			Success:      false,
			ResponseTime: responseTime,
			ErrorMessage: err.Error(),
			TestedAt:     time.Now(),
		}, nil
	}

	// Return successful test result
	return &dto.TestProviderResponse{
		Success:      true,
		ResponseTime: responseTime,
		TestResult: map[string]interface{}{
			"provider":       id,
			"language":       result.Language,
			"duration":       result.Duration.Seconds(),
			"text_preview":   truncateText(result.Text, 100),
			"segment_count":  len(result.Segments),
			"confidence":     result.Confidence,
		},
		TestedAt: time.Now(),
	}, nil
}

// truncateText truncates text to specified length with ellipsis
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}