# Dual Embedding System Implementation Plan (SOLID + TDD)

## Executive Summary

This document outlines a Test-Driven Development (TDD) approach to implementing a dual-embedding system supporting both OpenAI (1536 dimensions) and Google Gemini (768 dimensions) embeddings. The design follows SOLID principles throughout, ensuring maintainable, extensible, and testable code.

## 🎯 System Goals

### Primary Objectives
- Store both OpenAI and Gemini embeddings for each transcription
- Enable A/B testing between embedding providers
- Support fallback when one service is unavailable
- Allow cost optimization through selective embedding generation
- Maintain backwards compatibility while adding new providers

### Key Benefits of Dual Embeddings
1. **Redundancy**: If one API fails, use the other
2. **Comparison**: Evaluate which embeddings work better for your use case
3. **Migration**: Gradually transition from one provider to another
4. **Cost Optimization**: Use expensive embeddings only when necessary
5. **Feature Testing**: Different embeddings for different features

## 🗄️ Database Schema Design

### Enhanced Transcriptions Table

```sql
-- Add dual embedding columns with separate dimensions
ALTER TABLE transcriptions ADD COLUMN embedding_openai vector(1536);
ALTER TABLE transcriptions ADD COLUMN embedding_gemini vector(768);

-- Metadata for OpenAI embeddings
ALTER TABLE transcriptions ADD COLUMN embedding_openai_model varchar(50);
ALTER TABLE transcriptions ADD COLUMN embedding_openai_created_at timestamp;
ALTER TABLE transcriptions ADD COLUMN embedding_openai_version integer DEFAULT 1;
ALTER TABLE transcriptions ADD COLUMN embedding_openai_status varchar(20) DEFAULT 'pending';
ALTER TABLE transcriptions ADD COLUMN embedding_openai_error text;

-- Metadata for Gemini embeddings
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_model varchar(50);
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_created_at timestamp;
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_version integer DEFAULT 1;
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_status varchar(20) DEFAULT 'pending';
ALTER TABLE transcriptions ADD COLUMN embedding_gemini_error text;

-- Search configuration
ALTER TABLE transcriptions ADD COLUMN primary_embedding_provider varchar(20) DEFAULT 'openai';
ALTER TABLE transcriptions ADD COLUMN embedding_sync_status varchar(20) DEFAULT 'pending';

-- Performance optimization indexes
CREATE INDEX transcriptions_embedding_openai_idx ON transcriptions 
USING hnsw (embedding_openai vector_cosine_ops) WHERE embedding_openai IS NOT NULL;

CREATE INDEX transcriptions_embedding_gemini_idx ON transcriptions 
USING hnsw (embedding_gemini vector_cosine_ops) WHERE embedding_gemini IS NOT NULL;

CREATE INDEX transcriptions_openai_status_idx ON transcriptions (embedding_openai_status);
CREATE INDEX transcriptions_gemini_status_idx ON transcriptions (embedding_gemini_status);
```

### Supporting Tables

```sql
-- Embedding provider configuration
CREATE TABLE embedding_providers (
    id serial PRIMARY KEY,
    provider_name varchar(50) NOT NULL UNIQUE,
    model_name varchar(100) NOT NULL,
    dimension integer NOT NULL,
    is_active boolean DEFAULT true,
    cost_per_1k_tokens decimal(10,6),
    rate_limit_per_minute integer,
    created_at timestamp DEFAULT now()
);

-- Insert default providers
INSERT INTO embedding_providers (provider_name, model_name, dimension, cost_per_1k_tokens, rate_limit_per_minute) VALUES
('openai', 'text-embedding-ada-002', 1536, 0.0001, 3000),
('gemini', 'models/embedding-001', 768, 0.0001, 1500);

-- Embedding generation batches for tracking
CREATE TABLE embedding_batches (
    id serial PRIMARY KEY,
    batch_id varchar(50) NOT NULL UNIQUE,
    provider varchar(50) NOT NULL,
    started_at timestamp DEFAULT now(),
    completed_at timestamp,
    total_items integer NOT NULL,
    processed_items integer DEFAULT 0,
    failed_items integer DEFAULT 0,
    status varchar(20) DEFAULT 'running'
);
```

## 🏛️ SOLID Architecture Design

### 1. Single Responsibility Principle (SRP)

Each component has exactly one reason to change:

