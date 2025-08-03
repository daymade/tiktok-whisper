# Comprehensive Test Plan for tiktok-whisper

## Executive Summary

This document outlines a comprehensive testing strategy for the tiktok-whisper project. The goal is to achieve high test coverage across all major components to enable safe refactoring and ensure reliability.

## Current Test Coverage Assessment

### Existing Test Files
- ✅ `converter/convert_test.go` - Basic converter functionality
- ✅ `repository/sqlite/db_test.go` - SQLite connection testing  
- ✅ `repository/pg/pg_test.go` - PostgreSQL testing
- ✅ `embedding/provider/*_test.go` - Provider testing
- ✅ `embedding/orchestrator/*_test.go` - Orchestrator testing
- ✅ `storage/vector/*_test.go` - Vector storage testing
- ✅ `util/files/FileUtils_test.go` - File utilities testing
- ✅ `api/whisper_cpp/whisper_cpp_test.go` - Local transcriber testing

### Coverage Gaps
- ❌ CLI commands (0% coverage)
- ❌ Wire dependency injection (0% coverage)
- ❌ Audio processing utilities (partial coverage)
- ❌ Error handling paths (insufficient coverage)
- ❌ Concurrent processing (limited coverage)
- ❌ Integration tests (minimal coverage)

## Testing Strategy

### 1. Unit Testing Approach
- **Test Pyramid**: Focus on unit tests (70%), integration tests (20%), end-to-end tests (10%)
- **Test-Driven Development**: Write tests before implementing new features
- **Mock-based Testing**: Use mocks for external dependencies
- **Table-driven Tests**: Use Go's table-driven test pattern for comprehensive coverage

### 2. Test Categories

#### A. Unit Tests
- **Scope**: Individual functions and methods
- **Dependencies**: Mocked external dependencies
- **Coverage Target**: 90%+ for core business logic

#### B. Integration Tests  
- **Scope**: Component interaction testing
- **Dependencies**: Real database connections, file systems
- **Coverage Target**: Key workflows and data flows

#### C. Contract Tests
- **Scope**: Interface compliance testing
- **Dependencies**: Mock implementations
- **Coverage Target**: All public interfaces

#### D. Performance Tests
- **Scope**: Benchmarking critical paths
- **Dependencies**: Realistic data sets
- **Coverage Target**: Core processing functions

## Phase 1: Foundation and Infrastructure

### 1.1 Test Infrastructure Setup
- [ ] Create test utilities package (`internal/app/testutil`)
- [ ] Database test helpers (setup/teardown)
- [ ] Mock factory functions
- [ ] Test data fixtures
- [ ] Performance benchmarking utilities

### 1.2 Mock Implementations
- [ ] `MockTranscriber` for `api.Transcriber`
- [ ] `MockTranscriptionDAO` for `repository.TranscriptionDAO`
- [ ] `MockLogger` for logging interface
- [ ] `MockFileSystem` for file operations
- [ ] Enhanced `MockProvider` and `MockVectorStorage`

### 1.3 Test Data and Fixtures
- [ ] Sample audio files for testing
- [ ] Test transcription data
- [ ] Mock API responses
- [ ] Database seed data
- [ ] Configuration fixtures

## Phase 2: Core Business Logic Testing

### 2.1 Converter Package (`internal/app/converter`)
**Priority: HIGH** - Core business logic

#### Test Coverage Plan
- [ ] `NewConverter()` constructor testing
- [ ] `ConvertAudioDir()` batch audio conversion
- [ ] `ConvertVideoDir()` batch video conversion
- [ ] `ConvertAudios()` parallel audio processing
- [ ] `ConvertVideos()` parallel video processing
- [ ] Error handling in all conversion methods
- [ ] Concurrent processing behavior
- [ ] Progress tracking and reporting

#### Test Types
- Unit tests with mocked dependencies
- Integration tests with real databases
- Performance benchmarks for batch operations
- Error scenario testing (corrupted files, network issues)

### 2.2 Embedding System (`internal/app/embedding`)
**Priority: HIGH** - Recently implemented, critical for new features

#### Provider Testing (`internal/app/embedding/provider`)
- [ ] `NewOpenAIProvider()` constructor
- [ ] `NewGeminiProvider()` constructor  
- [ ] `NewMockProvider()` constructor
- [ ] `GenerateEmbedding()` for all providers
- [ ] `GetProviderInfo()` metadata retrieval
- [ ] Error handling (API failures, network issues)
- [ ] Rate limiting behavior
- [ ] Input validation

