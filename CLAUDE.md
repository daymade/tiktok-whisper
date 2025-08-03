# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based CLI tool called `tiktok-whisper` that batch converts videos/audio to text transcriptions using either local whisper.cpp (with coreML acceleration on macOS) or remote OpenAI Whisper API. The project supports downloading content from sources like Xiaoyuzhou podcasts and YouTube, then transcribing them with timestamp-aligned text output.

## Environment Setup

### API Key Configuration

**Security-first approach using .env files:**
```bash
# Copy the example file
cp .env.example .env

# Edit .env file with your API keys
# Note: .env files are automatically ignored by git for security
```

**Required API Keys:**
- `OPENAI_API_KEY` - For OpenAI text-embedding-ada-002 (1536 dimensions) and Whisper transcription
- `GEMINI_API_KEY` - For Google Gemini embedding-001 (768 dimensions)
- `ELEVENLABS_API_KEY` - For ElevenLabs Speech-to-Text API (optional)

**Environment Variables:**
```bash
# Option 1: Use .env file (recommended for development)
echo "OPENAI_API_KEY=sk-your-openai-key-here" >> .env
echo "GEMINI_API_KEY=AIza-your-gemini-key-here" >> .env

# Option 2: Set system environment variables
export OPENAI_API_KEY="sk-your-openai-key-here"
export GEMINI_API_KEY="AIza-your-gemini-key-here"
export DB_PASSWORD="passwd"  # Required for pgvector database connection
```

**Fail-fast validation:**
- The application validates API key formats on startup
- At least one API key must be configured
- Invalid or missing keys will cause immediate startup failure

## Common Development Commands

### Building the Project

**Main binary build:**
```bash
# Standard build
go build -o v2t ./cmd/v2t/main.go

# Build with CGO enabled (required for SQLite)
CGO_ENABLED=1 go build -o v2t ./cmd/v2t/main.go

# Run web server with database connection
CGO_ENABLED=1 DB_PASSWORD=passwd go run ./cmd/v2t/main.go web --port :8081

# Windows build
go build -o v2t.exe .\cmd\v2t\main.go
```

**Dependency injection setup (required after changing wire.go):**
```bash
cd ./internal/app
go install github.com/google/wire/cmd/wire@latest
wire
```

### Testing

**Run all tests:**
```bash
go test ./...
```

**Run tests with verbose output:**
```bash
go test -v ./...
```

**Run specific test packages:**
```bash
go test ./internal/app/converter/
go test ./internal/app/repository/sqlite/
go test ./internal/app/embedding/provider/
```

### Development Workflow

**Check formatting:**
```bash
go fmt ./...
```

**Run static analysis:**
```bash
go vet ./...
```

**Tidy dependencies:**
```bash
go mod tidy
```

## Architecture Overview

### Core Components

1. **CLI Layer** (`cmd/v2t/`): Cobra-based command-line interface with subcommands:
   - `download` - Download content from various sources
   - `convert` - Convert audio/video to text
   - `embed` - Generate embeddings for similarity search and duplicate detection
   - `export` - Export transcription results
   - `providers` - Manage transcription providers (NEW)
   - `config` - Configuration management
   - `version` - Version information

2. **Application Core** (`internal/app/`):
   - **API Layer**: Transcriber interface with local (whisper.cpp) and remote (OpenAI) implementations
   - **Repository Layer**: Database abstraction supporting SQLite and PostgreSQL with pgvector
   - **Converter**: Core business logic orchestrating the transcription process
   - **Embedding System**: Dual embedding support (OpenAI + Gemini) for similarity search
   - **Models**: Data structures for transcriptions, file info, and media metadata

3. **Dependency Injection**: Uses Google Wire for DI container configuration

### Key Interfaces

**Transcriber Interface** (`internal/app/api/transcriber.go`):
```go
type Transcriber interface {
    Transcript(inputFilePath string) (string, error)
}
```

**Database Interface** (`internal/app/repository/dao.go`):
```go
type TranscriptionDAO interface {
    // CRUD operations for transcriptions
}
```

### Configuration Options

