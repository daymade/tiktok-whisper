# Test Utilities Package

This package provides comprehensive testing utilities for the tiktok-whisper application, including database helpers, mock implementations, test fixtures, and performance benchmarking tools.

## Components

### 1. Database Test Helpers (`db_helpers.go`)

Provides utilities for setting up and managing test databases:

- **SetupTestDB**: Creates test databases with automatic cleanup
- **SetupTestSQLite**: Creates SQLite test databases  
- **SetupTestPostgres**: Creates PostgreSQL test databases
- **SeedTestData**: Populates databases with sample data
- **WithTestDB**: Helper function that provides a test database to a test function
- **WithSeekedTestDB**: Helper function that provides a pre-populated test database

### 2. Mock Factory Functions (`mock_factory.go`)

Mock implementations of core interfaces:

- **MockTranscriber**: Mock implementation of the Transcriber interface
- **MockTranscriptionDAO**: Mock implementation of the TranscriptionDAO interface
- **MockLogger**: Mock logger for testing logging behavior
- **EnhancedMockLogger**: Advanced mock logger with testify/mock integration
- **MockFileSystem**: Mock file system for testing file operations

### 3. Test Data Fixtures (`fixtures.go`)

Predefined test data and utilities:

- **TestTranscriptions**: Sample transcription data
- **TestFileInfos**: Sample file information
- **TestUsers**: Sample user data
- **MockAPIResponses**: Sample API responses
- **TestScenarios**: Different test scenarios with expected outcomes

### 4. Performance Benchmarking (`benchmark.go`)

Tools for performance testing:

- **BenchmarkHelper**: Utilities for timing and memory usage tracking
- **BenchmarkRunner**: Configurable benchmark execution
- **BenchmarkResults**: Results analysis and reporting
- **Performance assertion helpers**: Assert duration and memory usage

## Quick Start

### Database Testing

```go
func TestDatabaseOperation(t *testing.T) {
    testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
        // Your database test code here
        // Database is automatically cleaned up
    })
}
```

### Mock Usage

```go
func TestTranscription(t *testing.T) {
    mockTranscriber := testutil.NewMockTranscriber().
        WithDefaultResponse("Test transcription").
        WithError("/error/path.mp3", errors.New("test error"))
    
    result, err := mockTranscriber.Transcript("/path/to/audio.mp3")
    assert.NoError(t, err)
    assert.Equal(t, "Test transcription", result)
}
```

### Enhanced Mock Logger

```go
func TestEmbeddingOrchestrator(t *testing.T) {
    // Create enhanced mock logger with testify/mock integration
    logger := testutil.NewEnhancedMockLogger().WithMockingEnabled(true)
    
    // Set up expectations
    logger.ExpectInfo("Processing transcription", "transcriptionID", 123)
    logger.ExpectAnyError().Times(0) // Expect no errors
    
    // Your code under test
    orchestrator := NewEmbeddingOrchestrator(providers, storage, logger)
    err := orchestrator.ProcessTranscription(ctx, 123, "test text")
    
    // Verify expectations and structured data
    assert.NoError(t, err)
    logger.AssertExpectations(t)
    assert.True(t, logger.HasMessageWithField("transcriptionID", 123))
    assert.False(t, logger.HasError())
}
```

### Benchmarking

```go
func TestPerformance(t *testing.T) {
    bh := testutil.NewBenchmarkHelper("MyOperation")
    bh.Start()
    
    // Your code to benchmark
    doSomeWork()
    
    bh.Stop()
    bh.AssertDurationLessThan(t, 1*time.Second)
    t.Log(bh.Report())
}
```

## Environment Configuration

For PostgreSQL testing, set environment variables:

```bash
export POSTGRES_TEST_URL="postgres://user:pass@localhost/testdb?sslmode=disable"
# OR individual variables:
export POSTGRES_TEST_HOST="localhost"
export POSTGRES_TEST_USER="testuser"
export POSTGRES_TEST_PASSWORD="testpass"
export POSTGRES_TEST_DB="testdb"
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run benchmarks
go test -bench=.

# Run specific test file
go test ./internal/app/testutil/
```

## Best Practices

1. **Use helper functions** for database setup to ensure proper cleanup
2. **Use mocks** to isolate components under test
3. **Leverage fixtures** for consistent test data
4. **Use benchmarking** to validate performance requirements
5. **Always clean up** resources in tests
6. **Use meaningful test names** that describe what is being tested

## Thread Safety

All mock implementations are thread-safe and can be used in parallel tests. The benchmark helpers use mutexes to ensure thread-safe operations.

## Memory Management

Database test helpers automatically clean up temporary files and database connections. Mock implementations have `Reset()` methods to clear state between tests.

## Examples

See `examples_test.go` for comprehensive usage examples and `doc.go` for detailed API documentation.