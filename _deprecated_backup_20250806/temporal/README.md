# V2T Distributed Transcription System

A distributed video-to-text transcription system using Temporal workflow orchestration, supporting multiple machines with GPU/NPU acceleration.

## Features

- **Distributed Processing**: Utilize multiple machines (M2 + 2x M4) for parallel transcription
- **ETL Pipeline**: Complete workflow from YouTube URL to transcription
- **Multiple Providers**: Support for whisper.cpp, OpenAI, faster-whisper, and more
- **GPU Acceleration**: Automatic GPU/NPU detection and utilization
- **Fault Tolerance**: Automatic retries and failover between providers
- **MinIO Storage**: Distributed object storage for files and results

## Quick Start

See [DOCKER_PROFILES.md](DOCKER_PROFILES.md) for deployment options using Docker Compose profiles.

### 1. Start Services

```bash
# Single node development
docker-compose up -d

# Distributed production setup
docker-compose --profile distributed up -d

# With monitoring
docker-compose --profile distributed --profile monitoring up -d
```

### 2. Start Workers

#### Go Worker (for local whisper.cpp)
```bash
cd temporal/worker/
go run main.go
```

#### Python Worker (for faster-whisper with GPU)
```bash
cd temporal/python-worker/
uv sync
uv run python worker.py
```

### 3. Run ETL Pipeline

Process a YouTube video:
```bash
# Submit job
v2t etl --url "https://www.youtube.com/watch?v=VIDEO_ID" --language zh

# Submit and wait for result
v2t etl --url "https://www.youtube.com/watch?v=VIDEO_ID" --wait

# Check job status
v2t job status --workflow-id etl-abc123-1234567890
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   M2 Machine    │     │   M4 Machine 1  │     │   M4 Machine 2  │
│                 │     │                 │     │                 │
│ - Temporal      │     │ - Go Worker     │     │ - Python Worker │
│ - MinIO         │     │ - whisper.cpp   │     │ - faster-whisper│
│ - Go Worker     │     │                 │     │ - GPU/NPU       │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         └───────────────────────┴───────────────────────┘
                          Temporal Network
```

## ETL Workflow

1. **Download**: yt-dlp downloads audio from YouTube
2. **Convert**: FFmpeg converts to optimal format (16kHz mono WAV)
3. **Transcribe**: faster-whisper performs GPU-accelerated transcription
4. **Store**: Results saved to MinIO for distributed access

## Configuration

### Environment Variables

```bash
# MinIO
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin

# Temporal
export TEMPORAL_HOST=localhost:7233

# API Keys (for remote providers)
export OPENAI_API_KEY=sk-...
export ELEVENLABS_API_KEY=...
```

### Provider Configuration

Edit `~/.tiktok-whisper/providers.yaml`:

```yaml
providers:
  whisper_cpp:
    type: whisper_cpp
    settings:
      binary_path: /path/to/whisper.cpp/main
      model_path: /path/to/models/ggml-large-v2.bin
  
  faster_whisper:
    type: faster_whisper
    settings:
      model_size: large-v3
      device: cuda  # or cpu
```

## Deployment

### Single Machine Mode

```bash
# Use local providers only
v2t convert --input video.mp4
```

### Distributed Mode

```bash
# Use distributed system
v2t convert --input video.mp4 --distributed

# Process batch with parallelism
v2t convert --input /videos/ --distributed --parallel 4
```

### Multi-Machine Setup

1. **Control Node (M2)**:
   ```bash
   # Start core services
   docker-compose up -d
   ```

2. **Worker Nodes (M4)**:
   ```bash
   # Set Temporal host
   export TEMPORAL_HOST=192.168.1.100:7233
   
   # Start worker
   cd temporal/python-worker/
   uv run python worker.py
   ```

## Monitoring

- **Temporal Web UI**: http://localhost:8088
- **MinIO Console**: http://localhost:9001
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000

## Performance Tuning

### Python Worker

```python
# Adjust in worker.py
whisper_models[cache_key] = WhisperModel(
    model_size,
    device="cuda",  # Use GPU
    compute_type="float16",  # Faster on GPU
    num_workers=4,  # Parallel decoding
)
```

### Batch Processing

```bash
# Process multiple files in parallel
v2t etl --url "URL1" &
v2t etl --url "URL2" &
v2t etl --url "URL3" &
wait
```

## Troubleshooting

### Check Service Health

```bash
# Temporal
curl http://localhost:7233/health

# MinIO
curl http://localhost:9000/minio/health/live

# Worker logs
docker-compose logs -f
```

### Common Issues

1. **Worker not connecting**: Check TEMPORAL_HOST environment variable
2. **GPU not detected**: Verify CUDA installation with `nvidia-smi`
3. **MinIO access denied**: Check MINIO_ACCESS_KEY and MINIO_SECRET_KEY

## Development

### Running Tests

```bash
# Test ETL pipeline
./temporal/examples/test-etl.sh

# Test Go components
go test ./temporal/...

# Test Python components
cd temporal/python-worker/
uv run pytest
```

### Adding New Providers

1. Implement provider interface in Go
2. Register in provider factory
3. Add configuration to providers.yaml
4. Optional: Create Python activity for ML-based providers

## Security Notes

- No TLS/HTTPS required for local network
- Use environment variables for API keys
- MinIO uses default credentials (change in production)
- No Vault integration (as requested)