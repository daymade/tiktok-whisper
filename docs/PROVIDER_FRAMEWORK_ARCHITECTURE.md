# Provider Framework Architecture Documentation

## Overview

The tiktok-whisper project features a sophisticated, extensible transcription provider framework that abstracts multiple transcription services into a unified, configurable system. This framework follows SOLID design principles and provides a clean architecture for adding new providers without modifying existing code.

## Core Architecture

### System Design Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        tiktok-whisper CLI                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   Convert Cmd   │  │   Download Cmd  │  │    Other Cmds   │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│           │                     │                     │         │
│           └─────────────────────┼─────────────────────┘         │
│                                 │                               │
│          ┌─────────────────────────────────────┐               │
│          │       Provider Framework Core       │               │
│          └─────────────────────────────────────┘               │
│                                 │                               │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                  Provider Registry                          │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────┐ │  │
│  │  │whisper_cpp  │ │   openai    │ │ elevenlabs  │ │  ...  │ │  │
│  │  │   Provider  │ │  Provider   │ │  Provider   │ │       │ │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘ └───────┘ │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                 │                               │
│          ┌─────────────────────────────────────┐               │
│          │           Orchestrator             │               │
│          │  - Fallback Chain                   │               │
│          │  - Load Balancing                   │               │
│          │  - Provider Selection               │               │
│          └─────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │   Provider Instances   │
                    │  (whisper.cpp, OpenAI, │
                    │   ElevenLabs, SSH,    │
                    │   whisper-server,     │
                    │   etc.)               │
                    └───────────────────────┘
```

### Key Design Principles

1. **Single Responsibility**: Each provider handles only its specific transcription service
2. **Open/Closed Principle**: New providers can be added without modifying existing code
3. **Dependency Inversion**: High-level modules don't depend on low-level implementation details
4. **Interface Segregation**: Providers implement only the interfaces they need
5. **Factory Pattern**: Provider creation is abstracted through factory functions

## Component Architecture

### 1. Core Interfaces (`internal/app/api/provider/`)

#### Provider Interface
```go
type Provider interface {
    // Core transcription functionality
    Transcript(ctx context.Context, audioPath string) (*TranscriptionResult, error)
    
    // Health check
    HealthCheck(ctx context.Context) error
    
    // Provider metadata
    GetType() string
    GetName() string
    GetVersion() string
    IsEnabled() bool
    
    // Configuration
    GetConfig() ProviderConfig
    ValidateConfig() error
}
```

#### ProviderFactory Interface
```go
type ProviderFactory interface {
    Create(config ProviderConfig) (Provider, error)
    GetConfigTemplate() ProviderConfig
    ValidateConfig(config ProviderConfig) error
}
```

### 2. Registry System (`internal/app/api/provider/registry.go`)

The registry maintains a central collection of all available providers:

```go
type ProviderRegistry struct {
    factories map[string]ProviderFactory
    providers map[string]Provider
    mutex     sync.RWMutex
}
```

**Key Features:**
- Thread-safe provider registration and retrieval
- Factory-based provider instantiation
- Configuration validation
- Provider lifecycle management

### 3. Orchestrator (`internal/app/api/provider/orchestrator.go`)

The orchestrator handles intelligent provider selection and execution:

```go
type Orchestrator struct {
    registry      *ProviderRegistry
    config        *OrchestratorConfig
    metrics       *MetricsCollector
    loadBalancer  LoadBalancer
    fallbackChain []string
}
```

**Responsibilities:**
- Execute fallback chains
- Implement load balancing strategies
- Collect performance metrics
- Handle provider failures gracefully

### 4. Configuration System (`internal/app/api/provider/config.go`)

YAML-based configuration with environment variable support:

```yaml
providers:
  whisper_cpp:
    type: whisper_cpp
    enabled: true
    settings:
      binary_path: "/path/to/whisper"
      model_path: "/path/to/model"
    performance:
      timeout_sec: 300
      max_concurrency: 2

orchestrator:
  fallback_chain: ["whisper_cpp", "openai"]
  load_balancing:
    strategy: round_robin
