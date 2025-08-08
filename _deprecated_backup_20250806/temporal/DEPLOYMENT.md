# Distributed v2t Deployment Guide

This guide will help you deploy the distributed v2t transcription system across multiple machines using Temporal and MinIO.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│  M2 Machine (Control Node)                                          │
├─────────────────────────────────────────────────────────────────────┤
│  - Temporal Server & UI         - MinIO Cluster (3 nodes)          │
│  - PostgreSQL                   - Nginx Load Balancer              │
│  - v2t Workers (2 instances)    - Redis (optional)                 │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                 ┌──────────────────┴──────────────────┐
                 │                                     │
┌────────────────▼─────────────┐      ┌───────────────▼──────────────┐
│  M4 Machine #1               │      │  M4 Machine #2               │
├──────────────────────────────┤      ├──────────────────────────────┤
│  - v2t Workers (3 instances) │      │  - v2t Workers (3 instances) │
│  - Local whisper.cpp         │      │  - Local whisper.cpp         │
│  - Node Exporter             │      │  - Node Exporter             │
└──────────────────────────────┘      └──────────────────────────────┘
```

## Prerequisites

1. Docker and Docker Compose installed on all machines
2. Network connectivity between all machines
3. Whisper.cpp binary and models available
4. API keys for cloud providers (OpenAI, Gemini, ElevenLabs)

## Step 1: Prepare the Control Node (M2)

### 1.1 Clone and Build

```bash
# On M2 machine
cd /Volumes/SSD2T/workspace/go/tiktok-whisper

# Build the worker image
docker build -f temporal/Dockerfile.worker -t v2t-worker:latest .

# Create necessary directories
mkdir -p temporal/dynamicconfig
```

### 1.2 Configure Environment

Create `.env` file in the temporal directory:

```bash
cat > temporal/.env << EOF
# API Keys
OPENAI_API_KEY=sk-your-openai-key
GEMINI_API_KEY=AIza-your-gemini-key
ELEVENLABS_API_KEY=your-elevenlabs-key

# Network Configuration
TEMPORAL_HOST=localhost:7233
MINIO_ENDPOINT=localhost:9000
EOF
```

### 1.3 Start the Control Node

```bash
cd temporal
docker-compose up -d

# Check services are running
docker-compose ps

# View logs
docker-compose logs -f temporal
```

### 1.4 Initialize MinIO

```bash
# Access MinIO console at http://localhost:9001
# Default credentials: minioadmin/minioadmin

# Create bucket using MinIO client
docker run --rm --network temporal_v2t-network \
  minio/mc alias set myminio http://minio-nginx:9000 minioadmin minioadmin

docker run --rm --network temporal_v2t-network \
  minio/mc mb myminio/v2t-transcriptions
```

## Step 2: Deploy Worker Nodes (M4 Machines)

### 2.1 Prepare Worker Machines

On each M4 machine:

```bash
# Create working directory
mkdir -p ~/v2t-worker
cd ~/v2t-worker

# Copy files from M2
scp user@m2-machine:/path/to/tiktok-whisper/temporal/docker-compose.worker.yml .
scp user@m2-machine:/path/to/tiktok-whisper/temporal/providers.yaml .

