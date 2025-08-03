package config

import (
	"fmt"
	"os"
)

// NetworkConfig holds network-related configuration
type NetworkConfig struct {
	// Common hosts
	LocalHost      string
	RemoteHost     string
	RemoteUser     string
	
	// Ports
	HTTPPort       string
	SSHPort        string
	PostgresPort   string
	
	// URLs
	WhisperServerURL string
	DatabaseURL      string
}

// GetNetworkConfig returns network configuration from environment or defaults
func GetNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		LocalHost:      getEnvOrDefault("LOCAL_HOST", "localhost"),
		RemoteHost:     getEnvOrDefault("REMOTE_HOST", "mac-mini-m4-1.local"),
		RemoteUser:     getEnvOrDefault("REMOTE_USER", "daymade"),
		HTTPPort:       getEnvOrDefault("HTTP_PORT", DefaultHTTPPort),
		SSHPort:        getEnvOrDefault("SSH_PORT", DefaultSSHPort),
		PostgresPort:   getEnvOrDefault("POSTGRES_PORT", "5432"),
		WhisperServerURL: getEnvOrDefault("WHISPER_SERVER_URL", ""),
		DatabaseURL:     getEnvOrDefault("DATABASE_URL", ""),
	}
}

// GetWhisperServerURL constructs the whisper server URL
func (nc *NetworkConfig) GetWhisperServerURL() string {
	if nc.WhisperServerURL != "" {
		return nc.WhisperServerURL
	}
	return fmt.Sprintf("http://%s:%s", nc.LocalHost, nc.HTTPPort)
}

// GetSSHHost constructs the SSH host string
func (nc *NetworkConfig) GetSSHHost() string {
	if nc.RemoteUser != "" {
		return fmt.Sprintf("%s@%s", nc.RemoteUser, nc.RemoteHost)
	}
	return nc.RemoteHost
}

// GetPostgresConnectionString constructs PostgreSQL connection string
func (nc *NetworkConfig) GetPostgresConnectionString() string {
	if nc.DatabaseURL != "" {
		return nc.DatabaseURL
	}
	
	host := getEnvOrDefault("DB_HOST", nc.LocalHost)
	port := getEnvOrDefault("DB_PORT", nc.PostgresPort)
	user := getEnvOrDefault("DB_USER", "postgres")
	password := getEnvOrDefault("DB_PASSWORD", "")
	dbname := getEnvOrDefault("DB_NAME", "postgres")
	
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}