```

## Provider Implementations

### 1. Local whisper.cpp Provider

**Location**: `internal/app/api/whisper_cpp/`

**Architecture**:
```
┌─────────────────┐
│  WhisperCpp     │
│    Provider     │
├─────────────────┤
│ - binaryPath    │
│ - modelPath     │
│ - language      │
│ - prompt        │
└─────────────────┘
         │
    ┌────▼────┐
    │ Command │
    │ Executor│
    └────┬────┘
         │
┌────────▼─────────┐
│ Local Whisper.cpp│
│  Binary Process  │
└──────────────────┘
```

**Key Features**:
- Command-line interface execution
- Model loading optimization
- Language-specific prompting
- Process management and cleanup

### 2. OpenAI API Provider

**Location**: `internal/app/api/openai/whisper/`

**Architecture**:
```
┌─────────────────┐
│   OpenAI        │
│   Provider      │
├─────────────────┤
│ - API Key       │
│ - Model         │
│ - Format        │
└─────────────────┘
         │
    ┌────▼────┐
    │ HTTP    │
    │ Client  │
    └────┬────┘
         │
┌────────▼─────────┐
│  OpenAI Whisper  │
│      API         │
└──────────────────┘
```

**Key Features**:
- REST API integration
- Rate limiting handling
- Response format options
- Authentication management

### 3. ElevenLabs Provider

**Location**: `internal/app/api/elevenlabs/`

**Architecture**:
```
┌─────────────────┐
│  ElevenLabs     │
│    Provider     │
├─────────────────┤
│ - API Key       │
│ - Model ID      │
│ - Language      │
└─────────────────┘
         │
    ┌────▼────┐
    │ HTTP    │
    │ Client  │
    └────┬────┘
         │
┌────────▼─────────┐
│ ElevenLabs STT   │
│      API         │
└──────────────────┘
```

**Key Features**:
- WebSocket support for real-time transcription
- Multiple language models
- Word-level timestamps
- Custom vocabulary support

### 4. SSH Whisper Provider

**Location**: `internal/app/api/ssh_whisper/`

**Architecture**:
```
┌─────────────────┐
│   SSH Whisper   │
│    Provider     │
├─────────────────┤
│ - SSH Config    │
│ - Remote Paths  │
│ - File Transfer │
└─────────────────┘
         │
    ┌────▼────┐
    │ SSH     │
    │ Client  │
    └────┬────┘
         │
┌────────▼─────────┐    ┌─────────────────┐
│   SSH Session    │───▶│ Remote Whisper  │
│  & File Transfer │    │      Server     │
└──────────────────┘    └─────────────────┘
```

**Key Features**:
- Secure SSH connection management
- Automatic file transfer via SCP
- Remote command execution
- Session pooling for efficiency

### 5. Whisper-Server HTTP Provider

**Location**: `internal/app/api/whisper_server/`

**Architecture**:
```
┌─────────────────┐
│ Whisper-Server  │
│    Provider     │
├─────────────────┤
│ - Base URL      │
│ - Timeout       │
│ - Format        │
│ - Headers       │
└─────────────────┘
         │
    ┌────▼────┐
    │ HTTP    │
    │ Client  │
    └────┬────┘
         │
┌────────▼─────────┐    ┌─────────────────┐
│   HTTP Request   │───▶│ Whisper-Server  │
│  & Multipart     │    │   Middleware    │
└──────────────────┘    └─────────────────┘
                                │
                       ┌────────▼─────────┐
                       │  Whisper.cpp    │
                       │   Backend       │
                       └─────────────────┘
```

**Key Features**:
- HTTP/HTTPS communication
- Multipart form data upload
- Configurable endpoints
- Response format support (JSON, SRT, VTT, verbose_json)

## Data Flow Architecture

### 1. Request Flow

```
User Command → CLI Parser → Convert Command → Provider Framework
    ↓
Provider Selection (Orchestrator)
    ↓
Provider Instantiation (Factory)
    ↓
Health Check (Optional)
    ↓
Transcription Request
    ↓
Result Processing & Formatting
    ↓
