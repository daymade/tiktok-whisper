package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProvidersConfig represents the overall configuration for all providers
type ProvidersConfig struct {
	DefaultProvider string                     `yaml:"default_provider"`
	Providers       map[string]ProviderConfig  `yaml:"providers"`
	Orchestrator    OrchestratorConfig         `yaml:"orchestrator,omitempty"`
}

// ProviderConfig represents configuration for a single provider
type ProviderConfig struct {
	Type        string                 `yaml:"type"`
	Enabled     bool                   `yaml:"enabled"`
	Priority    int                    `yaml:"priority,omitempty"`
	Auth        map[string]interface{} `yaml:"auth,omitempty"`
	Settings    map[string]interface{} `yaml:"settings,omitempty"`
	Performance PerformanceConfig      `yaml:"performance,omitempty"`
	Retry       RetryConfig            `yaml:"retry,omitempty"`
}

// PerformanceConfig represents performance settings for a provider
type PerformanceConfig struct {
	TimeoutSec      int `yaml:"timeout_sec,omitempty"`
	MaxConcurrency  int `yaml:"max_concurrency,omitempty"`
	RateLimitRPM    int `yaml:"rate_limit_rpm,omitempty"`
}

// RetryConfig represents retry settings for a provider
type RetryConfig struct {
	MaxAttempts int `yaml:"max_attempts,omitempty"`
	BackoffSec  int `yaml:"backoff_sec,omitempty"`
}

// OrchestratorConfig represents orchestration settings
type OrchestratorConfig struct {
	FallbackChain []string          `yaml:"fallback_chain,omitempty"`
	PreferLocal   bool              `yaml:"prefer_local,omitempty"`
	RouterRules   RouterRulesConfig `yaml:"router_rules,omitempty"`
}

// RouterRulesConfig represents routing rules for the orchestrator
type RouterRulesConfig struct {
	ByFileSize map[string]string `yaml:"by_file_size,omitempty"`
	ByLanguage map[string]string `yaml:"by_language,omitempty"`
}

// LoadProvidersConfig loads provider configuration from a YAML file
func LoadProvidersConfig(configPath string) (*ProvidersConfig, error) {
	// Expand environment variables in path
	configPath = os.ExpandEnv(configPath)
	
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}
	
	// Read file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse YAML
	var config ProvidersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// Expand environment variables in configuration
	config.expandEnvironmentVariables()
	
	// Set defaults
	config.setDefaults()
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return &config, nil
}

