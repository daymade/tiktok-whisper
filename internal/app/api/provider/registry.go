package provider

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultProviderRegistry implements ProviderRegistry interface
type DefaultProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]TranscriptionProvider
	default_  string
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *DefaultProviderRegistry {
	return &DefaultProviderRegistry{
		providers: make(map[string]TranscriptionProvider),
	}
}

// RegisterProvider registers a new transcription provider
func (r *DefaultProviderRegistry) RegisterProvider(name string, provider TranscriptionProvider) error {
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate registration
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider '%s' already registered", name)
	}

	// Validate the provider configuration
	if err := provider.ValidateConfiguration(); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	r.providers[name] = provider

	// Set as default if it's the first provider
	if r.default_ == "" {
		r.default_ = name
	}

	return nil
}

// GetProvider retrieves a provider by name
func (r *DefaultProviderRegistry) GetProvider(name string) (TranscriptionProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	return provider, nil
}

// ListProviders returns a list of all registered provider names
func (r *DefaultProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// GetDefaultProvider returns the default provider
func (r *DefaultProviderRegistry) GetDefaultProvider() (TranscriptionProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.default_ == "" {
		return nil, fmt.Errorf("no default provider set")
	}

	provider, exists := r.providers[r.default_]
	if !exists {
		return nil, fmt.Errorf("default provider '%s' not found", r.default_)
	}

	return provider, nil
}

// SetDefaultProvider sets the default provider
func (r *DefaultProviderRegistry) SetDefaultProvider(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; !exists {
		return fmt.Errorf("provider '%s' not found", name)
	}

	r.default_ = name
	return nil
}

// HealthCheckAll performs health checks on all registered providers
func (r *DefaultProviderRegistry) HealthCheckAll(ctx context.Context) map[string]error {
	r.mu.RLock()
	providers := make(map[string]TranscriptionProvider)
	for name, provider := range r.providers {
		providers[name] = provider
	}
	r.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, provider := range providers {
		wg.Add(1)
		go func(name string, provider TranscriptionProvider) {
			defer wg.Done()
			
			err := provider.HealthCheck(ctx)
			
			mu.Lock()
			results[name] = err
			mu.Unlock()
		}(name, provider)
	}

	wg.Wait()
	return results
}

// DefaultTranscriptionOrchestrator implements TranscriptionOrchestrator interface
type DefaultTranscriptionOrchestrator struct {
	registry ProviderRegistry
	metrics  ProviderMetrics
	config   OrchestratorConfig
	mu       sync.RWMutex
	stats    OrchestratorStats
}


// NewTranscriptionOrchestrator creates a new transcription orchestrator
func NewTranscriptionOrchestrator(registry ProviderRegistry, metrics ProviderMetrics, config OrchestratorConfig) *DefaultTranscriptionOrchestrator {
	return &DefaultTranscriptionOrchestrator{
		registry: registry,
		metrics:  metrics,
		config:   config,
		stats:    OrchestratorStats{
			ProviderUsage:    make(map[string]int64),
			ErrorsByProvider: make(map[string]int64),
			LastHealthCheck:  make(map[string]bool),
		},
	}
}

// Transcribe performs transcription with automatic provider selection
func (o *DefaultTranscriptionOrchestrator) Transcribe(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error) {
	startTime := time.Now()
	
	o.mu.Lock()
	o.stats.TotalRequests++
	o.mu.Unlock()
	
	// Get recommended providers
	providers, err := o.RecommendProvider(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider recommendations: %w", err)
	}
	
	// Try each provider in order
	var lastErr error
	for _, providerName := range providers {
		response, err := o.TranscribeWithProvider(ctx, providerName, request)
		if err != nil {
			lastErr = err
			continue
		}
		
		// Record success
		latency := time.Since(startTime).Milliseconds()
		o.metrics.RecordSuccess(providerName, latency, 0) // TODO: extract audio duration
		
		o.mu.Lock()
		o.stats.SuccessfulRequests++
		o.stats.ProviderUsage[providerName]++
		o.mu.Unlock()
		
		return response, nil
	}
	
	// All providers failed
	o.mu.Lock()
	o.stats.FailedRequests++
	o.mu.Unlock()
	
	return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
}

// TranscribeWithProvider transcribes with a specific provider, with fallback
func (o *DefaultTranscriptionOrchestrator) TranscribeWithProvider(ctx context.Context, providerName string, request *TranscriptionRequest) (*TranscriptionResponse, error) {
	provider, err := o.registry.GetProvider(providerName)
	if err != nil {
		o.metrics.RecordFailure(providerName, "provider_not_found")
		return nil, fmt.Errorf("failed to get provider '%s': %w", providerName, err)
	}
	
	// Try the primary provider
	response, err := o.tryProvider(ctx, provider, request)
	if err == nil {
		return response, nil
	}
	
	// Record failure
	o.metrics.RecordFailure(providerName, "transcription_failed")
	o.mu.Lock()
	o.stats.ErrorsByProvider[providerName]++
	o.mu.Unlock()
	
	// Try fallback providers if configured
	for _, fallbackName := range o.config.FallbackChain {
		if fallbackName == providerName {
			continue // Skip the already tried provider
		}
		
		fallbackProvider, err := o.registry.GetProvider(fallbackName)
		if err != nil {
			continue
		}
		
		response, err := o.tryProvider(ctx, fallbackProvider, request)
		if err == nil {
			return response, nil
		}
		
		o.metrics.RecordFailure(fallbackName, "fallback_failed")
	}
	
	return nil, fmt.Errorf("provider '%s' and all fallbacks failed: %w", providerName, err)
}

// tryProvider attempts transcription with retry logic
func (o *DefaultTranscriptionOrchestrator) tryProvider(ctx context.Context, provider TranscriptionProvider, request *TranscriptionRequest) (*TranscriptionResponse, error) {
	var lastErr error
	
	for attempt := 0; attempt <= o.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(o.config.RetryDelay):
			}
		}
		
		// Try transcription
		response, err := provider.TranscriptWithOptions(ctx, request)
		if err == nil {
			return response, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if transcriptErr, ok := err.(*TranscriptionError); ok && !transcriptErr.Retryable {
			break // Don't retry non-retryable errors
		}
	}
	
	return nil, lastErr
}