Return to User
```

### 2. Error Handling Flow

```
Provider Failure
    ↓
Error Detection
    ↓
Fallback Chain Check
    ↓
Next Provider Selection
    ↓
Retry with New Provider
    ↓
Success or Final Error
```

### 3. Configuration Loading Flow

```
Application Start
    ↓
Load providers.yaml
    ↓
Expand Environment Variables
    ↓
Validate Configuration
    ↓
Register Providers
    ↓
Initialize Orchestrator
    ↓
Ready for Requests
```

## Extensibility Architecture

### Adding New Providers

1. **Implement Provider Interface**
```go
type MyProvider struct {
    config MyProviderConfig
}

func (p *MyProvider) Transcript(ctx context.Context, audioPath string) (*TranscriptionResult, error) {
    // Implementation
}
```

2. **Create Factory**
```go
type MyProviderFactory struct{}

func (f *MyProviderFactory) Create(config ProviderConfig) (Provider, error) {
    return &MyProvider{config: config.(*MyProviderConfig)}, nil
}
```

3. **Register Provider**
```go
registry.Register("my_provider", &MyProviderFactory{})
```

### Plugin Architecture (Future Enhancement)

The framework is designed to support dynamic plugin loading:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Main App      │    │   Plugin 1      │    │   Plugin 2      │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ Provider    │ │    │ │ Custom      │ │    │ │ Custom      │ │
│ │ Registry    │ │◄───┤ │ Provider    │ │◄───┤ │ Provider    │ │
│ └─────────────┘ │    │ │ Factory     │ │    │ │ Factory     │ │
│                 │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Performance Architecture

### 1. Concurrency Model

```
┌─────────────────┐
│   Request Pool  │
├─────────────────┤
│ Request 1 ──────┼──▶ Provider 1
│ Request 2 ──────┼──▶ Provider 2
│ Request 3 ──────┼──▶ Provider 3
│    ...          │
└─────────────────┘
```

### 2. Resource Management

- **Connection Pooling**: Reuse HTTP connections
- **Process Management**: Clean up subprocesses
- **Memory Management**: Stream large audio files
- **Rate Limiting**: Respect API limits

### 3. Caching Strategy (Planned)

```
┌─────────────────┐    ┌─────────────────┐
│   Request       │    │   Cache Layer   │
│                 │    │                 │
│ Audio File ─────┼───▶│ Hash Generation │
│ Provider  ──────┼───▶│ Cache Lookup    │
│ Settings  ──────┼───▶│ Result Storage │
└─────────────────┘    └─────────────────┘
```

## Security Architecture

### 1. Authentication

```
┌─────────────────┐    ┌─────────────────┐
│   Provider      │    │   Auth Manager  │
│                 │    │                 │
│ API Keys ───────┼───▶│ Secure Storage  │
│ Tokens ──────────┼───▶│ Key Rotation    │
│ Certificates ────┼───▶│ Validation      │
└─────────────────┘    └─────────────────┘
```

### 2. Data Protection

- **In Transit**: HTTPS/TLS encryption
- **At Rest**: Optional file encryption
- **Audit Logging**: Request tracking
- **Access Control**: Provider-specific permissions

## Monitoring & Observability

### 1. Metrics Collection

```
┌─────────────────┐    ┌─────────────────┐
│   Provider      │    │   Metrics       │
│                 │    │                 │
│ Success Rate ───┼───▶│ Prometheus      │
│ Response Time ──┼───▶│ Histograms      │
│ Error Count ────┼───▶│ Counters        │
│ Queue Size ─────┼───▶│ Gauges          │
└─────────────────┘    └─────────────────┘
```

### 2. Logging Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   Structured     │    │   Log           │
│      Logging     │    │   Aggregation   │
│                 │    │                 │
│ Request ID ─────┼───▶│ ELK Stack       │
│ Provider ───────┼───▶│ Splunk          │
│ Duration ───────┼───▶│ CloudWatch      │
│ Error ──────────┼───▶│ Custom          │
└─────────────────┘    └─────────────────┘
```

## Testing Architecture

