//go:build integration
// +build integration

package pg

import (
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"tiktok-whisper/internal/app/testutil"

	_ "github.com/lib/pq"
)

// TestNewPostgresDB_Integration tests the constructor with real database
func TestNewPostgresDB_Integration(t *testing.T) {
	// Skip if PostgreSQL is not available
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	tests := []struct {
		name             string
		connectionString string
		expectError      bool
	}{
		{
			name:             "valid_connection_string",
			connectionString: getTestConnectionString(),
			expectError:      false,
		},
		{
			name:             "invalid_host",
			connectionString: "postgres://postgres:postgres@invalid-host:5432/test_db?sslmode=disable",
			expectError:      true,
		},
		{
			name:             "invalid_credentials",
			connectionString: "postgres://wrong:wrong@localhost/test_db?sslmode=disable",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postgresDB, err := NewPostgresDB(tt.connectionString)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if postgresDB != nil {
					t.Error("Expected nil PostgresDB on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if postgresDB == nil {
					t.Error("Expected non-nil PostgresDB")
				} else {
					// Test that the database connection is working
					err = postgresDB.db.Ping()
					if err != nil {
						t.Errorf("Expected database connection to be working, got error: %v", err)
					}
					// Clean up
					postgresDB.Close()
				}
			}
		})
	}
}

// TestPostgresDB_FullIntegration tests complete workflow with real database
func TestPostgresDB_FullIntegration(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		// Test complete workflow
		user := "integration_test_user"
		fileName := "integration_test.mp3"
		transcription := "This is an integration test transcription"

		// 1. Check if file is already processed (should not be)
		_, err := postgresDB.CheckIfFileProcessed(fileName)
		if err == nil {
			t.Error("Expected error for non-existent file, got none")
		}

		// 2. Record a new transcription
		postgresDB.RecordToDB(
			user,
			"/test/integration",
			fileName,
			fileName,
			300,
			transcription,
			time.Now(),
			0,
			"",
		)

		// 3. Check if file is now processed
		id, err := postgresDB.CheckIfFileProcessed(fileName)
		if err != nil {
			t.Errorf("Expected no error for existing file, got: %v", err)
		}
		if id <= 0 {
			t.Errorf("Expected valid ID, got: %d", id)
		}

		// 4. Get all transcriptions for the user
		transcriptions, err := postgresDB.GetAllByUser(user)
		if err != nil {
			t.Errorf("Expected no error getting user transcriptions, got: %v", err)
		}

		// Find our transcription
		found := false
		for _, trans := range transcriptions {
			if trans.Mp3FileName == fileName {
				found = true
				if trans.Transcription != transcription {
					t.Errorf("Expected transcription '%s', got '%s'", transcription, trans.Transcription)
				}
				if trans.AudioDuration != 300 {
					t.Errorf("Expected duration 300, got %d", trans.AudioDuration)
				}
				break
			}
		}

		if !found {
			t.Error("Could not find the recorded transcription")
		}
	})
}

// TestPostgresDB_ConcurrentAccess_Integration tests concurrent database access
func TestPostgresDB_ConcurrentAccess_Integration(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		const numGoroutines = 10
		const recordsPerGoroutine = 5

		done := make(chan bool, numGoroutines)

		// Launch multiple goroutines to test concurrent access
		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				defer func() { done <- true }()

				for j := 0; j < recordsPerGoroutine; j++ {
					user := testutil.RandomTestUser()
					fileName := testutil.RandomTestAudioFile()
					transcription := testutil.RandomTestTranscriptionText()

					postgresDB.RecordToDB(
						user,
						"/test/concurrent",
						fileName,
						fileName,
						120,
						transcription,
						time.Now(),
						0,
						"",
					)

					// Test concurrent CheckIfFileProcessed calls
					_, _ = postgresDB.CheckIfFileProcessed(fileName)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify the total number of records
		count := testutil.GetTestDataCount(t, db)
		expectedCount := numGoroutines * recordsPerGoroutine
		if count != expectedCount {
			t.Errorf("Expected %d total records, got %d", expectedCount, count)
		}
	})
}

