# v2t Distributed System Performance Tuning Guide

## Overview

This guide provides comprehensive performance tuning recommendations for the v2t distributed transcription system.

## Baseline Performance Metrics

### Target SLAs
- **Single File Transcription**: < 2x real-time (30min audio in < 60min)
- **Batch Processing**: 100 files/hour per worker
- **API Latency**: p99 < 100ms for workflow submission
- **Storage Throughput**: 1GB/s aggregate read/write

### Current Benchmarks
```
Hardware: M2 Pro (8 CPU, 16GB RAM)
Provider: whisper_cpp with large-v2 model

File Size | Duration | Processing Time | Speed
----------|----------|-----------------|-------
10MB      | 5 min    | 2.5 min        | 2.0x
50MB      | 25 min   | 15 min         | 1.67x
100MB     | 50 min   | 35 min         | 1.43x
```

## Worker Optimization

### 1. CPU Optimization

#### Thread Configuration
```yaml
# Optimal thread counts by CPU type
whisper_cpp:
  settings:
    threads:
      M2_8core: 6    # Leave 2 for system
      M4_10core: 8   # Leave 2 for system
      M4_12core: 10  # Leave 2 for system
```

#### CPU Affinity
```bash
# Pin worker to specific CPUs
docker run --cpuset-cpus="0-5" v2t-worker
```

#### NUMA Optimization
```go
// For multi-socket systems
os.Setenv("OMP_PROC_BIND", "true")
os.Setenv("OMP_PLACES", "cores")
```

### 2. Memory Optimization

#### Memory Limits
```yaml
# Docker memory configuration
deploy:
  resources:
    limits:
      memory: 14G  # Leave 2GB for system
    reservations:
      memory: 8G
```

#### Garbage Collection Tuning
```go
// Reduce GC pressure
os.Setenv("GOGC", "200")  // Less frequent GC
os.Setenv("GOMEMLIMIT", "14GiB")  // Go 1.19+
```

#### Memory Pool
```go
// Reuse buffers for file operations
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 64*1024) // 64KB buffers
    },
}
```

### 3. GPU/NPU Acceleration

#### Metal Performance Shaders (Apple Silicon)
```bash
# Build whisper.cpp with CoreML
cd whisper.cpp
WHISPER_COREML=1 make -j

# Generate CoreML model
./models/generate-coreml-model.sh large-v2
```

#### GPU Memory Management
```python
# For Python workers with GPU
import torch
torch.cuda.empty_cache()  # Clear cache between jobs
torch.backends.cudnn.benchmark = True  # Enable cuDNN autotuner
```

## Provider Optimization

### 1. Provider Selection Strategy

```go
// Intelligent provider routing
type RouterConfig struct {
    Rules []RoutingRule
}

type RoutingRule struct {
    FileSize   Range
    Duration   Range
    Language   string
    Provider   string
    Priority   int
}

// Example configuration
rules := []RoutingRule{
    // Small files to local provider
    {FileSize: Range{0, 10*MB}, Provider: "whisper_cpp", Priority: 1},
    // Large files to API (better batching)
    {FileSize: Range{100*MB, MaxInt}, Provider: "openai", Priority: 1},
    // Chinese content to specialized model
    {Language: "zh", Provider: "whisper_cpp_zh", Priority: 2},
}
```

### 2. Connection Pooling

```go
// HTTP client optimization
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  true, // Audio already compressed
        ForceAttemptHTTP2:   true,
    },
    Timeout: 5 * time.Minute,
}
```

### 3. Batch Processing

```go
// Optimal batch sizes
const (
    APIBatchSize   = 5   // API providers
    LocalBatchSize = 1   // Local providers (CPU bound)
    GPUBatchSize   = 10  // GPU providers
)

// Dynamic batching based on queue depth
func calculateBatchSize(queueDepth int) int {
    switch {
    case queueDepth > 1000:
        return 20
    case queueDepth > 100:
        return 10
    default:
        return 5
    }
}
```

## Storage Optimization

### 1. MinIO Performance

#### Erasure Coding
```yaml
# Optimal erasure coding for 3 nodes
minio:
  erasure_set_drive_count: 6  # 2 drives per node
  data_blocks: 4               # Data blocks
  parity_blocks: 2             # Can lose 1 node
```

