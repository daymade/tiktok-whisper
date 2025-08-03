# Whisper-Server HTTP Provider Documentation

## Overview

The Whisper-Server HTTP Provider enables transcription using a remote whisper-server HTTP API, supporting LAN-based whisper.cpp instances over HTTP/HTTPS. This provider is ideal for centralized transcription services, GPU-accelerated servers, and multi-user environments.

## Architecture

### Data Flow Diagram

```
┌─────────────────┐    HTTP Request    ┌─────────────────┐    Whisper.cpp    ┌─────────────────┐
│   tiktok-whisper│ ──────────────────>│  Whisper-Server │ ─────────────────>│   Transcription │
│                 │                    │                 │                    │     Service      │
│                 │                    │                 │                    │                 │
│                 │                    │                 │                    │                 │
│                 │                    │                 │                    │                 │
│                 │                    │                 │                    │                 │
│                 │                    │                 │                    │                 │
└─────────────────┘ <─────────────────└─────────────────┘ <─────────────────└─────────────────┘
     Client            HTTP Response          Middleware              Backend
```

### Provider Components

1. **HTTP Client Layer**
   - Multipart form data handling
   - Configurable timeouts and retries
   - Custom header support
   - Connection pooling

2. **Request Processing**
   - Audio file upload via multipart/form-data
   - Parameter validation and formatting
   - Language and model selection
   - Response format handling

3. **Response Parsing**
   - JSON parsing with metadata extraction
   - Text/SRT/VTT format support
   - Verbose JSON with segments and words
   - Error handling and status codes

## Installation and Setup

### Prerequisites

1. **Whisper-Server Instance**
   - Running whisper-server on accessible host
   - Appropriate whisper models downloaded
   - Network connectivity (LAN/WAN)

2. **Network Configuration**
   - Open firewall ports (default: 8080)
   - Proxy configuration (if required)
   - DNS resolution or static IP

### Starting Whisper-Server

#### On macOS (mac-mini-m4-1.local)

```bash
# SSH into the server
ssh daymade@mac-mini-m4-1.local

# Navigate to whisper.cpp directory
cd /Users/daymade/Workspace/cpp/whisper.cpp

# Start whisper-server
./build/bin/whisper-server \
    --host 0.0.0.0 \
    --port 8080 \
    --model models/ggml-base.en.bin \
    --public examples/server/public

# Or start in background
nohup ./build/bin/whisper-server \
    --host 0.0.0.0 \
    --port 8080 \
    --model models/ggml-base.en.bin \
    > whisper-server.log 2>&1 &
```

#### With GPU Acceleration

```bash
./build/bin/whisper-server \
    --host 0.0.0.0 \
    --port 8080 \
    --model models/ggml-large-v3.bin \
    --gpu-layers 99 \
    --n-threads 8
```

## Configuration

### Basic Configuration

```yaml
# providers.yaml
default_provider: "whisper_server"

providers:
  whisper_server:
    type: "whisper_server"
    enabled: true
    settings:
      base_url: "http://192.168.31.151:8080"
      timeout: 120
      language: "auto"
      response_format: "json"
```

### Advanced Configuration

```yaml
providers:
  whisper_server_optimized:
    type: "whisper_server"
    enabled: true
    settings:
      # Server Configuration
      base_url: "https://whisper-server.example.com"
      inference_path: "/inference"
      load_path: "/load"
      timeout: 300
      
      # Transcription Options
      language: "en"
      response_format: "verbose_json"
      temperature: 0.0
      translate: false
      no_timestamps: false
      word_threshold: 0.01
      max_length: 1000
      
      # Security
      custom_headers:
        Authorization: "Bearer ${WHISPER_SERVER_TOKEN}"
        X-Client-ID: "tiktok-whisper"
      
      # SSL/TLS
      insecure_skip_tls: false
      
    performance:
      timeout_sec: 300
      max_concurrency: 2
      
    error_handling:
      max_retries: 3
      retry_delay_ms: 2000
```

### LAN Deployment Configuration

```yaml
providers:
  whisper_server_lan:
    type: "whisper_server"
    enabled: true
    settings:
      base_url: "http://mac-mini-m4-1.local:8080"
      # Or use IP directly
      # base_url: "http://192.168.31.151:8080"
      language: "zh"
      response_format: "verbose_json"
      temperature: 0.0
      word_threshold: 0.01
      custom_headers:
        User-Agent: "tiktok-whisper-lan-client/1.0"
```

## Usage Examples

### Command Line Interface

```bash
# Basic transcription
v2t convert --audio --input audio.wav --provider whisper_server

# With specific language
v2t convert --audio --input chinese_audio.wav --provider whisper_server_lan

# Generate subtitles
v2t convert --audio --input video.mp4 --provider whisper_server \
    --format srt

# Test provider
v2t providers test whisper_server --file test.wav

# Check provider status
v2t providers status
```

### Go SDK Usage