// RecommendProvider recommends providers based on request characteristics
func (o *DefaultTranscriptionOrchestrator) RecommendProvider(request *TranscriptionRequest) ([]string, error) {
	// Start with configured fallback chain or all providers
	var candidates []string
	if len(o.config.FallbackChain) > 0 {
		candidates = append(candidates, o.config.FallbackChain...)
	} else {
		candidates = o.registry.ListProviders()
	}
	
	// Apply routing rules
	candidates = o.applyRoutingRules(request, candidates)
	
	// If prefer local is enabled, prioritize local providers
	if o.config.PreferLocal {
		candidates = o.prioritizeLocalProviders(candidates)
	}
	
	// Ensure we have at least one candidate
	if len(candidates) == 0 {
		allProviders := o.registry.ListProviders()
		if len(allProviders) == 0 {
			return nil, fmt.Errorf("no providers available")
		}
		candidates = allProviders
	}
	
	return candidates, nil
}

// applyRoutingRules applies routing rules to filter and order providers
func (o *DefaultTranscriptionOrchestrator) applyRoutingRules(request *TranscriptionRequest, candidates []string) []string {
	// This is a simplified implementation - in a real system you'd analyze the file
	// For now, just return the candidates as-is
	return candidates
}

// prioritizeLocalProviders moves local providers to the front of the list
func (o *DefaultTranscriptionOrchestrator) prioritizeLocalProviders(candidates []string) []string {
	var local, remote []string
	
	for _, name := range candidates {
		provider, err := o.registry.GetProvider(name)
		if err != nil {
			continue
		}
		
		info := provider.GetProviderInfo()
		if info.Type == ProviderTypeLocal {
			local = append(local, name)
		} else {
			remote = append(remote, name)
		}
	}
	
	// Local providers first, then remote
	result := append(local, remote...)
	return result
}

// GetStats returns current orchestrator statistics
func (o *DefaultTranscriptionOrchestrator) GetStats() OrchestratorStats {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	stats := o.stats
	stats.ProviderUsage = make(map[string]int64)
	stats.ErrorsByProvider = make(map[string]int64)
	stats.LastHealthCheck = make(map[string]bool)
	
	for k, v := range o.stats.ProviderUsage {
		stats.ProviderUsage[k] = v
	}
	for k, v := range o.stats.ErrorsByProvider {
		stats.ErrorsByProvider[k] = v
	}
	for k, v := range o.stats.LastHealthCheck {
		stats.LastHealthCheck[k] = v
	}
	
	return stats
}