**Transcriber Selection** (modify `internal/app/wire.go`):
- For local whisper.cpp: `provideLocalTranscriber` 
- For OpenAI API: `provideRemoteTranscriber`

**Local Whisper.cpp Setup:**
- Requires setting `binaryPath` and `modelPath` in wire.go
- Binary must be compiled with coreML support on macOS: `WHISPER_COREML=1 make -j`

**OpenAI API Setup:**
- Set environment variable: `OPENAI_API_KEY`
- Switch wire configuration to use `provideRemoteTranscriber`
- Also used for embedding generation in dual embedding system

## Provider Framework (NEW)

The project now features a flexible transcription provider framework that abstracts the transcription process into a configurable, extensible system following SOLID design principles.

### Architecture Overview

For a complete understanding of the provider framework architecture, see:
- [Provider Framework Architecture Documentation](docs/PROVIDER_FRAMEWORK_ARCHITECTURE.md) - Comprehensive technical design
- [Provider Quick Start Guide](docs/PROVIDER_QUICK_START.md) - Quick setup and usage
- [SSH Whisper Provider](docs/SSH_WHISPER_PROVIDER.md) - SSH-based remote transcription
- [Whisper-Server Provider](docs/WHISPER_SERVER_PROVIDER.md) - HTTP whisper-server integration

### Available Providers

**Built-in Providers:**
- **whisper_cpp** - Local whisper.cpp binary (default)
- **openai** - OpenAI Whisper API 
- **elevenlabs** - ElevenLabs Speech-to-Text API
- **ssh_whisper** - Remote SSH whisper.cpp provider
- **whisper_server** - HTTP whisper-server provider
- **custom_http** - Generic HTTP-based whisper services (planned)

### Provider Configuration

**Configuration File:** `~/.tiktok-whisper/providers.yaml` (auto-created)

```yaml
default_provider: "whisper_cpp"

providers:
  whisper_cpp:
    type: whisper_cpp
    enabled: true
    settings:
      binary_path: "/Volumes/SSD2T/workspace/cpp/whisper.cpp/main"
      model_path: "/Volumes/SSD2T/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"
      language: "zh"
      prompt: "以下是简体中文普通话:"
    performance:
      timeout_sec: 300
      max_concurrency: 2
  
  openai:
    type: openai
    enabled: false
    auth:
      api_key: "${OPENAI_API_KEY}"
    settings:
      model: "whisper-1"
      response_format: "text"
    performance:
      timeout_sec: 60
      rate_limit_rpm: 50

orchestrator:
  fallback_chain: ["whisper_cpp", "openai"]
  prefer_local: true
  router_rules:
    by_file_size:
      small: "whisper_cpp"
      large: "openai"
```

### Provider Management CLI

**List all available providers:**
```bash
v2t providers list
```

**Check provider health:**
```bash
v2t providers status
```

**Get provider details:**
```bash
v2t providers info openai
```

**Test a provider:**
```bash
v2t providers test whisper_cpp --file test.wav
```

**Show configuration:**
```bash
v2t providers config
```

### Enhanced Features

**Intelligent Orchestration:**
- Automatic provider selection based on file characteristics
- Fallback chains for reliability
- Load balancing and cost optimization

**Comprehensive Monitoring:**
- Health checks for all providers
- Usage statistics and performance metrics
- Error tracking with retry suggestions

**Advanced Configuration:**
- Environment variable expansion
- YAML-based configuration with validation
- Runtime provider switching

### Backward Compatibility

The framework maintains 100% backward compatibility:
- All existing CLI commands work unchanged
- Default behavior remains local whisper.cpp
- Original `Transcriber` interface still supported
- Automatic fallback if provider framework fails

### Database Support

**SQLite** (default):
- Embedded database, no setup required
- Uses `github.com/mattn/go-sqlite3` driver

**PostgreSQL with pgvector** (for embedding storage):
- Requires external PostgreSQL instance with pgvector extension
- Uses `github.com/lib/pq` driver
- Migration scripts available in `scripts/pg/sql/`
- Dual embedding storage for OpenAI and Gemini embeddings

