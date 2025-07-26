# Transcription Embeddings Feature Design

> **⚠️ DEPRECATED DOCUMENT**  
> **Status**: Archived (2025-07-23)  
> **Reason**: Superseded by actual implementation in DUAL_EMBEDDING_TDD_PLAN.md and IMPLEMENTATION_SUMMARY.md  
> **Current Implementation**: Dual embedding system (OpenAI + Gemini) with pgvector is fully operational  

## Executive Summary

**NOTE: This document represents early design concepts. The actual implementation differs significantly and is documented in:**
- `DUAL_EMBEDDING_TDD_PLAN.md` - Current architecture and implementation plan
- `IMPLEMENTATION_SUMMARY.md` - Completed implementation status
- `../CLAUDE.md` - Current system configuration and usage

This document outlines the design for adding embedding functionality to the tiktok-whisper project. The feature will enable:
- Automatic generation of embeddings for transcriptions
- Finding duplicate or similar transcriptions using vector similarity
- Foundation for future RAG (Retrieval-Augmented Generation) capabilities

The implementation follows a phased approach, starting with mock services and progressing to real Google Gemini embeddings.

## Architecture Overview

### Core Components

```
┌─────────────────────┐     ┌──────────────────────┐     ┌─────────────────────┐
│  CLI Commands       │────▶│  Vector Processor    │────▶│  Vector Repository  │
│  - v2t embed       │     │  - Batch processing  │     │  - SQLite storage   │
│  - v2t find-similar│     │  - Similarity search │     │  - Vector operations│
└─────────────────────┘     └──────────────────────┘     └─────────────────────┘
                                      │
                                      ▼
                            ┌──────────────────────┐
                            │  Embedding Service   │
                            │  - Mock (Phase 1)    │
                            │  - Google (Phase 2)  │
                            └──────────────────────┘
```

### Database Design

**Database File**: `/data/transcription-vector.db` (separate from main database)

**Schema**:
```sql
-- Main embeddings table
CREATE TABLE transcription_embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transcription_id INTEGER NOT NULL,
    user TEXT NOT NULL,
    mp3_file_name TEXT NOT NULL,
    transcription_text TEXT NOT NULL,
    embedding_vector TEXT,        -- JSON array of floats
    embedding_model TEXT,
    embedding_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT,                -- JSON for additional classification data
    UNIQUE(transcription_id)
);

-- Metadata for embedding models
CREATE TABLE embedding_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_name TEXT NOT NULL,
    embedding_dimension INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_user ON transcription_embeddings(user);
CREATE INDEX idx_mp3_file_name ON transcription_embeddings(mp3_file_name);
```

### Package Structure

```
internal/app/
├── embedding/                    # Embedding service layer
│   ├── service.go               # EmbeddingService interface
│   ├── mock.go                  # Mock implementation
│   └── google.go                # Google Gemini implementation (future)
├── repository/
│   └── vector/                  # Vector database operations
│       ├── interface.go         # VectorRepository interface
│       ├── sqlite.go            # SQLite implementation
│       └── models.go            # Data models
└── vector/                      # Vector utilities
    ├── similarity.go            # Similarity calculations
    └── processor.go             # Batch processing logic
```

## Interface Design

### EmbeddingService Interface

```go
type EmbeddingService interface {
    // Generate embedding for a single text
    GenerateEmbedding(text string) ([]float32, error)
    
    // Generate embeddings for multiple texts
    GenerateBatchEmbeddings(texts []string) ([][]float32, error)
    
    // Get the dimension of embeddings produced
    GetEmbeddingDimension() int
    
    // Get the model name
    GetModelName() string
}
```

### VectorRepository Interface

```go
type VectorRepository interface {
    // Storage operations
    StoreEmbedding(transcriptionID int, user, mp3FileName, text string, 
                   embedding []float32, model string) error
    GetEmbedding(transcriptionID int) (*TranscriptionEmbedding, error)
    GetEmbeddingsByUser(user string) ([]*TranscriptionEmbedding, error)
    
    // Similarity search operations
    FindSimilar(embedding []float32, topK int, threshold float32) ([]*SimilarityResult, error)
    FindDuplicates(threshold float32) ([][]int, error)
    
    // Batch operations
    BatchStoreEmbeddings(embeddings []*TranscriptionEmbedding) error
    
    // Lifecycle
    Close() error
}
```

### Data Models

```go
type TranscriptionEmbedding struct {
    ID                int
    TranscriptionID   int
    User              string
    Mp3FileName       string
    TranscriptionText string
    EmbeddingVector   []float32
    EmbeddingModel    string
    Metadata          map[string]interface{}
    Timestamp         time.Time
}

type SimilarityResult struct {
    TranscriptionID int
    Similarity      float32
    User            string
    Mp3FileName     string
}
```

## Mock Embedding Service

The mock service generates deterministic embeddings for testing without API calls:

1. **Deterministic Generation**:
   - SHA256 hash of text → consistent pseudo-embeddings
   - Same text always produces same embedding
   - Preserves semantic relationships for testing

