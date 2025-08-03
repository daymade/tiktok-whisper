package provider

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ProviderConfiguration represents the complete provider configuration
type ProviderConfiguration struct {
	// Default provider to use when none is specified
	DefaultProvider string `yaml:"default_provider"`
	
	// Provider-specific configurations
	Providers map[string]ProviderConfig `yaml:"providers"`
	
	// Orchestrator configuration
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	
	// Global settings
	Global GlobalConfig `yaml:"global"`
}

// ProviderConfig represents configuration for a single provider
type ProviderConfig struct {
	// Provider type (whisper_cpp, openai, elevenlabs, custom_http, etc.)
	Type string `yaml:"type"`
	
	// Whether this provider is enabled
	Enabled bool `yaml:"enabled"`
	
	// Provider-specific settings
	Settings map[string]interface{} `yaml:"settings"`
	
	// Authentication settings
	Auth AuthConfig `yaml:"auth,omitempty"`
	
	// Performance settings
	Performance PerformanceConfig `yaml:"performance,omitempty"`
	
	// Retry and error handling
	ErrorHandling ErrorHandlingConfig `yaml:"error_handling,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	// API key (can be environment variable reference like ${OPENAI_API_KEY})
	APIKey string `yaml:"api_key,omitempty"`
	
	// Additional headers for HTTP-based providers
	Headers map[string]string `yaml:"headers,omitempty"`
	
	// Base URL for custom HTTP providers
	BaseURL string `yaml:"base_url,omitempty"`
}

// PerformanceConfig represents performance-related configuration
type PerformanceConfig struct {
	// Timeout for transcription requests
	TimeoutSec int `yaml:"timeout_sec,omitempty"`
	
	// Maximum concurrent requests
	MaxConcurrency int `yaml:"max_concurrency,omitempty"`
	
	// Rate limiting (requests per minute)
	RateLimitRPM int `yaml:"rate_limit_rpm,omitempty"`
}

// ErrorHandlingConfig represents error handling configuration
type ErrorHandlingConfig struct {
	// Maximum number of retries
	MaxRetries int `yaml:"max_retries,omitempty"`
	
	// Delay between retries
	RetryDelayMs int `yaml:"retry_delay_ms,omitempty"`
	
	// Whether to use exponential backoff
	ExponentialBackoff bool `yaml:"exponential_backoff,omitempty"`
}

// GlobalConfig represents global configuration settings
type GlobalConfig struct {
	// Global timeout for all operations
	GlobalTimeoutSec int `yaml:"global_timeout_sec,omitempty"`
	
	// Temporary directory for file processing
	TempDir string `yaml:"temp_dir,omitempty"`
	
	// Logging configuration
	LogLevel string `yaml:"log_level,omitempty"`
	
	// Metrics collection settings
	Metrics MetricsConfig `yaml:"metrics,omitempty"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	// Whether to collect metrics
	Enabled bool `yaml:"enabled"`
	
	// Metrics retention period
	RetentionDays int `yaml:"retention_days,omitempty"`
	
	// Export format (prometheus, json, etc.)
	ExportFormat string `yaml:"export_format,omitempty"`
}

// ConfigManager manages provider configuration
type ConfigManager struct {
	configPath string
	config     *ProviderConfiguration
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadConfig loads configuration from the YAML file
func (cm *ConfigManager) LoadConfig() (*ProviderConfiguration, error) {
	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Create default configuration if file doesn't exist
		defaultConfig := cm.createDefaultConfig()
		if err := cm.SaveConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		cm.config = defaultConfig
		return defaultConfig, nil
	}
	
	// Read config file
	data, err := ioutil.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse YAML
	var config ProviderConfiguration
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}
	
	// Expand environment variables
	if err := cm.expandEnvironmentVariables(&config); err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}
	
	// Validate configuration
	if err := cm.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	cm.config = &config
	return &config, nil
}

