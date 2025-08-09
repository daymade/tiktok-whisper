package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration values
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	External   ExternalConfig
	Storage    StorageConfig
	Redis      RedisConfig
	Auth       AuthConfig
	Providers  ProviderConfig
	Logging    LoggingConfig
	Security   SecurityConfig
	Features   FeatureConfig
	Performance PerformanceConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string
	Port         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type        string
	Path        string
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	SSLMode     string
	Pgvector    PgvectorConfig
}

// PgvectorConfig holds pgvector-specific configuration
type PgvectorConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// ExternalConfig holds external service configuration
type ExternalConfig struct {
	OpenAI      OpenAIConfig
	Gemini      GeminiConfig
	ElevenLabs  ElevenLabsConfig
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey    string
	OrgID     string
	BaseURL   string
	Whisper   OpenAIWhisperConfig
	Embedding OpenAIEmbeddingConfig
}

// OpenAIWhisperConfig holds OpenAI Whisper configuration
type OpenAIWhisperConfig struct {
	Enabled        bool
	Model          string
	ResponseFormat string
}

// OpenAIEmbeddingConfig holds OpenAI embedding configuration
type OpenAIEmbeddingConfig struct {
	Enabled bool
	Model   string
}

// GeminiConfig holds Google Gemini configuration
type GeminiConfig struct {
	APIKey    string
	Embedding GeminiEmbeddingConfig
}

// GeminiEmbeddingConfig holds Gemini embedding configuration
type GeminiEmbeddingConfig struct {
	Enabled bool
	Model   string
}

// ElevenLabsConfig holds ElevenLabs configuration
type ElevenLabsConfig struct {
	APIKey       string
	Enabled      bool
	Model        string
	Language     string
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	EnableMinIO  bool
	MinIO        MinIOConfig
	AWS          AWSConfig
	MaxUploadSize int64
}

// MinIOConfig holds MinIO configuration
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	Region    string
}

// AWSConfig holds AWS configuration
type AWSConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	S3Bucket        string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	URL      string
	PoolSize int
	Enabled  bool
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret            string
	JWTIssuer           string
	JWTAudience         string
	JWTExpiration       time.Duration
	JWTRefreshExpiration time.Duration
	APIKeyAuthEnabled   bool
	APIKeyHeader        string
}

// ProviderConfig holds provider framework configuration
type ProviderConfig struct {
	DefaultProvider string
	Timeout        time.Duration
	MaxConcurrency int
	FallbackChain  []string
	WhisperCpp     WhisperCppConfig
}

// WhisperCppConfig holds whisper.cpp configuration
type WhisperCppConfig struct {
	Enabled    bool
	BinaryPath string
	ModelPath  string
	Language   string
	Model      string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level                string
	Format               string
	EnableRequestLogging bool
	EnableMetrics        bool
	MetricsPort          string
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	CORSEnabled         bool
	CORSMaxAge          int
	CORSAllowCredentials bool
	EnableHTTPS         bool
	HTTPSCertFile       string
	HTTPSKeyFile        string
	RateLimitEnabled    bool
	RateLimitRPM        int
	RateLimitBurst      int
}

// FeatureConfig holds feature flag configuration
type FeatureConfig struct {
	EnableDualEmbeddings  bool
	EnableWebInterface    bool
	EnableBatchProcessing bool
	EnableAutoRetries     bool
	EnableHealthChecks    bool
	EnableSwaggerUI       bool
}

// PerformanceConfig holds performance tuning configuration
type PerformanceConfig struct {
	MaxConcurrentUploads      int
	MaxConcurrentTranscriptions int
	CacheTTL                  time.Duration
	WorkerPoolSize            int
}

// Load loads configuration from environment variables
func Load(envFile string) (*Config, error) {
	// Load .env file if it exists
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", envFile, err)
		}
	} else {
		godotenv.Load() // Try to load .env by default
	}

	config := &Config{
		Server:     loadServerConfig(),
		Database:   loadDatabaseConfig(),
		External:   loadExternalConfig(),
		Storage:    loadStorageConfig(),
		Redis:      loadRedisConfig(),
		Auth:       loadAuthConfig(),
		Providers:  loadProviderConfig(),
		Logging:    loadLoggingConfig(),
		Security:   loadSecurityConfig(),
		Features:   loadFeatureConfig(),
		Performance: loadPerformanceConfig(),
	}

	return config, nil
}

