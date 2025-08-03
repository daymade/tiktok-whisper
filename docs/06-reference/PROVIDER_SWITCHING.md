# Provider Switching

This document describes how to use and configure different transcription providers in tiktok-whisper.

## Overview

The tiktok-whisper tool supports multiple transcription providers:

- `whisper_cpp` - Local whisper.cpp binary (default)
- `openai` - OpenAI Whisper API
- `elevenlabs` - ElevenLabs Speech-to-Text API
- `whisper_server` - HTTP-based whisper server
- `ssh_whisper` - Remote whisper.cpp via SSH
- `custom_http` - Generic HTTP transcription service

## Configuration

### Configuration File

Create a `providers.yaml` file in your project directory or at `~/.tiktok-whisper/providers.yaml`:

```yaml
default_provider: "whisper_cpp"

providers:
  whisper_cpp:
    type: "whisper_cpp"
    enabled: true
    settings:
      binary_path: "/path/to/whisper.cpp/main"  # Required
      model_path: "/path/to/models/ggml-base.bin"  # Required
      language: "en"
      prompt: ""
      
  openai:
    type: "openai"
    enabled: true
    auth:
      api_key: "sk-..."  # Required
    settings:
      model: "whisper-1"
      language: "en"
      response_format: "text"
      temperature: 0.0
      
  whisper_server:
    type: "whisper_server"
    enabled: true
    settings:
      base_url: "http://localhost:8080"  # Required
      inference_path: "/inference"
      language: "auto"
      response_format: "json"
```

### Important Notes

1. **No Default Values**: Providers require explicit configuration. There are no hardcoded defaults.
2. **Fail Fast**: If a provider is not properly configured, the program will exit with an error.
3. **Config Priority**: Local `providers.yaml` takes precedence over `~/.tiktok-whisper/providers.yaml`.

## Usage

### Using Default Provider

```bash
# Uses the default_provider from config
./tiktok-whisper convert -i audio.mp3 -a
```

### Switching Providers

Use the `--provider` flag to select a specific provider:

```bash
# Use whisper_cpp provider
./tiktok-whisper convert -i audio.mp3 -a --provider whisper_cpp

# Use OpenAI provider
./tiktok-whisper convert -i audio.mp3 -a --provider openai

# Use whisper server
./tiktok-whisper convert -i audio.mp3 -a --provider whisper_server
```

### List Available Providers

```bash
./tiktok-whisper providers list
```

### Check Provider Status

```bash
./tiktok-whisper providers status
```

## Provider-Specific Configuration

### whisper_cpp

Required settings:
- `binary_path`: Path to whisper.cpp main executable
- `model_path`: Path to whisper model file

Optional settings:
- `language`: Language code (default: "auto")
- `prompt`: Context prompt for better accuracy
- `output_format`: Output format (default: "txt")
- `max_concurrent`: Max concurrent transcriptions
- `temp_dir`: Temporary directory for processing

### openai

Required settings:
- `auth.api_key`: OpenAI API key

Optional settings:
- `model`: Model to use (default: "whisper-1")
- `language`: Language code
- `prompt`: Context prompt
- `response_format`: Response format (text/json/srt/vtt)
- `temperature`: Temperature for sampling
- `base_url`: Custom API endpoint

### whisper_server

Required settings:
- `base_url`: Base URL of whisper server

Optional settings:
- `inference_path`: Inference endpoint path
- `load_path`: Model load endpoint path
- `language`: Language code
- `response_format`: Response format
- `timeout`: Request timeout in seconds

## Error Handling

The system follows a fail-fast approach:

1. **Missing Configuration**: If `providers.yaml` is not found, the program exits.
2. **Invalid Provider**: If specified provider doesn't exist in config, the program exits.
3. **Missing Required Settings**: If a provider lacks required settings, the program exits.
4. **Provider Creation Failed**: If provider initialization fails, the program exits.

Example error messages:

```
Failed to load provider configuration from providers.yaml: open providers.yaml: no such file or directory
Provider 'invalid_provider' not found in configuration
Failed to create provider 'whisper_cpp': whisper_cpp provider requires 'binary_path' setting
Failed to create provider 'openai': openai provider requires 'api_key' in auth configuration
```

## Best Practices

1. **Keep Sensitive Data Secure**: Don't commit API keys to version control. Use environment variables or secure vaults.
2. **Test Configuration**: Use `providers test` command to verify provider setup.
3. **Use Local Config**: Place `providers.yaml` in your project directory for project-specific settings.
4. **Explicit Configuration**: Always explicitly configure all required settings - there are no defaults.