// TestPostgresDB_DataIntegrity_Integration tests data integrity with real PostgreSQL
func TestPostgresDB_DataIntegrity_Integration(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		// Test with various data types and edge cases
		testCases := []struct {
			name          string
			user          string
			fileName      string
			transcription string
			duration      int
		}{
			{
				name:          "empty_strings",
				user:          "",
				fileName:      "",
				transcription: "",
				duration:      0,
			},
			{
				name:          "unicode_characters",
				user:          "用户名测试",
				fileName:      "测试文件.mp3",
				transcription: "这是一个中文转录测试 🎵 with émojis and àccents",
				duration:      180,
			},
			{
				name:          "json_like_data",
				user:          "json_user",
				fileName:      "json_data.mp3",
				transcription: `{"type": "transcription", "content": "JSON-like data with quotes and brackets"}`,
				duration:      300,
			},
			{
				name:          "very_long_text",
				user:          "long_text_user",
				fileName:      "long_text.mp3",
				transcription: generateLongText(50000), // 50KB of text
				duration:      7200,
			},
			{
				name:          "special_sql_characters",
				user:          "sql_test'; DROP TABLE transcriptions; --",
				fileName:      "sql_injection.mp3",
				transcription: "'; SELECT * FROM users; --",
				duration:      60,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				postgresDB.RecordToDB(
					tc.user,
					"/test/integrity",
					tc.fileName,
					tc.fileName,
					tc.duration,
					tc.transcription,
					time.Now(),
					0,
					"",
				)

				// Verify data was stored correctly
				var storedUser, storedTranscription string
				var storedDuration int
				err := db.QueryRow(`
					SELECT "user", audio_duration, transcription 
					FROM transcriptions 
					WHERE file_name = $1
				`, tc.fileName).Scan(&storedUser, &storedDuration, &storedTranscription)

				if err != nil {
					t.Fatalf("Failed to retrieve data: %v", err)
				}

				if storedUser != tc.user {
					t.Errorf("User mismatch: expected %s, got %s", tc.user, storedUser)
				}
				if storedTranscription != tc.transcription {
					t.Errorf("Transcription mismatch: lengths %d vs %d", len(tc.transcription), len(storedTranscription))
				}
				if storedDuration != tc.duration {
					t.Errorf("Duration mismatch: expected %d, got %d", tc.duration, storedDuration)
				}

				// Verify that the table still exists (no SQL injection)
				var tableExists bool
				err = db.QueryRow(`
					SELECT EXISTS (
						SELECT FROM information_schema.tables 
						WHERE table_name = 'transcriptions'
					)
				`).Scan(&tableExists)

				if err != nil || !tableExists {
					t.Error("Table 'transcriptions' should still exist after insert")
				}
			})
		}
	})
}

// TestPostgresDB_TransactionHandling_Integration tests transaction behavior
func TestPostgresDB_TransactionHandling_Integration(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		// Test that individual operations are atomic
		initialCount := testutil.GetTestDataCount(t, db)

		// This should succeed
		postgresDB.RecordToDB(
			"transaction_test_user",
			"/test/transaction",
			"valid_file.mp3",
			"valid_file.mp3",
			120,
			"Valid transcription",
			time.Now(),
			0,
			"",
		)

		afterValidCount := testutil.GetTestDataCount(t, db)
		if afterValidCount != initialCount+1 {
			t.Errorf("Expected count to increase by 1, got %d -> %d", initialCount, afterValidCount)
		}
	})
}

// TestPostgresDB_GetAllByUser_Integration tests GetAllByUser with real data
func TestPostgresDB_GetAllByUser_Integration(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		tests := []struct {
			name         string
			userNickname string
			expectError  bool
			minCount     int // minimum expected count
		}{
			{
				name:         "existing_user_with_records",
				userNickname: "test_user_1",
				expectError:  false,
				minCount:     2, // Based on seeded data
			},
			{
				name:         "existing_user_with_one_record",
				userNickname: "test_user_2",
				expectError:  false,
				minCount:     1,
			},
			{
				name:         "non_existing_user",
				userNickname: "non_existent_user",
				expectError:  false,
				minCount:     0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				transcriptions, err := postgresDB.GetAllByUser(tt.userNickname)

				if tt.expectError && err == nil {
					t.Errorf("Expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}

				if len(transcriptions) < tt.minCount {
					t.Errorf("Expected at least %d transcriptions, got %d", tt.minCount, len(transcriptions))
				}

				// Verify all returned transcriptions belong to the user and have no errors
				for i, transcription := range transcriptions {
					if transcription.User != tt.userNickname {
						t.Errorf("Expected user %s, got %s at index %d", tt.userNickname, transcription.User, i)
					}
					if transcription.ErrorMessage != "" {
						t.Errorf("Expected empty error message, got %s at index %d", transcription.ErrorMessage, i)
					}
				}

				// Verify ordering (should be DESC by last_conversion_time)
				if len(transcriptions) > 1 {
					for i := 1; i < len(transcriptions); i++ {
						if transcriptions[i-1].LastConversionTime.Before(transcriptions[i].LastConversionTime) {
							t.Errorf("Expected transcriptions to be ordered by last_conversion_time DESC")
						}
					}
				}
			})
		}
	})
}

