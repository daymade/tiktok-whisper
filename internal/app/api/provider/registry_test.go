package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockTranscriptionProvider implements TranscriptionProvider interface for testing
type MockTranscriptionProvider struct {
	name            string
	info            ProviderInfo
	transcriptFunc  func(string) (string, error)
	validateFunc    func() error
	healthCheckFunc func(context.Context) error
}

func (m *MockTranscriptionProvider) Transcript(inputFilePath string) (string, error) {
	if m.transcriptFunc != nil {
		return m.transcriptFunc(inputFilePath)
	}
	return "mock transcription result", nil
}

func (m *MockTranscriptionProvider) TranscriptWithOptions(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error) {
	if m.transcriptFunc != nil {
		text, err := m.transcriptFunc(request.InputFilePath)
		if err != nil {
			return nil, err
		}
		return &TranscriptionResponse{
			Text:           text,
			ProcessingTime: 100 * time.Millisecond,
			ModelUsed:      "mock-model",
		}, nil
	}
	
	return &TranscriptionResponse{
		Text:           "mock transcription result",
		ProcessingTime: 100 * time.Millisecond,
		ModelUsed:      "mock-model",
	}, nil
}

func (m *MockTranscriptionProvider) GetProviderInfo() ProviderInfo {
	if m.info.Name != "" {
		return m.info
	}
	
	return ProviderInfo{
		Name:        m.name,
		DisplayName: "Mock Provider",
		Type:        ProviderTypeLocal,
		Version:     "1.0.0",
		SupportedFormats: []AudioFormat{
			FormatWAV,
			FormatMP3,
		},
		RequiresInternet: false,
		RequiresAPIKey:   false,
	}
}

func (m *MockTranscriptionProvider) ValidateConfiguration() error {
	if m.validateFunc != nil {
		return m.validateFunc()
	}
	return nil
}

func (m *MockTranscriptionProvider) HealthCheck(ctx context.Context) error {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

// Test DefaultProviderRegistry

func TestProviderRegistry_RegisterProvider(t *testing.T) {
	registry := NewProviderRegistry()
	
	// Test successful registration
	provider := &MockTranscriptionProvider{name: "test-provider"}
	err := registry.RegisterProvider("test-provider", provider)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Test duplicate registration
	err = registry.RegisterProvider("test-provider", provider)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
	
	// Test empty name
	err = registry.RegisterProvider("", provider)
	if err == nil {
		t.Error("Expected error for empty provider name")
	}
	
	// Test nil provider
	err = registry.RegisterProvider("nil-provider", nil)
	if err == nil {
		t.Error("Expected error for nil provider")
	}
}

func TestProviderRegistry_GetProvider(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &MockTranscriptionProvider{name: "test-provider"}
	
	// Register provider
	err := registry.RegisterProvider("test-provider", provider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}
	
	// Test successful retrieval
	retrievedProvider, err := registry.GetProvider("test-provider")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if retrievedProvider != provider {
		t.Error("Retrieved provider does not match registered provider")
	}
	
	// Test non-existent provider
	_, err = registry.GetProvider("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

func TestProviderRegistry_ListProviders(t *testing.T) {
	registry := NewProviderRegistry()
	
	// Empty registry
	providers := registry.ListProviders()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
	
	// Add providers
	provider1 := &MockTranscriptionProvider{name: "provider1"}
	provider2 := &MockTranscriptionProvider{name: "provider2"}
	
	registry.RegisterProvider("provider1", provider1)
	registry.RegisterProvider("provider2", provider2)
	
	providers = registry.ListProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
	
	// Check that both providers are listed
	found1, found2 := false, false
	for _, name := range providers {
		if name == "provider1" {
			found1 = true
		}
		if name == "provider2" {
			found2 = true
		}
	}
	
	if !found1 || !found2 {
		t.Error("Not all registered providers were listed")
	}
}

func TestProviderRegistry_DefaultProvider(t *testing.T) {
	registry := NewProviderRegistry()
	
	// Test no default provider
	_, err := registry.GetDefaultProvider()
	if err == nil {
		t.Error("Expected error when no default provider is set")
	}
	
	// Register provider and check it becomes default
	provider := &MockTranscriptionProvider{name: "test-provider"}
	err = registry.RegisterProvider("test-provider", provider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}
	
	defaultProvider, err := registry.GetDefaultProvider()
	if err != nil {
		t.Fatalf("Expected no error getting default provider, got %v", err)
	}
	
	if defaultProvider != provider {
		t.Error("Default provider does not match registered provider")
	}
	
	// Test setting different default
	provider2 := &MockTranscriptionProvider{name: "provider2"}
	err = registry.RegisterProvider("provider2", provider2)
	if err != nil {
		t.Fatalf("Failed to register second provider: %v", err)
	}
	
	err = registry.SetDefaultProvider("provider2")
	if err != nil {
		t.Fatalf("Failed to set default provider: %v", err)
	}
	
	defaultProvider, err = registry.GetDefaultProvider()
	if err != nil {
		t.Fatalf("Failed to get default provider: %v", err)
	}
	
	if defaultProvider != provider2 {
		t.Error("Default provider was not updated")
	}
	
	// Test setting non-existent default
	err = registry.SetDefaultProvider("non-existent")
	if err == nil {
		t.Error("Expected error when setting non-existent default provider")
	}
}

func TestProviderRegistry_HealthCheckAll(t *testing.T) {
	registry := NewProviderRegistry()
	
	// Create providers with different health check results
	healthyProvider := &MockTranscriptionProvider{
		name: "healthy",
		healthCheckFunc: func(ctx context.Context) error {
			return nil
		},
	}
	
	unhealthyProvider := &MockTranscriptionProvider{
		name: "unhealthy",
		healthCheckFunc: func(ctx context.Context) error {
			return errors.New("provider is unhealthy")
		},
	}
	
	timeoutProvider := &MockTranscriptionProvider{
		name: "timeout",
		healthCheckFunc: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				return nil
			}
		},
	}
	
	// Register providers
	registry.RegisterProvider("healthy", healthyProvider)
	registry.RegisterProvider("unhealthy", unhealthyProvider)
	registry.RegisterProvider("timeout", timeoutProvider)
	
	// Run health checks with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	
	results := registry.HealthCheckAll(ctx)
	
	// Check results
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
	
	// Healthy provider should have no error
	if results["healthy"] != nil {
		t.Errorf("Expected healthy provider to have no error, got %v", results["healthy"])
	}
	
	// Unhealthy provider should have error
	if results["unhealthy"] == nil {
		t.Error("Expected unhealthy provider to have error")
	}
	
	// Timeout provider should have timeout error
	if results["timeout"] == nil {
		t.Error("Expected timeout provider to have error")
	}
}

// Test DefaultProviderMetrics

func TestProviderMetrics_RecordSuccess(t *testing.T) {
	metrics := NewProviderMetrics()
	
	// Record success
	metrics.RecordSuccess("test-provider", 1000, 60.0)
	
	// Get metrics
	stats := metrics.GetProviderMetrics("test-provider")
	
	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", stats.TotalRequests)
	}
	
	if stats.SuccessfulRequests != 1 {
		t.Errorf("Expected 1 successful request, got %d", stats.SuccessfulRequests)
	}
	
	if stats.FailedRequests != 0 {
		t.Errorf("Expected 0 failed requests, got %d", stats.FailedRequests)
	}
	
	if stats.SuccessRate != 1.0 {
		t.Errorf("Expected success rate of 1.0, got %f", stats.SuccessRate)
	}
	
	if stats.AverageLatencyMs != 1000.0 {
		t.Errorf("Expected average latency of 1000ms, got %f", stats.AverageLatencyMs)
	}
	
	if stats.TotalAudioProcessed != 60.0 {
		t.Errorf("Expected 60 seconds of audio processed, got %f", stats.TotalAudioProcessed)
	}
}

