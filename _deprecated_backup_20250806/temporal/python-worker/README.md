# Python Worker for v2t Distributed System

This Python worker demonstrates cross-language support in the v2t distributed transcription system. It can work alongside Go workers in the same Temporal cluster.

## Features

- **Whisper Python**: Local transcription with GPU acceleration
- **OpenAI API**: Alternative Python implementation
- **Advanced Audio Analysis**: ML-based audio analysis
- **Post-processing**: NLP-based text enhancement
- **Cross-language Workflows**: Python workflows callable from Go

## Setup

### Local Development

```bash
# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Set environment variables
cp .env.example .env
# Edit .env with your configuration

# Run the worker
python worker.py
```

### Docker Deployment

```bash
# Build the image
docker build -t v2t-python-worker:latest .

# Run the worker
docker run -d \
  -e TEMPORAL_HOST=temporal:7233 \
  -e OPENAI_API_KEY=your-key \
  -e MINIO_ENDPOINT=minio:9000 \
  v2t-python-worker:latest
```

### Add to Docker Compose

Add this service to your `docker-compose.yml`:

```yaml
python-worker:
  image: v2t-python-worker:latest
  environment:
    - TEMPORAL_HOST=temporal:7233
    - MINIO_ENDPOINT=minio-nginx:9000
    - OPENAI_API_KEY=${OPENAI_API_KEY}
  depends_on:
    - temporal
    - minio-nginx
  networks:
    - v2t-network
  deploy:
    replicas: 1
```

## Usage

### From Go Workflows

You can call Python activities from Go workflows:

```go
// In your Go workflow
var result map[string]interface{}
err := workflow.ExecuteActivity(ctx, "transcribe_with_whisper_python", request).Get(ctx, &result)
```

### Python-specific Workflows

Submit Python workflows using the CLI:

```bash
# Use the Python ML workflow
./v2t-distributed transcribe audio.mp3 \
  --workflow PythonTranscriptionWorkflow \
  --post-process \
  --add-punctuation
```

## Activities

### transcribe_with_whisper_python
- Uses OpenAI Whisper Python library
- Supports GPU acceleration
- Configurable model sizes (tiny, base, small, medium, large)

### transcribe_with_openai_api
- Alternative implementation using OpenAI API
- Useful for comparison and fallback

### advanced_audio_analysis
- Speaker diarization (planned)
- Emotion detection (planned)
- Audio quality assessment
- Background noise analysis

### post_process_transcription
- Punctuation restoration
- Text summarization
- Key phrase extraction
- Translation

## Extending

Add new ML capabilities:

```python
@activity.defn
async def custom_ml_activity(request: Dict[str, Any]) -> Dict[str, Any]:
    # Your ML code here
    pass

# Register in main()
activities=[
    # ... existing activities
    custom_ml_activity,
]
```

## Performance Tuning

### GPU Support

```python
# Enable GPU in Whisper
whisper_model = whisper.load_model(model_size, device="cuda")

# Use fp16 for faster inference
result = whisper_model.transcribe(audio, fp16=True)
```

### Batch Processing

```python
@activity.defn
async def batch_transcribe(files: List[str]) -> List[Dict[str, Any]]:
    # Process multiple files efficiently
    pass
```

## Monitoring

View Python worker status in Temporal UI:
- Worker identity: `v2t-python-worker-{hostname}`
- Task queue: `v2t-transcription-queue`
- Available activities and workflows

## Troubleshooting

### Common Issues

1. **CUDA/GPU not available**
   ```bash
   # Check GPU availability
   python -c "import torch; print(torch.cuda.is_available())"
   ```

2. **Memory issues with large models**
   ```python
   # Use smaller model or increase memory
   whisper_model = whisper.load_model("base")  # Instead of "large"
   ```

3. **MinIO connection issues**
   ```python
   # Test MinIO connection
   minio_client.list_buckets()
   ```

## Future Enhancements

- [ ] Speaker diarization with pyannote
- [ ] Real-time transcription support
- [ ] Custom fine-tuned models
- [ ] Batch processing optimization
- [ ] Streaming transcription
- [ ] Multi-language detection