package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// APIKeys holds all API keys loaded from environment
type APIKeys struct {
	OpenAI string
	Gemini string
}

// LoadEnv loads environment variables from .env file if it exists
// This function implements fail-fast principle - it will exit if critical configuration is missing
func LoadEnv() error {
	// Try to load .env file from current directory or project root
	envPaths := []string{
		".env",
		".env.local",
		"../.env",
		"../../.env",
	}

	// Look for .env file, but don't fail if not found (environment variables might be set system-wide)
	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err != nil {
				return fmt.Errorf("error loading %s file: %w", envPath, err)
			}
			fmt.Printf("✅ Loaded environment variables from %s\n", envPath)
			break
		}
	}

	return nil
}

// GetAPIKeys retrieves and validates API keys from environment variables
// Implements fail-fast: returns error immediately if required keys are missing
func GetAPIKeys() (*APIKeys, error) {
	apiKeys := &APIKeys{
		OpenAI: strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		Gemini: strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
	}

	// Validate API keys format (basic checks)
	if apiKeys.OpenAI != "" {
		if !strings.HasPrefix(apiKeys.OpenAI, "sk-") {
			return nil, fmt.Errorf("invalid OPENAI_API_KEY format: must start with 'sk-'")
		}
		if len(apiKeys.OpenAI) < 20 {
			return nil, fmt.Errorf("invalid OPENAI_API_KEY format: too short")
		}
	}

	if apiKeys.Gemini != "" {
		if !strings.HasPrefix(apiKeys.Gemini, "AIza") {
			return nil, fmt.Errorf("invalid GEMINI_API_KEY format: must start with 'AIza'")
		}
		if len(apiKeys.Gemini) < 30 {
			return nil, fmt.Errorf("invalid GEMINI_API_KEY format: too short")
		}
	}

	return apiKeys, nil
}

// ValidateAPIKeys checks if at least one API key is available
// Returns helpful information about available keys without failing
func ValidateAPIKeys(apiKeys *APIKeys) error {
	var availableKeys []string
	if apiKeys.OpenAI != "" {
		availableKeys = append(availableKeys, "OpenAI")
	}
	if apiKeys.Gemini != "" {
		availableKeys = append(availableKeys, "Gemini")
	}

	if len(availableKeys) > 0 {
		fmt.Printf("✅ API keys available: %s\n", strings.Join(availableKeys, ", "))
	} else {
		fmt.Printf("ℹ️  No API keys configured (embedding features will be unavailable)\n")
	}
	
	return nil
}

// RequireAPIKeys validates that at least one API key is available (for embedding operations)
// This implements fail-fast behavior for operations that specifically need API keys
func RequireAPIKeys(apiKeys *APIKeys) error {
	if apiKeys.OpenAI == "" && apiKeys.Gemini == "" {
		return fmt.Errorf("embedding operations require at least one API key - please set OPENAI_API_KEY or GEMINI_API_KEY in environment or .env file")
	}
	return nil
}

// GetProjectRoot finds the project root directory by looking for go.mod
func GetProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find project root (go.mod not found)")
}

// InitializeConfig loads environment and validates configuration
// This is the main entry point for configuration loading
func InitializeConfig() (*APIKeys, error) {
	// Load .env file if available
	if err := LoadEnv(); err != nil {
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	// Get and validate API keys
	apiKeys, err := GetAPIKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	// Show available keys without failing
	ValidateAPIKeys(apiKeys)

	return apiKeys, nil
}