func TestProviderMetrics_RecordFailure(t *testing.T) {
	metrics := NewProviderMetrics()
	
	// Record failure
	metrics.RecordFailure("test-provider", "network_error")
	
	// Get metrics
	stats := metrics.GetProviderMetrics("test-provider")
	
	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", stats.TotalRequests)
	}
	
	if stats.SuccessfulRequests != 0 {
		t.Errorf("Expected 0 successful requests, got %d", stats.SuccessfulRequests)
	}
	
	if stats.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", stats.FailedRequests)
	}
	
	if stats.SuccessRate != 0.0 {
		t.Errorf("Expected success rate of 0.0, got %f", stats.SuccessRate)
	}
	
	// Check error breakdown
	if stats.ErrorBreakdown["network_error"] != 1 {
		t.Errorf("Expected 1 network_error, got %d", stats.ErrorBreakdown["network_error"])
	}
}

func TestProviderMetrics_GetOverallMetrics(t *testing.T) {
	metrics := NewProviderMetrics()
	
	// Record some metrics for multiple providers
	metrics.RecordSuccess("provider1", 1000, 30.0)
	metrics.RecordSuccess("provider1", 1500, 45.0)
	metrics.RecordFailure("provider1", "timeout")
	
	metrics.RecordSuccess("provider2", 800, 20.0)
	metrics.RecordFailure("provider2", "auth_error")
	metrics.RecordFailure("provider2", "auth_error")
	
	// Get overall metrics
	overall := metrics.GetOverallMetrics()
	
	if overall.TotalProviders != 2 {
		t.Errorf("Expected 2 total providers, got %d", overall.TotalProviders)
	}
	
	if overall.TotalRequests != 6 {
		t.Errorf("Expected 6 total requests, got %d", overall.TotalRequests)
	}
	
	if overall.SuccessfulRequests != 3 {
		t.Errorf("Expected 3 successful requests, got %d", overall.SuccessfulRequests)
	}
	
	expectedSuccessRate := 3.0 / 6.0
	if overall.OverallSuccessRate != expectedSuccessRate {
		t.Errorf("Expected success rate of %f, got %f", expectedSuccessRate, overall.OverallSuccessRate)
	}
}

// Benchmark tests

func BenchmarkProviderRegistry_GetProvider(b *testing.B) {
	registry := NewProviderRegistry()
	provider := &MockTranscriptionProvider{name: "test-provider"}
	registry.RegisterProvider("test-provider", provider)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetProvider("test-provider")
	}
}

func BenchmarkProviderMetrics_RecordSuccess(b *testing.B) {
	metrics := NewProviderMetrics()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordSuccess("test-provider", 1000, 60.0)
	}
}