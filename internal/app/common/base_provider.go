package common

// BaseProvider provides common implementation for all providers
type BaseProvider struct {
	Name                     string
	DisplayName              string
	Type                     ProviderType
	Version                  string
	SupportedFormats         []AudioFormat
	SupportedLanguages       []string
	MaxFileSizeMB            int
	MaxDurationSec           int
	SupportsTimestamps       bool
	SupportsWordLevel        bool
	SupportsConfidence       bool
	SupportsLanguageDetection bool
	SupportsStreaming        bool
	RequiresInternet         bool
	RequiresAPIKey           bool
	RequiresBinary           bool
	DefaultModel             string
	AvailableModels          []string
	ConfigSchema             map[string]interface{}
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name, displayName string, providerType ProviderType, version string) BaseProvider {
	return BaseProvider{
		Name:               name,
		DisplayName:        displayName,
		Type:               providerType,
		Version:            version,
		SupportedFormats:   []AudioFormat{FormatWAV, FormatMP3},
		SupportsTimestamps: true,
	}
}

// GetProviderInfo returns provider information
func (b BaseProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:                     b.Name,
		DisplayName:              b.DisplayName,
		Type:                     b.Type,
		Version:                  b.Version,
		SupportedFormats:         b.SupportedFormats,
		SupportedLanguages:       b.SupportedLanguages,
		MaxFileSizeMB:            b.MaxFileSizeMB,
		MaxDurationSec:           b.MaxDurationSec,
		SupportsTimestamps:       b.SupportsTimestamps,
		SupportsWordLevel:        b.SupportsWordLevel,
		SupportsConfidence:       b.SupportsConfidence,
		SupportsLanguageDetection: b.SupportsLanguageDetection,
		SupportsStreaming:        b.SupportsStreaming,
		RequiresInternet:         b.RequiresInternet,
		RequiresAPIKey:           b.RequiresAPIKey,
		RequiresBinary:           b.RequiresBinary,
		DefaultModel:             b.DefaultModel,
		AvailableModels:          b.AvailableModels,
		ConfigSchema:             b.ConfigSchema,
	}
}

// SetSupportedFormats sets the supported audio formats
func (b *BaseProvider) SetSupportedFormats(formats []AudioFormat) {
	b.SupportedFormats = formats
}

// AddSupportedFormat adds a supported audio format
func (b *BaseProvider) AddSupportedFormat(format AudioFormat) {
	b.SupportedFormats = append(b.SupportedFormats, format)
}