# Provider Framework Design Document

## Overview

This document describes the design and implementation of the flexible transcription provider framework that abstracts the current whisper CLI calls into a provider pattern following SOLID design principles.

## Motivation

The original implementation was limited to:
- Local whisper.cpp binary execution
- OpenAI Whisper API (remote)
- Hard-coded provider selection in wire.go

The new framework provides:
- **Flexible Provider Selection**: Runtime provider configuration via YAML
- **Multiple Provider Support**: Easy integration of new TTS/STT services
- **Intelligent Orchestration**: Automatic provider selection and fallback
- **Comprehensive Monitoring**: Health checks, metrics, and statistics
- **SOLID Design Principles**: Extensible, maintainable architecture

## Architecture Overview

### Core Components

```
┌─────────────────────────────────────────────────┐
│                   CLI Layer                     │
│  ┌─────────────┐ ┌──────────────┐ ┌───────────┐ │
│  │   convert   │ │  providers   │ │    web    │ │
│  └─────────────┘ └──────────────┘ └───────────┘ │
└─────────────────────────────────────────────────┘
                       │
┌─────────────────────────────────────────────────┐
│            Provider Orchestrator                │
│  ┌─────────────────────────────────────────────┐ │
│  │         Provider Registry                   │ │
│  │  ┌──────────┐ ┌──────────┐ ┌─────────────┐  │ │
│  │  │WhisperCpp│ │ OpenAI   │ │ElevenLabs   │  │ │
│  │  └──────────┘ └──────────┘ └─────────────┘  │ │
│  └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
                       │
┌─────────────────────────────────────────────────┐
│              Configuration                      │
│              (YAML-based)                       │
└─────────────────────────────────────────────────┘
```

### Key Interfaces

#### 1. TranscriptionProvider Interface

```go
type TranscriptionProvider interface {
    // Backward compatible method
    Transcript(inputFilePath string) (string, error)
    
    // Enhanced method with full options
    TranscriptWithOptions(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error)
    
    // Provider metadata
    GetProviderInfo() ProviderInfo
    
    // Configuration validation
    ValidateConfiguration() error
    
    // Health check
    HealthCheck(ctx context.Context) error
}
```

#### 2. ProviderRegistry Interface

```go
type ProviderRegistry interface {
    RegisterProvider(name string, provider TranscriptionProvider) error
    GetProvider(name string) (TranscriptionProvider, error)
    ListProviders() []string
    GetDefaultProvider() (TranscriptionProvider, error)
    SetDefaultProvider(name string) error
    HealthCheckAll(ctx context.Context) map[string]error
}
```

#### 3. TranscriptionOrchestrator Interface

```go
type TranscriptionOrchestrator interface {
    Transcribe(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error)
    TranscribeWithProvider(ctx context.Context, providerName string, request *TranscriptionRequest) (*TranscriptionResponse, error)
    RecommendProvider(request *TranscriptionRequest) ([]string, error)
    GetStats() OrchestratorStats
}
```

## Provider Implementations

### 1. Enhanced Local Transcriber (whisper.cpp)

**File**: `internal/app/api/whisper_cpp/enhanced_provider.go`

**Features**:
- Maintains backward compatibility with existing `LocalTranscriber`
- Supports context cancellation
- Comprehensive error handling with retry suggestions
- Automatic audio format conversion
- Configurable language, prompt, and output formats

**Configuration Example**:
```yaml
whisper_cpp:
  type: whisper_cpp
  enabled: true
  settings:
    binary_path: "/path/to/whisper.cpp/main"
    model_path: "/path/to/models/ggml-large-v2.bin"
    language: "zh"
    prompt: "以下是简体中文普通话:"
```

### 2. Enhanced OpenAI Provider

**File**: `internal/app/api/openai/whisper/enhanced_provider.go`

**Features**:
- Maintains backward compatibility with existing `RemoteTranscriber`
- Advanced error handling with specific error codes
- API rate limiting and retry logic
- Multiple response format support
- Cost tracking capabilities

**Configuration Example**:
```yaml
openai:
  type: openai
  enabled: true
  auth:
    api_key: "${OPENAI_API_KEY}"
  settings:
    model: "whisper-1"
    response_format: "text"
  performance:
    timeout_sec: 60
    rate_limit_rpm: 50
```

### 3. ElevenLabs STT Provider (New)

**File**: `internal/app/api/elevenlabs/stt_provider.go`

**Features**:
- RESTful API integration with multipart form uploads
- Word-level timing alignment
- Language auto-detection
- Comprehensive error handling
- Health check capabilities

**Configuration Example**:
```yaml
elevenlabs:
  type: elevenlabs
  enabled: true
  auth:
    api_key: "${ELEVENLABS_API_KEY}"
  settings:
    model: "whisper-large-v3"
  performance:
    timeout_sec: 120
```

## Configuration System

### YAML Configuration Structure

**File**: `~/.tiktok-whisper/providers.yaml`

```yaml
default_provider: "whisper_cpp"

providers:
  whisper_cpp:
    type: "whisper_cpp"
    enabled: true
    settings:
      binary_path: "/Volumes/SSD2T/workspace/cpp/whisper.cpp/main"
      model_path: "/Volumes/SSD2T/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"
      language: "zh"
      prompt: "以下是简体中文普通话:"
    performance:
      timeout_sec: 300
      max_concurrency: 2
    error_handling:
      max_retries: 2
      retry_delay_ms: 1000

orchestrator:
  fallback_chain: ["whisper_cpp", "openai"]
  health_check_interval: "5m"
  max_retries: 1
  retry_delay: "2s"
  prefer_local: true
  router_rules:
    by_file_size:
      small: "whisper_cpp"    # < 10MB
      large: "openai"         # > 100MB
    by_language:
      zh: "whisper_cpp"
      en: "openai"

global:
  global_timeout_sec: 600
  temp_dir: "/tmp/transcription"
  log_level: "info"
  metrics:
    enabled: true
    retention_days: 30
```