```go
// EmbeddingProvider - Only responsible for generating embeddings
type EmbeddingProvider interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// VectorStorage - Only responsible for storing vectors
type VectorStorage interface {
    StoreEmbedding(ctx context.Context, id int, provider string, embedding []float32) error
}

// SimilarityCalculator - Only responsible for similarity calculations
type SimilarityCalculator interface {
    Calculate(a, b []float32) (float32, error)
}

// ProgressTracker - Only responsible for tracking progress
type ProgressTracker interface {
    UpdateProgress(processed, total int) error
    GetProgress() (processed, total int, err error)
}
```

### 2. Open/Closed Principle (OCP)

System is open for extension but closed for modification:

```go
// Base interface - closed for modification
type EmbeddingProvider interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
    GetProviderInfo() ProviderInfo
}

// Provider info
type ProviderInfo struct {
    Name      string
    Model     string
    Dimension int
}

// Open for extension - add new providers without changing existing code
type OpenAIProvider struct {
    client *openai.Client
    model  string
}

type GeminiProvider struct {
    client *genai.Client
    model  string
}

type MockProvider struct {
    dimension int
}

// Future providers can be added without modifying existing code
type CohereProvider struct { /* ... */ }
type HuggingFaceProvider struct { /* ... */ }
```

### 3. Liskov Substitution Principle (LSP)

All providers are perfectly substitutable:

```go
// Any provider can be used wherever EmbeddingProvider is expected
func ProcessTranscription(provider EmbeddingProvider, text string) error {
    embedding, err := provider.GenerateEmbedding(context.Background(), text)
    if err != nil {
        return err
    }
    // Process embedding...
    return nil
}

// Works with any provider
ProcessTranscription(&OpenAIProvider{}, text)
ProcessTranscription(&GeminiProvider{}, text)
ProcessTranscription(&MockProvider{}, text)
```

### 4. Interface Segregation Principle (ISP)

Multiple specific interfaces instead of one large interface:

```go
// Segregated interfaces for different responsibilities
type EmbeddingGenerator interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

type BatchEmbeddingGenerator interface {
    GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

type EmbeddingMetadata interface {
    GetProviderName() string
    GetModelName() string
    GetDimension() int
}

type RateLimiter interface {
    Wait(ctx context.Context) error
    Limit() rate.Limit
}

type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}

// Providers implement only what they need
type OpenAIProvider struct{}

func (p *OpenAIProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) { /* ... */ }
func (p *OpenAIProvider) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) { /* ... */ }
func (p *OpenAIProvider) GetProviderName() string { return "openai" }
func (p *OpenAIProvider) GetModelName() string { return "text-embedding-ada-002" }
func (p *OpenAIProvider) GetDimension() int { return 1536 }
```

### 5. Dependency Inversion Principle (DIP)

High-level modules depend on abstractions:

```go
// High-level orchestrator depends on interfaces, not concrete implementations
type EmbeddingOrchestrator struct {
    providers map[string]EmbeddingProvider  // Interface, not concrete type
    storage   VectorStorage                 // Interface, not concrete type
    logger    Logger                        // Interface, not concrete type
}

// Constructor accepts interfaces
func NewEmbeddingOrchestrator(
    providers map[string]EmbeddingProvider,
    storage VectorStorage,
    logger Logger,
) *EmbeddingOrchestrator {
    return &EmbeddingOrchestrator{
        providers: providers,
        storage:   storage,
        logger:    logger,
    }
}

// Dependency injection at runtime
orchestrator := NewEmbeddingOrchestrator(
    map[string]EmbeddingProvider{
        "openai": openaiProvider,
        "gemini": geminiProvider,
    },
    pgVectorStorage,
    structuredLogger,
)
```

## 🧪 TDD Implementation Cycles

### Cycle 1: Basic EmbeddingProvider Interface

#### 1.1 RED - Write Failing Test
```go
// embedding_provider_test.go
func TestEmbeddingProviderInterface(t *testing.T) {
    // This test will fail - interface doesn't exist yet
    var provider EmbeddingProvider
    provider = &MockProvider{}
    
    embedding, err := provider.GenerateEmbedding(context.Background(), "test text")
    
    assert.NoError(t, err)
    assert.NotNil(t, embedding)
}
```

#### 1.2 GREEN - Minimal Implementation
```go
// embedding_provider.go
type EmbeddingProvider interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// mock_provider.go
type MockProvider struct{}

func (m *MockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
    return make([]float32, 768), nil
}
```

#### 1.3 REFACTOR - Improve Design
```go
// embedding_provider.go
type EmbeddingProvider interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
    GetProviderInfo() ProviderInfo
}

type ProviderInfo struct {
    Name      string
    Model     string
    Dimension int
}
```

