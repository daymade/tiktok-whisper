# v2t Distributed System Architecture

## Overview

The v2t distributed transcription system leverages Temporal for workflow orchestration, MinIO for distributed storage, and supports multiple transcription providers across a cluster of machines.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Control Plane (M2)                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐ │
│  │  Temporal   │  │   Temporal   │  │   MinIO      │  │   Prometheus   │ │
│  │   Server    │  │     UI       │  │  Cluster     │  │   + Grafana    │ │
│  └──────┬──────┘  └──────────────┘  └──────┬───────┘  └────────┬────────┘ │
│         │                                   │                    │          │
└─────────┼───────────────────────────────────┼────────────────────┼─────────┘
          │                                   │                    │
          │         gRPC/HTTP                 │ S3 API            │ Metrics
          │                                   │                    │
┌─────────┴───────────────────────────────────┴────────────────────┴─────────┐
│                           Data Plane (Workers)                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐           │
│  │   M2 Workers    │  │   M4 Worker #1  │  │   M4 Worker #2  │           │
│  │  ┌───────────┐  │  │  ┌───────────┐  │  │  ┌───────────┐  │           │
│  │  │ v2t Core  │  │  │  │ v2t Core  │  │  │  │ v2t Core  │  │           │
│  │  │ Providers │  │  │  │ Providers │  │  │  │ Providers │  │           │
│  │  └───────────┘  │  │  └───────────┘  │  │  └───────────┘  │           │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘           │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Control Plane Components

#### Temporal Server
- **Purpose**: Workflow orchestration and state management
- **Components**:
  - Frontend Service: gRPC API for workflow operations
  - History Service: Event sourcing for workflow state
  - Matching Service: Task distribution to workers
  - Worker Service: Background jobs and timers
- **Storage**: PostgreSQL for workflow metadata
- **Port**: 7233 (gRPC)

#### MinIO Cluster
- **Purpose**: Distributed object storage for audio files and results
- **Configuration**: 3-node erasure-coded cluster
- **Features**:
  - S3-compatible API
  - Automatic replication
  - Fault tolerance with 1 node failure
- **Ports**: 9000 (API), 9001 (Console)

#### Monitoring Stack
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Visualization and dashboards
- **Node Exporter**: System metrics
- **Alerts**: Worker health, provider availability, system resources

### 2. Worker Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        v2t Worker Process                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Temporal Worker                       │   │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │  Workflow   │  │  Activity    │  │   Health     │  │   │
│  │  │  Executor   │  │  Executor    │  │   Server     │  │   │
│  │  └──────┬──────┘  └──────┬───────┘  └──────────────┘  │   │
│  └─────────┼────────────────┼──────────────────────────────┘   │
│            │                │                                   │
│  ┌─────────▼────────────────▼───────────────────────────────┐  │
│  │              v2t Provider Framework                       │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────────┐ │  │
│  │  │ WhisperCpp │  │   OpenAI   │  │   SSH Whisper     │ │  │
│  │  │  Provider  │  │  Provider  │  │    Provider       │ │  │
│  │  └────────────┘  └────────────┘  └────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 3. Workflow Patterns

#### Single File Transcription
```
Client → Submit Workflow → Temporal Server
                              ↓
                         Schedule Task → Worker Pool
                                            ↓
                                    1. Download from MinIO
                                    2. Select Provider
                                    3. Transcribe
                                    4. Upload Result
                                    5. Update Database
                                            ↓
                                        Complete ← Result
```

#### Batch Processing with Parallelism
```
Batch Request → Split into Child Workflows
                    ↓
            ┌───────┼───────┐
            ▼       ▼       ▼
         File 1  File 2  File N
            │       │       │
            └───────┴───────┘
                    ▼
            Aggregate Results
```

#### Provider Fallback Chain
```
Primary Provider (whisper_cpp)
    ↓ (failure)
Secondary Provider (openai)
    ↓ (failure)
Tertiary Provider (elevenlabs)
    ↓
Success or Final Failure
```

## Data Flow

### 1. File Upload Flow
```
User → CLI → MinIO API → Storage → Event Notification → Workflow Trigger
```

### 2. Transcription Flow
```
Workflow → Activity → MinIO Download → Local Cache
                           ↓
                    Provider Selection
                           ↓
                    Transcription Engine
                           ↓
                    Result Processing
                           ↓
                    MinIO Upload → Database Update
```

### 3. Result Retrieval
```
User → CLI → Temporal Query → Workflow State
                ↓
         MinIO Direct Access → Download Result
```

## Provider Integration

### Provider Types

1. **Local Providers**
   - whisper_cpp: Direct binary execution
   - GPU-accelerated: CUDA/Metal support

2. **Remote API Providers**
   - OpenAI: REST API with auth
   - ElevenLabs: REST API with auth
   - Whisper Server: HTTP API

3. **Hybrid Providers**
   - SSH Whisper: Remote execution via SSH
   - Custom HTTP: Generic HTTP integration

### Provider Selection Logic

```go
type ProviderSelector struct {
    Rules []SelectionRule
}

type SelectionRule struct {
    Condition  func(file FileInfo) bool
    Provider   string
    Priority   int
}

// Example rules:
// - Files > 100MB → OpenAI (better for large files)
// - Chinese audio → whisper_cpp with zh model
// - Real-time priority → GPU provider
// - Cost optimization → Local providers first
```

## Scalability Patterns