#### Orchestrator Testing (`internal/app/embedding/orchestrator`)
- [ ] `NewEmbeddingOrchestrator()` constructor
- [ ] `ProcessTranscription()` single processing
- [ ] `GetEmbeddingStatus()` status retrieval
- [ ] `NewBatchProcessor()` batch constructor
- [ ] `ProcessAllTranscriptions()` batch processing
- [ ] Concurrent processing coordination
- [ ] Error recovery and retry logic
- [ ] Progress reporting

#### Similarity Calculator (`internal/app/embedding/similarity`)
- [ ] `CosineSimilarity()` calculation
- [ ] `EuclideanDistance()` calculation
- [ ] Edge cases (zero vectors, identical vectors)
- [ ] Performance benchmarks
- [ ] Numerical accuracy testing

### 2.3 Storage Layer (`internal/app/storage`)
**Priority: HIGH** - Critical for data persistence

#### Vector Storage (`internal/app/storage/vector`)
- [ ] `NewPgVectorStorage()` constructor
- [ ] `StoreEmbedding()` single embedding storage
- [ ] `StoreDualEmbeddings()` dual embedding storage
- [ ] `GetEmbedding()` single embedding retrieval
- [ ] `GetDualEmbeddings()` dual embedding retrieval
- [ ] `FindSimilar()` similarity search
- [ ] Connection management
- [ ] Transaction handling
- [ ] Error recovery

### 2.4 Repository Layer (`internal/app/repository`)
**Priority: HIGH** - Database abstraction layer

#### SQLite Implementation (`internal/app/repository/sqlite`)
- [ ] `NewSQLiteDB()` constructor
- [ ] `GetAllByUser()` user-specific queries
- [ ] `CheckIfFileProcessed()` processing status
- [ ] `RecordToDB()` database record creation
- [ ] Transaction management
- [ ] Connection pooling
- [ ] Migration compatibility

#### PostgreSQL Implementation (`internal/app/repository/pg`)
- [ ] `NewPostgreSQLDB()` constructor
- [ ] All TranscriptionDAO methods
- [ ] Vector column operations
- [ ] Connection management
- [ ] Performance optimization
- [ ] Migration compatibility

## Phase 3: API and External Integration Testing

### 3.1 Transcriber API (`internal/app/api`)
**Priority: MEDIUM** - External API integration

#### OpenAI API (`internal/app/api/openai`)
- [ ] `GetClient()` client initialization
- [ ] `whisper.Transcript()` transcription API
- [ ] `chat.Chat()` chat completion API
- [ ] `embedding.GenerateEmbedding()` embedding API
- [ ] Error handling (API errors, rate limits)
- [ ] Authentication testing
- [ ] Response parsing

#### Whisper.cpp (`internal/app/api/whisper_cpp`)
- [ ] `NewLocalTranscriber()` constructor
- [ ] `Transcript()` local transcription
- [ ] Binary path validation
- [ ] Model file validation
- [ ] Process execution
- [ ] Output parsing
- [ ] Error handling

### 3.2 Audio Processing (`internal/app/audio`)
**Priority: MEDIUM** - Audio format handling

- [ ] `GetAudioInfo()` metadata extraction
- [ ] `ConvertToWav()` format conversion
- [ ] `GetDuration()` duration calculation
- [ ] File format detection
- [ ] Error handling (corrupted files)
- [ ] Performance benchmarks

### 3.3 File Utilities (`internal/app/util/files`)
**Priority: LOW** - Utility functions

- [ ] `GetAbsolutePath()` path resolution
- [ ] `EnsureDirectoryExists()` directory creation
- [ ] `GetFileExtension()` extension extraction
- [ ] `IsValidAudioFile()` file validation
- [ ] Error handling (permissions, disk space)

## Phase 4: CLI and Command Testing

### 4.1 Command Structure (`cmd/v2t/cmd`)
**Priority: MEDIUM** - User interface

#### Root Command (`cmd/v2t/cmd/root.go`)
- [ ] Command initialization
- [ ] Global flags handling
- [ ] Subcommand registration
- [ ] Help text generation
- [ ] Error handling

#### Subcommands
- [ ] `config` command functionality
- [ ] `convert` command with all flags
- [ ] `download` command integration
- [ ] `embed` command operations
- [ ] `export` command functionality
- [ ] `version` command output

### 4.2 CLI Integration Testing
- [ ] End-to-end command execution
- [ ] Flag parsing and validation
- [ ] Configuration file handling
- [ ] Output formatting
- [ ] Error message quality

## Phase 5: Integration and End-to-End Testing

### 5.1 Database Integration
**Priority: HIGH** - Critical data flows

- [ ] SQLite database operations
- [ ] PostgreSQL with pgvector
- [ ] Migration testing
- [ ] Data consistency
- [ ] Performance under load

### 5.2 External API Integration
**Priority: MEDIUM** - External dependencies

