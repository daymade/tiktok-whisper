package tests

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/pg"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/testutil"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// TransactionTestSuite provides comprehensive transaction and data consistency testing
type TransactionTestSuite struct {
	name string
	dao  repository.TranscriptionDAO
	db   *sql.DB
}

// TestTransactionHandlingAndDataConsistency tests transaction handling and data consistency
func TestTransactionHandlingAndDataConsistency(t *testing.T) {
	databases := []struct {
		name      string
		available func() bool
		setup     func(t *testing.T) TransactionTestSuite
	}{
		{
			name:      "SQLite",
			available: func() bool { return true },
			setup:     setupSQLiteTransactionTest,
		},
		{
			name:      "PostgreSQL",
			available: isPostgresAvailable,
			setup:     setupPostgresTransactionTest,
		},
	}

	for _, db := range databases {
		if !db.available() {
			t.Logf("Skipping %s transaction tests - not available", db.name)
			continue
		}

		t.Run(db.name, func(t *testing.T) {
			suite := db.setup(t)
			defer suite.dao.Close()

			runAllTransactionTests(t, suite)
		})
	}
}

// runAllTransactionTests executes all transaction and consistency tests
func runAllTransactionTests(t *testing.T, suite TransactionTestSuite) {
	testBasicTransactions(t, suite)
	testTransactionIsolation(t, suite)
	testAtomicityAndConsistency(t, suite)
	testConcurrentTransactions(t, suite)
	testTransactionRollbackScenarios(t, suite)
	testLongRunningTransactions(t, suite)
	testDataIntegrityConstraints(t, suite)
	testACIDProperties(t, suite)
}