### Horizontal Scaling

1. **Worker Scaling**
   - Add workers: `docker-compose scale v2t-worker=N`
   - Auto-scaling based on queue depth
   - Resource-aware task distribution

2. **Storage Scaling**
   - MinIO erasure coding: Add storage nodes
   - Automatic rebalancing
   - Multi-site replication

3. **Database Scaling**
   - Read replicas for queries
   - Partitioning by date/user
   - Archive old transcriptions

### Vertical Scaling

1. **GPU Utilization**
   - Dedicated GPU workers
   - Batch processing on GPU
   - Model optimization (int8 quantization)

2. **Memory Optimization**
   - Streaming transcription
   - Chunked file processing
   - Provider connection pooling

## Security Architecture

### Authentication & Authorization

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   CLI Client    │────▶│  API Gateway    │────▶│    Temporal     │
│   (API Key)     │     │  (TLS + Auth)   │     │   (mTLS)        │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                          │
                                                          ▼
                                                 ┌─────────────────┐
                                                 │    Workers      │
                                                 │  (Service Auth) │
                                                 └─────────────────┘
```

### Data Protection

1. **In Transit**
   - TLS 1.3 for all communications
   - mTLS between Temporal components
   - Encrypted MinIO traffic

2. **At Rest**
   - MinIO server-side encryption
   - Database encryption
   - Secure credential storage

3. **Access Control**
   - Temporal namespace isolation
   - MinIO bucket policies
   - Provider API key management

## Monitoring & Observability

### Metrics Collection

```
Workers → Prometheus Metrics
   │         ├── Worker Health
   │         ├── Provider Status
   │         ├── Task Latency
   │         └── Resource Usage
   │
   └────▶ Custom Metrics
            ├── Transcription Duration
            ├── Provider Success Rate
            ├── Queue Depth
            └── Cost per Minute
```

### Dashboards

1. **System Overview**
   - Worker status grid
   - Total throughput
   - Error rates
   - Resource utilization

2. **Provider Analytics**
   - Provider distribution
   - Success/failure rates
   - Latency percentiles
   - Cost analysis

3. **Workflow Monitoring**
   - Active workflows
   - Completion times
   - Retry patterns
   - Failure analysis

### Alerting Rules

```yaml
Critical:
  - Worker down > 5 minutes
  - No providers available
  - Temporal disconnected
  - Storage > 90%

Warning:
  - High failure rate > 10%
  - Long workflow duration > 30m
  - Provider degradation
  - Queue backlog > 100
```

## Deployment Patterns

### Production Deployment

```
┌─────────────────────────────────────────────────┐
│                Load Balancer                     │
│                 (HAProxy/Nginx)                  │
└─────────────┬───────────────┬───────────────────┘
              │               │
     ┌────────▼────────┐  ┌───▼────────────┐
     │   Temporal      │  │    MinIO       │
     │   Cluster       │  │   Cluster      │
     │  (3 nodes)      │  │  (3+ nodes)    │
     └─────────────────┘  └────────────────┘
              │
     ┌────────┴─────────────────────────┐
     │         Worker Fleet             │
     │   ┌─────────┐  ┌─────────┐     │
     │   │ GPU     │  │  CPU    │     │
     │   │ Workers │  │ Workers │     │
     │   └─────────┘  └─────────┘     │
     └──────────────────────────────────┘
```

### Development Setup

```
Single Docker Compose:
  - Temporal (dev mode)
  - MinIO (single node)
  - 1-2 Workers
  - PostgreSQL
  - Monitoring stack
```

## Performance Optimization

### Caching Strategy

```
L1: Worker Local Cache (Hot files)
    ↓
L2: Shared Redis Cache (Frequent files)
    ↓
L3: MinIO Storage (All files)
```

### Batch Processing

```go
// Optimal batch sizes
const (
    SmallFileBatch  = 20  // < 10MB files
    MediumFileBatch = 10  // 10-50MB files
    LargeFileBatch  = 5   // > 50MB files
)
```

### Resource Allocation

```yaml
Worker Types:
  - CPU Optimized: 8 cores, 16GB RAM
  - GPU Optimized: 4 cores, 32GB RAM, 1 GPU
  - Memory Optimized: 4 cores, 64GB RAM
```

## Troubleshooting Guide

### Common Issues

1. **Worker Not Processing Tasks**
   - Check Temporal connectivity
   - Verify task queue name
   - Check provider health
   - Review worker logs

2. **Slow Transcription**
   - Check provider selection
   - Verify network latency
   - Monitor resource usage
   - Consider provider fallback

3. **Storage Issues**
   - Check MinIO cluster health
   - Verify bucket permissions
   - Monitor disk usage
   - Check network connectivity

### Debug Commands

```bash
# Check worker health
curl http://worker:8081/health

# View Temporal workflows
temporal workflow list

# Check MinIO status
mc admin info myminio

# View provider status
v2t providers status

# Monitor logs
docker-compose logs -f v2t-worker
```

## Future Enhancements

1. **Real-time Transcription**
   - WebSocket streaming
   - Chunked processing
   - Live captioning

2. **Multi-language Models**
   - Language detection
   - Specialized models
   - Translation pipeline

3. **Advanced Analytics**
   - Speaker diarization
   - Sentiment analysis
   - Keyword extraction

4. **Cost Optimization**
   - Spot instance support
   - Predictive scaling
   - Provider cost routing