```go
package main

import (
    "context"
    "tiktok-whisper/internal/app/api/whisper_server"
)

func main() {
    // Create provider configuration
    config := whisper_server.WhisperServerConfig{
        BaseURL:        "http://192.168.31.151:8080",
        Timeout:        120 * time.Second,
        Language:       "auto",
        ResponseFormat: "verbose_json",
        Temperature:    0.0,
        CustomHeaders: map[string]string{
            "User-Agent": "tiktok-whisper-client/1.0",
        },
    }
    
    // Create provider instance
    provider := whisper_server.NewWhisperServerProvider(config)
    
    // Health check
    ctx := context.Background()
    err := provider.HealthCheck(ctx)
    if err != nil {
        panic(err)
    }
    
    // Transcribe audio
    result, err := provider.Transcript("audio.wav")
    if err != nil {
        panic(err)
    }
    
    println("Transcription:", result)
}
```

## API Reference

### WhisperServerConfig

```go
type WhisperServerConfig struct {
    BaseURL         string            `json:"base_url"`           // Required
    InferencePath   string            `json:"inference_path"`     // Default: "/inference"
    LoadPath        string            `json:"load_path"`          // Default: "/load"
    Timeout         time.Duration     `json:"timeout"`           // Request timeout
    Language        string            `json:"language"`          // Language code
    ResponseFormat  string            `json:"response_format"`    // json, text, srt, vtt, verbose_json
    Temperature     float32           `json:"temperature"`       // 0.0-1.0
    Translate       bool              `json:"translate"`         // Translate to English
    NoTimestamps    bool              `json:"no_timestamps"`     // Disable timestamps
    WordThreshold   float32           `json:"word_threshold"`    // Word-level threshold
    MaxLength       int               `json:"max_length"`         // Max text length
    CustomHeaders   map[string]string `json:"custom_headers"`    // Custom HTTP headers
    InsecureSkipTLS bool              `json:"insecure_skip_tls"` // Skip TLS verification
}
```

### WhisperServerResponse

```go
type WhisperServerResponse struct {
    Text     string                 `json:"text"`
    Language string                 `json:"language"`
    Duration float64                `json:"duration"`
    Segments []WhisperServerSegment `json:"segments,omitempty"`
    // Additional fields for verbose_json format
}
```

## Response Formats

### JSON Format
```json
{
    "text": "Hello world",
    "language": "english",
    "duration": 2.5
}
```

### Verbose JSON Format
```json
{
    "text": "Hello world",
    "language": "english",
    "duration": 2.5,
    "segments": [
        {
            "id": 0,
            "text": "Hello world",
            "start": 0.0,
            "end": 2.5,
            "words": [
                {
                    "word": "Hello",
                    "start": 0.0,
                    "end": 0.5,
                    "probability": 0.99
                },
                {
                    "word": "world",
                    "start": 0.5,
                    "end": 1.0,
                    "probability": 0.98
                }
            ]
        }
    ]
}
```

### SRT Format
```srt
1
00:00:00,000 --> 00:00:02,500
Hello world
```

## Performance Optimization

### Server Side

1. **Model Selection**
   - Use `ggml-base.en` for English-only (faster)
   - Use `ggml-base` for multilingual
   - Use `ggml-large-v3` for best accuracy (slower)

2. **GPU Acceleration**
   ```bash
   ./build/bin/whisper-server \
       --gpu-layers 99 \
       --n-threads 8 \
       --model models/ggml-large-v3.bin
   ```

3. **Batch Processing**
   - Configure appropriate timeout values
   - Use concurrent requests carefully
   - Monitor server resources

### Client Side

1. **Timeout Configuration**
   ```yaml
   settings:
     timeout: 300  # Adjust based on audio length
   ```

2. **Concurrent Requests**
   ```yaml
   performance:
     max_concurrency: 2  # Don't overload server
   ```

3. **Retry Strategy**
   ```yaml
   error_handling:
     max_retries: 3
     retry_delay_ms: 2000
   ```

## Troubleshooting

### Common Issues

1. **Connection Timeout**
   ```bash
   # Check server is running
   ssh daymade@mac-mini-m4-1.local "pgrep -f whisper-server"
   
   # Check port is open
   telnet 192.168.31.151 8080
   ```

2. **503 Service Unavailable**
   - Server still loading model
   - Insufficient resources
   - Check server logs:
     ```bash
     ssh daymade@mac-mini-m4-1.local "tail -f whisper-server.log"
     ```

3. **Proxy Issues**
   ```bash
   # Bypass proxy for LAN
   export NO_PROXY="192.168.31.151,mac-mini-m4-1.local"
   
   # Or use SSH tunnel
   ssh -L 8080:localhost:8080 daymade@mac-mini-m4-1.local -N
   ```

4. **Model Loading Errors**
   - Verify model path exists
   - Check file permissions
   - Ensure sufficient disk space

### Debug Commands