**Database Connection Information:**
- Container: `mypgvector` (Docker container with ankane/pgvector image)
- Host: localhost
- Port: 5432 (mapped from Docker)
- Username: postgres
- Password: `passwd` (required for application connections)
- Database: postgres
- Main table: `transcriptions`

**Manual Database Access Commands:**
```bash
# Connect using temporary Docker container (password: passwd)
PGPASSWORD=passwd docker run --rm --network container:mypgvector postgres:15-alpine psql -h localhost -U postgres -d postgres

# View table structure
PGPASSWORD=passwd docker run --rm --network container:mypgvector postgres:15-alpine psql -h localhost -U postgres -d postgres -c "\d transcriptions"

# Check data statistics
PGPASSWORD=passwd docker run --rm --network container:mypgvector postgres:15-alpine psql -h localhost -U postgres -d postgres -c "
SELECT COUNT(*) as total_records, 
COUNT(embedding_openai) as openai_embeddings, 
COUNT(embedding_gemini) as gemini_embeddings,
COUNT(CASE WHEN embedding_openai IS NOT NULL OR embedding_gemini IS NOT NULL THEN 1 END) as total_embeddings
FROM transcriptions;"

# View embedding status distribution
PGPASSWORD=passwd docker run --rm --network container:mypgvector postgres:15-alpine psql -h localhost -U postgres -d postgres -c "
SELECT 
  embedding_openai_status,
  COUNT(*) as openai_count,
  embedding_gemini_status,
  COUNT(*) as gemini_count
FROM transcriptions 
GROUP BY embedding_openai_status, embedding_gemini_status;"
```

**Current Data Status (as of latest check):**
- Total transcription records: 1,060
- OpenAI embeddings generated: 2
- Gemini embeddings generated: 50
- Total usable embeddings: 52

## Python Scripts Alternative

For Windows with CUDA GPU, Python scripts are available in `scripts/python/`:
- `whisperToText.py` - Single file/directory transcription
- `whisperToTextParallel.py` - Parallel processing across subdirectories
- `convertTo16KHz.py` - Audio format conversion utility

**Setup Python environment:**
```bash
pip install -r requirements.txt
```

## Testing Strategy

The project uses table-driven tests and includes:
- Unit tests for core components
- Integration tests for database operations  
- Test utilities for database setup and teardown

**Test file locations:**
- Repository tests: `internal/app/repository/*/test.go`
- Converter tests: `internal/app/converter/convert_test.go`
- API tests: `internal/app/api/*/test.go`
- Embedding tests: `internal/app/embedding/provider/*/test.go`

**Testing features:**
- Table-driven tests with comprehensive coverage
- Mock-based testing for external dependencies
- Integration tests with real databases and APIs
- Test utilities in `internal/app/testutil/`

## External Dependencies

**Whisper.cpp Setup** (for local transcription):
```bash
mkdir -p ~/workspace/cpp/ && cd ~/workspace/cpp/
git clone git@github.com:ggerganov/whisper.cpp.git
cd whisper.cpp
bash ./models/download-ggml-model.sh large
# ... follow README instructions for coreML model generation
```

**Required Tools:**
- `ffmpeg` - For audio/video processing
- `yt-dlp` - For YouTube downloads (optional)

## File Structure Patterns

- `/data/` - Downloaded content and converted audio files
- `/data/transcription/` - Text output files
- `/scripts/` - Database SQL scripts and Python utilities  
- `/test/data/` - Test audio/video files
- `/logs/` - Application logs

## Export to Markdown Tool

A powerful automation tool has been created at `tools/export-to-md/` that streamlines the entire process of exporting transcription data to Markdown files.

**Quick Usage:**
```bash
cd tools/export-to-md

# Initialize uv environment
uv sync

# List all users and their record counts
uv run python export_to_md.py list-users

# Export specific user data
uv run python export_to_md.py export --user "经纬第二期"

# Export all users (creates subdirectories)
uv run python export_to_md.py export-all
```

**Features:**
- Automatic JSON export from SQLite database
- Integration with html2md tool for conversion
- Batch processing with 50 records per Markdown file
- ZIP file generation with all Markdown files
- Colorized terminal output with progress indicators
- Configurable paths and options
- Error handling and validation

