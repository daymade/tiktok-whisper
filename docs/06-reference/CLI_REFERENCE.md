# CLI Command Reference

Complete reference for all v2t (video-to-text) CLI commands.

## Global Options

```bash
v2t [global options] command [command options] [arguments...]
```

### Global Flags
- `--help, -h` - Show help
- `--version, -v` - Print version information

## Commands

### 1. Download Commands

#### Download from Xiaoyuzhou

```bash
v2t download xiaoyuzhou [options]
```

**Options:**
- `-e, --episode URL` - Download single episode
- `-p, --podcast URL` - Download entire podcast  
- `-u, --user NICKNAME` - Specify user nickname

**Examples:**
```bash
# Download single episode
v2t download xiaoyuzhou -e https://www.xiaoyuzhoufm.com/episode/6626864fc3e09d8f37c3bde3

# Download entire podcast
v2t download xiaoyuzhou -p https://www.xiaoyuzhoufm.com/podcast/613753ef23c82a9a1ccfdf35 -u "podcast_name"
```

### 2. Convert Commands

#### Batch Convert

```bash
v2t convert batch [options]
```

**Options:**
- `-u, --user NICKNAME` - User nickname for organization
- `-p, --provider NAME` - Transcription provider to use
- `--parallel N` - Number of parallel conversions

**Examples:**
```bash
# Convert with default provider
v2t convert batch -u myname

# Convert with specific provider
v2t convert batch -u myname -p openai

# Convert with parallelism
v2t convert batch -u myname --parallel 3
```

### 3. Embedding Commands

#### Generate Embeddings

```bash
v2t embed generate [options]
```

**Options:**
- `-u, --user NICKNAME` - Generate for specific user
- `-p, --provider NAME` - Embedding provider (openai/gemini)
- `--force` - Regenerate existing embeddings

**Examples:**
```bash
# Generate all missing embeddings
v2t embed generate

# Generate for specific user
v2t embed generate -u myname

# Force regenerate with specific provider
v2t embed generate -p gemini --force
```

#### Search Embeddings

```bash
v2t embed search [options]
```

**Options:**
- `--text QUERY` - Search query text
- `--limit N` - Number of results (default: 10)
- `--threshold FLOAT` - Similarity threshold (0-1)

**Examples:**
```bash
# Search for similar content
v2t embed search --text "machine learning"

# Search with custom limit
v2t embed search --text "podcast about AI" --limit 20
```

#### Embedding Status

```bash
v2t embed status [options]
```

**Options:**
- `-u, --user NICKNAME` - Show status for specific user
- `--detailed` - Show detailed statistics

**Examples:**
```bash
# Show overall status
v2t embed status

# Show user-specific status
v2t embed status -u myname --detailed
```

### 4. Export Commands

#### Export to Markdown

```bash
v2t export markdown [options]
```

**Options:**
- `-u, --user NICKNAME` - Export specific user's data
- `-o, --output PATH` - Output directory
- `--format FORMAT` - Export format (single/split)

**Examples:**
```bash
# Export all transcriptions
v2t export markdown -o ./exports

# Export specific user
v2t export markdown -u myname -o ./exports/myname
```

#### Export to Excel

```bash
v2t export excel [options]
```

**Options:**
- `-u, --user NICKNAME` - Export specific user's data
- `-o, --output FILE` - Output file path

**Examples:**
```bash
# Export to Excel
v2t export excel -o transcriptions.xlsx
```

### 5. Provider Commands

#### List Providers

```bash
v2t providers list
```

**Output:**
- Provider name
- Type
- Status (enabled/disabled)
- Configuration summary

#### Provider Status

```bash
v2t providers status [provider_name]
```

**Examples:**
```bash
# Show all provider status
v2t providers status

# Show specific provider
v2t providers status openai
```

#### Test Provider

```bash
v2t providers test [provider_name]
```

**Examples:**
```bash
# Test specific provider
v2t providers test whisper_cpp

# Test all enabled providers
v2t providers test --all
```

### 6. Web Interface

#### Start Web Server

```bash
v2t web [options]
```

**Options:**
- `-p, --port PORT` - Server port (default: 8080)
- `--host HOST` - Bind host (default: localhost)
- `--no-browser` - Don't open browser automatically

**Examples:**
```bash
# Start with defaults
v2t web

# Start on custom port
v2t web -p 3000

# Start without opening browser
v2t web --no-browser
```

### 7. Configuration Commands

#### Show Config

```bash
v2t config show
```

**Output:**
- Current configuration file path
- Active providers
- Environment variables

#### Edit Config

```bash
v2t config edit
```

Opens configuration file in default editor.

#### Validate Config

```bash
v2t config validate
```

Validates configuration syntax and settings.

## Environment Variables

### Required
- `OPENAI_API_KEY` - OpenAI API key (for OpenAI provider)
- `GEMINI_API_KEY` - Google Gemini API key (for embeddings)

### Optional
- `WHISPER_CPP_BINARY` - Path to whisper.cpp binary
- `WHISPER_CPP_MODEL` - Path to whisper model file
- `ELEVENLABS_API_KEY` - ElevenLabs API key
- `DATABASE_URL` - Database connection string
- `HTTP_PORT` - Default HTTP port
- `REMOTE_HOST` - Remote host for SSH
- `REMOTE_USER` - Remote user for SSH

## Configuration Files

### Provider Configuration
Default location: `~/.tiktok-whisper/providers.yaml`

See [providers-example.yaml](../../providers-example.yaml) for full example.

### Application Configuration  
Default location: `~/.tiktok-whisper/config.yaml`

## Common Workflows

### 1. First Time Setup
```bash
# 1. Copy example configuration
cp providers-example.yaml ~/.tiktok-whisper/providers.yaml

# 2. Set API keys
export OPENAI_API_KEY="your-key"
export GEMINI_API_KEY="your-key"

# 3. Test providers
v2t providers test --all

# 4. Start using
v2t convert batch -u myname
```

### 2. Download and Convert Podcast
```bash
# Download podcast
v2t download xiaoyuzhou -p PODCAST_URL -u "podcast_name"

# Convert to text
v2t convert batch -u "podcast_name"

# Generate embeddings
v2t embed generate -u "podcast_name"

# Export to markdown
v2t export markdown -u "podcast_name" -o ./exports
```

### 3. Search Similar Content
```bash
# Generate embeddings first
v2t embed generate

# Search for content
v2t embed search --text "your search query"

# Start web interface for visual search
v2t web
```

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Provider error
- `4` - Database error
- `5` - Network error

## Debugging

Enable debug logging:
```bash
export V2T_DEBUG=1
v2t [command]
```

Enable verbose output:
```bash
v2t -v [command]
```