```bash
# Test connectivity
curl -I http://192.168.31.151:8080

# Test inference
curl -X POST http://192.168.31.151:8080/inference \
    -F "file=@audio.wav" \
    -F "response_format=json"

# View server logs
ssh daymade@mac-mini-m4-1.local "tail -f /path/to/whisper-server.log"

# Check server status
v2t providers status
```

## Security Considerations

### Network Security

1. **Firewall Configuration**
   - Only expose necessary ports
   - Use IP whitelisting if possible
   - Consider VPN for WAN access

2. **Authentication**
   ```yaml
   settings:
     custom_headers:
       Authorization: "Bearer ${API_TOKEN}"
       X-API-Key: "${API_KEY}"
   ```

3. **HTTPS/SSL**
   - Use reverse proxy (nginx/apache) for SSL termination
   - Configure valid certificates
   - Enable HTTPS in configuration

### Data Privacy

1. **Audio Data**
   - Transmissions are unencrypted by default
   - Use HTTPS for sensitive audio
   - Consider audio retention policies

2. **Transcription Results**
   - Results are transmitted in clear text
   - Store results securely if needed
   - Implement proper access controls

## Integration Guide

### With Load Balancer

```yaml
providers:
  whisper_server_1:
    type: "whisper_server"
    enabled: true
    settings:
      base_url: "http://server1.example.com:8080"
      
  whisper_server_2:
    type: "whisper_server"
    enabled: true
    settings:
      base_url: "http://server2.example.com:8080"
      
  whisper_server_3:
    type: "whisper_server"
    enabled: true
    settings:
      base_url: "http://server3.example.com:8080"

orchestrator:
  load_balancing:
    strategy: "round_robin"
    health_check_interval: 30
```

### With Docker

```dockerfile
FROM golang:1.21-alpine

# Copy application
COPY tiktok-whisper /usr/local/bin/
COPY providers.yaml /etc/tiktok-whisper/

# Environment variables
ENV WHISPER_SERVER_URL="http://whisper-server:8080"

# Run
CMD ["tiktok-whisper", "convert", "--provider", "whisper_server"]
```

### With Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tiktok-whisper
spec:
  template:
    spec:
      containers:
      - name: app
        image: tiktok-whisper:latest
        env:
        - name: WHISPER_SERVER_URL
          value: "http://whisper-server-service:8080"
        volumeMounts:
        - name: config
          mountPath: /etc/tiktok-whisper/
      volumes:
      - name: config
        configMap:
          name: tiktok-whisper-config
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./internal/app/api/whisper_server/...

# Run with verbose output
go test -v ./internal/app/api/whisper_server/...

# Run specific test
go test -run TestHealthCheck ./internal/app/api/whisper_server/
```

### Integration Tests

```bash
# Run test client
./test-whisper-server-client \
    -url "http://192.168.31.151:8080" \
    -file test/data/test.wav \
    -verbose

# Run full integration test suite
./scripts/test-whisper-server.sh
```

### Load Testing

```bash
# Concurrent requests
for i in {1..10}; do
    ./test-whisper-server-client \
        -url "http://192.168.31.151:8080" \
        -file test/data/test.wav &
done
wait
```

## Monitoring and Logging

### Provider Metrics

The provider automatically tracks:
- Request count and success rate
- Response times
- Error rates by type
- Server health status

### Log Configuration

```yaml
# Enable debug logging
export RUST_LOG=debug
export WHISPER_SERVER_DEBUG=1

# View logs
tail -f whisper-server.log | grep -E "(error|warning|info)"

# Monitor performance
curl -s http://192.168.31.151:8080/metrics | grep whisper_
```

## Best Practices

1. **Production Deployment**
   - Use multiple server instances
   - Implement proper monitoring
   - Set up alerts for failures
   - Use load balancing

2. **Performance**
   - Choose appropriate model size
   - Optimize timeout values
   - Monitor resource usage
   - Implement caching if applicable

3. **Reliability**
   - Implement retry logic
   - Use health checks
   - Have fallback providers
   - Monitor error rates

4. **Security**
   - Use authentication
   - Enable HTTPS
   - Audit logs regularly
   - Implement rate limiting

## Version Compatibility

| Provider Version | Whisper-Server Version | Features |
|-----------------|----------------------|----------|
| 1.0.0           | 1.5.0+               | Basic transcription |
| 1.0.0           | 1.6.0+               | Verbose JSON support |
| 1.0.0           | 1.7.0+               | GPU acceleration |

## Contributing

To contribute to the whisper-server provider:

1. Fork the repository
2. Create feature branch
3. Add tests for new functionality
4. Update documentation
5. Submit pull request

## License

This provider is part of the tiktok-whisper project and follows the same license terms.

## Support

For issues and questions:
- GitHub Issues: [tiktok-whisper/issues](https://github.com/your-repo/tiktok-whisper/issues)
- Documentation: [docs/](./docs/)
- Whisper.cpp: [ggerganov/whisper.cpp](https://github.com/ggerganov/whisper.cpp)