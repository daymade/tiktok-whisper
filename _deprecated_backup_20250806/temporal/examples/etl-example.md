# ETL Pipeline Example

This example demonstrates how to use the distributed ETL pipeline to download and transcribe YouTube videos.

## Prerequisites

1. Start the Temporal cluster:
```bash
cd temporal/
docker-compose up -d
```

2. Start the Go worker:
```bash
cd temporal/worker/
go run main.go
```

3. Start the Python worker:
```bash
cd temporal/python-worker/
uv sync
uv run python worker.py
```

## Usage Examples

### 1. Process a YouTube Video

Submit a YouTube video for transcription:

```bash
# Basic usage
v2t etl --url "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

# Specify language
v2t etl --url "https://www.youtube.com/watch?v=VIDEO_ID" --language zh

# Wait for completion
v2t etl --url "https://www.youtube.com/watch?v=VIDEO_ID" --wait
```

### 2. Check Job Status

Check the status of a running job:

```bash
# Check status using workflow ID
v2t job status --workflow-id etl-abc123-1234567890
```

### 3. Using the Distributed Mode in Convert Command

The existing `convert` command also supports distributed mode:

```bash
# Convert a local file using distributed system
v2t convert --input video.mp4 --distributed

# Convert with specific provider
v2t convert --input video.mp4 --distributed --provider openai
```

## Architecture

The ETL pipeline consists of:

1. **Download Stage**: Uses `yt-dlp` to download audio from YouTube
2. **Convert Stage**: Uses `ffmpeg` to convert to optimal format (16kHz mono WAV)
3. **Transcribe Stage**: Uses `faster-whisper` with GPU acceleration
4. **Storage Stage**: Stores results in MinIO for distributed access

## Python Worker Features

The Python worker uses:
- **faster-whisper**: CTranslate2-based Whisper for 4x faster transcription
- **GPU Acceleration**: Automatic CUDA detection and usage
- **Voice Activity Detection**: Filters out silence for better accuracy
- **Word-level Timestamps**: Provides precise timing information
- **Model Caching**: Loads models once and reuses for efficiency

## Configuration

### Environment Variables

```bash
# MinIO Configuration
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin

# Temporal Configuration
export TEMPORAL_HOST=localhost:7233

# GPU Configuration (optional)
export CUDA_VISIBLE_DEVICES=0  # Use specific GPU
```

### Python Dependencies

The Python worker uses `uv` for dependency management. All dependencies are specified in `pyproject.toml`:

- `faster-whisper`: Fast transcription engine
- `yt-dlp`: YouTube downloader
- `ffmpeg-python`: Audio conversion
- `temporalio`: Workflow orchestration
- `minio`: Object storage

## Monitoring

1. **Temporal Web UI**: http://localhost:8088
   - View workflow executions
   - Check activity history
   - Debug failed workflows

2. **MinIO Console**: http://localhost:9001
   - Browse stored transcriptions
   - Monitor storage usage

## Troubleshooting

### Common Issues

1. **GPU Not Detected**:
   ```bash
   # Check CUDA availability
   python -c "import torch; print(torch.cuda.is_available())"
   ```

2. **YouTube Download Fails**:
   ```bash
   # Update yt-dlp
   uv pip install --upgrade yt-dlp
   ```

3. **Worker Not Connecting**:
   ```bash
   # Check Temporal connection
   curl http://localhost:7233/health
   ```

### Debug Mode

Enable debug logging:

```bash
# Go worker
LOG_LEVEL=debug go run main.go

# Python worker
LOG_LEVEL=DEBUG uv run python worker.py
```

## Performance Tips

1. **Model Selection**:
   - `large-v3`: Best accuracy (default)
   - `medium`: Good balance of speed/accuracy
   - `small`: Fastest, lower accuracy

2. **Batch Processing**:
   ```bash
   # Process multiple URLs
   v2t etl --url "URL1" && v2t etl --url "URL2" && v2t etl --url "URL3"
   ```

3. **Resource Allocation**:
   - Python worker uses 4 CPU threads by default
   - GPU memory usage scales with model size
   - Monitor with `nvidia-smi` for GPU utilization