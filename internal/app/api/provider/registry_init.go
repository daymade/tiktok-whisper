package provider

import (
	"fmt"
	"sync"
)

// ProviderCreator is a function that creates a provider from configuration
type ProviderCreator func(config map[string]interface{}) (TranscriptionProvider, error)

// providerRegistry stores provider creation functions
var (
	providerRegistry = make(map[string]ProviderCreator)
	registryMutex    sync.RWMutex
)

// RegisterProvider registers a provider creator function
func RegisterProvider(providerType string, creator ProviderCreator) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	providerRegistry[providerType] = creator
}

// GetProviderCreator returns the creator function for a provider type
func GetProviderCreator(providerType string) (ProviderCreator, error) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	
	creator, ok := providerRegistry[providerType]
	if !ok {
		return nil, fmt.Errorf("provider type %s not registered", providerType)
	}
	return creator, nil
}

// ListRegisteredProviders returns all registered provider types
func ListRegisteredProviders() []string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	
	var providers []string
	for providerType := range providerRegistry {
		providers = append(providers, providerType)
	}
	return providers
}