#### Storage Class
```bash
# Configure storage tiers
mc admin tier add minio FAST /mnt/nvme
mc admin tier add minio COLD /mnt/hdd

# Lifecycle rules
mc ilm add --transition-days 7 --storage-class COLD minio/v2t-transcriptions
```

#### Client Configuration
```go
// Optimized MinIO client
minioClient, _ := minio.New(endpoint, &minio.Options{
    Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
    Secure: false,
    Transport: &http.Transport{
        MaxIdleConns:       100,
        IdleConnTimeout:    90 * time.Second,
        DisableCompression: true,
        // Enable S3 Transfer Acceleration
        ExpectContinueTimeout: 1 * time.Second,
    },
})

// Enable multipart upload for large files
minioClient.PutObject(ctx, bucket, object, reader, size,
    minio.PutObjectOptions{
        PartSize: 64 * 1024 * 1024, // 64MB parts
        NumThreads: 4,               // Parallel uploads
    })
```

### 2. Caching Strategy

#### Local File Cache
```go
type FileCache struct {
    root     string
    maxSize  int64
    strategy CacheStrategy
}

// LRU eviction
type LRUCache struct {
    mu       sync.Mutex
    size     int64
    maxSize  int64
    items    map[string]*CacheItem
    eviction *list.List
}

// Cache hot files locally
func (c *FileCache) Get(key string) (io.ReadCloser, error) {
    // Check local cache first
    if file, err := c.getLocal(key); err == nil {
        c.updateAccess(key)
        return file, nil
    }
    
    // Fetch from MinIO
    file, err := c.fetchRemote(key)
    if err != nil {
        return nil, err
    }
    
    // Cache if beneficial
    if c.shouldCache(key) {
        c.putLocal(key, file)
    }
    
    return file, nil
}
```

#### Redis Cache for Metadata
```go
// Cache transcription results
type ResultCache struct {
    redis *redis.Client
    ttl   time.Duration
}

func (c *ResultCache) Get(fileHash string) (*TranscriptionResult, error) {
    data, err := c.redis.Get(ctx, fileHash).Bytes()
    if err == redis.Nil {
        return nil, nil
    }
    
    var result TranscriptionResult
    json.Unmarshal(data, &result)
    return &result, nil
}
```

## Temporal Optimization

### 1. Workflow Configuration

```go
// Optimized workflow options
workflowOptions := client.StartWorkflowOptions{
    TaskQueue: "v2t-transcription-queue",
    WorkflowExecutionTimeout: 2 * time.Hour,
    WorkflowTaskTimeout: 10 * time.Second,
    // Reduce history size
    WorkflowIDReusePolicy: client.WorkflowIDReusePolicyAllowDuplicate,
}

// Activity options
activityOptions := workflow.ActivityOptions{
    StartToCloseTimeout: 30 * time.Minute,
    HeartbeatTimeout:    30 * time.Second,
    RetryPolicy: &temporal.RetryPolicy{
        InitialInterval:    time.Second,
        BackoffCoefficient: 2.0,
        MaximumInterval:    60 * time.Second,
        MaximumAttempts:    3,
    },
    // Activity scheduling optimization
    ScheduleToCloseTimeout: 45 * time.Minute,
    ScheduleToStartTimeout: 5 * time.Minute,
}
```

### 2. Worker Tuning

```go
// Worker configuration
workerOptions := worker.Options{
    MaxConcurrentActivityExecutionSize:     20,  // Concurrent activities
    MaxConcurrentWorkflowTaskExecutionSize: 10,  // Concurrent workflows
    MaxConcurrentActivityTaskPollers:       5,   // Pollers for activities
    MaxConcurrentWorkflowTaskPollers:       2,   // Pollers for workflows
    
    // Rate limiting
    WorkerActivitiesPerSecond: 100,
    TaskQueueActivitiesPerSecond: 1000,
    
    // Sticky execution
    StickyScheduleToStartTimeout: 5 * time.Second,
    
    // Local activity optimization
    LocalActivityWorkerOnly: false,
}
```

### 3. Database Optimization

```yaml
# PostgreSQL tuning for Temporal
postgresql:
  max_connections: 200
  shared_buffers: 4GB
  effective_cache_size: 12GB
  work_mem: 64MB
  maintenance_work_mem: 1GB
  
  # Write performance
  wal_buffers: 16MB
  checkpoint_completion_target: 0.9
  max_wal_size: 4GB
  
  # Query optimization
  random_page_cost: 1.1  # SSD storage
  effective_io_concurrency: 200
```

