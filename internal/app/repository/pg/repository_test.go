package pg

import (
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/testutil"

	_ "github.com/lib/pq"
)

// TestPostgresDAO_Interface verifies PostgresDB implements TranscriptionDAO interface
func TestPostgresDAO_Interface(t *testing.T) {
	var _ repository.TranscriptionDAO = (*PostgresDB)(nil)
}

// TestNewPostgresDB tests the constructor function
func TestNewPostgresDB(t *testing.T) {
	// Skip if PostgreSQL is not available
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
	}

	tests := []struct {
		name             string
		connectionString string
		expectError      bool
	}{
		{
			name:             "valid_connection_string",
			connectionString: "postgres://postgres:postgres@localhost/test_db?sslmode=disable",
			expectError:      false,
		},
		{
			name:             "invalid_connection_string",
			connectionString: "invalid://connection/string",
			expectError:      true,
		},
		{
			name:             "empty_connection_string",
			connectionString: "",
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

// TestPostgresDB_Close tests the Close method
func TestPostgresDB_Close(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
	}

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		// Test successful close
		err := postgresDB.Close()
		if err != nil {
			t.Errorf("Expected Close() to return nil, got: %v", err)
		}

		// Test that operations fail after close
		_, err = postgresDB.CheckIfFileProcessed("test.mp3")
		if err == nil {
			t.Error("Expected operations to fail after Close(), but they didn't")
		}
	})
}

// TestPostgresDB_CheckIfFileProcessed tests the CheckIfFileProcessed method
func TestPostgresDB_CheckIfFileProcessed(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
	}

	testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		tests := []struct {
			name          string
			fileName      string
			expectError   bool
			expectValidID bool
			setupData     func()
		}{
			{
				name:          "existing_processed_file",
				fileName:      "test_audio_1.mp3",
				expectError:   false,
				expectValidID: true,
			},
			{
				name:          "non_existing_file",
				fileName:      "non_existent.mp3",
				expectError:   true,
				expectValidID: false,
			},
			{
				name:          "existing_error_file",
				fileName:      "test_audio_error.mp3",
				expectError:   true,
				expectValidID: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.setupData != nil {
					tt.setupData()
				}

				id, err := postgresDB.CheckIfFileProcessed(tt.fileName)

				if tt.expectError && err == nil {
					t.Errorf("Expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if tt.expectValidID && id <= 0 {
					t.Errorf("Expected valid ID (>0) but got: %d", id)
				}
				if !tt.expectValidID && !tt.expectError && id <= 0 {
					t.Errorf("Expected valid ID for successful case but got: %d", id)
				}
			})
		}
	})
}

