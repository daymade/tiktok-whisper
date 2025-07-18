package tests

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/pg"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/testutil"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// CrossDatabaseTestSuite provides a test suite that works across different database implementations
type CrossDatabaseTestSuite struct {
	name string
	dao  repository.TranscriptionDAO
	db   *sql.DB
}

// TestCrossDatabaseCompatibility tests that both database implementations behave consistently
func TestCrossDatabaseCompatibility(t *testing.T) {
	suites := []CrossDatabaseTestSuite{}

	// Always test SQLite
	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		sqliteDAO := &sqlite.SQLiteDB{}
		sqliteDAO = &sqlite.SQLiteDB{} // Using raw struct for testing
		// Set the db field manually for testing
		type sqliteDBInternal struct {
			db *sql.DB
		}
		(*sqliteDBInternal)(sqliteDAO).db = db

		suites = append(suites, CrossDatabaseTestSuite{
			name: "SQLite",
			dao:  sqliteDAO,
			db:   db,
		})

		// Test PostgreSQL if available
		if isPostgresAvailable() {
			testutil.WithTestDB(t, func(t *testing.T, pgDB *sql.DB) {
				postgresDAO := &pg.PostgresDB{}
				// Set the db field manually for testing
				type postgresDBInternal struct {
					db *sql.DB
				}
				(*postgresDBInternal)(postgresDAO).db = pgDB

				suites = append(suites, CrossDatabaseTestSuite{
					name: "PostgreSQL",
					dao:  postgresDAO,
					db:   pgDB,
				})

				// Run all cross-database tests
				runCrossDatabaseTests(t, suites)
			})
		} else {
			t.Log("PostgreSQL not available, running SQLite-only tests")
			// Run tests with just SQLite
			runCrossDatabaseTests(t, suites)
		}
	})
}

// runCrossDatabaseTests executes the test suite across all available databases
func runCrossDatabaseTests(t *testing.T, suites []CrossDatabaseTestSuite) {
	for _, suite := range suites {
		t.Run(suite.name, func(t *testing.T) {
			testBasicOperations(t, suite)
			testDataConsistency(t, suite)
			testErrorHandling(t, suite)
			testBoundaryConditions(t, suite)
		})
	}

	// If we have multiple databases, test cross-database data migration
	if len(suites) > 1 {
		testDataMigration(t, suites)
	}
}

// testBasicOperations tests basic CRUD operations
func testBasicOperations(t *testing.T, suite CrossDatabaseTestSuite) {
	t.Run("BasicOperations", func(t *testing.T) {
		// Clear any existing data
		testutil.CleanTestData(t, suite.db)

		// Test RecordToDB
		testTime := time.Now()
		suite.dao.RecordToDB(
			"basic_test_user",
			"/test/basic",
			"basic_test.mp3",
			"basic_test.mp3",
			120,
			"Basic test transcription",
			testTime,
			0,
			"",
		)

		// Test CheckIfFileProcessed
		id, err := suite.dao.CheckIfFileProcessed("basic_test.mp3")
		if err != nil {
			t.Errorf("[%s] Expected no error from CheckIfFileProcessed, got: %v", suite.name, err)
		}
		if id <= 0 {
			t.Errorf("[%s] Expected valid ID from CheckIfFileProcessed, got: %d", suite.name, id)
		}

		// Test CheckIfFileProcessed for non-existent file
		_, err = suite.dao.CheckIfFileProcessed("non_existent.mp3")
		if err == nil {
			t.Errorf("[%s] Expected error for non-existent file, got none", suite.name)
		}

		// Test GetAllByUser (note: PostgreSQL may not be implemented)
		transcriptions, err := suite.dao.GetAllByUser("basic_test_user")
		if err != nil && err.Error() != "not implemented" {
			t.Errorf("[%s] Unexpected error from GetAllByUser: %v", suite.name, err)
		}
		if err == nil {
			if len(transcriptions) != 1 {
				t.Errorf("[%s] Expected 1 transcription, got %d", suite.name, len(transcriptions))
			}
			if len(transcriptions) > 0 && transcriptions[0].User != "basic_test_user" {
				t.Errorf("[%s] Expected user 'basic_test_user', got '%s'", suite.name, transcriptions[0].User)
			}
		}
	})
}

