// Package testutil provides comprehensive testing utilities for the tiktok-whisper application.
//
// This package contains four main components:
//
// 1. Database Test Helpers (db_helpers.go):
//   - SetupTestDB: Creates test databases with automatic cleanup
//   - SetupTestSQLite: Creates SQLite test databases
//   - SetupTestPostgres: Creates PostgreSQL test databases
//   - SeedTestData: Populates test databases with sample data
//   - Helper functions for database testing workflows
//
// 2. Mock Factory Functions (mock_factory.go):
//   - MockTranscriber: Mock implementation of the Transcriber interface
//   - MockTranscriptionDAO: Mock implementation of the TranscriptionDAO interface
//   - MockLogger: Mock logger for testing logging behavior
//   - MockFileSystem: Mock file system for testing file operations
//
// 3. Test Data Fixtures (fixtures.go):
//   - Predefined test data for transcriptions, file info, and configurations
//   - Sample API responses and error messages
//   - Helper functions for generating test data
//
// 4. Performance Benchmarking Utilities (benchmark.go):
//   - BenchmarkHelper: Utilities for timing and memory usage tracking
//   - BenchmarkRunner: Configurable benchmark execution
//   - Performance assertion helpers
//
// # Usage Examples
//
// ## Database Testing
//
//	func TestWithDatabase(t *testing.T) {
//	    testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
//	        // Your database test code here
//	        // Database is automatically cleaned up after the test
//	    })
//	}
//
//	func TestWithSeedData(t *testing.T) {
//	    testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
//	        // Your test code with pre-populated data
//	    })
//	}
//
// ## Mock Usage
//
//	func TestTranscription(t *testing.T) {
//	    // Create a mock transcriber
//	    mockTranscriber := testutil.NewMockTranscriber().
//	        WithDefaultResponse("Test transcription").
//	        WithError("/path/to/error.mp3", errors.New("test error"))
//
//	    // Create a mock DAO
//	    mockDAO := testutil.NewMockTranscriptionDAO().
//	        WithTranscriptions(testutil.TestTranscriptions).
//	        WithProcessedFile("test.mp3", 1)
//
//	    // Use mocks in your tests
//	    result, err := mockTranscriber.Transcript("/path/to/audio.mp3")
//	    // ... test assertions
//	}
//
// ## Benchmarking
//
//	func TestPerformance(t *testing.T) {
//	    bh := testutil.NewBenchmarkHelper("TranscriptionPerformance")
//	    bh.Start()
//
//	    // Your code to benchmark
//	    doSomeWork()
//
//	    bh.Stop()
//
//	    // Assert performance requirements
//	    bh.AssertDurationLessThan(t, 1*time.Second)
//	    bh.AssertMemoryUsageLessThan(t, 1024*1024) // 1MB
//
//	    t.Log(bh.Report())
//	}
//
// ## Using Test Fixtures
//
//	func TestWithFixtures(t *testing.T) {
//	    // Use predefined test data
//	    transcriptions := testutil.TestTranscriptions
//	    user := testutil.RandomTestUser()
//	    audioFile := testutil.RandomTestAudioFile()
//
//	    // Generate custom test data
//	    customTranscription := testutil.GenerateTestTranscription(
//	        1, "test_user", "test.mp3", 120.0, "Test transcription text")
//	}
//
// # Environment Configuration
//
// The database helpers support environment-based configuration:
//
//	POSTGRES_TEST_URL=postgres://user:pass@localhost/testdb?sslmode=disable
//	POSTGRES_TEST_HOST=localhost
//	POSTGRES_TEST_USER=testuser
//	POSTGRES_TEST_PASSWORD=testpass
//	POSTGRES_TEST_DB=testdb
//
// # Best Practices
//
// 1. Always use the helper functions for database setup to ensure proper cleanup
// 2. Use mocks to isolate components under test
// 3. Leverage fixtures for consistent test data
// 4. Use benchmarking utilities to validate performance requirements
// 5. Run tests with `go test -v ./...` to see detailed output
// 6. Use `go test -bench=.` to run benchmarks
//
// # Thread Safety
//
// All mock implementations are thread-safe and can be used in parallel tests.
// The benchmark helpers use mutexes to ensure thread-safe operations.
//
// # Memory Management
//
// Database test helpers automatically clean up temporary files and database connections.
// Mock implementations have Reset() methods to clear state between tests.
package testutil