# Copy whisper.cpp binary and models
mkdir -p whisper/models
scp -r user@m2-machine:/path/to/whisper.cpp/main whisper/
scp -r user@m2-machine:/path/to/whisper.cpp/models/*.bin whisper/models/
```

### 2.2 Configure Worker Environment

Create `.env` file on each M4:

```bash
cat > .env << EOF
# API Keys (same as control node)
OPENAI_API_KEY=sk-your-openai-key
GEMINI_API_KEY=AIza-your-gemini-key
ELEVENLABS_API_KEY=your-elevenlabs-key

# Point to M2 control node
TEMPORAL_HOST=192.168.1.100:7233  # Replace with M2's IP
MINIO_ENDPOINT=192.168.1.100:9000  # Replace with M2's IP

# Worker identity
HOSTNAME=m4-1  # or m4-2 for second machine
EOF
```

### 2.3 Pull and Start Workers

```bash
# Pull the worker image from M2's registry or build locally
docker pull v2t-worker:latest

# Or load from saved image
# On M2: docker save v2t-worker:latest | gzip > v2t-worker.tar.gz
# On M4: docker load < v2t-worker.tar.gz

# Start workers
docker-compose -f docker-compose.worker.yml up -d

# Check status
docker-compose -f docker-compose.worker.yml ps
```

## Step 3: Submit Transcription Jobs

### 3.1 Build CLI Client

On your development machine:

```bash
cd temporal/client
go build -o v2t-distributed cli.go
```

### 3.2 Single File Transcription

```bash
# Transcribe a single file
./v2t-distributed transcribe /path/to/audio.mp3 \
  --provider whisper_cpp \
  --language zh

# With automatic provider selection
./v2t-distributed transcribe /path/to/audio.mp3
```

### 3.3 Batch Transcription

```bash
# Transcribe all audio files in a directory
./v2t-distributed batch /path/to/audio/directory \
  --parallel 10 \
  --extension "mp3,wav,m4a" \
  --language en

# Check status
./v2t-distributed status batch-audio-1234567890

# List recent workflows
./v2t-distributed list --limit 20
```

## Step 4: Monitoring

### 4.1 Temporal UI

Access the Temporal Web UI at `http://m2-machine:8080`

- View running workflows
- Check worker status
- Debug failed workflows
- Monitor system health

### 4.2 MinIO Console

Access MinIO at `http://m2-machine:9001`

- Monitor storage usage
- View transcription results
- Check replication status

### 4.3 Worker Logs

```bash
# On control node
docker-compose logs -f v2t-worker

# On worker nodes
docker-compose -f docker-compose.worker.yml logs -f v2t-worker
```

## Step 5: Scaling

### Add More Workers

```bash
# Scale workers on any node
docker-compose -f docker-compose.worker.yml up -d --scale v2t-worker=5
```

### Add New Worker Node

1. Repeat Step 2 on the new machine
2. Workers automatically register with Temporal
3. Tasks are automatically distributed

## Troubleshooting

### Workers Not Connecting

```bash
# Check network connectivity
ping m2-machine
telnet m2-machine 7233

# Check Temporal server
curl http://m2-machine:7233/health

# Check worker logs
docker logs v2t-worker_1
```

### MinIO Issues

```bash
# Check MinIO cluster status
docker exec -it temporal_minio1_1 mc admin info myminio

# Check bucket access
docker exec -it temporal_minio1_1 mc ls myminio/v2t-transcriptions
```

### Performance Tuning

1. Adjust worker concurrency in docker-compose:
   ```yaml
   environment:
     - MAX_CONCURRENT_ACTIVITIES=20
   ```

2. Increase MinIO performance:
   ```yaml
   environment:
     - MINIO_CACHE_DRIVES="/cache1,/cache2"
     - MINIO_CACHE_QUOTA=80
   ```

## Best Practices

1. **Resource Allocation**: Reserve CPU/memory for workers based on workload
2. **Network**: Use dedicated network for cluster communication
3. **Storage**: Use SSD for MinIO data directories
4. **Monitoring**: Set up Prometheus/Grafana for production
5. **Backup**: Regular backups of MinIO data and Temporal database

## Security Considerations

1. **API Keys**: Use secrets management (Docker secrets, Vault)
2. **Network**: Use TLS for inter-node communication
3. **Access Control**: Configure MinIO policies and Temporal authorization
4. **Firewall**: Restrict ports to trusted networks only

## Next Steps

1. Set up monitoring with Prometheus/Grafana
2. Configure autoscaling based on queue depth
3. Implement backup and disaster recovery
4. Add more transcription providers
5. Create Python workers for ML workloads