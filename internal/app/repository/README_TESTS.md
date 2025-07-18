# Repository Layer Test Suite

This document describes the comprehensive test suite created for the repository layer, covering both SQLite and PostgreSQL implementations.

## Test Structure

The test suite is organized into the following components:

### 1. Individual Implementation Tests
- `/sqlite/repository_test.go` - Comprehensive tests for SQLite implementation
- `/pg/repository_test.go` - Comprehensive tests for PostgreSQL implementation

### 2. Cross-Database Tests
- `/tests/compatibility_test.go` - Cross-database compatibility tests
- `/tests/performance_test.go` - Performance benchmarks and regression tests
- `/tests/error_scenarios_test.go` - Error handling and edge case tests
- `/tests/transaction_test.go` - Transaction handling and ACID property tests

## Test Coverage

### Core Functionality Tests
- **Interface compliance** - Verifies both implementations satisfy the TranscriptionDAO interface
- **CRUD operations** - Tests Create (RecordToDB), Read (GetAllByUser, CheckIfFileProcessed), and Close operations
- **Data consistency** - Ensures data integrity across operations
- **Error handling** - Validates proper error responses for invalid inputs

### Data Integrity Tests
- **Unicode support** - Tests with various character encodings including Chinese, emojis, and special characters
- **Large data handling** - Tests with large transcription texts (up to 10MB)
- **Boundary conditions** - Tests with empty strings, null values, and edge cases
- **SQL injection prevention** - Tests resistance to SQL injection attacks

### Concurrency Tests
- **Concurrent writes** - Multiple goroutines writing simultaneously
- **Concurrent reads** - Multiple goroutines reading simultaneously
- **Race conditions** - Tests for race conditions in file processing checks
- **Deadlock prevention** - Tests for potential deadlock scenarios

### Performance Tests
- **Single operation benchmarks** - Individual operation performance
- **Batch operation benchmarks** - Bulk operation performance
- **Concurrent operation benchmarks** - Multi-threaded performance
- **Memory usage benchmarks** - Memory allocation and garbage collection patterns
- **Large data benchmarks** - Performance with large transcription texts

### Error Scenario Tests
- **Connection errors** - Database connection failures and recovery
- **Constraint violations** - Database constraint handling
- **Resource exhaustion** - Behavior under memory/disk pressure
- **Data corruption** - Handling of corrupted or malformed data
- **Network issues** - PostgreSQL network timeout scenarios

### Transaction Tests
- **ACID properties** - Atomicity, Consistency, Isolation, Durability
- **Transaction isolation** - Tests for different isolation levels
- **Rollback scenarios** - Explicit and error-triggered rollbacks
- **Long-running transactions** - Extended transaction behavior
- **Concurrent transactions** - Multiple simultaneous transactions

## Running Tests

### Prerequisites
- Go 1.19 or later
- SQLite support (included)
- PostgreSQL (optional, for PostgreSQL tests)

### Environment Setup

For PostgreSQL tests, set environment variables:
```bash
export POSTGRES_TEST_URL="postgres://user:password@localhost/test_db?sslmode=disable"
# OR individual variables:
export POSTGRES_TEST_HOST="localhost"
export POSTGRES_TEST_USER="postgres"
export POSTGRES_TEST_PASSWORD="postgres"
export POSTGRES_TEST_DB="tiktok_whisper_test"
```

### Running Individual Tests

#### SQLite Tests
```bash
# Run all SQLite tests
go test ./internal/app/repository/sqlite/ -v

# Run specific test categories
go test ./internal/app/repository/sqlite/ -v -run TestSQLiteDB_Interface
go test ./internal/app/repository/sqlite/ -v -run TestSQLiteDB_CheckIfFileProcessed
go test ./internal/app/repository/sqlite/ -v -run TestSQLiteDB_RecordToDB
go test ./internal/app/repository/sqlite/ -v -run TestSQLiteDB_ConcurrentAccess
go test ./internal/app/repository/sqlite/ -v -run TestSQLiteDB_DataIntegrity

# Run benchmarks
go test ./internal/app/repository/sqlite/ -bench=. -benchmem
```

#### PostgreSQL Tests
```bash
# Run all PostgreSQL tests (requires PostgreSQL setup)
go test ./internal/app/repository/pg/ -v

# Run specific PostgreSQL tests
go test ./internal/app/repository/pg/ -v -run TestPostgresDB_Interface
go test ./internal/app/repository/pg/ -v -run TestPostgresDB_RecordToDB
go test ./internal/app/repository/pg/ -v -run TestPostgresDB_ConcurrentAccess

# Run PostgreSQL benchmarks
go test ./internal/app/repository/pg/ -bench=. -benchmem
```