### Cycle 2: Mock Provider with Deterministic Output

#### 2.1 RED - Test Deterministic Behavior
```go
func TestMockProviderDeterministic(t *testing.T) {
    provider := NewMockProvider(768)
    
    // Same input should produce same output
    embedding1, _ := provider.GenerateEmbedding(context.Background(), "hello world")
    embedding2, _ := provider.GenerateEmbedding(context.Background(), "hello world")
    
    assert.Equal(t, embedding1, embedding2)
    
    // Different input should produce different output
    embedding3, _ := provider.GenerateEmbedding(context.Background(), "goodbye world")
    assert.NotEqual(t, embedding1, embedding3)
}
```

#### 2.2 GREEN - Implement Deterministic Mock
```go
type MockProvider struct {
    dimension int
}

func NewMockProvider(dimension int) *MockProvider {
    return &MockProvider{dimension: dimension}
}

func (m *MockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
    // Use SHA256 for deterministic output
    hash := sha256.Sum256([]byte(text))
    embedding := make([]float32, m.dimension)
    
    for i := 0; i < m.dimension; i++ {
        byteIndex := i % len(hash)
        embedding[i] = (float32(hash[byteIndex]) / 255.0) * 2 - 1
    }
    
    return embedding, nil
}

func (m *MockProvider) GetProviderInfo() ProviderInfo {
    return ProviderInfo{
        Name:      "mock",
        Model:     "mock-model",
        Dimension: m.dimension,
    }
}
```

### Cycle 3: Vector Storage Interface

#### 3.1 RED - Test Storage Interface
```go
func TestVectorStorage(t *testing.T) {
    storage := NewMockVectorStorage()
    embedding := []float32{0.1, 0.2, 0.3}
    
    err := storage.StoreEmbedding(context.Background(), 1, "openai", embedding)
    assert.NoError(t, err)
    
    retrieved, err := storage.GetEmbedding(context.Background(), 1, "openai")
    assert.NoError(t, err)
    assert.Equal(t, embedding, retrieved)
}
```

#### 3.2 GREEN - Implement Storage
```go
type VectorStorage interface {
    StoreEmbedding(ctx context.Context, id int, provider string, embedding []float32) error
    GetEmbedding(ctx context.Context, id int, provider string) ([]float32, error)
}

type MockVectorStorage struct {
    embeddings map[string][]float32
    mu         sync.RWMutex
}

func NewMockVectorStorage() *MockVectorStorage {
    return &MockVectorStorage{
        embeddings: make(map[string][]float32),
    }
}

func (s *MockVectorStorage) StoreEmbedding(ctx context.Context, id int, provider string, embedding []float32) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    key := fmt.Sprintf("%d-%s", id, provider)
    s.embeddings[key] = embedding
    return nil
}

func (s *MockVectorStorage) GetEmbedding(ctx context.Context, id int, provider string) ([]float32, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    key := fmt.Sprintf("%d-%s", id, provider)
    embedding, exists := s.embeddings[key]
    if !exists {
        return nil, errors.New("embedding not found")
    }
    return embedding, nil
}
```

### Cycle 4: Dual Embedding Storage

#### 4.1 RED - Test Dual Storage
```go
func TestDualEmbeddingStorage(t *testing.T) {
    storage := NewPgVectorStorage(db)
    
    openaiEmbedding := make([]float32, 1536)
    geminiEmbedding := make([]float32, 768)
    
    // Store both embeddings
    err := storage.StoreDualEmbeddings(context.Background(), 1, 
        openaiEmbedding, geminiEmbedding)
    assert.NoError(t, err)
    
    // Retrieve both
    retrieved, err := storage.GetDualEmbeddings(context.Background(), 1)
    assert.NoError(t, err)
    assert.Equal(t, openaiEmbedding, retrieved.OpenAI)
    assert.Equal(t, geminiEmbedding, retrieved.Gemini)
}
```