// SaveProvidersConfig saves provider configuration to a YAML file
func SaveProvidersConfig(config *ProvidersConfig, configPath string) error {
	// Expand environment variables in path
	configPath = os.ExpandEnv(configPath)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	
	// Write file
	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// expandEnvironmentVariables expands environment variables in the configuration
func (c *ProvidersConfig) expandEnvironmentVariables() {
	for _, provider := range c.Providers {
		// Expand in auth section
		for key, value := range provider.Auth {
			if strValue, ok := value.(string); ok {
				if strings.HasPrefix(strValue, "${") && strings.HasSuffix(strValue, "}") {
					envVar := strings.TrimSuffix(strings.TrimPrefix(strValue, "${"), "}")
					provider.Auth[key] = os.Getenv(envVar)
				}
			}
		}
		
		// Expand in settings section
		for key, value := range provider.Settings {
			if strValue, ok := value.(string); ok {
				if strings.HasPrefix(strValue, "${") && strings.HasSuffix(strValue, "}") {
					envVar := strings.TrimSuffix(strings.TrimPrefix(strValue, "${"), "}")
					provider.Settings[key] = os.Getenv(envVar)
				}
			}
		}
	}
}

// setDefaults sets default values for the configuration
func (c *ProvidersConfig) setDefaults() {
	// Set default provider if not specified
	if c.DefaultProvider == "" && len(c.Providers) > 0 {
		// Try to find whisper_cpp first
		if _, ok := c.Providers["whisper_cpp"]; ok {
			c.DefaultProvider = "whisper_cpp"
		} else {
			// Use the first enabled provider
			for name, provider := range c.Providers {
				if provider.Enabled {
					c.DefaultProvider = name
					break
				}
			}
		}
	}
	
	// Set default performance values
	for name, provider := range c.Providers {
		if provider.Performance.TimeoutSec == 0 {
			provider.Performance.TimeoutSec = 300 // 5 minutes default
		}
		if provider.Performance.MaxConcurrency == 0 {
			provider.Performance.MaxConcurrency = 1
		}
		if provider.Retry.MaxAttempts == 0 {
			provider.Retry.MaxAttempts = 3
		}
		if provider.Retry.BackoffSec == 0 {
			provider.Retry.BackoffSec = 2
		}
		c.Providers[name] = provider
	}
}

// Validate validates the configuration
func (c *ProvidersConfig) Validate() error {
	// Check if at least one provider is enabled
	hasEnabledProvider := false
	for _, provider := range c.Providers {
		if provider.Enabled {
			hasEnabledProvider = true
			break
		}
	}
	
	if !hasEnabledProvider {
		return fmt.Errorf("at least one provider must be enabled")
	}
	
	// Check if default provider exists and is enabled
	if c.DefaultProvider != "" {
		provider, exists := c.Providers[c.DefaultProvider]
		if !exists {
			return fmt.Errorf("default provider '%s' does not exist", c.DefaultProvider)
		}
		if !provider.Enabled {
			return fmt.Errorf("default provider '%s' is not enabled", c.DefaultProvider)
		}
	}
	
	// Validate provider types
	validTypes := map[string]bool{
		"whisper_cpp":    true,
		"openai":         true,
		"elevenlabs":     true,
		"ssh_whisper":    true,
		"whisper_server": true,
		"custom_http":    true,
	}
	
	for name, provider := range c.Providers {
		if !validTypes[provider.Type] {
			return fmt.Errorf("invalid provider type '%s' for provider '%s'", provider.Type, name)
		}
	}
	
	return nil
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	// Check environment variable first
	if path := os.Getenv("PROVIDERS_CONFIG_PATH"); path != "" {
		return path
	}
	
	// Use home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "providers.yaml"
	}
	
	return filepath.Join(home, ".tiktok-whisper", "providers.yaml")
}

// CreateDefaultConfig creates a default configuration
func CreateDefaultConfig() *ProvidersConfig {
	return &ProvidersConfig{
		DefaultProvider: "whisper_cpp",
		Providers: map[string]ProviderConfig{
			"whisper_cpp": {
				Type:    "whisper_cpp",
				Enabled: true,
				Settings: map[string]interface{}{
					"binary_path": "/usr/local/bin/whisper",
					"model_path":  "/models/ggml-large-v2.bin",
					"language":    "zh",
					"prompt":      "以下是简体中文普通话:",
				},
				Performance: PerformanceConfig{
					TimeoutSec:     300,
					MaxConcurrency: 2,
				},
			},
			"openai": {
				Type:    "openai",
				Enabled: false,
				Auth: map[string]interface{}{
					"api_key": "${OPENAI_API_KEY}",
				},
				Settings: map[string]interface{}{
					"model":           "whisper-1",
					"response_format": "text",
				},
				Performance: PerformanceConfig{
					TimeoutSec:   60,
					RateLimitRPM: 50,
				},
			},
			"elevenlabs": {
				Type:    "elevenlabs",
				Enabled: false,
				Auth: map[string]interface{}{
					"api_key": "${ELEVENLABS_API_KEY}",
				},
				Settings: map[string]interface{}{
					"model": "whisper-large-v3",
				},
			},
		},
		Orchestrator: OrchestratorConfig{
			FallbackChain: []string{"whisper_cpp", "openai"},
			PreferLocal:   true,
			RouterRules: RouterRulesConfig{
				ByFileSize: map[string]string{
					"small": "whisper_cpp",
					"large": "openai",
				},
			},
		},
	}
}