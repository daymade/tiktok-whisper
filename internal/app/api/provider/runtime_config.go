package provider

import "sync"

// RuntimeConfig holds runtime configuration for provider selection
type RuntimeConfig struct {
	ProviderName      string
	ForceRetranscribe bool
}

var (
	runtimeConfig   *RuntimeConfig
	runtimeConfigMu sync.RWMutex
)

// SetRuntimeConfig sets the runtime configuration.
// If config is nil, it initializes with an empty default.
func SetRuntimeConfig(config *RuntimeConfig) {
	runtimeConfigMu.Lock()
	defer runtimeConfigMu.Unlock()
	if config == nil {
		config = &RuntimeConfig{}
	}
	runtimeConfig = config
}

// GetRuntimeConfig gets the runtime configuration
func GetRuntimeConfig() *RuntimeConfig {
	runtimeConfigMu.RLock()
	defer runtimeConfigMu.RUnlock()
	return runtimeConfig
}

// ResolveProviderType returns the active provider name, defaulting to "whisper_cpp".
func ResolveProviderType() string {
	if cfg := GetRuntimeConfig(); cfg != nil && cfg.ProviderName != "" {
		return cfg.ProviderName
	}
	return "whisper_cpp"
}