// testDataConsistency tests data consistency across operations
func testDataConsistency(t *testing.T, suite CrossDatabaseTestSuite) {
	t.Run("DataConsistency", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		// Insert multiple records for the same user
		testData := []struct {
			fileName      string
			transcription string
			hasError      int
		}{
			{"file1.mp3", "Transcription 1", 0},
			{"file2.mp3", "Transcription 2", 0},
			{"file3.mp3", "", 1}, // Error case
			{"file4.mp3", "Transcription 4", 0},
		}

		for _, data := range testData {
			suite.dao.RecordToDB(
				"consistency_user",
				"/test/consistency",
				data.fileName,
				data.fileName,
				120,
				data.transcription,
				time.Now(),
				data.hasError,
				"",
			)
		}

		// Check that all files are recognized as processed (including error ones)
		for _, data := range testData {
			if data.hasError == 0 {
				id, err := suite.dao.CheckIfFileProcessed(data.fileName)
				if err != nil {
					t.Errorf("[%s] Expected file %s to be processed, got error: %v", suite.name, data.fileName, err)
				}
				if id <= 0 {
					t.Errorf("[%s] Expected valid ID for processed file %s, got: %d", suite.name, data.fileName, id)
				}
			} else {
				// Error files should not be considered "processed"
				_, err := suite.dao.CheckIfFileProcessed(data.fileName)
				if err == nil {
					t.Errorf("[%s] Expected error file %s to not be considered processed", suite.name, data.fileName)
				}
			}
		}

		// Count total records in database
		totalCount := testutil.GetTestDataCount(t, suite.db)
		if totalCount != len(testData) {
			t.Errorf("[%s] Expected %d total records, got %d", suite.name, len(testData), totalCount)
		}
	})
}

// testErrorHandling tests error scenarios
func testErrorHandling(t *testing.T, suite CrossDatabaseTestSuite) {
	t.Run("ErrorHandling", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		// Test error record insertion
		suite.dao.RecordToDB(
			"error_test_user",
			"/test/error",
			"error_file.mp3",
			"error_file.mp3",
			0,
			"",
			time.Now(),
			1,
			"Test error message",
		)

		// Test that error files are not considered processed
		_, err := suite.dao.CheckIfFileProcessed("error_file.mp3")
		if err == nil {
			t.Errorf("[%s] Expected error file to not be considered processed", suite.name)
		}

		// Test GetAllByUser should not return error records (if implemented)
		transcriptions, err := suite.dao.GetAllByUser("error_test_user")
		if err != nil && err.Error() != "not implemented" {
			t.Errorf("[%s] Unexpected error from GetAllByUser: %v", suite.name, err)
		}
		if err == nil {
			if len(transcriptions) != 0 {
				t.Errorf("[%s] Expected 0 transcriptions for user with only error records, got %d", suite.name, len(transcriptions))
			}
		}
	})
}

// testBoundaryConditions tests boundary conditions and edge cases
func testBoundaryConditions(t *testing.T, suite CrossDatabaseTestSuite) {
	t.Run("BoundaryConditions", func(t *testing.T) {
		testutil.CleanTestData(t, suite.db)

		// Test with empty strings
		suite.dao.RecordToDB("", "", "", "", 0, "", time.Now(), 0, "")

		// Test with very long strings
		longText := generateLongText(1000)
		suite.dao.RecordToDB(
			longText[:50], // Truncate user name to reasonable length
			"/very/long/path/"+longText[:50],
			longText[:100], // File names shouldn't be too long
			longText[:100],
			999999,
			longText,
			time.Now(),
			0,
			"",
		)

		// Test with unicode characters
		suite.dao.RecordToDB(
			"测试用户",
			"/测试/路径",
			"测试文件.mp3",
			"测试文件.mp3",
			180,
			"这是一个中文转录测试 🎵",
			time.Now(),
			0,
			"",
		)

		// Test with special characters that might cause SQL issues
		suite.dao.RecordToDB(
			"user'; DROP TABLE transcriptions; --",
			"/test/injection",
			"injection.mp3",
			"injection.mp3",
			60,
			"'; SELECT * FROM users; --",
			time.Now(),
			0,
			"",
		)

		// Verify the table still exists and has the expected number of records
		count := testutil.GetTestDataCount(t, suite.db)
		if count != 4 {
			t.Errorf("[%s] Expected 4 boundary test records, got %d", suite.name, count)
		}
	})
}

// testDataMigration tests data migration between different database implementations
func testDataMigration(t *testing.T, suites []CrossDatabaseTestSuite) {
	if len(suites) < 2 {
		return
	}

	t.Run("DataMigration", func(t *testing.T) {
		source := suites[0]
		target := suites[1]

		// Clean both databases
		testutil.CleanTestData(t, source.db)
		testutil.CleanTestData(t, target.db)

		// Insert test data into source
		testData := []struct {
			user          string
			fileName      string
			transcription string
			duration      int
			hasError      int
		}{
			{"migration_user_1", "file1.mp3", "Migration test 1", 120, 0},
			{"migration_user_1", "file2.mp3", "Migration test 2", 180, 0},
			{"migration_user_2", "file3.mp3", "Migration test 3", 240, 0},
			{"migration_user_2", "file4.mp3", "", 0, 1}, // Error record
		}

		for _, data := range testData {
			source.dao.RecordToDB(
				data.user,
				"/test/migration",
				data.fileName,
				data.fileName,
				data.duration,
				data.transcription,
				time.Now(),
				data.hasError,
				"",
			)
		}

		// Simulate migration by reading from source and writing to target
		// This tests that the data can be extracted and inserted consistently
		for _, data := range testData {
			// Check if file exists in source
			if data.hasError == 0 {
				_, err := source.dao.CheckIfFileProcessed(data.fileName)
				if err != nil {
					t.Errorf("Source [%s] should have processed file %s", source.name, data.fileName)
				}
			}

			// Insert into target
			target.dao.RecordToDB(
				data.user,
				"/test/migration",
				data.fileName,
				data.fileName,
				data.duration,
				data.transcription,
				time.Now(),
				data.hasError,
				"",
			)
		}

		// Verify both databases have the same number of records
		sourceCount := testutil.GetTestDataCount(t, source.db)
		targetCount := testutil.GetTestDataCount(t, target.db)

		if sourceCount != targetCount {
			t.Errorf("Migration failed: source [%s] has %d records, target [%s] has %d records",
				source.name, sourceCount, target.name, targetCount)
		}

		// Verify processed files are consistent
		for _, data := range testData {
			if data.hasError == 0 {
				_, sourceErr := source.dao.CheckIfFileProcessed(data.fileName)
				_, targetErr := target.dao.CheckIfFileProcessed(data.fileName)

				if (sourceErr == nil) != (targetErr == nil) {
					t.Errorf("File processing status inconsistent for %s: source error=%v, target error=%v",
						data.fileName, sourceErr, targetErr)
				}
			}
		}

		t.Logf("Successfully migrated %d records from %s to %s",
			sourceCount, source.name, target.name)
	})
}