// loadServerConfig loads server configuration
func loadServerConfig() ServerConfig {
	return ServerConfig{
		Host:         getEnv("SERVER_HOST", "0.0.0.0"),
		Port:         getEnv("SERVER_PORT", "8081"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:  getDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
	}
}

// loadDatabaseConfig loads database configuration
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Type:     getEnv("DB_TYPE", "sqlite"),
		Path:     getEnv("DB_PATH", "./data/tiktok-whisper.db"),
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		Name:     getEnv("DB_NAME", "postgres"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		Pgvector: PgvectorConfig{
			Host:     getEnv("PGVECTOR_HOST", "localhost"),
			Port:     getEnv("PGVECTOR_PORT", "5432"),
			User:     getEnv("PGVECTOR_USER", "postgres"),
			Password: getEnv("PGVECTOR_PASSWORD", ""),
			DBName:   getEnv("PGVECTOR_DBNAME", "postgres"),
			SSLMode:  getEnv("PGVECTOR_SSL_MODE", "disable"),
		},
	}
}

// loadExternalConfig loads external service configuration
func loadExternalConfig() ExternalConfig {
	return ExternalConfig{
		OpenAI: OpenAIConfig{
			APIKey:  getEnv("OPENAI_API_KEY", ""),
			OrgID:   getEnv("OPENAI_ORG_ID", ""),
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			Whisper: OpenAIWhisperConfig{
				Enabled:        getBoolEnv("OPENAI_WHISPER_ENABLED", true),
				Model:          getEnv("OPENAI_WHISPER_MODEL", "whisper-1"),
				ResponseFormat: getEnv("OPENAI_WHISPER_RESPONSE_FORMAT", "text"),
			},
			Embedding: OpenAIEmbeddingConfig{
				Enabled: getBoolEnv("OPENAI_EMBEDDING_ENABLED", true),
				Model:   getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
			},
		},
		Gemini: GeminiConfig{
			APIKey: getEnv("GEMINI_API_KEY", ""),
			Embedding: GeminiEmbeddingConfig{
				Enabled: getBoolEnv("GEMINI_EMBEDDING_ENABLED", true),
				Model:   getEnv("GEMINI_EMBEDDING_MODEL", "text-embedding-004"),
			},
		},
		ElevenLabs: ElevenLabsConfig{
			APIKey:   getEnv("ELEVENLABS_API_KEY", ""),
			Enabled:  getBoolEnv("ELEVENLABS_ENABLED", false),
			Model:    getEnv("ELEVENLABS_MODEL", "eleven_multilingual_v2"),
			Language: getEnv("ELEVENLABS_LANGUAGE", "zh"),
		},
	}
}

// loadStorageConfig loads storage configuration
func loadStorageConfig() StorageConfig {
	return StorageConfig{
		EnableMinIO:  getBoolEnv("ENABLE_MINIO", false),
		MaxUploadSize: getInt64Env("MAX_UPLOAD_SIZE", 100*1024*1024), // 100MB
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			Bucket:    getEnv("MINIO_BUCKET", "tiktok-whisper"),
			UseSSL:    getBoolEnv("MINIO_USE_SSL", false),
			Region:    getEnv("MINIO_REGION", "us-east-1"),
		},
		AWS: AWSConfig{
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Region:          getEnv("AWS_REGION", "us-east-1"),
			S3Bucket:        getEnv("AWS_S3_BUCKET", ""),
		},
	}
}

// loadRedisConfig loads Redis configuration
func loadRedisConfig() RedisConfig {
	return RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getIntEnv("REDIS_DB", 0),
		URL:      getEnv("REDIS_URL", ""),
		PoolSize: getIntEnv("REDIS_POOL_SIZE", 10),
		Enabled:  getBoolEnv("REDIS_ENABLED", true),
	}
}

// loadAuthConfig loads authentication configuration
func loadAuthConfig() AuthConfig {
	return AuthConfig{
		JWTSecret:            getEnv("JWT_SECRET", "your-secret-key"),
		JWTIssuer:           getEnv("JWT_ISSUER", "tiktok-whisper"),
		JWTAudience:         getEnv("JWT_AUDIENCE", "tiktok-whisper-api"),
		JWTExpiration:       getDuration("JWT_EXPIRATION", 24*time.Hour),
		JWTRefreshExpiration: getDuration("JWT_REFRESH_EXPIRATION", 168*time.Hour),
		APIKeyAuthEnabled:   getBoolEnv("API_KEY_AUTH_ENABLED", false),
		APIKeyHeader:        getEnv("API_KEY_HEADER", "X-API-Key"),
	}
}