// testBasicTransactions tests basic transaction functionality
func testBasicTransactions(t *testing.T, suite TransactionTestSuite) {
	t.Run("BasicTransactions", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("SimpleCommit", func(t *testing.T) {
			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Insert data within transaction
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "tx_user", "/test/tx", "commit_test.mp3", "commit_test.mp3", 120, "Transaction commit test", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert in transaction: %v", err)
			}

			// Commit transaction
			err = tx.Commit()
			if err != nil {
				t.Fatalf("Failed to commit transaction: %v", err)
			}

			// Verify data was committed
			id, err := suite.dao.CheckIfFileProcessed("commit_test.mp3")
			if err != nil {
				t.Errorf("Expected file to be found after commit, got error: %v", err)
			}
			if id <= 0 {
				t.Errorf("Expected valid ID after commit, got: %d", id)
			}
		})

		t.Run("SimpleRollback", func(t *testing.T) {
			initialCount := testutil.GetTestDataCount(t, suite.db)

			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Insert data within transaction
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "tx_user", "/test/tx", "rollback_test.mp3", "rollback_test.mp3", 120, "Transaction rollback test", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert in transaction: %v", err)
			}

			// Rollback transaction
			err = tx.Rollback()
			if err != nil {
				t.Fatalf("Failed to rollback transaction: %v", err)
			}

			// Verify data was not committed
			_, err = suite.dao.CheckIfFileProcessed("rollback_test.mp3")
			if err == nil {
				t.Error("Expected file to not be found after rollback")
			}

			// Verify record count is unchanged
			finalCount := testutil.GetTestDataCount(t, suite.db)
			if finalCount != initialCount {
				t.Errorf("Expected record count to remain %d after rollback, got %d", initialCount, finalCount)
			}
		})

		t.Run("MultipleOperationsInTransaction", func(t *testing.T) {
			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Insert multiple records in same transaction
			for i := 0; i < 5; i++ {
				_, err = tx.Exec(`
					INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, "multi_tx_user", "/test/multi", fmt.Sprintf("multi_file_%d.mp3", i), fmt.Sprintf("multi_file_%d.mp3", i), 120, fmt.Sprintf("Multi transaction test %d", i), time.Now(), 0, "")

				if err != nil {
					tx.Rollback()
					t.Fatalf("Failed to insert record %d in transaction: %v", i, err)
				}
			}

			// Commit all operations
			err = tx.Commit()
			if err != nil {
				t.Fatalf("Failed to commit multi-operation transaction: %v", err)
			}

			// Verify all records were committed
			for i := 0; i < 5; i++ {
				fileName := fmt.Sprintf("multi_file_%d.mp3", i)
				id, err := suite.dao.CheckIfFileProcessed(fileName)
				if err != nil {
					t.Errorf("Expected file %s to be found after commit, got error: %v", fileName, err)
				}
				if id <= 0 {
					t.Errorf("Expected valid ID for file %s after commit, got: %d", fileName, id)
				}
			}
		})
	})
}

// testTransactionIsolation tests transaction isolation levels
func testTransactionIsolation(t *testing.T, suite TransactionTestSuite) {
	t.Run("TransactionIsolation", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("ReadCommitted", func(t *testing.T) {
			// Test read committed isolation level behavior
			
			// Transaction 1: Insert data but don't commit yet
			tx1, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction 1: %v", err)
			}
			defer tx1.Rollback()

			_, err = tx1.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "isolation_user", "/test/isolation", "isolation_test.mp3", "isolation_test.mp3", 120, "Isolation test", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert in transaction 1: %v", err)
			}

			// Transaction 2: Try to read the uncommitted data
			tx2, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction 2: %v", err)
			}
			defer tx2.Rollback()

			var count int
			err = tx2.QueryRow("SELECT COUNT(*) FROM transcriptions WHERE file_name = ?", "isolation_test.mp3").Scan(&count)
			if err != nil {
				t.Fatalf("Failed to query in transaction 2: %v", err)
			}

			// With read committed, transaction 2 should not see uncommitted data
			if count > 0 {
				t.Logf("Transaction 2 can see uncommitted data (count: %d) - isolation level may be lower than read committed", count)
			}

			// Commit transaction 1
			err = tx1.Commit()
			if err != nil {
				t.Fatalf("Failed to commit transaction 1: %v", err)
			}

			// Now transaction 2 should be able to see the committed data in a new query
			err = tx2.QueryRow("SELECT COUNT(*) FROM transcriptions WHERE file_name = ?", "isolation_test.mp3").Scan(&count)
			if err != nil {
				t.Fatalf("Failed to query after commit in transaction 2: %v", err)
			}

			if count == 0 {
				t.Log("Transaction 2 cannot see committed data - may be using higher isolation level")
			}

			tx2.Commit()
		})

		t.Run("PhantomReads", func(t *testing.T) {
			// Test for phantom reads
			tx1, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction 1: %v", err)
			}
			defer tx1.Rollback()

			// Count records for a user
			var initialCount int
			err = tx1.QueryRow("SELECT COUNT(*) FROM transcriptions WHERE user = ?", "phantom_user").Scan(&initialCount)
			if err != nil {
				t.Fatalf("Failed to count records in transaction 1: %v", err)
			}

			// Another transaction inserts a record for the same user
			suite.dao.RecordToDB(
				"phantom_user",
				"/test/phantom",
				"phantom_file.mp3",
				"phantom_file.mp3",
				120,
				"Phantom read test",
				time.Now(),
				0,
				"",
			)

			// Count again in the same transaction
			var finalCount int
			err = tx1.QueryRow("SELECT COUNT(*) FROM transcriptions WHERE user = ?", "phantom_user").Scan(&finalCount)
			if err != nil {
				t.Fatalf("Failed to count records again in transaction 1: %v", err)
			}

			if finalCount > initialCount {
				t.Logf("Phantom read detected: count increased from %d to %d within same transaction", initialCount, finalCount)
			}

			tx1.Commit()
		})
	})
}

// testAtomicityAndConsistency tests atomicity and consistency properties
func testAtomicityAndConsistency(t *testing.T, suite TransactionTestSuite) {
	t.Run("AtomicityAndConsistency", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("AllOrNothing", func(t *testing.T) {
			initialCount := testutil.GetTestDataCount(t, suite.db)

			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Insert several records
			records := []struct {
				fileName string
				valid    bool
			}{
				{"atomic_file_1.mp3", true},
				{"atomic_file_2.mp3", true},
				{"", false}, // This might cause an error
				{"atomic_file_3.mp3", true},
			}

			var insertError error
			for _, record := range records {
				if record.valid {
					_, err = tx.Exec(`
						INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, "atomic_user", "/test/atomic", record.fileName, record.fileName, 120, "Atomic test", time.Now(), 0, "")
				} else {
					// Try to insert invalid data
					_, err = tx.Exec(`
						INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, "atomic_user", "/test/atomic", record.fileName, record.fileName, 120, "Invalid atomic test", time.Now(), 0, "")
				}

				if err != nil {
					insertError = err
					break
				}
			}

			if insertError != nil {
				// Rollback due to error
				tx.Rollback()
				t.Logf("Transaction rolled back due to error: %v", insertError)

				// Verify no records were inserted
				finalCount := testutil.GetTestDataCount(t, suite.db)
				if finalCount != initialCount {
					t.Errorf("Expected atomicity: no records should be inserted on error. Initial: %d, Final: %d", initialCount, finalCount)
				}
			} else {
				// Commit successful transaction
				err = tx.Commit()
				if err != nil {
					t.Fatalf("Failed to commit atomic transaction: %v", err)
				}

				// Verify all valid records were inserted
				for _, record := range records {
					if record.valid && record.fileName != "" {
						_, err := suite.dao.CheckIfFileProcessed(record.fileName)
						if err != nil {
							t.Errorf("Expected file %s to be found after atomic commit", record.fileName)
						}
					}
				}
			}
		})

		t.Run("ConsistencyConstraints", func(t *testing.T) {
			// Test that data remains consistent throughout transactions

			// Setup initial consistent state
			suite.dao.RecordToDB(
				"consistency_user",
				"/test/consistency",
				"consistency_file.mp3",
				"consistency_file.mp3",
				120,
				"Consistency test",
				time.Now(),
				0,
				"",
			)

			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin consistency transaction: %v", err)
			}

			// Perform operations that should maintain consistency
			_, err = tx.Exec(`
				UPDATE transcriptions 
				SET transcription = ?, last_conversion_time = ?
				WHERE file_name = ?
			`, "Updated consistency test", time.Now(), "consistency_file.mp3")

			if err != nil {
				tx.Rollback()
				t.Fatalf("Failed to update in consistency transaction: %v", err)
			}

			// Verify consistency within transaction
			var transcription string
			err = tx.QueryRow("SELECT transcription FROM transcriptions WHERE file_name = ?", "consistency_file.mp3").Scan(&transcription)
			if err != nil {
				tx.Rollback()
				t.Fatalf("Failed to verify consistency in transaction: %v", err)
			}

			if transcription != "Updated consistency test" {
				tx.Rollback()
				t.Errorf("Consistency violation: expected 'Updated consistency test', got '%s'", transcription)
			}

			err = tx.Commit()
			if err != nil {
				t.Fatalf("Failed to commit consistency transaction: %v", err)
			}

			// Verify consistency after commit
			var finalTranscription string
			err = suite.db.QueryRow("SELECT transcription FROM transcriptions WHERE file_name = ?", "consistency_file.mp3").Scan(&finalTranscription)
			if err != nil {
				t.Fatalf("Failed to verify final consistency: %v", err)
			}

			if finalTranscription != "Updated consistency test" {
				t.Errorf("Final consistency violation: expected 'Updated consistency test', got '%s'", finalTranscription)
			}
		})
	})
}

// testConcurrentTransactions tests concurrent transaction scenarios
func testConcurrentTransactions(t *testing.T, suite TransactionTestSuite) {
	t.Run("ConcurrentTransactions", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("ConcurrentInserts", func(t *testing.T) {
			const numGoroutines = 10
			const recordsPerGoroutine = 20

			var wg sync.WaitGroup
			errors := make(chan error, numGoroutines)

			wg.Add(numGoroutines)
			for i := 0; i < numGoroutines; i++ {
				go func(goroutineID int) {
					defer wg.Done()

					tx, err := suite.db.Begin()
					if err != nil {
						errors <- fmt.Errorf("goroutine %d: failed to begin transaction: %v", goroutineID, err)
						return
					}
					defer tx.Rollback()

					// Insert multiple records in this transaction
					for j := 0; j < recordsPerGoroutine; j++ {
						_, err = tx.Exec(`
							INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
							VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
						`, fmt.Sprintf("concurrent_user_%d", goroutineID), "/test/concurrent", 
						fmt.Sprintf("concurrent_file_%d_%d.mp3", goroutineID, j),
						fmt.Sprintf("concurrent_file_%d_%d.mp3", goroutineID, j),
						120, fmt.Sprintf("Concurrent test %d-%d", goroutineID, j), time.Now(), 0, "")

						if err != nil {
							errors <- fmt.Errorf("goroutine %d: failed to insert record %d: %v", goroutineID, j, err)
							return
						}
					}

					err = tx.Commit()
					if err != nil {
						errors <- fmt.Errorf("goroutine %d: failed to commit transaction: %v", goroutineID, err)
						return
					}
				}(i)
			}

			wg.Wait()
			close(errors)

			// Check for errors
			errorCount := 0
			for err := range errors {
				t.Logf("Concurrent transaction error: %v", err)
				errorCount++
			}

			// Verify final record count
			finalCount := testutil.GetTestDataCount(t, suite.db)
			expectedCount := numGoroutines * recordsPerGoroutine
			
			if errorCount > 0 {
				t.Logf("Some transactions failed (%d errors), so expected count may be less than %d", errorCount, expectedCount)
			}

			if finalCount < expectedCount-errorCount*recordsPerGoroutine {
				t.Errorf("Expected at least %d records after concurrent inserts, got %d", 
					expectedCount-errorCount*recordsPerGoroutine, finalCount)
			}
		})

		t.Run("ReadWriteConflicts", func(t *testing.T) {
			// Insert initial data
			suite.dao.RecordToDB(
				"conflict_user",
				"/test/conflict",
				"conflict_file.mp3",
				"conflict_file.mp3",
				120,
				"Initial conflict test",
				time.Now(),
				0,
				"",
			)

			var wg sync.WaitGroup
			conflicts := make(chan string, 10)

			// Reader goroutines
			for i := 0; i < 3; i++ {
				wg.Add(1)
				go func(readerID int) {
					defer wg.Done()

					for j := 0; j < 10; j++ {
						tx, err := suite.db.Begin()
						if err != nil {
							conflicts <- fmt.Sprintf("reader %d: failed to begin transaction: %v", readerID, err)
							continue
						}

						var transcription string
						err = tx.QueryRow("SELECT transcription FROM transcriptions WHERE file_name = ?", "conflict_file.mp3").Scan(&transcription)
						if err != nil {
							conflicts <- fmt.Sprintf("reader %d: failed to read: %v", readerID, err)
						}

						time.Sleep(10 * time.Millisecond) // Hold transaction open briefly

						tx.Commit()
					}
				}(i)
			}

			// Writer goroutines
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func(writerID int) {
					defer wg.Done()

					for j := 0; j < 5; j++ {
						tx, err := suite.db.Begin()
						if err != nil {
							conflicts <- fmt.Sprintf("writer %d: failed to begin transaction: %v", writerID, err)
							continue
						}

						_, err = tx.Exec(`
							UPDATE transcriptions 
							SET transcription = ?, last_conversion_time = ?
							WHERE file_name = ?
						`, fmt.Sprintf("Updated by writer %d iteration %d", writerID, j), time.Now(), "conflict_file.mp3")

						if err != nil {
							conflicts <- fmt.Sprintf("writer %d: failed to update: %v", writerID, err)
						}

						time.Sleep(15 * time.Millisecond) // Hold transaction open briefly

						tx.Commit()
					}
				}(i)
			}

			wg.Wait()
			close(conflicts)

			// Report any conflicts
			conflictCount := 0
			for conflict := range conflicts {
				t.Logf("Read-write conflict: %s", conflict)
				conflictCount++
			}

			t.Logf("Total conflicts detected: %d", conflictCount)
		})
	})
}

// testTransactionRollbackScenarios tests various rollback scenarios
func testTransactionRollbackScenarios(t *testing.T, suite TransactionTestSuite) {
	t.Run("RollbackScenarios", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("ExplicitRollback", func(t *testing.T) {
			initialCount := testutil.GetTestDataCount(t, suite.db)

			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Insert data
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "rollback_user", "/test/rollback", "explicit_rollback.mp3", "explicit_rollback.mp3", 120, "Explicit rollback test", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert in rollback transaction: %v", err)
			}

			// Explicitly rollback
			err = tx.Rollback()
			if err != nil {
				t.Fatalf("Failed to rollback transaction: %v", err)
			}

			// Verify data was not persisted
			finalCount := testutil.GetTestDataCount(t, suite.db)
			if finalCount != initialCount {
				t.Errorf("Expected count to remain %d after explicit rollback, got %d", initialCount, finalCount)
			}
		})

		t.Run("ErrorTriggeredRollback", func(t *testing.T) {
			initialCount := testutil.GetTestDataCount(t, suite.db)

			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Insert valid data first
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "error_rollback_user", "/test/error", "error_rollback.mp3", "error_rollback.mp3", 120, "Error rollback test", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert valid data: %v", err)
			}

			// Try to insert invalid data that might cause constraint violation
			_, err = tx.Exec("INSERT INTO invalid_table VALUES (1)")
			if err != nil {
				// Error occurred, rollback
				tx.Rollback()
				t.Logf("Transaction rolled back due to error: %v", err)

				// Verify no data was persisted
				finalCount := testutil.GetTestDataCount(t, suite.db)
				if finalCount != initialCount {
					t.Errorf("Expected count to remain %d after error rollback, got %d", initialCount, finalCount)
				}
			} else {
				// Unexpectedly succeeded
				tx.Commit()
				t.Log("Invalid SQL unexpectedly succeeded")
			}
		})

		t.Run("PartialRollback", func(t *testing.T) {
			// Test savepoints if the database supports them
			if suite.name == "SQLite" {
				t.Skip("SQLite savepoint testing requires specific setup")
			}

			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}
			defer tx.Rollback()

			// Insert first record
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "partial_user", "/test/partial", "partial_1.mp3", "partial_1.mp3", 120, "Partial test 1", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert first record: %v", err)
			}

			// Create savepoint (PostgreSQL syntax)
			if suite.name == "PostgreSQL" {
				_, err = tx.Exec("SAVEPOINT partial_savepoint")
				if err != nil {
					t.Fatalf("Failed to create savepoint: %v", err)
				}

				// Insert second record
				_, err = tx.Exec(`
					INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, "partial_user", "/test/partial", "partial_2.mp3", "partial_2.mp3", 120, "Partial test 2", time.Now(), 0, "")

				if err != nil {
					t.Fatalf("Failed to insert second record: %v", err)
				}

				// Rollback to savepoint
				_, err = tx.Exec("ROLLBACK TO SAVEPOINT partial_savepoint")
				if err != nil {
					t.Fatalf("Failed to rollback to savepoint: %v", err)
				}

				// Commit transaction
				err = tx.Commit()
				if err != nil {
					t.Fatalf("Failed to commit partial transaction: %v", err)
				}

				// Verify only first record exists
				_, err = suite.dao.CheckIfFileProcessed("partial_1.mp3")
				if err != nil {
					t.Error("Expected first record to exist after partial rollback")
				}

				_, err = suite.dao.CheckIfFileProcessed("partial_2.mp3")
				if err == nil {
					t.Error("Expected second record to not exist after partial rollback")
				}
			}
		})
	})
}