## Network Optimization

### 1. Network Configuration

```yaml
# Docker network optimization
networks:
  v2t-network:
    driver: bridge
    driver_opts:
      com.docker.network.driver.mtu: 9000  # Jumbo frames
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### 2. TCP Tuning

```bash
# System TCP optimization
sysctl -w net.core.rmem_max=134217728
sysctl -w net.core.wmem_max=134217728
sysctl -w net.ipv4.tcp_rmem="4096 87380 134217728"
sysctl -w net.ipv4.tcp_wmem="4096 65536 134217728"
sysctl -w net.ipv4.tcp_congestion_control=bbr
```

### 3. Load Balancing

```nginx
# Nginx optimization for MinIO
upstream minio_backend {
    least_conn;
    keepalive 32;
    
    server minio1:9000 max_fails=1 fail_timeout=10s;
    server minio2:9000 max_fails=1 fail_timeout=10s;
    server minio3:9000 max_fails=1 fail_timeout=10s;
}

server {
    listen 9000;
    
    # Buffer optimization
    client_body_buffer_size 16M;
    client_max_body_size 5G;
    
    # Timeout optimization
    proxy_connect_timeout 300s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;
    
    # Keep-alive
    keepalive_timeout 65;
    keepalive_requests 100;
    
    location / {
        proxy_pass http://minio_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        
        # Buffering off for streaming
        proxy_buffering off;
        proxy_request_buffering off;
    }
}
```

## Monitoring & Profiling

### 1. Performance Metrics

```go
// Custom metrics
var (
    transcriptionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "v2t_transcription_duration_seconds",
            Help: "Transcription duration by provider",
            Buckets: []float64{10, 30, 60, 120, 300, 600, 1800},
        },
        []string{"provider", "file_size_range"},
    )
    
    providerThroughput = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "v2t_provider_throughput_mbps",
            Help: "Provider throughput in MB/s",
        },
        []string{"provider"},
    )
)
```

### 2. Profiling

```go
// Enable profiling endpoints
import _ "net/http/pprof"

go func() {
    http.ListenAndServe("localhost:6060", nil)
}()

// CPU profiling
// go tool pprof http://localhost:6060/debug/pprof/profile

// Memory profiling
// go tool pprof http://localhost:6060/debug/pprof/heap

// Goroutine profiling
// go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### 3. Tracing

```go
// OpenTelemetry tracing
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

tracer := otel.Tracer("v2t-worker")

func TranscribeWithTracing(ctx context.Context, file string) error {
    ctx, span := tracer.Start(ctx, "transcribe",
        trace.WithAttributes(
            attribute.String("file", file),
            attribute.Int64("size", fileSize),
        ),
    )
    defer span.End()
    
    // Transcription logic
    return transcribe(ctx, file)
}
```

## Optimization Checklist

### Pre-deployment
- [ ] Build whisper.cpp with optimizations (AVX2, CoreML)
- [ ] Configure thread counts based on CPU
- [ ] Set memory limits and GC tuning
- [ ] Configure MinIO erasure coding
- [ ] Optimize PostgreSQL for Temporal
- [ ] Set up monitoring and alerting

### Runtime
- [ ] Monitor CPU and memory usage
- [ ] Check provider success rates
- [ ] Analyze transcription latencies
- [ ] Review storage I/O patterns
- [ ] Monitor network throughput
- [ ] Check cache hit rates

### Periodic
- [ ] Profile CPU and memory usage
- [ ] Analyze slow queries
- [ ] Review provider selection logic
- [ ] Optimize batch sizes
- [ ] Clean up old data
- [ ] Update model versions

## Performance Testing

### Load Testing Script
```bash
#!/bin/bash
# Load test with increasing concurrency

for workers in 1 5 10 20 50; do
    echo "Testing with $workers concurrent workers..."
    
    # Submit workflows
    for i in $(seq 1 $workers); do
        v2t-distributed transcribe test-$i.mp3 &
    done
    
    # Wait and collect metrics
    wait
    
    # Query metrics
    curl -s http://localhost:9090/api/v1/query?query=v2t_transcription_duration_seconds
done
```

### Baseline Establishment
```sql
-- Query p50, p95, p99 latencies
SELECT 
    provider,
    PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY duration) as p50,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration) as p95,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration) as p99,
    COUNT(*) as total
FROM transcriptions
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY provider;
```