### Running Cross-Database Tests

```bash
# Compatibility tests (compares SQLite vs PostgreSQL behavior)
go test ./internal/app/repository/tests/ -v -run TestCrossDatabaseCompatibility

# Performance comparison benchmarks
go test ./internal/app/repository/tests/ -bench=BenchmarkRepositoryPerformance -benchmem
go test ./internal/app/repository/tests/ -bench=BenchmarkCrossDatabasePerformance -benchmem

# Error scenario tests
go test ./internal/app/repository/tests/ -v -run TestErrorScenarios

# Transaction and consistency tests
go test ./internal/app/repository/tests/ -v -run TestTransactionHandlingAndDataConsistency
```

### Running Performance Regression Tests

```bash
# Test for performance regressions against baselines
go test ./internal/app/repository/tests/ -v -run TestPerformanceRegression

# Comprehensive performance analysis
go test ./internal/app/repository/tests/ -bench=. -benchmem -benchtime=10s
```

### Running All Repository Tests

```bash
# Run all repository tests (SQLite only)
go test ./internal/app/repository/... -v

# Run all tests including PostgreSQL (if available)
POSTGRES_TEST_URL="postgres://postgres:postgres@localhost/test?sslmode=disable" \
go test ./internal/app/repository/... -v

# Run all tests with benchmarks
go test ./internal/app/repository/... -v -bench=. -benchmem
```

## Test Features

### Automatic Database Setup
- Tests automatically create and teardown test databases
- SQLite uses temporary files that are automatically cleaned up
- PostgreSQL creates and drops test databases with unique names

### Test Data Management
- Comprehensive test fixtures with realistic data
- Automatic seeding of test data when needed
- Cleanup between tests to ensure isolation

### Error Recovery Testing
- Tests validate that databases can recover from various error conditions
- Ensures data integrity is maintained even after errors
- Tests proper cleanup and resource management

### Performance Monitoring
- Benchmarks track operations per second
- Memory allocation tracking
- Garbage collection impact analysis
- Performance regression detection

### Cross-Platform Compatibility
- Tests work on Linux, macOS, and Windows
- Handle different SQLite configurations
- Support various PostgreSQL versions

## Test Configuration

### Database-Specific Settings
Tests adapt to different database configurations:
- SQLite: WAL mode, foreign key constraints, journal modes
- PostgreSQL: Isolation levels, connection pooling, schema validation

### Environment Variables
- `POSTGRES_TEST_URL` - Full PostgreSQL connection string
- `POSTGRES_TEST_HOST` - PostgreSQL host (default: localhost)
- `POSTGRES_TEST_USER` - PostgreSQL user (default: postgres)
- `POSTGRES_TEST_PASSWORD` - PostgreSQL password (default: postgres)
- `POSTGRES_TEST_DB` - PostgreSQL database name (default: tiktok_whisper_test)

## Troubleshooting

### Common Issues

1. **PostgreSQL Connection Failed**
   - Ensure PostgreSQL is running
   - Check connection parameters
   - Verify user permissions

2. **SQLite Permission Denied**
   - Check file system permissions
   - Ensure temporary directory is writable

3. **Test Timeouts**
   - Some tests may take longer with large datasets
   - Use `-timeout` flag to increase test timeout

4. **Memory Issues**
   - Large data tests may require significant memory
   - Monitor system resources during testing

### Debugging Tests

```bash
# Verbose output with detailed logging
go test ./internal/app/repository/... -v -test.v

# Run with race detection
go test ./internal/app/repository/... -race

# CPU profiling for performance analysis
go test ./internal/app/repository/tests/ -bench=. -cpuprofile=cpu.prof

# Memory profiling
go test ./internal/app/repository/tests/ -bench=. -memprofile=mem.prof
```

## Test Maintenance

### Adding New Tests
1. Follow existing naming conventions
2. Use the testutil package for database setup
3. Ensure tests are isolated and don't depend on external state
4. Add both positive and negative test cases

### Updating Benchmarks
1. Update baseline performance expectations in regression tests
2. Add new benchmark categories for new features
3. Monitor performance over time

### Database Schema Changes
When the database schema changes:
1. Update the `createTestTables` function in testutil
2. Update test fixtures to match new schema
3. Add migration tests if applicable
4. Verify both SQLite and PostgreSQL compatibility

This comprehensive test suite ensures the repository layer is robust, performant, and reliable across different database backends and usage scenarios.