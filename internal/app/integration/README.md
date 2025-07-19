# Integration Tests for Network Failure Handling and External API Integration

This directory contains comprehensive integration tests that verify network resilience, error handling, and external API integration across the entire tiktok-whisper project.

## Test Structure

### Core Test Files

1. **`network_resilience_test.go`** - Network failure handling and connectivity issues
2. **`api_integration_test.go`** - OpenAI API integration with mocking and real API tests
3. **`failure_recovery_test.go`** - Retry logic, circuit breaker patterns, and recovery mechanisms
4. **`end_to_end_test.go`** - Complete workflow testing with external dependencies
5. **`mock_servers_test.go`** - Comprehensive mock HTTP servers for various failure scenarios
6. **`database_filesystem_test.go`** - Database connectivity resilience and file system operations

## Test Categories

### 1. Network Resilience Testing

Tests various network failure scenarios:
- **Connection timeouts** - Configurable timeout scenarios
- **DNS failures** - Non-existent domains and invalid hosts
- **Connection refused** - Unreachable services
- **Intermittent connectivity** - Unstable network conditions
- **Rate limiting** - API rate limit handling
- **Network partitions** - Temporary service unavailability

### 2. External API Integration

Comprehensive testing of OpenAI API integration:
- **Mock API responses** - Controlled response testing
- **Real API testing** - Integration with actual OpenAI API (when API key available)
- **Error response handling** - Various API error scenarios
- **Response validation** - JSON parsing and validation
- **Concurrent API requests** - Parallel request handling
- **Context cancellation** - Request timeout and cancellation

### 3. Failure Recovery Patterns

Implementation and testing of resilience patterns:
- **Retry logic** - Exponential backoff strategies
- **Circuit breaker** - Failure threshold and recovery testing
- **Graceful degradation** - Fallback mechanisms
- **Resource cleanup** - Proper cleanup after failures
- **Error propagation** - Error handling through system layers

### 4. End-to-End Integration

Complete workflow testing:
- **Full transcription workflow** - Database + API + file processing
- **Multiple transcriber support** - Local whisper.cpp and remote OpenAI API
- **Concurrent processing** - Multiple file processing
- **Workflow recovery** - Recovery from partial failures
- **Resource management** - Memory and file handle management

### 5. Mock Server Infrastructure

Sophisticated mock servers for testing:
- **Configurable responses** - Custom response scenarios
- **Failure mode simulation** - Various server failure types
- **Rate limiting simulation** - API rate limit testing
- **Response timing control** - Latency and timeout testing
- **Request logging** - Request verification and debugging

### 6. Database and File System Resilience

Infrastructure resilience testing:
- **PostgreSQL connectivity** - Connection failures and recovery
- **SQLite operations** - Local database resilience
- **File system operations** - File I/O failure handling
- **Concurrent access** - Multi-threaded file and database operations
- **Resource cleanup** - Proper resource management

## Running the Tests

### Prerequisites

1. **Go 1.23+** installed
2. **Build tags support** for integration tests
3. **Optional dependencies** for full testing:
   - OpenAI API key for real API tests
   - PostgreSQL instance for database tests
   - whisper.cpp binary and models for local transcription tests

### Basic Integration Test Execution

```bash
# Run all integration tests
go test -tags=integration ./internal/app/integration/...

# Run specific test file
go test -tags=integration ./internal/app/integration/ -run TestNetworkTimeout

# Run with verbose output
go test -tags=integration -v ./internal/app/integration/...

# Run in short mode (skips long-running tests)
go test -tags=integration -short ./internal/app/integration/...
```

### Environment Setup

```bash
# For OpenAI API tests (optional)
export OPENAI_API_KEY="your-api-key-here"

# For PostgreSQL tests (optional)
# Ensure PostgreSQL is running locally or adjust connection strings in tests

# For whisper.cpp tests (optional)
# Ensure whisper.cpp binary and models are available at expected paths
```

### Test Configuration

The tests are designed to be flexible and handle missing dependencies gracefully:

- **Missing API keys** - Real API tests are skipped automatically
- **Missing databases** - Database-specific tests are skipped
- **Missing binaries** - Local transcription tests are skipped
- **Short mode** - Long-running tests are skipped with `-short` flag

## Test Scenarios Covered

### Network Failure Scenarios

```go
// Example: Testing timeout scenarios
TestNetworkTimeout:
- Fast response within timeout
- Slow response within timeout  
- Timeout exceeded scenarios
- Very short timeout handling

TestConnectionRefused:
- Invalid hostnames
- Unreachable ports
- Invalid URL schemes
```

### API Integration Scenarios

```go
// Example: Testing API response handling
TestOpenAIAPIIntegration:
- Successful transcription responses
- Invalid API key errors
- Rate limit exceeded errors
- Malformed JSON responses
- Server error responses

TestAPIResponseValidation:
- Valid response parsing
- Missing field handling
- Invalid JSON handling
- Non-string text fields
```

### Failure Recovery Scenarios