// loadProviderConfig loads provider configuration
func loadProviderConfig() ProviderConfig {
	return ProviderConfig{
		DefaultProvider: getEnv("DEFAULT_PROVIDER", "whisper_cpp"),
		Timeout:        getDuration("PROVIDER_TIMEOUT", 300*time.Second),
		MaxConcurrency: getIntEnv("PROVIDER_MAX_CONCURRENCY", 2),
		FallbackChain:  getStringSlice("PROVIDER_FALLBACK_CHAIN", []string{"whisper_cpp", "openai", "elevenlabs"}),
		WhisperCpp: WhisperCppConfig{
			Enabled:    getBoolEnv("WHISPER_CPP_ENABLED", true),
			BinaryPath: getEnv("WHISPER_CPP_BINARY_PATH", ""),
			ModelPath:  getEnv("WHISPER_CPP_MODEL_PATH", ""),
			Language:   getEnv("WHISPER_CPP_LANGUAGE", "zh"),
			Model:      getEnv("WHISPER_CPP_MODEL", "large"),
		},
	}
}

// loadLoggingConfig loads logging configuration
func loadLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:                getEnv("LOG_LEVEL", "info"),
		Format:               getEnv("LOG_FORMAT", "text"),
		EnableRequestLogging: getBoolEnv("ENABLE_REQUEST_LOGGING", true),
		EnableMetrics:        getBoolEnv("ENABLE_METRICS", true),
		MetricsPort:          getEnv("METRICS_PORT", "9090"),
	}
}

// loadSecurityConfig loads security configuration
func loadSecurityConfig() SecurityConfig {
	return SecurityConfig{
		CORSEnabled:         getBoolEnv("CORS_ENABLED", true),
		CORSMaxAge:          getIntEnv("CORS_MAX_AGE", 86400),
		CORSAllowCredentials: getBoolEnv("CORS_ALLOW_CREDENTIALS", true),
		EnableHTTPS:         getBoolEnv("ENABLE_HTTPS", false),
		HTTPSCertFile:       getEnv("HTTPS_CERT_FILE", ""),
		HTTPSKeyFile:        getEnv("HTTPS_KEY_FILE", ""),
		RateLimitEnabled:    getBoolEnv("RATE_LIMIT_ENABLED", true),
		RateLimitRPM:        getIntEnv("RATE_LIMIT_REQUESTS_PER_MINUTE", 100),
		RateLimitBurst:      getIntEnv("RATE_LIMIT_BURST", 10),
	}
}

// loadFeatureConfig loads feature flag configuration
func loadFeatureConfig() FeatureConfig {
	return FeatureConfig{
		EnableDualEmbeddings:  getBoolEnv("ENABLE_DUAL_EMBEDDINGS", true),
		EnableWebInterface:    getBoolEnv("ENABLE_WEB_INTERFACE", true),
		EnableBatchProcessing: getBoolEnv("ENABLE_BATCH_PROCESSING", true),
		EnableAutoRetries:     getBoolEnv("ENABLE_AUTO_RETRIES", true),
		EnableHealthChecks:    getBoolEnv("ENABLE_HEALTH_CHECKS", true),
		EnableSwaggerUI:       getBoolEnv("ENABLE_SWAGGER_UI", true),
	}
}

// loadPerformanceConfig loads performance configuration
func loadPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		MaxConcurrentUploads:      getIntEnv("MAX_CONCURRENT_UPLOADS", 5),
		MaxConcurrentTranscriptions: getIntEnv("MAX_CONCURRENT_TRANSCRIPTIONS", 10),
		CacheTTL:                  getDuration("CACHE_TTL", 300*time.Second),
		WorkerPoolSize:            getIntEnv("WORKER_POOL_SIZE", 10),
	}
}

// Helper functions for environment variable parsing

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// getIntEnv gets an integer environment variable
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getInt64Env gets an int64 environment variable
func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

// getDuration gets a duration environment variable
func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

// getStringSlice gets a string slice environment variable
func getStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}