**Configuration:**
```bash
# Show current configuration
uv run python export_to_md.py config --show

# Update paths if needed
uv run python export_to_md.py config --set html2md_path="/path/to/html2md/main.py"
```

See `tools/export-to-md/README.md` for complete documentation.

## Embedding System

The project features a dual embedding system for similarity search and duplicate detection:

**CLI Commands:**
```bash
# Generate embeddings for all transcriptions
v2t embed generate

# Generate embeddings for specific user
v2t embed generate --user "username" --provider gemini

# Check embedding status with user distribution
v2t embed status

# Find similar transcriptions
v2t embed search --text "search query" --limit 10

# Calculate similarity between transcriptions
v2t embed similarity --id1 123 --id2 456

# Find duplicates for specific user
v2t embed duplicates --user "username" --threshold 0.95
```

**Architecture:**
- **Dual Provider Support**: OpenAI text-embedding-3-small and Gemini text-embedding-004
- **PostgreSQL Integration**: Uses pgvector extension for efficient similarity search
- **User-Specific Processing**: Targeted embedding generation with user filtering and statistics
- **Batch Processing**: Handles large datasets with progress tracking
- **Error Resilience**: Robust error handling and retry mechanisms

**Configuration:**
- Requires `OPENAI_API_KEY` and `GEMINI_API_KEY` environment variables
- PostgreSQL with pgvector extension for storage (see Database Connection Information above)
- Migration scripts: `scripts/pg/sql/add_dual_embeddings.sql`

**Data Access for Development:**
- Use DataGrip or similar tools to connect to pgvector database
- Connection details: localhost:5432, user: postgres, db: postgres
- For command-line access, use the Docker commands listed in Database Connection Information section

**Key Files:**
- `internal/app/embedding/provider/` - Embedding providers (OpenAI, Gemini)
- `cmd/v2t/cmd/embed/` - CLI commands for embedding operations
- `docs/DUAL_EMBEDDING_TDD_PLAN.md` - Implementation documentation

## Trackpad Gesture System

The project features a Jon Ive-level natural trackpad interaction system for the 3D visualization:

**Features:**
- **智能手势识别**: Distinguishes between single-finger rotation, two-finger pinch-zoom, and two-finger pan
- **真正的双指缩放**: Proper pinch-to-zoom with logarithmic scaling for linear feel
- **动量支持**: Natural momentum decay after gestures end with physics-based animation
- **跨浏览器兼容**: Safari GestureEvent and standard TouchEvent support
- **设备自适应**: Automatic mouse vs trackpad detection with manual override
- **高性能优化**: 120fps updates, Map-based touch tracking, anti-jitter algorithms

**Technical Implementation:**
- Enhanced touch state management with Map data structure
- Intelligent gesture recognition with hysteresis to prevent switching
- Natural zoom control using logarithmic scaling
- Momentum physics system with realistic decay
- Cross-browser compatibility layer
- Device-specific sensitivity settings

**Files:**
- `web/static/js/visualization.js` - Core gesture system implementation
- `web/static/debug.html` - Touch event debugging page
- `docs/TRACKPAD_GESTURE_SYSTEM.md` - Complete technical documentation

**Testing:**
- Visit `/debug.html` to test touch events and gesture recognition
- Console logs with `[TOUCH]`, `[GESTURE]`, and `[INPUT]` prefixes for debugging
- Device indicator in UI shows current input device and confidence level

## Important Configuration Files

- `go.mod` - Go module dependencies
- `requirements.txt` - Python dependencies for scripts
- `internal/app/wire.go` - Dependency injection configuration
- `internal/app/wire_gen.go` - Generated DI code (don't edit manually)
- `tools/export-to-md/config.json` - Export tool configuration
- `tools/export-to-md/pyproject.toml` - uv project configuration
- `scripts/pg/sql/add_dual_embeddings.sql` - PostgreSQL embedding schema
- `docs/DUAL_EMBEDDING_TDD_PLAN.md` - Embedding system documentation
- `docs/TRACKPAD_GESTURE_SYSTEM.md` - Trackpad gesture system technical documentation