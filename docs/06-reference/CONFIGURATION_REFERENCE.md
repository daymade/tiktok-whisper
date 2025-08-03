# Configuration Reference

Complete reference for all configuration options in tiktok-whisper.

## Configuration Files

### 1. Provider Configuration (`providers.yaml`)

Location priority:
1. `./providers.yaml` (current directory)
2. `~/.tiktok-whisper/providers.yaml` (home directory)

For detailed provider configuration and usage, see [Provider Switching Documentation](PROVIDER_SWITCHING.md).

#### Top-Level Fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `default_provider` | string | Provider to use when none specified | `"whisper_cpp"` |
| `providers` | map | Provider configurations | - |
| `orchestrator` | object | Orchestration settings | - |

#### Provider Configuration

Each provider in the `providers` map has these fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Provider type identifier |
| `enabled` | bool | No | Whether provider is active | `true` |
| `auth` | object | No | Authentication settings |
| `settings` | map | No | Provider-specific settings |
| `performance` | object | No | Performance tuning |
| `error_handling` | object | No | Error handling configuration |

#### Authentication Configuration

| Field | Type | Description |
|-------|------|-------------|
| `api_key` | string | API key (supports env vars) |
| `username` | string | Username for authentication |
| `password` | string | Password for authentication |

#### Performance Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `timeout_sec` | int | Request timeout in seconds | Provider-specific |
| `max_concurrency` | int | Max parallel operations | Provider-specific |

#### Error Handling Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `max_retries` | int | Maximum retry attempts | 2 |
| `retry_delay_ms` | int | Delay between retries (ms) | 1000 |

### 2. Provider-Specific Settings

#### Whisper.cpp Provider

| Setting | Type | Required | Description |
|---------|------|----------|-------------|
| `binary_path` | string | **Yes** | Path to whisper binary |
| `model_path` | string | **Yes** | Path to model file |
| `language` | string | No | Target language (default: `"auto"`) |
| `prompt` | string | No | Initial prompt |
| `threads` | int | No | CPU threads to use |

#### OpenAI Provider

| Setting | Type | Required | Description | Default |
|---------|------|----------|-------------|---------|
| `api_key` | string | **Yes** | API key (in `auth` section) | - |
| `model` | string | No | Model name | `"whisper-1"` |
| `language` | string | No | Target language | `"auto"` |
| `prompt` | string | No | Initial prompt | - |
| `temperature` | float | No | Sampling temperature | 0.0 |
| `response_format` | string | No | Output format | `"text"` |

#### ElevenLabs Provider

| Setting | Type | Description | Default |
|---------|------|-------------|---------|
| `language_code` | string | Language code | `"zh"` |
| `model` | string | Model identifier | `"eleven_whisper_v2"` |

#### SSH Whisper Provider

| Setting | Type | Description | Default |
|---------|------|-------------|---------|
| `host` | string | SSH host (user@host) | - |
| `remote_dir` | string | Remote whisper directory | - |
| `binary_path` | string | Remote binary path | - |
| `model_path` | string | Remote model path | - |
| `language` | string | Target language | `"zh"` |
| `prompt` | string | Initial prompt | - |
| `threads` | int | CPU threads | 4 |

#### HTTP Whisper Server Provider

| Setting | Type | Description | Default |
|---------|------|-------------|---------|
| `base_url` | string | Server base URL | - |
| `inference_path` | string | Inference endpoint | `"/inference"` |
| `load_path` | string | Model load endpoint | `"/load"` |
| `timeout` | int | Request timeout (sec) | 60 |
| `language` | string | Target language | `"auto"` |
| `response_format` | string | Response format | `"json"` |
| `temperature` | float | Sampling temperature | 0.0 |
| `translate` | bool | Translate to English | false |
| `no_timestamps` | bool | Disable timestamps | false |
| `word_threshold` | float | Word confidence threshold | 0.01 |
| `max_length` | int | Max segment length | 0 |
| `custom_headers` | map | Additional HTTP headers | {} |
| `insecure_skip_tls` | bool | Skip TLS verification | false |

