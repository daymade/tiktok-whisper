# Provider Framework Quick Start Guide

## Overview

The tiktok-whisper project now supports multiple transcription providers through a unified framework. This guide helps you quickly set up and use different providers.

## Available Providers

1. **whisper_cpp** - Local whisper.cpp binary
2. **openai** - OpenAI Whisper API
3. **elevenlabs** - ElevenLabs Speech-to-Text API
4. **ssh_whisper** - Remote SSH whisper.cpp
5. **whisper_server** - HTTP whisper-server API (NEW!)

## Quick Setup

### 1. Configure Providers

Create or edit `~/.tiktok-whisper/providers.yaml`:

```yaml
# Example configuration with multiple providers
default_provider: "whisper_server"

providers:
  # Local whisper.cpp
  whisper_cpp:
    type: "whisper_cpp"
    enabled: true
    settings:
      binary_path: "/path/to/whisper.cpp/main"
      model_path: "/path/to/models/ggml-base.en.bin"
      language: "auto"

  # OpenAI Whisper API
  openai:
    type: "openai"
    enabled: false  # Requires API key
    settings:
      model: "whisper-1"
      response_format: "json"

  # Whisper-Server on LAN (NEW!)
  whisper_server:
    type: "whisper_server"
    enabled: true
    settings:
      base_url: "http://192.168.31.151:8080"
      language: "auto"
      response_format: "verbose_json"
      timeout: 120

  # SSH Remote Whisper
  ssh_whisper:
    type: "ssh_whisper"
    enabled: false
    settings:
      host: "user@remote-server.com"
      remote_dir: "/path/to/whisper.cpp"
      binary_path: "./build/bin/whisper-cli"
      model_path: "models/ggml-base.en.bin"

# Intelligent routing
orchestrator:
  fallback_chain: ["whisper_server", "whisper_cpp", "openai"]
  router_rules:
    by_file_size:
      small: "whisper_cpp"
      large: "whisper_server"
```

### 2. Set Environment Variables

```bash
# For OpenAI provider
export OPENAI_API_KEY="sk-your-api-key-here"

# For ElevenLabs provider
export ELEVENLABS_API_KEY="your-elevenlabs-key"

# For whisper-server authentication (optional)
export WHISPER_SERVER_TOKEN="your-token"
```

## Usage Examples

### Basic Transcription

```bash
# Use default provider
v2t convert --audio --input audio.wav

# Use specific provider
v2t convert --audio --input audio.wav --provider whisper_server

# Generate subtitles
v2t convert --audio --input video.mp4 --provider whisper_server --format srt
```

### Provider Management

```bash
# List all providers
v2t providers list

# Check provider status
v2t providers status

# Test a provider
v2t providers test whisper_server --file test.wav

# View provider configuration
v2t providers config
```

## Setting Up Whisper-Server (Recommended)

### 1. Start Whisper-Server

```bash
# SSH into your server (e.g., mac-mini-m4-1.local)
ssh daymade@mac-mini-m4-1.local

# Navigate to whisper.cpp directory
cd /Users/daymade/Workspace/cpp/whisper.cpp

# Start the server
./build/bin/whisper-server \
    --host 0.0.0.0 \
    --port 8080 \
    --model models/ggml-base.en.bin

# Or start in background
nohup ./build/bin/whisper-server \
    --host 0.0.0.0 \
    --port 8080 \
    --model models/ggml-base.en.bin \
    > whisper-server.log 2>&1 &
```

### 2. Test Connection

```bash
# Test from your local machine
curl -s http://192.168.31.151:8080

# Or use our test client
./test-whisper-server-client -health -url "http://192.168.31.151:8080"
```

### 3. Configure in providers.yaml

```yaml
whisper_server_lan:
  type: "whisper_server"
  enabled: true
  settings:
    base_url: "http://192.168.31.151:8080"
    language: "auto"
    response_format: "verbose_json"
```

## Migration from Old Configuration

If you were using the old wire-based configuration, the migration is seamless. The provider framework maintains 100% backward compatibility.

Your existing commands will continue to work:
```bash
v2t convert --audio --input audio.wav  # Still works!
```

## Troubleshooting

### Common Issues

1. **Provider not found**
   ```bash
   v2t providers list  # Check if provider is registered
   ```

2. **Connection failed**
   ```bash
   v2t providers status  # Check provider health
   ```

3. **Authentication error**
   ```bash
   export OPENAI_API_KEY="your-key"  # Set required environment variables
   ```

### Debug Mode

Enable debug logging:
```bash
export RUST_LOG=debug
v2t convert --audio --input audio.wav --provider whisper_server
```

## Performance Tips

1. **For large files**: Use whisper-server with GPU acceleration
2. **For many small files**: Use local whisper_cpp
3. **For highest accuracy**: Use OpenAI or large whisper-server model
4. **For batch processing**: Configure max_concurrency appropriately

## Next Steps

1. Read detailed provider documentation:
   - [Whisper-Server Provider](../01-architecture/WHISPER_SERVER_PROVIDER.md)
   - [SSH Whisper Provider](../01-architecture/SSH_WHISPER_PROVIDER.md)

2. Explore advanced features:
   - Provider orchestration
   - Load balancing
   - Fallback chains

3. Set up monitoring:
   - Health checks
   - Performance metrics
   - Error tracking

## Support

For help and issues:
- Check the provider status: `v2t providers status`
- Run diagnostics: `v2t providers test <provider> --file test.wav`
- Review logs and error messages
- Open an issue on GitHub