#### 4.2 GREEN - Implement Dual Storage
```go
type DualEmbedding struct {
    OpenAI []float32
    Gemini []float32
}

type DualVectorStorage interface {
    StoreDualEmbeddings(ctx context.Context, id int, openai, gemini []float32) error
    GetDualEmbeddings(ctx context.Context, id int) (*DualEmbedding, error)
}

type PgVectorStorage struct {
    db *sql.DB
}

func (s *PgVectorStorage) StoreDualEmbeddings(ctx context.Context, id int, openai, gemini []float32) error {
    query := `
        UPDATE transcriptions 
        SET embedding_openai = $1,
            embedding_openai_model = 'text-embedding-ada-002',
            embedding_openai_created_at = now(),
            embedding_openai_status = 'completed',
            embedding_gemini = $2,
            embedding_gemini_model = 'models/embedding-001',
            embedding_gemini_created_at = now(),
            embedding_gemini_status = 'completed'
        WHERE id = $3
    `
    
    openaiStr := vectorToString(openai)
    geminiStr := vectorToString(gemini)
    
    _, err := s.db.ExecContext(ctx, query, openaiStr, geminiStr, id)
    return err
}
```

### Cycle 5: Similarity Calculator

#### 5.1 RED - Test Similarity Calculation
```go
func TestCosineSimilarity(t *testing.T) {
    calc := NewCosineSimilarityCalculator()
    
    // Test identical vectors
    a := []float32{1, 0, 0}
    similarity, err := calc.Calculate(a, a)
    assert.NoError(t, err)
    assert.Equal(t, float32(1.0), similarity)
    
    // Test orthogonal vectors
    b := []float32{0, 1, 0}
    similarity, err = calc.Calculate(a, b)
    assert.NoError(t, err)
    assert.Equal(t, float32(0.0), similarity)
}
```

#### 5.2 GREEN - Implement Calculator
```go
type SimilarityCalculator interface {
    Calculate(a, b []float32) (float32, error)
}

type CosineSimilarityCalculator struct{}

func NewCosineSimilarityCalculator() *CosineSimilarityCalculator {
    return &CosineSimilarityCalculator{}
}

func (c *CosineSimilarityCalculator) Calculate(a, b []float32) (float32, error) {
    if len(a) != len(b) {
        return 0, errors.New("vectors must have same dimension")
    }
    
    var dotProduct, normA, normB float32
    for i := range a {
        dotProduct += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    
    if normA == 0 || normB == 0 {
        return 0, nil
    }
    
    return dotProduct / (float32(math.Sqrt(float64(normA))) * 
                        float32(math.Sqrt(float64(normB)))), nil
}
```

### Cycle 6: Embedding Orchestrator

#### 6.1 RED - Test Orchestrator
```go
func TestEmbeddingOrchestrator(t *testing.T) {
    // Setup mocks
    providers := map[string]EmbeddingProvider{
        "openai": NewMockProvider(1536),
        "gemini": NewMockProvider(768),
    }
    storage := NewMockVectorStorage()
    logger := NewMockLogger()
    
    orchestrator := NewEmbeddingOrchestrator(providers, storage, logger)
    
    // Test processing
    err := orchestrator.ProcessTranscription(context.Background(), 1, "test text")
    assert.NoError(t, err)
    
    // Verify both embeddings were generated and stored
    status, err := orchestrator.GetEmbeddingStatus(context.Background(), 1)
    assert.NoError(t, err)
    assert.True(t, status.OpenAICompleted)
    assert.True(t, status.GeminiCompleted)
}
```

#### 6.2 GREEN - Implement Orchestrator
```go
type EmbeddingOrchestrator struct {
    providers map[string]EmbeddingProvider
    storage   VectorStorage
    logger    Logger
}

func NewEmbeddingOrchestrator(
    providers map[string]EmbeddingProvider,
    storage VectorStorage,
    logger Logger,
) *EmbeddingOrchestrator {
    return &EmbeddingOrchestrator{
        providers: providers,
        storage:   storage,
        logger:    logger,
    }
}

func (o *EmbeddingOrchestrator) ProcessTranscription(ctx context.Context, id int, text string) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(o.providers))
    
    for providerName, provider := range o.providers {
        wg.Add(1)
        go func(name string, p EmbeddingProvider) {
            defer wg.Done()
            
            embedding, err := p.GenerateEmbedding(ctx, text)
            if err != nil {
                o.logger.Error("Failed to generate embedding", 
                    "provider", name, "error", err)
                errors <- err
                return
            }
            
            err = o.storage.StoreEmbedding(ctx, id, name, embedding)
            if err != nil {
                o.logger.Error("Failed to store embedding", 
                    "provider", name, "error", err)
                errors <- err
                return
            }
            
            o.logger.Info("Successfully processed embedding", 
                "provider", name, "id", id)
        }(providerName, provider)
    }
    
    wg.Wait()
    close(errors)
    
    // Check if any errors occurred
    var errs []error
    for err := range errors {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("embedding generation failed: %v", errs)
    }
    
    return nil
}
```