// TestDatabaseSpecificFeatures tests features specific to each database implementation
func TestDatabaseSpecificFeatures(t *testing.T) {
	t.Run("SQLiteFeatures", func(t *testing.T) {
		testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
			// Test SQLite-specific features
			
			// Test WAL mode (if applicable)
			var walMode string
			err := db.QueryRow("PRAGMA journal_mode").Scan(&walMode)
			if err != nil {
				t.Logf("Could not query journal mode: %v", err)
			} else {
				t.Logf("SQLite journal mode: %s", walMode)
			}

			// Test foreign key enforcement
			var fkEnabled int
			err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
			if err != nil {
				t.Logf("Could not query foreign keys: %v", err)
			} else {
				t.Logf("SQLite foreign keys enabled: %d", fkEnabled)
			}
		})
	})

	if isPostgresAvailable() {
		t.Run("PostgreSQLFeatures", func(t *testing.T) {
			testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
				// Test PostgreSQL-specific features
				
				// Test version
				var version string
				err := db.QueryRow("SELECT version()").Scan(&version)
				if err != nil {
					t.Logf("Could not query PostgreSQL version: %v", err)
				} else {
					t.Logf("PostgreSQL version: %s", version[:50]+"...") // Truncate for readability
				}

				// Test transaction isolation level
				var isolationLevel string
				err = db.QueryRow("SHOW default_transaction_isolation").Scan(&isolationLevel)
				if err != nil {
					t.Logf("Could not query isolation level: %v", err)
				} else {
					t.Logf("PostgreSQL default isolation level: %s", isolationLevel)
				}

				// Test that we can use PostgreSQL-specific SQL
				var tableExists bool
				err = db.QueryRow(`
					SELECT EXISTS (
						SELECT FROM information_schema.tables 
						WHERE table_name = 'transcriptions'
					)
				`).Scan(&tableExists)
				
				if err != nil {
					t.Errorf("PostgreSQL information_schema query failed: %v", err)
				}
				if !tableExists {
					t.Error("transcriptions table should exist")
				}
			})
		})
	}
}

// Helper functions

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

	err = db.Ping()
	return err == nil
}

// generateLongText generates text of specified length for testing
func generateLongText(length int) string {
	text := "This is test content for compatibility testing. It includes various characters: áéíóú, 测试, 🎵, and symbols: !@#$%^&*(). "
	result := ""
	for len(result) < length {
		result += text
	}
	return result[:length]
}

// BenchmarkCrossDatabasePerformance compares performance across database implementations
func BenchmarkCrossDatabasePerformance(b *testing.B) {
	// Benchmark SQLite
	b.Run("SQLite", func(b *testing.B) {
		testutil.WithTestDB(&testing.T{}, func(t *testing.T, db *sql.DB) {
			sqliteDAO := &sqlite.SQLiteDB{}
			type sqliteDBInternal struct {
				db *sql.DB
			}
			(*sqliteDBInternal)(sqliteDAO).db = db

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sqliteDAO.RecordToDB(
					"benchmark_user",
					"/benchmark/input",
					"benchmark_file.mp3",
					"benchmark_file.mp3",
					120,
					"Benchmark transcription",
					time.Now(),
					0,
					"",
				)
			}
		})
	})

	// Benchmark PostgreSQL if available
	if isPostgresAvailable() {
		b.Run("PostgreSQL", func(b *testing.B) {
			testutil.WithTestDB(&testing.T{}, func(t *testing.T, db *sql.DB) {
				postgresDAO := &pg.PostgresDB{}
				type postgresDBInternal struct {
					db *sql.DB
				}
				(*postgresDBInternal)(postgresDAO).db = db

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					postgresDAO.RecordToDB(
						"benchmark_user",
						"/benchmark/input",
						"benchmark_file.mp3",
						"benchmark_file.mp3",
						120,
						"Benchmark transcription",
						time.Now(),
						0,
						"",
					)
				}
			})
		})
	}
}