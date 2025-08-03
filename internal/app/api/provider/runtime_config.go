package provider

import "sync"

// RuntimeConfig holds runtime configuration for provider selection
type RuntimeConfig struct {
	ProviderName string
}

var (
	runtimeConfig     *RuntimeConfig
	runtimeConfigOnce sync.Once
	runtimeConfigMu   sync.RWMutex
)

// SetRuntimeConfig sets the runtime configuration
func SetRuntimeConfig(config *RuntimeConfig) {
	runtimeConfigMu.Lock()
	defer runtimeConfigMu.Unlock()
	runtimeConfig = config
}

// GetRuntimeConfig gets the runtime configuration
func GetRuntimeConfig() *RuntimeConfig {
	runtimeConfigMu.RLock()
	defer runtimeConfigMu.RUnlock()
	return runtimeConfig
}

// InitializeRuntimeConfig initializes runtime config with defaults
func InitializeRuntimeConfig() {
	runtimeConfigOnce.Do(func() {
		if runtimeConfig == nil {
			runtimeConfig = &RuntimeConfig{}
		}
	})
}