### 1. Unit Testing

```
┌─────────────────┐    ┌─────────────────┐
│   Test Cases    │    │   Mock          │
│                 │    │                 │
│ Provider Logic  │◄───┤│ Providers       │
│ Error Handling  │◄───┤│ HTTP Servers    │
│ Config Parsing  │◄───┤│ SSH Sessions    │
└─────────────────┘    └─────────────────┘
```

### 2. Integration Testing

```
┌─────────────────┐    ┌─────────────────┐
│   Test          │    │   Docker        │
│   Environment   │    │   Containers    │
│                 │    │                 │
│ Test Runner ────┼───▶│ Whisper.cpp     │
│ Provider Tests ─┼───▶│ API Mocks       │
│ Data Cleanup ───┼───▶│ SSH Servers     │
└─────────────────┘    └─────────────────┘
```

## Deployment Architecture

### 1. Single Machine Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│                     Single Host                                │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  CLI Tool   │  │ Provider    │  │   Local Whisper.cpp    │  │
│  │             │  │ Framework  │  │      or               │  │
│  └─────────────┘  └─────────────┘  │   Remote Services     │  │
│                                        └─────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 2. Distributed Deployment

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client        │    │   Orchestrator  │    │   Provider      │
│   Machines      │    │   Service       │    │   Servers       │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ tiktok-     │ │    │ │ Load        │ │    │ │ Whisper.cpp │ │
│ │ whisper     │ │◄───┤ │ Balancer    │ │◄───┤ │ Servers     │ │
│ │ CLI         │ │    │ │ Health      │ │    │ │ OpenAI      │ │
│ └─────────────┘ │    │ │ Check       │ │    │ │ ElevenLabs  │ │
│                 │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Configuration Management

### Environment-Based Configuration

```yaml
# Development
providers:
  whisper_cpp:
    enabled: true
    settings:
      binary_path: "./debug/whisper"

# Production
providers:
  whisper_server:
    enabled: true
    settings:
      base_url: "${WHISPER_SERVER_URL}"
      timeout: 300
```

### Dynamic Configuration Updates

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Config        │    │   Watcher        │    │   Provider      │
│   Changes       │    │   Service        │    │   Registry      │
│                 │    │                 │    │                 │
│ YAML Update ────┼───▶│ File Watcher    │◄───┤│ Hot Reload      │
│ Signal ─────────┼───▶│ Config Parser   │◄───┤│ No Downtime    │
│ API Call ───────┼───▶│ Validator       │◄───┤│ Graceful       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Future Enhancements

### 1. Streaming Transcription

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Audio Stream  │    │   Stream        │    │   Real-time     │
│                 │    │   Processor     │    │   Results       │
│                 │    │                 │    │                 │
│ Microphone ─────┼───▶│ Buffer         │◄───┤│ WebSocket      │
│ File Stream ────┼───▶│ Chunking       │◄───┤│ SSE            │
│ Network ────────┼───▶│ Real-time      │◄───┤│ Push Updates   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 2. Machine Learning Optimization

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   ML             │    │   Provider      │    │   Optimized     │
│   Models         │    │   Selection     │    │   Routing       │
│                 │    │                 │    │                 │
│ Audio Features ──┼───▶│ Classification │◄───┤│ Best Provider  │
│ Content Type ────┼───▶│ Quality Score  │◄───┤│ Cost Analysis  │
│ Length Estimate ─┼───▶│ Complexity     │◄───┤│ Performance    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Conclusion

The provider framework architecture demonstrates a mature, production-ready system that:

1. **Scales Horizontally**: Easy to add new providers and distribute load
2. **Maintains Stability**: Robust error handling and fallback mechanisms
3. **Provides Flexibility**: Configurable routing and load balancing
4. **Ensures Security**: Authentication, encryption, and audit capabilities
5. **Supports Growth**: Plugin architecture and streaming capabilities

This design enables tiktok-whisper to support virtually any transcription service while maintaining a consistent, reliable user experience. The architecture prioritizes extensibility, performance, and maintainability, making it suitable for both small-scale and enterprise deployments.