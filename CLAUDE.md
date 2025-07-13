# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based CLI tool called `tiktok-whisper` that batch converts videos/audio to text transcriptions using either local whisper.cpp (with coreML acceleration on macOS) or remote OpenAI Whisper API. The project supports downloading content from sources like Xiaoyuzhou podcasts and YouTube, then transcribing them with timestamp-aligned text output.

## Common Development Commands

### Building the Project

**Main binary build:**
```bash
# Standard build
go build -o v2t ./cmd/v2t/main.go

# Build with CGO enabled (required for SQLite)
CGO_ENABLED=1 go build -o v2t ./cmd/v2t/main.go

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

**Run specific test files:**
```bash
go test ./internal/app/converter/
go test ./internal/app/repository/sqlite/
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
   - `export` - Export transcription results
   - `config` - Configuration management

2. **Application Core** (`internal/app/`):
   - **API Layer**: Transcriber interface with local (whisper.cpp) and remote (OpenAI) implementations
   - **Repository Layer**: Database abstraction supporting SQLite and PostgreSQL
   - **Converter**: Core business logic orchestrating the transcription process
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

### Database Support

**SQLite** (default):
- Embedded database, no setup required
- Uses `github.com/mattn/go-sqlite3` driver

**PostgreSQL** (optional):
- Requires external PostgreSQL instance
- Uses `github.com/lib/pq` driver
- Migration scripts available in `scripts/pg/sql/`

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

## Important Configuration Files

- `go.mod` - Go module dependencies
- `requirements.txt` - Python dependencies for scripts
- `internal/app/wire.go` - Dependency injection configuration
- `internal/app/wire_gen.go` - Generated DI code (don't edit manually)
- `tools/export-to-md/config.json` - Export tool configuration
- `tools/export-to-md/pyproject.toml` - uv project configuration