- [ ] OpenAI API integration
- [ ] Whisper.cpp binary integration
- [ ] Network failure handling
- [ ] Rate limiting compliance
- [ ] Authentication flows

### 5.3 File System Integration
**Priority: MEDIUM** - File operations

- [ ] Batch file processing
- [ ] Directory traversal
- [ ] File format conversion
- [ ] Temporary file cleanup
- [ ] Permission handling

## Phase 6: Performance and Load Testing

### 6.1 Performance Benchmarks
- [ ] Batch processing performance
- [ ] Concurrent transcription limits
- [ ] Database query performance
- [ ] Memory usage optimization
- [ ] CPU utilization

### 6.2 Load Testing
- [ ] Large file processing
- [ ] Batch size optimization
- [ ] Concurrent user simulation
- [ ] Resource exhaustion testing
- [ ] Recovery testing

## Test Implementation Guidelines

### 1. Test Organization
```go
// Package structure
package converter_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "tiktok-whisper/internal/app/converter"
    "tiktok-whisper/internal/app/testutil"
)
```

### 2. Test Naming Convention
- Test functions: `TestFunctionName_Scenario`
- Benchmark functions: `BenchmarkFunctionName`
- Example functions: `ExampleFunctionName`

### 3. Mock Usage Pattern
```go
// Mock interface implementation
type MockTranscriber struct {
    mock.Mock
}

func (m *MockTranscriber) Transcript(inputFilePath string) (string, error) {
    args := m.Called(inputFilePath)
    return args.String(0), args.Error(1)
}
```

### 4. Table-Driven Tests
```go
func TestConvertAudioDir(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    error
        wantErr bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 5. Integration Test Setup
```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup test database
    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)
    
    // Test implementation
}
```

## Test Execution Strategy

### 1. Continuous Integration
- Run unit tests on every commit
- Run integration tests on pull requests
- Run performance tests on releases
- Generate coverage reports

### 2. Test Categories
- `go test -short ./...` - Unit tests only
- `go test ./...` - All tests including integration
- `go test -bench=. ./...` - Performance benchmarks
- `go test -race ./...` - Race condition detection

### 3. Coverage Targets
- **Unit tests**: 90%+ coverage for core packages
- **Integration tests**: 80%+ coverage for key workflows
- **Overall**: 85%+ total coverage

### 4. Performance Baselines
- Batch processing: <100ms per file
- Database operations: <10ms per query
- API calls: <5s timeout with retries
- Memory usage: <1GB for 1000 files

## Success Criteria

### 1. Quantitative Metrics
- [ ] 90%+ unit test coverage for core packages
- [ ] 80%+ integration test coverage
- [ ] 85%+ overall test coverage
- [ ] 0 critical bugs in production
- [ ] <5% performance regression

### 2. Qualitative Metrics
- [ ] Comprehensive error handling
- [ ] Maintainable test code
- [ ] Clear test documentation
- [ ] Reliable CI/CD pipeline
- [ ] Developer confidence in refactoring

## Risk Assessment

### 1. Technical Risks
- **External API dependencies**: OpenAI API changes
- **Binary dependencies**: Whisper.cpp compatibility
- **Database migrations**: Data loss during upgrades
- **Concurrency issues**: Race conditions in parallel processing

### 2. Mitigation Strategies
- **API versioning**: Pin to specific API versions
- **Binary testing**: Test multiple whisper.cpp versions
- **Database backups**: Automated backup before migrations
- **Race detection**: Use `-race` flag in CI

## Timeline Estimation

### Phase 1: Foundation (Week 1-2)
- Test infrastructure setup
- Mock implementations
- Test data fixtures

### Phase 2: Core Logic (Week 3-6)
- Converter package testing
- Embedding system testing
- Storage layer testing
- Repository layer testing

### Phase 3: API Integration (Week 7-8)
- Transcriber API testing
- Audio processing testing
- File utilities testing

### Phase 4: CLI Testing (Week 9-10)
- Command structure testing
- CLI integration testing

### Phase 5: Integration Testing (Week 11-12)
- Database integration
- External API integration
- File system integration

### Phase 6: Performance Testing (Week 13-14)
- Performance benchmarks
- Load testing
- Optimization

## Conclusion

This comprehensive test plan will establish a robust testing foundation for the tiktok-whisper project. The phased approach ensures that critical components are tested first while building the necessary infrastructure for long-term maintainability. The high test coverage will enable safe refactoring and confident feature development.

The success of this plan depends on:
1. Consistent execution of each phase
2. Regular review and adjustment based on findings
3. Team commitment to test-driven development
4. Continuous integration and monitoring

By following this plan, the project will achieve the reliability and maintainability required for successful long-term development and deployment.