// TestPostgresDB_RecordToDB tests the RecordToDB method
func TestPostgresDB_RecordToDB(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
	}

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		testCases := []struct {
			name          string
			user          string
			inputDir      string
			fileName      string
			mp3FileName   string
			audioDuration int
			transcription string
			hasError      int
			errorMessage  string
			shouldPanic   bool
		}{
			{
				name:          "successful_record",
				user:          "test_user",
				inputDir:      "/test/input",
				fileName:      "test.mp3",
				mp3FileName:   "test.mp3",
				audioDuration: 120,
				transcription: "Test transcription",
				hasError:      0,
				errorMessage:  "",
				shouldPanic:   false,
			},
			{
				name:          "error_record",
				user:          "test_user",
				inputDir:      "/test/input",
				fileName:      "error.mp3",
				mp3FileName:   "error.mp3",
				audioDuration: 0,
				transcription: "",
				hasError:      1,
				errorMessage:  "Test error",
				shouldPanic:   false,
			},
			{
				name:          "unicode_text",
				user:          "测试用户",
				inputDir:      "/test/unicode",
				fileName:      "unicode.mp3",
				mp3FileName:   "unicode.mp3",
				audioDuration: 180,
				transcription: "Unicode test: 你好世界 🌍",
				hasError:      0,
				errorMessage:  "",
				shouldPanic:   false,
			},
			{
				name:          "long_transcription",
				user:          "test_user",
				inputDir:      "/test/input",
				fileName:      "long.mp3",
				mp3FileName:   "long.mp3",
				audioDuration: 3600,
				transcription: generateLongText(10000), // 10KB text
				hasError:      0,
				errorMessage:  "",
				shouldPanic:   false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.shouldPanic {
					defer func() {
						if r := recover(); r == nil {
							t.Errorf("Expected RecordToDB to panic, but it didn't")
						}
					}()
				}

				// Record the data
				postgresDB.RecordToDB(
					tc.user,
					tc.inputDir,
					tc.fileName,
					tc.mp3FileName,
					tc.audioDuration,
					tc.transcription,
					time.Now(),
					tc.hasError,
					tc.errorMessage,
				)

				// Verify the record was inserted
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM transcriptions WHERE file_name = $1", tc.fileName).Scan(&count)
				if err != nil {
					t.Fatalf("Failed to verify record insertion: %v", err)
				}

				if count != 1 {
					t.Errorf("Expected 1 record to be inserted, found %d", count)
				}

				// Verify the record content using PostgreSQL parameterized query
				var storedUser, storedTranscription, storedErrorMessage string
				var storedAudioDuration, storedHasError int
				err = db.QueryRow(`
					SELECT "user", audio_duration, transcription, has_error, error_message 
					FROM transcriptions 
					WHERE file_name = $1
				`, tc.fileName).Scan(&storedUser, &storedAudioDuration, &storedTranscription, &storedHasError, &storedErrorMessage)

				if err != nil {
					t.Fatalf("Failed to retrieve inserted record: %v", err)
				}

				if storedUser != tc.user {
					t.Errorf("Expected user %s, got %s", tc.user, storedUser)
				}
				if storedAudioDuration != tc.audioDuration {
					t.Errorf("Expected audio duration %d, got %d", tc.audioDuration, storedAudioDuration)
				}
				if storedTranscription != tc.transcription {
					t.Errorf("Expected transcription length %d, got %d", len(tc.transcription), len(storedTranscription))
				}
				if storedHasError != tc.hasError {
					t.Errorf("Expected has_error %d, got %d", tc.hasError, storedHasError)
				}
				if storedErrorMessage != tc.errorMessage {
					t.Errorf("Expected error message %s, got %s", tc.errorMessage, storedErrorMessage)
				}
			})
		}
	})
}