2. **Characteristics**:
   - 768-dimensional vectors (matching common embedding models)
   - Values normalized to [-1, 1] range
   - Supports batch operations

3. **Implementation**:
   ```go
   func (m *MockEmbeddingService) GenerateEmbedding(text string) ([]float32, error) {
       // Generate deterministic embedding from text hash
       hash := sha256.Sum256([]byte(text))
       embedding := make([]float32, m.dimension)
       
       // Convert hash to float values
       for i := 0; i < m.dimension; i++ {
           byteIndex := i % len(hash)
           embedding[i] = (float32(hash[byteIndex]) / 255.0) * 2 - 1
       }
       
       return embedding, nil
   }
   ```

## CLI Commands

### 1. Generate Embeddings
```bash
# Generate embeddings for all transcriptions
v2t embed --all

# Generate embeddings for specific user
v2t embed --user "经纬第二期"

# Generate embeddings for new transcriptions only
v2t embed --new-only
```

### 2. Find Similar Transcriptions
```bash
# Find top 5 similar transcriptions to ID 123
v2t find-similar --id 123 --top 5

# Find similar transcriptions with threshold
v2t find-similar --id 123 --threshold 0.8
```

### 3. Find Duplicates
```bash
# Find duplicate transcriptions (similarity > 0.95)
v2t find-duplicates --threshold 0.95

# Find duplicates for specific user
v2t find-duplicates --user "经纬第二期" --threshold 0.9
```

## Similarity Calculation

Using cosine similarity for vector comparison:

```go
func CosineSimilarity(a, b []float32) float32 {
    if len(a) != len(b) {
        return 0
    }
    
    var dotProduct, normA, normB float32
    for i := range a {
        dotProduct += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    
    if normA == 0 || normB == 0 {
        return 0
    }
    
    return dotProduct / (float32(math.Sqrt(float64(normA))) * 
                        float32(math.Sqrt(float64(normB))))
}
```

## Implementation Phases

### Phase 1: Foundation (Current)
- [x] Design interfaces and data models
- [ ] Implement mock embedding service
- [ ] Create SQLite vector repository
- [ ] Add basic similarity search
- [ ] Implement CLI commands

### Phase 2: Integration
- [ ] Add batch processing for existing transcriptions
- [ ] Implement duplicate detection
- [ ] Create comprehensive tests
- [ ] Add progress tracking and logging

### Phase 3: Production
- [ ] Implement Google Gemini embedding service
- [ ] Add vector indexing for performance
- [ ] Implement semantic search
- [ ] Add caching layer

## Configuration

### Wire Integration

```go
// Add to wire.go
func provideEmbeddingService() EmbeddingService {
    // Phase 1: Return mock service
    return &MockEmbeddingService{
        dimension: 768,
        modelName: "mock-embedding-v1",
    }
}

func provideVectorRepository(dbPath string) (VectorRepository, error) {
    return NewSQLiteVectorRepository(dbPath)
}
```

### Environment Variables

```bash
# Future Google integration
GOOGLE_API_KEY=your-api-key
GOOGLE_EMBEDDING_MODEL=models/embedding-001

# Vector database path
VECTOR_DB_PATH=/data/transcription-vector.db
```

## Performance Considerations

1. **Batch Processing**:
   - Process embeddings in batches of 100
   - Use goroutines for parallel processing
   - Implement rate limiting for API calls

2. **Storage Optimization**:
   - JSON arrays for initial implementation
   - Consider binary format for production
   - Implement compression for large datasets

3. **Search Optimization**:
   - Limit search scope by user or date
   - Implement approximate nearest neighbor for large datasets
   - Cache frequently accessed embeddings

## Testing Strategy

1. **Unit Tests**:
   - Mock service consistency
   - Similarity calculation accuracy
   - Repository CRUD operations

2. **Integration Tests**:
   - End-to-end embedding generation
   - Duplicate detection accuracy
   - CLI command functionality

3. **Performance Tests**:
   - Batch processing speed
   - Similarity search performance
   - Memory usage under load

## Future Enhancements

1. **Vector Database Migration**:
   - Consider specialized vector databases (Pinecone, Weaviate)
   - Implement proper vector indexing

2. **Advanced Features**:
   - Clustering for content categorization
   - Semantic search with natural language queries
   - Cross-lingual similarity detection

3. **RAG Integration**:
   - Use embeddings for context retrieval
   - Integrate with LLM for question answering
   - Build knowledge base from transcriptions

## Security Considerations

1. **API Key Management**:
   - Store keys in environment variables
   - Never commit keys to repository
   - Implement key rotation

2. **Data Privacy**:
   - Option to exclude sensitive transcriptions
   - Implement access controls
   - Consider on-premise embedding models

## Conclusion

This design provides a solid foundation for adding embedding functionality to tiktok-whisper. The phased approach allows for rapid development and testing with mock services before investing in external API integration. The architecture is extensible and follows the existing patterns in the codebase.