### 3. Orchestrator Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `fallback_chain` | []string | Provider fallback order | - |
| `health_check_interval` | string | Health check frequency | `"10m"` |
| `health_check_timeout` | string | Health check timeout | `"30s"` |
| `global_timeout_sec` | int | Global operation timeout | 600 |
| `max_retries` | int | Orchestrator-level retries | 1 |
| `retry_delay` | string | Retry delay duration | `"5s"` |
| `prefer_local` | bool | Prefer local providers | true |
| `router_rules` | object | Routing configuration | - |
| `load_balancing` | object | Load balancing config | - |

#### Router Rules

| Field | Type | Description |
|-------|------|-------------|
| `by_language` | map | Route by detected language |
| `by_file_size` | map | Route by file size category |
| `by_quality` | map | Route by quality requirement |

#### Load Balancing

| Field | Type | Description | Options |
|-------|------|-------------|---------|
| `strategy` | string | Balancing algorithm | `round_robin`, `least_connections`, `weighted` |
| `weights` | map | Provider weights | Provider name → weight |

## Environment Variables

### Core Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | OpenAI API key | - |
| `GEMINI_API_KEY` | Google Gemini API key | - |
| `ELEVENLABS_API_KEY` | ElevenLabs API key | - |
| `WHISPER_CPP_BINARY` | Whisper.cpp binary path | - |
| `WHISPER_CPP_MODEL` | Model file path | - |

### Network Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LOCAL_HOST` | Local hostname | `localhost` |
| `REMOTE_HOST` | Remote hostname | `mac-mini-m4-1.local` |
| `REMOTE_USER` | SSH username | `daymade` |
| `HTTP_PORT` | HTTP server port | `8080` |
| `SSH_PORT` | SSH port | `22` |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `WHISPER_SERVER_URL` | Whisper server URL | - |

### Database Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | Full database URL | - |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | `postgres` |

### Application Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `V2T_DEBUG` | Enable debug logging | `0` |
| `V2T_CONFIG_PATH` | Config directory path | `~/.tiktok-whisper` |
| `V2T_DATA_PATH` | Data directory path | `./data` |
| `NO_PROXY` | Proxy bypass list | - |

## Configuration Precedence

1. Command-line flags (highest priority)
2. Environment variables
3. Configuration file settings
4. Built-in defaults (lowest priority)

## Variable Expansion

Configuration values support environment variable expansion:

```yaml
api_key: "${OPENAI_API_KEY}"
binary_path: "${WHISPER_CPP_BINARY:-./whisper.cpp/main}"
```

Syntax:
- `${VAR}` - Use environment variable
- `${VAR:-default}` - Use environment variable with fallback

## Validation Rules

### API Keys
- OpenAI: Must start with `sk-`, minimum 20 characters
- Gemini: Must start with `AIza`, minimum 30 characters
- ElevenLabs: Minimum 32 characters

### Timeouts
- Must be positive
- Maximum 30 minutes (1800 seconds)

### Concurrency
- Must be positive
- Maximum 100

### Retries
- Cannot be negative
- Maximum 10

### Ports
- Must be valid port numbers (1-65535)
- Must be numeric strings

## Example Configurations

### Minimal Configuration
```yaml
default_provider: "whisper_cpp"
providers:
  whisper_cpp:
    type: "whisper_cpp"
    enabled: true
```

### Multi-Provider with Fallback
```yaml
default_provider: "openai"
providers:
  openai:
    type: "openai"
    enabled: true
    auth:
      api_key: "${OPENAI_API_KEY}"
  whisper_cpp:
    type: "whisper_cpp"
    enabled: true
orchestrator:
  fallback_chain: ["openai", "whisper_cpp"]
```

### Production Configuration
See [providers-example.yaml](../../providers-example.yaml) for a complete production-ready configuration.

## Troubleshooting

### Configuration Not Loading
1. Check file location: `~/.tiktok-whisper/providers.yaml`
2. Validate YAML syntax
3. Check file permissions

### Environment Variables Not Working
1. Ensure variables are exported
2. Check variable names match exactly
3. Verify no typos in expansion syntax

### Provider Not Available
1. Check `enabled: true` in configuration
2. Verify required settings are present
3. Test with `v2t providers test [name]`