// SaveConfig saves configuration to the YAML file
func (cm *ConfigManager) SaveConfig(config *ProviderConfiguration) error {
	// Ensure directory exists
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	
	// Write to file
	if err := ioutil.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	cm.config = config
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *ProviderConfiguration {
	return cm.config
}

// UpdateProviderConfig updates configuration for a specific provider
func (cm *ConfigManager) UpdateProviderConfig(providerName string, config ProviderConfig) error {
	if cm.config == nil {
		return fmt.Errorf("configuration not loaded")
	}
	
	if cm.config.Providers == nil {
		cm.config.Providers = make(map[string]ProviderConfig)
	}
	
	cm.config.Providers[providerName] = config
	return cm.SaveConfig(cm.config)
}

// createDefaultConfig creates a default configuration
func (cm *ConfigManager) createDefaultConfig() *ProviderConfiguration {
	return &ProviderConfiguration{
		DefaultProvider: "whisper_cpp",
		Providers: map[string]ProviderConfig{
			"whisper_cpp": {
				Type:    "whisper_cpp",
				Enabled: true,
				Settings: map[string]interface{}{
					"binary_path": "${WHISPER_CPP_BINARY:-./whisper.cpp/main}",
					"model_path":  "${WHISPER_CPP_MODEL:-./whisper.cpp/models/ggml-large-v2.bin}",
					"language":    "zh",
					"prompt":      "以下是简体中文普通话:",
				},
				Performance: PerformanceConfig{
					TimeoutSec:     300, // 5 minutes
					MaxConcurrency: 2,
				},
				ErrorHandling: ErrorHandlingConfig{
					MaxRetries:   2,
					RetryDelayMs: 1000,
				},
			},
			"openai": {
				Type:    "openai",
				Enabled: false, // Disabled by default since it requires API key
				Auth: AuthConfig{
					APIKey: "${OPENAI_API_KEY}",
				},
				Settings: map[string]interface{}{
					"model":           "whisper-1",
					"response_format": "text",
				},
				Performance: PerformanceConfig{
					TimeoutSec:     60,
					MaxConcurrency: 5,
					RateLimitRPM:   50,
				},
				ErrorHandling: ErrorHandlingConfig{
					MaxRetries:         3,
					RetryDelayMs:       2000,
					ExponentialBackoff: true,
				},
			},
			"elevenlabs": {
				Type:    "elevenlabs",
				Enabled: false, // Disabled by default
				Auth: AuthConfig{
					APIKey: "${ELEVENLABS_API_KEY}",
				},
				Settings: map[string]interface{}{
					"model": "whisper-large-v3",
				},
				Performance: PerformanceConfig{
					TimeoutSec:     120,
					MaxConcurrency: 3,
					RateLimitRPM:   100,
				},
				ErrorHandling: ErrorHandlingConfig{
					MaxRetries:   2,
					RetryDelayMs: 1500,
				},
			},
		},
		Orchestrator: OrchestratorConfig{
			FallbackChain:       []string{"whisper_cpp", "openai"},
			HealthCheckInterval: 5 * time.Minute,
			MaxRetries:          1,
			RetryDelay:          2 * time.Second,
			PreferLocal:         true,
			RouterRules: RouterRules{
				ByFileSize: map[string]string{
					"small":  "whisper_cpp", // < 10MB
					"medium": "whisper_cpp", // 10MB - 100MB
					"large":  "openai",      // > 100MB
				},
				ByLanguage: map[string]string{
					"zh": "whisper_cpp",
					"en": "openai",
				},
			},
		},
		Global: GlobalConfig{
			GlobalTimeoutSec: 600, // 10 minutes
			TempDir:          "/tmp/transcription",
			LogLevel:         "info",
			Metrics: MetricsConfig{
				Enabled:       true,
				RetentionDays: 30,
				ExportFormat:  "json",
			},
		},
	}
}

// expandEnvironmentVariables expands environment variable references in the config
func (cm *ConfigManager) expandEnvironmentVariables(config *ProviderConfiguration) error {
	for name, providerConfig := range config.Providers {
		// Expand API key
		if providerConfig.Auth.APIKey != "" {
			expanded := os.ExpandEnv(providerConfig.Auth.APIKey)
			providerConfig.Auth.APIKey = expanded
		}
		
		// Expand base URL
		if providerConfig.Auth.BaseURL != "" {
			expanded := os.ExpandEnv(providerConfig.Auth.BaseURL)
			providerConfig.Auth.BaseURL = expanded
		}
		
		// Expand headers
		for key, value := range providerConfig.Auth.Headers {
			providerConfig.Auth.Headers[key] = os.ExpandEnv(value)
		}
		
		// Update the config
		config.Providers[name] = providerConfig
	}
	
	// Expand global temp dir
	if config.Global.TempDir != "" {
		config.Global.TempDir = os.ExpandEnv(config.Global.TempDir)
	}
	
	return nil
}

// validateConfig validates the configuration
func (cm *ConfigManager) validateConfig(config *ProviderConfiguration) error {
	// Check that default provider exists and is enabled
	if config.DefaultProvider != "" {
		defaultConfig, exists := config.Providers[config.DefaultProvider]
		if !exists {
			return fmt.Errorf("default provider '%s' not found in providers", config.DefaultProvider)
		}
		if !defaultConfig.Enabled {
			return fmt.Errorf("default provider '%s' is disabled", config.DefaultProvider)
		}
	}
	
	// Validate provider configurations
	for name, providerConfig := range config.Providers {
		if providerConfig.Type == "" {
			return fmt.Errorf("provider '%s' has no type specified", name)
		}
		
		// Validate timeouts
		if providerConfig.Performance.TimeoutSec < 0 {
			return fmt.Errorf("provider '%s' has invalid timeout", name)
		}
		
		// Validate concurrency
		if providerConfig.Performance.MaxConcurrency < 0 {
			return fmt.Errorf("provider '%s' has invalid max concurrency", name)
		}
		
		// Validate retry settings
		if providerConfig.ErrorHandling.MaxRetries < 0 {
			return fmt.Errorf("provider '%s' has invalid max retries", name)
		}
	}
	
	return nil
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	// Try user home directory first
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".tiktok-whisper", "providers.yaml")
	}
	
	// Fallback to current directory
	return "./config/providers.yaml"
}