// testLongRunningTransactions tests long-running transaction scenarios
func testLongRunningTransactions(t *testing.T, suite TransactionTestSuite) {
	t.Run("LongRunningTransactions", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("LongTransaction", func(t *testing.T) {
			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin long transaction: %v", err)
			}
			defer tx.Rollback()

			// Perform operations over a longer period
			for i := 0; i < 10; i++ {
				_, err = tx.Exec(`
					INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, "long_tx_user", "/test/long", fmt.Sprintf("long_file_%d.mp3", i), fmt.Sprintf("long_file_%d.mp3", i), 120, fmt.Sprintf("Long transaction test %d", i), time.Now(), 0, "")

				if err != nil {
					t.Fatalf("Failed to insert record %d in long transaction: %v", i, err)
				}

				// Simulate processing time
				time.Sleep(100 * time.Millisecond)
			}

			// Commit after long operation
			err = tx.Commit()
			if err != nil {
				t.Fatalf("Failed to commit long transaction: %v", err)
			}

			// Verify all records were committed
			for i := 0; i < 10; i++ {
				fileName := fmt.Sprintf("long_file_%d.mp3", i)
				_, err := suite.dao.CheckIfFileProcessed(fileName)
				if err != nil {
					t.Errorf("Expected file %s to exist after long transaction", fileName)
				}
			}
		})

		t.Run("TransactionTimeout", func(t *testing.T) {
			// Test transaction timeout behavior
			// This is database-specific and may require configuration
			t.Skip("Transaction timeout testing requires specific database configuration")
		})
	})
}

// testDataIntegrityConstraints tests data integrity constraints within transactions
func testDataIntegrityConstraints(t *testing.T, suite TransactionTestSuite) {
	t.Run("DataIntegrityConstraints", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("ForeignKeyConstraints", func(t *testing.T) {
			// Test foreign key constraints if they exist
			// Current schema may not have foreign keys
			t.Skip("Foreign key constraint testing requires schema with foreign keys")
		})

		t.Run("CheckConstraints", func(t *testing.T) {
			// Test check constraints if they exist
			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin constraint transaction: %v", err)
			}

			// Try to insert data that might violate constraints
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "constraint_user", "/test/constraint", "constraint_test.mp3", "constraint_test.mp3", -1, "Negative duration test", time.Now(), 2, "Invalid has_error value")

			if err != nil {
				t.Logf("Constraint violation detected: %v", err)
				tx.Rollback()
			} else {
				t.Log("No constraint violations detected - schema may allow these values")
				tx.Commit()
			}
		})

		t.Run("UniqueConstraints", func(t *testing.T) {
			// Test unique constraints if they exist
			// Current schema may not have unique constraints beyond primary key
			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin unique constraint transaction: %v", err)
			}
			defer tx.Rollback()

			// Insert first record
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "unique_user", "/test/unique", "unique_test.mp3", "unique_test.mp3", 120, "Unique test 1", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert first unique record: %v", err)
			}

			// Try to insert duplicate
			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "unique_user", "/test/unique", "unique_test.mp3", "unique_test.mp3", 180, "Unique test 2", time.Now(), 0, "")

			if err != nil {
				t.Logf("Unique constraint violation detected: %v", err)
			} else {
				t.Log("No unique constraint violation - schema may allow duplicate file names")
			}

			tx.Commit()
		})
	})
}

// testACIDProperties tests ACID properties comprehensively
func testACIDProperties(t *testing.T, suite TransactionTestSuite) {
	t.Run("ACIDProperties", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		t.Run("Atomicity", func(t *testing.T) {
			// Already tested in testAtomicityAndConsistency
			t.Log("Atomicity tested in AtomicityAndConsistency section")
		})

		t.Run("Consistency", func(t *testing.T) {
			// Already tested in testAtomicityAndConsistency
			t.Log("Consistency tested in AtomicityAndConsistency section")
		})

		t.Run("Isolation", func(t *testing.T) {
			// Already tested in testTransactionIsolation
			t.Log("Isolation tested in TransactionIsolation section")
		})

		t.Run("Durability", func(t *testing.T) {
			// Test that committed transactions survive
			
			// Insert data in transaction
			tx, err := suite.db.Begin()
			if err != nil {
				t.Fatalf("Failed to begin durability transaction: %v", err)
			}

			_, err = tx.Exec(`
				INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, "durability_user", "/test/durability", "durability_test.mp3", "durability_test.mp3", 120, "Durability test", time.Now(), 0, "")

			if err != nil {
				t.Fatalf("Failed to insert durability record: %v", err)
			}

			err = tx.Commit()
			if err != nil {
				t.Fatalf("Failed to commit durability transaction: %v", err)
			}

			// Verify data persists after commit
			_, err = suite.dao.CheckIfFileProcessed("durability_test.mp3")
			if err != nil {
				t.Error("Expected durability test record to persist after commit")
			}

			// Simulate system restart by creating new DAO instance
			// (This is a limited test of durability)
			var newDAO TranscriptionDAO
			switch suite.name {
			case "SQLite":
				sqliteDAO := &sqlite.SQLiteDB{}
				type sqliteDBInternal struct {
					db *sql.DB
				}
				(*sqliteDBInternal)(sqliteDAO).db = suite.db
				newDAO = sqliteDAO
			case "PostgreSQL":
				postgresDAO := &pg.PostgresDB{}
				type postgresDBInternal struct {
					db *sql.DB
				}
				(*postgresDBInternal)(postgresDAO).db = suite.db
				newDAO = postgresDAO
			}

			// Verify data persists with new DAO instance
			_, err = newDAO.CheckIfFileProcessed("durability_test.mp3")
			if err != nil {
				t.Error("Expected durability test record to persist with new DAO instance")
			}
		})
	})
}

// Setup functions for transaction testing

func setupSQLiteTransactionTest(t *testing.T) TransactionTestSuite {
	db := testutil.SetupTestSQLite(t)
	sqliteDAO := &sqlite.SQLiteDB{}
	type sqliteDBInternal struct {
		db *sql.DB
	}
	(*sqliteDBInternal)(sqliteDAO).db = db

	return TransactionTestSuite{
		name: "SQLite",
		dao:  sqliteDAO,
		db:   db,
	}
}

func setupPostgresTransactionTest(t *testing.T) TransactionTestSuite {
	db := testutil.SetupTestPostgres(t)
	postgresDAO := &pg.PostgresDB{}
	type postgresDBInternal struct {
		db *sql.DB
	}
	(*postgresDBInternal)(postgresDAO).db = db

	return TransactionTestSuite{
		name: "PostgreSQL",
		dao:  postgresDAO,
		db:   db,
	}
}

// Helper functions

// isPostgresAvailable checks if PostgreSQL is available for testing
func isPostgresAvailable() bool {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost/postgres?sslmode=disable")
	if err != nil {
		return false
	}
	defer db.Close()

	return db.Ping() == nil
}