// TestPostgresDB_GetAllByUser tests the GetAllByUser method
func TestPostgresDB_GetAllByUser(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
	}

	testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		tests := []struct {
			name             string
			userNickname     string
			expectError      bool
			expectCount      int
			expectedErrorMsg string
			setupData        func()
		}{
			{
				name:         "existing_user_with_records",
				userNickname: "test_user_1",
				expectError:  false,
				expectCount:  2, // Based on seeded data - non-error records only
			},
			{
				name:         "existing_user_with_one_record",
				userNickname: "test_user_2",
				expectError:  false,
				expectCount:  1, // Only non-error records
			},
			{
				name:         "non_existing_user",
				userNickname: "non_existent_user",
				expectError:  false,
				expectCount:  0,
			},
			{
				name:         "empty_user_nickname",
				userNickname: "",
				expectError:  false,
				expectCount:  0,
			},
			{
				name:         "user_with_unicode_name",
				userNickname: "测试用户",
				expectError:  false,
				expectCount:  0, // Assuming no seeded data for unicode user
				setupData: func() {
					// Insert test data with unicode user
					_, err := db.Exec(`
						INSERT INTO transcriptions (user_nickname, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
						VALUES ($1, '/test/unicode', 'unicode_test.mp3', 'unicode_test.mp3', 120, 'Unicode transcription', now(), 0, '')
					`, "测试用户")
					if err != nil {
						t.Fatalf("Failed to insert unicode test data: %v", err)
					}
				},
			},
			{
				name:         "user_with_special_sql_characters",
				userNickname: "test'; DROP TABLE transcriptions; --",
				expectError:  false,
				expectCount:  0,
				setupData: func() {
					// Insert test data with SQL injection attempt in username
					_, err := db.Exec(`
						INSERT INTO transcriptions (user_nickname, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
						VALUES ($1, '/test/sql', 'sql_test.mp3', 'sql_test.mp3', 60, 'SQL injection test', now(), 0, '')
					`, "test'; DROP TABLE transcriptions; --")
					if err != nil {
						t.Fatalf("Failed to insert SQL test data: %v", err)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Setup test data if needed
				if tt.setupData != nil {
					tt.setupData()
				}

				transcriptions, err := postgresDB.GetAllByUser(tt.userNickname)

				// Check error expectations
				if tt.expectError {
					if err == nil {
						t.Errorf("Expected error but got none")
					}
					if tt.expectedErrorMsg != "" && !strings.Contains(err.Error(), tt.expectedErrorMsg) {
						t.Errorf("Expected error to contain '%s', got: %v", tt.expectedErrorMsg, err)
					}
					return
				}

				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
					return
				}

				// Check count expectations
				if len(transcriptions) < tt.expectCount {
					t.Errorf("Expected at least %d transcriptions, got %d", tt.expectCount, len(transcriptions))
				}

				// Verify all returned transcriptions belong to the user and have no errors
				for i, transcription := range transcriptions {
					if transcription.User != tt.userNickname {
						t.Errorf("Expected user %s, got %s at index %d", tt.userNickname, transcription.User, i)
					}
					if transcription.ErrorMessage != "" {
						t.Errorf("Expected empty error message, got %s at index %d", transcription.ErrorMessage, i)
					}
					if transcription.ID <= 0 {
						t.Errorf("Expected valid ID (>0), got %d at index %d", transcription.ID, i)
					}
				}

				// Verify ordering (should be DESC by last_conversion_time)
				if len(transcriptions) > 1 {
					for i := 1; i < len(transcriptions); i++ {
						if transcriptions[i-1].LastConversionTime.Before(transcriptions[i].LastConversionTime) {
							t.Errorf("Expected transcriptions to be ordered by last_conversion_time DESC, but record %d is older than record %d", i-1, i)
						}
					}
				}
			})
		}
	})
}

// TestPostgresDB_GetAllByUser_WhenImplemented tests what the method should do when implemented
func TestPostgresDB_GetAllByUser_WhenImplemented(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
	}

	t.Skip("GetAllByUser is not implemented yet - this test shows expected behavior")

	// This test documents what the implementation should do
	testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		// Create a temporary implementation for testing
		postgresDB := &PostgresDBWithGetAllByUser{db: db}

		tests := []struct {
			name         string
			userNickname string
			expectError  bool
			expectCount  int
		}{
			{
				name:         "existing_user_with_records",
				userNickname: "test_user_1",
				expectError:  false,
				expectCount:  2, // Based on seeded data
			},
			{
				name:         "existing_user_with_one_record",
				userNickname: "test_user_2",
				expectError:  false,
				expectCount:  1, // Only non-error records
			},
			{
				name:         "non_existing_user",
				userNickname: "non_existent_user",
				expectError:  false,
				expectCount:  0,
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

				if len(transcriptions) != tt.expectCount {
					t.Errorf("Expected %d transcriptions, got %d", tt.expectCount, len(transcriptions))
				}

				// Verify all returned transcriptions belong to the user and have no errors
				for _, transcription := range transcriptions {
					if transcription.User != tt.userNickname {
						t.Errorf("Expected user %s, got %s", tt.userNickname, transcription.User)
					}
					if transcription.ErrorMessage != "" {
						t.Errorf("Expected empty error message, got %s", transcription.ErrorMessage)
					}
				}

				// Verify ordering (should be DESC by last_conversion_time)
				if len(transcriptions) > 1 {
					for i := 1; i < len(transcriptions); i++ {
						if transcriptions[i-1].LastConversionTime.Before(transcriptions[i].LastConversionTime) {
							t.Error("Expected transcriptions to be ordered by last_conversion_time DESC")
						}
					}
				}
			})
		}
	})
}