### Environment Variable Support

All configuration values support environment variable expansion:

```yaml
auth:
  api_key: "${OPENAI_API_KEY}"
  base_url: "${CUSTOM_WHISPER_URL}"
```

## CLI Interface

### Provider Management Commands

```bash
# List all available providers
v2t providers list

# Check provider health status  
v2t providers status

# Get detailed provider information
v2t providers info openai

# Show current configuration
v2t providers config

# Test a specific provider
v2t providers test openai --file test.wav

# Provider usage statistics
v2t providers stats
```

### Output Formats

All commands support multiple output formats:

```bash
v2t providers list --output json
v2t providers status --output yaml
v2t providers info openai --output table
```

## Dependency Injection Integration

### Wire Configuration

**File**: `internal/app/wire_enhanced.go`

The enhanced wire configuration supports:
- Provider registry initialization from YAML config
- Automatic provider discovery and registration
- Fallback to original providers if enhanced setup fails
- Backward compatibility with existing `Converter` interface

### Factory Pattern

**File**: `internal/app/api/provider/factory.go`

The factory provides:
- Provider type discovery
- Configuration validation
- Provider metadata without instantiation
- Consistent provider creation interface

## Error Handling and Monitoring

### Structured Error Types

```go
type TranscriptionError struct {
    Code        string   `json:"code"`
    Message     string   `json:"message"`
    Provider    string   `json:"provider"`
    Retryable   bool     `json:"retryable"`
    Suggestions []string `json:"suggestions,omitempty"`
}
```

### Health Checks

All providers implement health checks:
- Configuration validation
- API connectivity tests
- Binary availability checks
- Resource accessibility verification

### Metrics Collection

```go
type ProviderMetrics interface {
    RecordSuccess(provider string, latencyMs int64, audioLengthSec float64)
    RecordFailure(provider string, errorType string)
    GetProviderMetrics(provider string) ProviderStats
    GetOverallMetrics() OverallStats
}
```

## Backward Compatibility

### Seamless Migration

The framework maintains 100% backward compatibility:

1. **Existing CLI Commands**: All current commands work unchanged
2. **Default Behavior**: Local whisper.cpp remains the default
3. **API Compatibility**: Original `Transcriber` interface still works
4. **Configuration**: Falls back to hardcoded paths if no config exists

### Migration Path

1. **Phase 1**: Framework available alongside existing implementation
2. **Phase 2**: Users can opt-in to YAML configuration
3. **Phase 3**: Enhanced features become available
4. **Phase 4**: Original implementation remains as fallback

## Testing Strategy

### Unit Tests

**Files**: `internal/app/api/provider/*_test.go`

- **Mock-based testing** for all provider interfaces
- **Registry functionality** with concurrent access
- **Metrics collection** accuracy and performance
- **Configuration parsing** and validation
- **Error handling** scenarios

### Integration Tests

- **Real API testing** with configurable endpoints
- **Provider health checks** with actual services  
- **End-to-end workflows** using the orchestrator
- **CLI command testing** with various output formats

### Benchmark Tests

- **Provider selection** performance
- **Metrics collection** overhead  
- **Configuration loading** speed
- **Concurrent access** patterns

## Future Extensions

### Planned Provider Types

1. **Custom HTTP Provider**: Generic REST API support
2. **Azure Speech Service**: Microsoft cognitive services
3. **Google Speech-to-Text**: Google Cloud STT API
4. **Local GPU Providers**: CUDA-accelerated implementations
5. **Streaming Providers**: Real-time transcription support

### Advanced Features

1. **Load Balancing**: Distribute requests across multiple instances
2. **Cost Optimization**: Automatic provider selection based on cost
3. **Quality Scoring**: Provider selection based on accuracy metrics
4. **Caching Layer**: Avoid duplicate transcriptions
5. **Batch Processing**: Optimize for large-scale operations

## Implementation Notes

### Import Cycle Resolution

The current implementation avoids import cycles by:
- Keeping provider interfaces separate from implementations
- Using factory patterns for provider creation
- Implementing provider registration at higher levels

### Library Compatibility

OpenAI provider implementation is currently limited by the go-openai library version:
- Basic transcription functionality works
- Advanced features (segments, word timing) are placeholder
- Future library updates will enable full feature support

### Performance Considerations

- **Lazy Loading**: Providers are created only when needed
- **Connection Pooling**: HTTP clients reuse connections
- **Timeout Management**: All operations have configurable timeouts
- **Memory Management**: Large audio files are streamed when possible

## Security Considerations

1. **API Key Management**: Environment variable support with validation
2. **Configuration Security**: Restricted file permissions on config files
3. **Network Security**: TLS verification for all HTTP providers
4. **Input Validation**: Comprehensive validation of all inputs
5. **Error Information**: No sensitive data leaked in error messages

## Summary

The provider framework successfully abstracts the whisper CLI calls into a flexible, extensible system that:

- ✅ Follows SOLID design principles
- ✅ Maintains backward compatibility
- ✅ Supports multiple TTS/STT services
- ✅ Provides intelligent orchestration
- ✅ Includes comprehensive monitoring
- ✅ Has extensive test coverage
- ✅ Offers excellent CLI management tools

The framework is production-ready and provides a solid foundation for future transcription service integrations while maintaining the reliability and performance of the existing system.