### Cycle 7: Batch Processing

#### 7.1 RED - Test Batch Processing
```go
func TestBatchProcessor(t *testing.T) {
    processor := NewBatchProcessor(orchestrator, repository)
    
    // Create test transcriptions
    transcriptions := []Transcription{
        {ID: 1, Text: "First text"},
        {ID: 2, Text: "Second text"},
        {ID: 3, Text: "Third text"},
    }
    
    // Process batch
    results, err := processor.ProcessBatch(context.Background(), transcriptions, 2)
    assert.NoError(t, err)
    assert.Equal(t, 3, results.Processed)
    assert.Equal(t, 0, results.Failed)
}
```

#### 7.2 GREEN - Implement Batch Processor
```go
type BatchProcessor struct {
    orchestrator *EmbeddingOrchestrator
    repository   TranscriptionRepository
    batchSize    int
    concurrency  int
}

type BatchResult struct {
    Processed int
    Failed    int
    Errors    []error
}

func (p *BatchProcessor) ProcessBatch(ctx context.Context, transcriptions []Transcription, batchSize int) (*BatchResult, error) {
    result := &BatchResult{}
    
    // Process in batches
    for i := 0; i < len(transcriptions); i += batchSize {
        end := min(i+batchSize, len(transcriptions))
        batch := transcriptions[i:end]
        
        // Process batch concurrently
        var wg sync.WaitGroup
        for _, t := range batch {
            wg.Add(1)
            go func(transcription Transcription) {
                defer wg.Done()
                
                err := p.orchestrator.ProcessTranscription(ctx, transcription.ID, transcription.Text)
                if err != nil {
                    result.Failed++
                    result.Errors = append(result.Errors, err)
                } else {
                    result.Processed++
                }
            }(t)
        }
        wg.Wait()
    }
    
    return result, nil
}
```

## 🚀 Implementation Roadmap

### Phase 1: Foundation (Week 1)
1. **TDD Cycle 1-2**: Basic interfaces and mock providers
2. **Database Migration**: Add dual embedding columns
3. **Configuration**: Setup provider configurations

### Phase 2: Core Services (Week 2)
1. **TDD Cycle 3-4**: Storage implementation
2. **Provider Implementation**: OpenAI and Gemini providers
3. **Error Handling**: Comprehensive error handling

### Phase 3: Orchestration (Week 3)
1. **TDD Cycle 5-6**: Orchestrator and similarity
2. **Batch Processing**: Implement batch processor
3. **Progress Tracking**: Add progress monitoring

### Phase 4: CLI Integration (Week 4)
1. **TDD Cycle 7-8**: CLI commands
2. **User Interface**: Interactive features
3. **Documentation**: User guides

### Phase 5: Testing & Optimization (Week 5)
1. **Integration Tests**: Full system testing
2. **Performance Testing**: Load and stress testing
3. **Optimization**: Performance tuning

## 📋 CLI Commands for Dual Embeddings

```bash
# Generate embeddings using specific provider
v2t embed generate --provider openai --all
v2t embed generate --provider gemini --user "username"

# Generate using both providers
v2t embed generate --provider both --all

# User-specific embedding generation (✅ IMPLEMENTED)
v2t embed generate --user "username" --provider gemini --batch-size 10
v2t embed generate --user "username" --provider both

# Check embedding status with user distribution (✅ IMPLEMENTED)
v2t embed status

# Search for similar transcriptions (✅ IMPLEMENTED)
v2t embed search --text "search query" --limit 10 --provider gemini

# Calculate similarity between transcriptions (✅ IMPLEMENTED)
v2t embed similarity --id1 123 --id2 456 --provider gemini

# Find duplicates with user filtering (✅ IMPLEMENTED)
v2t embed duplicates --user "username" --threshold 0.95 --provider gemini

# Compare embeddings from different providers
v2t embed compare --id 123
v2t embed compare --provider1 openai --provider2 gemini --top 10
```

## 🏗️ Project Structure Following SOLID