```go
// Example: Testing retry and circuit breaker patterns
TestRetryLogic:
- Exponential backoff verification
- Maximum retry limits
- Success on various attempt numbers

TestCircuitBreakerPattern:
- Failure threshold detection
- Circuit open state handling
- Recovery after timeout
```

### End-to-End Scenarios

```go
// Example: Complete workflow testing
TestEndToEndWorkflow:
- Remote API workflow
- Local API workflow  
- Mock workflow
- Database persistence verification

TestConcurrentWorkflow:
- Multiple file processing
- Race condition handling
- Resource contention management
```

## Mock Server Capabilities

The mock server infrastructure provides:

### Failure Mode Simulation

```go
// Available failure modes
SetFailureMode("timeout")           // Request timeouts
SetFailureMode("connection_reset")  // Connection drops
SetFailureMode("malformed_response") // Invalid JSON
SetFailureMode("empty_response")    // Empty responses
SetFailureMode("random_errors")     // Intermittent failures
```

### Rate Limiting Simulation

```go
// Enable rate limiting for testing
mockServer.EnableRateLimit(10) // 10 requests per minute
```

### Custom Response Queuing

```go
// Queue specific responses
mockServer.QueueResponse(MockResponse{
    StatusCode: http.StatusOK,
    Body:       `{"text": "Custom response"}`,
    Delay:      200 * time.Millisecond,
})
```

## Database Testing

### SQLite Tests
- Connection handling
- Concurrent access
- Transaction management
- File locking behavior

### PostgreSQL Tests
- Connection pooling
- Network connectivity issues
- Authentication failures
- Database availability

## File System Testing

### Operation Failures
- Permission denied scenarios
- Disk space limitations
- Invalid file paths
- Concurrent file access

### Resource Management
- File handle cleanup
- Temporary file management
- Directory operations
- Cross-platform compatibility

## Best Practices for Integration Tests

### 1. Test Isolation
- Each test creates its own temporary resources
- No dependencies between test cases
- Proper cleanup in defer blocks

### 2. Realistic Scenarios
- Tests simulate real-world failure conditions
- Use realistic timeouts and delays
- Cover edge cases and boundary conditions

### 3. Performance Considerations
- Long-running tests are marked and can be skipped
- Resource usage is minimized
- Concurrent tests don't overload the system

### 4. Error Handling
- All error conditions are tested
- Error messages are validated
- Recovery scenarios are verified

## Debugging Integration Tests

### Verbose Logging
```bash
# Enable detailed test output
go test -tags=integration -v ./internal/app/integration/ -run TestSpecificTest

# Add timing information
go test -tags=integration -v ./internal/app/integration/ -timeout 10m
```

### Mock Server Debugging
```go
// Check request logs
requests := mockServer.GetRequestLog()
for _, req := range requests {
    t.Logf("Request: %s %s", req.Method, req.Path)
}
```

### Test Data Inspection
```go
// Examine database state
transcriptions, _ := dao.GetAllByUser("test_user")
t.Logf("Found %d transcriptions", len(transcriptions))
```

## Contributing to Integration Tests

### Adding New Tests

1. **Follow naming conventions** - Use descriptive test names
2. **Use build tags** - Mark tests with `//go:build integration`
3. **Handle missing dependencies** - Skip tests gracefully when dependencies unavailable
4. **Add proper documentation** - Document test purpose and scenarios
5. **Include cleanup** - Ensure proper resource cleanup

### Test Structure Template

```go
//go:build integration
// +build integration

func TestNewFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping test in short mode")
    }
    
    // Check dependencies
    if !isDependencyAvailable() {
        t.Skip("Dependency not available")
    }
    
    // Setup
    testResource := setupTestResource(t)
    defer cleanupTestResource(testResource)
    
    // Test scenarios
    tests := []struct{
        name string
        // test parameters
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Performance Benchmarks

Some integration tests include performance benchmarks:

```bash
# Run benchmarks
go test -tags=integration -bench=. ./internal/app/integration/

# Run specific benchmark
go test -tags=integration -bench=BenchmarkConcurrentAPI ./internal/app/integration/
```

## Continuous Integration

These tests are designed to run in CI environments:

- **Timeout handling** - Tests have appropriate timeouts
- **Dependency checking** - Missing dependencies cause skips, not failures
- **Resource limits** - Tests respect CI resource constraints
- **Parallel execution** - Tests can run in parallel safely

## Troubleshooting

### Common Issues

1. **Test timeouts** - Increase timeout with `-timeout` flag
2. **Port conflicts** - Tests use ephemeral ports to avoid conflicts
3. **File permissions** - Ensure test runner has necessary file permissions
4. **Network connectivity** - Some tests require internet access for real API calls

### Test Debugging

1. **Use verbose mode** - `-v` flag for detailed output
2. **Run specific tests** - Use `-run` flag to isolate issues
3. **Check test logs** - Review logged error messages and request details
4. **Verify dependencies** - Ensure all required services are available

This comprehensive integration test suite ensures the tiktok-whisper application handles network failures, API integration issues, and external dependency problems gracefully while maintaining data integrity and providing appropriate error handling throughout the system.