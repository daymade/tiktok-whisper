package provider

import (
	"time"
)

// OrchestratorConfig defines orchestration rules for provider selection
type OrchestratorConfig struct {
	// FallbackChain defines the order of providers to try if one fails
	FallbackChain []string `yaml:"fallback_chain" json:"fallback_chain"`
	
	// PreferLocal indicates if local providers should be preferred over remote ones
	PreferLocal bool `yaml:"prefer_local" json:"prefer_local"`
	
	// RouterRules defines routing rules based on file characteristics
	RouterRules RouterRules `yaml:"router_rules" json:"router_rules"`
	
	// HealthCheckInterval defines how often to check provider health
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
	
	// MaxRetries defines the maximum number of retry attempts
	MaxRetries int `yaml:"max_retries" json:"max_retries"`
	
	// RetryDelay defines the delay between retry attempts
	RetryDelay time.Duration `yaml:"retry_delay" json:"retry_delay"`
	
	// RetryPolicy defines retry behavior for failed transcriptions
	RetryPolicy RetryPolicy `yaml:"retry_policy" json:"retry_policy"`
	
	// LoadBalancing defines load balancing strategy
	LoadBalancing LoadBalancingConfig `yaml:"load_balancing" json:"load_balancing"`
}

// RouterRules defines routing rules based on file characteristics
type RouterRules struct {
	// ByFileSize maps file size ranges to providers
	ByFileSize map[string]string `yaml:"by_file_size" json:"by_file_size"`
	
	// ByLanguage maps languages to preferred providers
	ByLanguage map[string]string `yaml:"by_language" json:"by_language"`
	
	// ByFormat maps file formats to providers
	ByFormat map[string]string `yaml:"by_format" json:"by_format"`
	
	// ByDuration maps duration ranges to providers
	ByDuration map[string]string `yaml:"by_duration" json:"by_duration"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts     int           `yaml:"max_attempts" json:"max_attempts"`
	InitialInterval time.Duration `yaml:"initial_interval" json:"initial_interval"`
	MaxInterval     time.Duration `yaml:"max_interval" json:"max_interval"`
	Multiplier      float64       `yaml:"multiplier" json:"multiplier"`
}

// LoadBalancingConfig defines load balancing configuration
type LoadBalancingConfig struct {
	Strategy string                 `yaml:"strategy" json:"strategy"` // round-robin, least-connections, weighted
	Weights  map[string]int         `yaml:"weights" json:"weights"`   // Provider weights for weighted strategy
	Options  map[string]interface{} `yaml:"options" json:"options"`   // Additional strategy-specific options
}

// DefaultOrchestratorConfig returns a default orchestrator configuration
func DefaultOrchestratorConfig() *OrchestratorConfig {
	return &OrchestratorConfig{
		FallbackChain: []string{"whisper_cpp", "openai"},
		PreferLocal:   true,
		RouterRules: RouterRules{
			ByFileSize: map[string]string{
				"small":  "whisper_cpp",  // < 50MB
				"medium": "whisper_cpp",  // 50-200MB
				"large":  "openai",       // > 200MB
			},
			ByLanguage: map[string]string{
				"zh": "whisper_cpp",
				"en": "openai",
			},
		},
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
		RetryDelay:          2 * time.Second,
		RetryPolicy: RetryPolicy{
			MaxAttempts:     3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
		},
		LoadBalancing: LoadBalancingConfig{
			Strategy: "round-robin",
		},
	}
}

// ProviderSelector implements intelligent provider selection
type ProviderSelector interface {
	// SelectProvider selects the best provider for a request
	SelectProvider(req *TranscriptionRequest) (TranscriptionProvider, error)
	
	// GetFallbackChain returns the fallback chain for a request
	GetFallbackChain(req *TranscriptionRequest) []TranscriptionProvider
	
	// UpdateProviderHealth updates the health status of a provider
	UpdateProviderHealth(providerName string, isHealthy bool)
}

// SmartProviderSelector implements intelligent provider selection
type SmartProviderSelector struct {
	config           *OrchestratorConfig
	registry         ProviderRegistry
	healthStatus     map[string]bool
	lastHealthCheck  map[string]time.Time
	roundRobinIndex  int
}

// NewSmartProviderSelector creates a new smart provider selector
func NewSmartProviderSelector(config *OrchestratorConfig, registry ProviderRegistry) *SmartProviderSelector {
	return &SmartProviderSelector{
		config:          config,
		registry:        registry,
		healthStatus:    make(map[string]bool),
		lastHealthCheck: make(map[string]time.Time),
	}
}

// SelectProvider selects the best provider based on rules and health
func (s *SmartProviderSelector) SelectProvider(req *TranscriptionRequest) (TranscriptionProvider, error) {
	// Check routing rules first
	if provider := s.checkRoutingRules(req); provider != nil {
		return provider, nil
	}
	
	// Fall back to default selection
	return s.registry.GetDefaultProvider()
}

// checkRoutingRules checks routing rules to select a provider
func (s *SmartProviderSelector) checkRoutingRules(req *TranscriptionRequest) TranscriptionProvider {
	// Check language-based routing
	if req.Language != "" && s.config.RouterRules.ByLanguage != nil {
		if providerName, ok := s.config.RouterRules.ByLanguage[req.Language]; ok {
			if provider, err := s.registry.GetProvider(providerName); err == nil {
				if s.isHealthy(providerName) {
					return provider
				}
			}
		}
	}
	
	// Check format-based routing
	if req.InputFilePath != "" && s.config.RouterRules.ByFormat != nil {
		// Extract format from file path
		format := extractFileFormat(req.InputFilePath)
		if providerName, ok := s.config.RouterRules.ByFormat[format]; ok {
			if provider, err := s.registry.GetProvider(providerName); err == nil {
				if s.isHealthy(providerName) {
					return provider
				}
			}
		}
	}
	
	return nil
}

// isHealthy checks if a provider is healthy
func (s *SmartProviderSelector) isHealthy(providerName string) bool {
	// Check if we need to update health status
	lastCheck, exists := s.lastHealthCheck[providerName]
	if !exists || time.Since(lastCheck) > s.config.HealthCheckInterval {
		// Perform health check
		if provider, err := s.registry.GetProvider(providerName); err == nil {
			healthy := provider.HealthCheck(nil) == nil
			s.healthStatus[providerName] = healthy
			s.lastHealthCheck[providerName] = time.Now()
			return healthy
		}
		return false
	}
	
	// Return cached health status
	return s.healthStatus[providerName]
}

// GetFallbackChain returns the fallback chain for a request
func (s *SmartProviderSelector) GetFallbackChain(req *TranscriptionRequest) []TranscriptionProvider {
	var chain []TranscriptionProvider
	
	for _, providerName := range s.config.FallbackChain {
		if provider, err := s.registry.GetProvider(providerName); err == nil {
			if s.isHealthy(providerName) {
				chain = append(chain, provider)
			}
		}
	}
	
	return chain
}

// UpdateProviderHealth updates the health status of a provider
func (s *SmartProviderSelector) UpdateProviderHealth(providerName string, isHealthy bool) {
	s.healthStatus[providerName] = isHealthy
	s.lastHealthCheck[providerName] = time.Now()
}

// extractFileFormat extracts the file format from a file path
func extractFileFormat(filePath string) string {
	// Simple implementation - can be enhanced
	if len(filePath) > 4 {
		ext := filePath[len(filePath)-4:]
		switch ext {
		case ".mp3", ".wav", ".m4a", ".aac":
			return "audio"
		case ".mp4", ".avi", ".mov", ".mkv":
			return "video"
		}
	}
	return "unknown"
}