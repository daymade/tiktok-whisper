package config

import (
	"fmt"
	"strings"
	"time"
)

// ValidateTimeout validates timeout duration
func ValidateTimeout(timeout time.Duration, name string) error {
	if timeout <= 0 {
		return fmt.Errorf("%s timeout must be positive", name)
	}
	if timeout > 30*time.Minute {
		return fmt.Errorf("%s timeout too large (max 30 minutes)", name)
	}
	return nil
}

// ValidateConcurrency validates concurrency setting
func ValidateConcurrency(concurrency int, name string) error {
	if concurrency <= 0 {
		return fmt.Errorf("%s concurrency must be positive", name)
	}
	if concurrency > 100 {
		return fmt.Errorf("%s concurrency too high (max 100)", name)
	}
	return nil
}

// ValidateRetries validates retry count
func ValidateRetries(retries int, name string) error {
	if retries < 0 {
		return fmt.Errorf("%s retries cannot be negative", name)
	}
	if retries > 10 {
		return fmt.Errorf("%s retries too high (max 10)", name)
	}
	return nil
}

// ValidateRetryDelay validates retry delay
func ValidateRetryDelay(delayMs int, name string) error {
	if delayMs < 0 {
		return fmt.Errorf("%s retry delay cannot be negative", name)
	}
	if delayMs > 60000 {
		return fmt.Errorf("%s retry delay too high (max 60 seconds)", name)
	}
	return nil
}

// ValidateAPIKey validates API key format
func ValidateAPIKey(apiKey string, keyType string) error {
	if apiKey == "" {
		return fmt.Errorf("%s API key is required", keyType)
	}
	
	switch keyType {
	case "OpenAI":
		if !strings.HasPrefix(apiKey, "sk-") {
			return fmt.Errorf("invalid OpenAI API key format: must start with 'sk-'")
		}
		if len(apiKey) < 20 {
			return fmt.Errorf("invalid OpenAI API key format: too short")
		}
	case "Gemini":
		if !strings.HasPrefix(apiKey, "AIza") {
			return fmt.Errorf("invalid Gemini API key format: must start with 'AIza'")
		}
		if len(apiKey) < 30 {
			return fmt.Errorf("invalid Gemini API key format: too short")
		}
	case "ElevenLabs":
		if len(apiKey) < 32 {
			return fmt.Errorf("invalid ElevenLabs API key format: too short")
		}
	}
	
	return nil
}

// ValidateURL validates URL format
func ValidateURL(url string, name string) error {
	if url == "" {
		return fmt.Errorf("%s URL is required", name)
	}
	
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("%s URL must start with http:// or https://", name)
	}
	
	return nil
}

// ValidatePort validates port number
func ValidatePort(port string, name string) error {
	if port == "" {
		return fmt.Errorf("%s port is required", name)
	}
	
	// Simple validation - could be enhanced
	if len(port) > 5 {
		return fmt.Errorf("%s port invalid", name)
	}
	
	return nil
}

// ValidateProviderConfig validates common provider configuration
func ValidateProviderConfig(timeout time.Duration, concurrency int, retries int, retryDelayMs int, providerName string) error {
	if err := ValidateTimeout(timeout, providerName); err != nil {
		return err
	}
	
	if err := ValidateConcurrency(concurrency, providerName); err != nil {
		return err
	}
	
	if err := ValidateRetries(retries, providerName); err != nil {
		return err
	}
	
	if err := ValidateRetryDelay(retryDelayMs, providerName); err != nil {
		return err
	}
	
	return nil
}