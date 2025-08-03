# Testing Strategy

This document outlines the testing strategy for the tiktok-whisper project, including the separation of unit tests and integration tests.

## Test Types

### Unit Tests

Unit tests are fast, isolated tests that don't require external dependencies like databases or API services.

**Characteristics:**
- Run with `go test -short`
- Use mocks and stubs for external dependencies
- Use in-memory databases (e.g., SQLite `:memory:`)
- No network calls
- Fast execution (< 1 second per test)

**Running unit tests:**
```bash
# Run all unit tests
make test

# Or directly with go
go test -short -v ./...

# Run tests for a specific package
go test -short -v ./internal/app/storage/...
```

### Integration Tests

Integration tests verify the interaction between components and require real external services.

**Characteristics:**
- Tagged with `//go:build integration` or `// +build integration`
- Require real PostgreSQL database
- May require API keys (from .env file)
- Make actual network calls
- Slower execution

**Running integration tests:**
```bash
# Run all integration tests
make test-integration-go

# Or directly with go
go test -tags=integration -v ./...

# With environment variables
POSTGRES_TEST_URL=postgres://user:pass@localhost/testdb go test -tags=integration -v ./...
```

## Test Organization

### Directory Structure

```
internal/
├── app/
│   ├── storage/
│   │   └── vector/
│   │       ├── pgvector.go                 # Implementation
│   │       ├── pgvector_test.go            # Unit tests (with mocks)
│   │       ├── pgvector_unit_test.go       # Additional unit tests
│   │       └── pgvector_integration_test.go # Integration tests
│   └── integration/
│       ├── user_embedding_integration_test.go  # Full workflow tests
│       └── ...
```

### Build Tags

Integration tests should include build tags at the top of the file:

```go
//go:build integration
// +build integration

package mypackage
```

### Test Patterns

#### Unit Test Example

```go
// pgvector_unit_test.go
package vector

import (
    "testing"
    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/assert"
)

func TestPgVectorStorage_GetEmbedding_Unit(t *testing.T) {
    // Create mock database
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    // Setup expectations
    mock.ExpectQuery("SELECT embedding_openai FROM transcriptions").
        WithArgs(1).
        WillReturnRows(sqlmock.NewRows([]string{"embedding_openai"}).
            AddRow("[0.1,0.2,0.3]"))

    // Test
    storage := NewPgVectorStorage(db)
    embedding, err := storage.GetEmbedding(ctx, 1, "openai")
    
    // Assert
    assert.NoError(t, err)
    assert.Len(t, embedding, 3)
}
```

#### Integration Test Example

```go
//go:build integration
// +build integration

package vector

import (
    "testing"
    "os"
    "database/sql"
)

func TestPgVectorStorage_Integration(t *testing.T) {
    // Skip if no database URL
    pgURL := os.Getenv("POSTGRES_TEST_URL")
    if pgURL == "" {
        t.Skip("POSTGRES_TEST_URL not set")
    }

    // Connect to real database
    db, err := sql.Open("postgres", pgURL)
    require.NoError(t, err)
    defer db.Close()

    // Run tests with real database
    storage := NewPgVectorStorage(db)
    // ... test implementation
}
```

## Environment Setup

### For Unit Tests

No special setup required. Unit tests should run out of the box.

### For Integration Tests

1. **PostgreSQL Database**
   ```bash
   # Set test database URL
   export POSTGRES_TEST_URL="postgres://postgres:postgres@localhost/testdb?sslmode=disable"
   ```

2. **API Keys** (if needed)
   ```bash
   # Copy .env.example to .env
   cp .env.example .env
   
   # Edit .env with your API keys
   OPENAI_API_KEY=your-key-here
   GEMINI_API_KEY=your-key-here
   ```

3. **pgvector Extension**
   ```sql
   -- Install in your test database
   CREATE EXTENSION IF NOT EXISTS vector;
   ```

## CI/CD Considerations

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run unit tests
        run: make test

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: pgvector/pgvector:pg16
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run integration tests
        env:
          POSTGRES_TEST_URL: postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
        run: make test-integration-go
```

## Best Practices

1. **Fast Feedback**: Unit tests should run quickly to provide fast feedback during development.

2. **Isolation**: Unit tests should not depend on external services or state.

3. **Deterministic**: Tests should produce the same results every time they run.

4. **Clear Naming**: Use descriptive test names that explain what is being tested.

5. **Table-Driven Tests**: Use table-driven tests for testing multiple scenarios:
   ```go
   tests := []struct {
       name     string
       input    string
       expected string
       wantErr  bool
   }{
       {"valid input", "hello", "HELLO", false},
       {"empty input", "", "", true},
   }
   
   for _, tt := range tests {
       t.Run(tt.name, func(t *testing.T) {
           // test implementation
       })
   }
   ```

6. **Mock External Dependencies**: Use interfaces and mocks for external services:
   ```go
   type EmbeddingProvider interface {
       GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
   }
   
   type MockProvider struct {
       mock.Mock
   }
   
   func (m *MockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
       args := m.Called(ctx, text)
       return args.Get(0).([]float32), args.Error(1)
   }
   ```

## Troubleshooting

### Common Issues

1. **"Skipping PostgreSQL tests in short mode"**
   - Solution: Don't use `-short` flag or run integration tests specifically

2. **"POSTGRES_TEST_URL not set"**
   - Solution: Set the environment variable or create a .env file

3. **"pq: SSL is not enabled on the server"**
   - Solution: Add `?sslmode=disable` to your connection string

4. **Mock expectations not met**
   - Solution: Ensure your mock expectations match the actual queries

## Future Improvements

1. **Use pglite for unit tests**: Investigate using an in-memory PostgreSQL-compatible database for unit tests that need vector operations.

2. **Test containers**: Use testcontainers-go for spinning up PostgreSQL instances for tests.

3. **Parallel test execution**: Ensure tests can run in parallel safely.

4. **Coverage reports**: Integrate coverage reporting in CI/CD pipeline.