// TestPostgresDB_SpecialCharacters_Integration tests Unicode and special character support
func TestPostgresDB_SpecialCharacters_Integration(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		// Insert data with Unicode username
		unicodeUser := "测试用户"
		_, err := db.Exec(`
			INSERT INTO transcriptions (user_nickname, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
			VALUES ($1, '/test/unicode', 'unicode_test.mp3', 'unicode_test.mp3', 120, 'Unicode transcription', now(), 0, '')
		`, unicodeUser)
		if err != nil {
			t.Fatalf("Failed to insert unicode test data: %v", err)
		}

		// Test retrieving Unicode user data
		transcriptions, err := postgresDB.GetAllByUser(unicodeUser)
		if err != nil {
			t.Errorf("Failed to get transcriptions for Unicode user: %v", err)
		}

		if len(transcriptions) != 1 {
			t.Errorf("Expected 1 transcription for Unicode user, got %d", len(transcriptions))
		}

		// Insert data with SQL injection attempt in username
		sqlInjectionUser := "test'; DROP TABLE transcriptions; --"
		_, err = db.Exec(`
			INSERT INTO transcriptions (user_nickname, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
			VALUES ($1, '/test/sql', 'sql_test.mp3', 'sql_test.mp3', 60, 'SQL injection test', now(), 0, '')
		`, sqlInjectionUser)
		if err != nil {
			t.Fatalf("Failed to insert SQL test data: %v", err)
		}

		// Test retrieving data with SQL injection attempt
		transcriptions, err = postgresDB.GetAllByUser(sqlInjectionUser)
		if err != nil {
			t.Errorf("Failed to get transcriptions for SQL injection user: %v", err)
		}

		if len(transcriptions) != 1 {
			t.Errorf("Expected 1 transcription for SQL injection user, got %d", len(transcriptions))
		}

		// Verify table still exists
		var tableExists bool
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_name = 'transcriptions'
			)
		`).Scan(&tableExists)

		if err != nil || !tableExists {
			t.Error("Table 'transcriptions' should still exist after SQL injection attempt")
		}
	})
}

// Benchmark tests for PostgreSQL performance

// BenchmarkPostgresDB_RecordToDB benchmarks the RecordToDB method
func BenchmarkPostgresDB_RecordToDB(b *testing.B) {
	if !isPostgresAvailable() {
		b.Skip("PostgreSQL not available, skipping benchmark")
	}

	testutil.WithTestDB(&testing.T{}, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			postgresDB.RecordToDB(
				"benchmark_user",
				"/benchmark/input",
				"benchmark_file.mp3",
				"benchmark_file.mp3",
				120,
				"Benchmark transcription text for performance testing",
				time.Now(),
				0,
				"",
			)
		}
	})
}

// BenchmarkPostgresDB_CheckIfFileProcessed benchmarks the CheckIfFileProcessed method
func BenchmarkPostgresDB_CheckIfFileProcessed(b *testing.B) {
	if !isPostgresAvailable() {
		b.Skip("PostgreSQL not available, skipping benchmark")
	}

	testutil.WithSeekedTestDB(&testing.T{}, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = postgresDB.CheckIfFileProcessed("test_audio_1.mp3")
		}
	})
}

// Helper functions

// isPostgresAvailable checks if PostgreSQL is available for testing
func isPostgresAvailable() bool {
	// Check for environment variable indicating PostgreSQL is available
	if os.Getenv("POSTGRES_TEST_URL") != "" {
		return true
	}

	// Try to connect to default PostgreSQL instance
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost/postgres?sslmode=disable")
	if err != nil {
		return false
	}
	defer db.Close()

	err = db.Ping()
	return err == nil
}

// getTestConnectionString returns the test database connection string
func getTestConnectionString() string {
	if url := os.Getenv("POSTGRES_TEST_URL"); url != "" {
		return url
	}
	return "postgres://postgres:postgres@localhost/postgres?sslmode=disable"
}

// generateLongText generates text of specified length for testing
func generateLongText(length int) string {
	text := "This is a test transcription with repeated content for PostgreSQL testing. It includes unicode: 测试 and symbols: !@#$%^&*(). "
	result := ""
	for len(result) < length {
		result += text
	}
	return result[:length]
}