```
internal/app/
├── embedding/
│   ├── provider/
│   │   ├── interface.go          # EmbeddingProvider interface
│   │   ├── openai.go            # OpenAI implementation
│   │   ├── gemini.go            # Gemini implementation
│   │   ├── mock.go              # Mock implementation
│   │   └── factory.go           # Provider factory
│   ├── orchestrator/
│   │   ├── orchestrator.go      # Main orchestrator
│   │   ├── batch_processor.go   # Batch processing
│   │   └── progress_tracker.go  # Progress tracking
│   └── similarity/
│       ├── calculator.go        # Similarity interfaces
│       ├── cosine.go           # Cosine implementation
│       └── euclidean.go        # Euclidean implementation
├── storage/
│   ├── vector/
│   │   ├── interface.go        # VectorStorage interface
│   │   ├── pgvector.go        # PostgreSQL implementation
│   │   └── mock.go            # Mock implementation
│   └── repository/
│       ├── transcription.go    # Transcription repository
│       └── batch.go           # Batch tracking
├── cmd/
│   └── v2t/
│       ├── embed.go           # Embedding commands
│       ├── similar.go         # Similarity commands
│       └── provider.go        # Provider commands
└── wire.go                    # Dependency injection
```

## 🧪 Testing Strategy

### Unit Tests (80% coverage target)
- Provider implementations
- Storage operations
- Similarity calculations
- Orchestrator logic

### Integration Tests
- End-to-end embedding generation
- Database operations
- Provider switching
- Error recovery

### Performance Tests
- Batch processing speed
- Concurrent operations
- Memory usage
- Database query performance

### A/B Testing Framework
```go
type ABTest struct {
    ID           string
    ProviderA    string
    ProviderB    string
    SampleSize   int
    Metrics      ABTestMetrics
}

type ABTestMetrics struct {
    AvgLatencyA      time.Duration
    AvgLatencyB      time.Duration
    SuccessRateA     float64
    SuccessRateB     float64
    SimilarityScores []float32  // Cross-provider similarity
}
```

## 🔧 Configuration

### Environment Variables
```bash
# Provider configurations
OPENAI_API_KEY=sk-...
OPENAI_MODEL=text-embedding-ada-002
OPENAI_RATE_LIMIT=3000

GEMINI_API_KEY=...
GEMINI_MODEL=models/embedding-001
GEMINI_RATE_LIMIT=1500

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=postgres
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_password_here

# Processing
BATCH_SIZE=10
CONCURRENCY=5
PRIMARY_PROVIDER=openai
FALLBACK_PROVIDER=gemini

# Feature flags
ENABLE_DUAL_EMBEDDINGS=true
ENABLE_AB_TESTING=false
ENABLE_FALLBACK=true
```

## 📊 Monitoring & Observability

### Metrics to Track
1. **Provider Metrics**
   - Requests per minute
   - Average latency
   - Error rates
   - Token usage

2. **Processing Metrics**
   - Embeddings generated per provider
   - Batch processing time
   - Queue depth
   - Failure rates

3. **Storage Metrics**
   - Vector operations per second
   - Index performance
   - Storage space usage
   - Query latency

### Health Checks
```go
type HealthCheck struct {
    Provider   string
    Status     string
    Latency    time.Duration
    LastCheck  time.Time
    ErrorCount int
}

func (o *EmbeddingOrchestrator) HealthCheck(ctx context.Context) []HealthCheck {
    checks := []HealthCheck{}
    
    for name, provider := range o.providers {
        start := time.Now()
        err := provider.HealthCheck(ctx)
        
        checks = append(checks, HealthCheck{
            Provider:  name,
            Status:    getStatus(err),
            Latency:   time.Since(start),
            LastCheck: time.Now(),
        })
    }
    
    return checks
}
```

## 🎯 Success Criteria

### Functional Requirements
- ✅ Both OpenAI and Gemini embeddings generated for all transcriptions
- ✅ Fallback mechanism when one provider fails
- ✅ A/B testing capability between providers
- ✅ Backward compatibility maintained

### Performance Requirements
- ✅ <100ms similarity search for both embedding types
- ✅ Batch processing handles 100+ transcriptions efficiently
- ✅ Concurrent processing respects rate limits
- ✅ Memory usage <1GB for typical workloads

### Quality Requirements
- ✅ 80%+ test coverage
- ✅ All SOLID principles followed
- ✅ TDD approach for all features
- ✅ Comprehensive error handling

## 🔄 Migration Strategy

### Phase 1: Dual Storage
1. Add new columns without removing old ones
2. Generate embeddings for both providers
3. Monitor performance and costs

### Phase 2: Evaluation
1. Run A/B tests comparing providers
2. Analyze similarity search quality
3. Compare costs and performance

### Phase 3: Optimization
1. Choose primary provider based on results
2. Implement smart fallback strategies
3. Optimize for cost/performance balance

This implementation plan ensures a robust, maintainable, and extensible dual-embedding system that follows best practices in software design and development.