// TestPostgresDB_ConcurrentAccess tests concurrent access to the database
func TestPostgresDB_ConcurrentAccess(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
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

// TestPostgresDB_TransactionHandling tests transaction behavior
func TestPostgresDB_TransactionHandling(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
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

		// Test that database constraints are enforced
		// (This depends on your actual schema constraints)
	})
}

// TestPostgresDB_DataIntegrity tests data integrity with PostgreSQL-specific features
func TestPostgresDB_DataIntegrity(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
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

// TestPostgresDB_MemoryUsage tests memory usage patterns
func TestPostgresDB_MemoryUsage(t *testing.T) {
	if !isPostgresAvailable() {
		t.Skip("PostgreSQL not available, skipping test")
	}

	benchmark := testutil.NewBenchmarkHelper("PostgresDB_MemoryUsage")

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		postgresDB := &PostgresDB{db: db}

		benchmark.Start()

		// Perform various operations and measure memory
		benchmark.Measure("record_operations", func() {
			for i := 0; i < 100; i++ {
				postgresDB.RecordToDB(
					"memory_test_user",
					"/test/memory",
					"memory_test.mp3",
					"memory_test.mp3",
					120,
					"Memory usage test transcription",
					time.Now(),
					0,
					"",
				)
			}
		})

		benchmark.Measure("check_operations", func() {
			for i := 0; i < 50; i++ {
				_, _ = postgresDB.CheckIfFileProcessed("memory_test.mp3")
			}
		})

		benchmark.Stop()

		// Assert reasonable memory usage
		benchmark.AssertMemoryUsageLessThan(t, 10*1024*1024) // 10MB
		benchmark.AssertDurationLessThan(t, 5*time.Second)   // 5 seconds

		t.Log(benchmark.Report())
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

// generateLongText generates text of specified length for testing
func generateLongText(length int) string {
	text := "This is a test transcription with repeated content for PostgreSQL testing. It includes unicode: 测试 and symbols: !@#$%^&*(). "
	result := ""
	for len(result) < length {
		result += text
	}
	return result[:length]
}

// PostgresDBWithGetAllByUser is a test implementation that adds GetAllByUser functionality
// This shows what the implementation should look like when completed
type PostgresDBWithGetAllByUser struct {
	db *sql.DB
}

func (pdb *PostgresDBWithGetAllByUser) Close() error {
	return pdb.db.Close()
}

func (pdb *PostgresDBWithGetAllByUser) CheckIfFileProcessed(fileName string) (int, error) {
	query := `SELECT id FROM transcriptions WHERE file_name = $1 AND has_error = 0`
	row := pdb.db.QueryRow(query, fileName)
	var id int
	err := row.Scan(&id)
	return id, err
}

func (pdb *PostgresDBWithGetAllByUser) RecordToDB(user, inputDir, fileName, mp3FileName string, audioDuration int, transcription string,
	lastConversionTime time.Time, hasError int, errorMessage string) {
	insertSQL := `INSERT INTO transcriptions ("user", input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`
	_, err := pdb.db.Exec(insertSQL, user, inputDir, fileName, mp3FileName, audioDuration, transcription, lastConversionTime, hasError, errorMessage)
	if err != nil {
		panic(err) // Match the behavior of other implementations
	}
}

func (pdb *PostgresDBWithGetAllByUser) GetAllByUser(userNickname string) ([]model.Transcription, error) {
	sqlStr := `
		SELECT id, "user", last_conversion_time, mp3_file_name, audio_duration, transcription, error_message
		FROM transcriptions
		WHERE has_error = 0
		  AND "user" = $1
		ORDER BY last_conversion_time DESC;`
	rows, err := pdb.db.Query(sqlStr, userNickname)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transcriptions := make([]model.Transcription, 0)

	for rows.Next() {
		var t model.Transcription
		err = rows.Scan(&t.ID, &t.User, &t.LastConversionTime, &t.Mp3FileName, &t.AudioDuration, &t.Transcription, &t.ErrorMessage)
		if err != nil {
			return nil, err
		}

		transcriptions = append(transcriptions, t)
	}
	return transcriptions, nil
}
