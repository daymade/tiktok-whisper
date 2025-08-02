package sqlite

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/testutil"

	_ "github.com/mattn/go-sqlite3"
)

// TestSQLiteDAO_Interface verifies SQLiteDB implements TranscriptionDAO interface
func TestSQLiteDAO_Interface(t *testing.T) {
	var _ repository.TranscriptionDAO = (*SQLiteDB)(nil)
}

// TestNewSQLiteDB tests the constructor function
func TestNewSQLiteDB(t *testing.T) {
	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		// Create a temporary database file for this test
		tmpFile := "/tmp/test_sqlite_" + time.Now().Format("20060102150405") + ".db"
		defer os.Remove(tmpFile)

		// Test successful creation
		sqliteDB := NewSQLiteDB(tmpFile)
		if sqliteDB == nil {
			t.Fatal("Expected NewSQLiteDB to return a non-nil instance")
		}

		// Test that the database connection is working
		err := sqliteDB.db.Ping()
		if err != nil {
			t.Fatalf("Expected database connection to be working, got error: %v", err)
		}

		// Clean up
		err = sqliteDB.Close()
		if err != nil {
			t.Errorf("Expected Close() to succeed, got error: %v", err)
		}
	})
}

// TestSQLiteDB_Close tests the Close method
func TestSQLiteDB_Close(t *testing.T) {
	tmpFile := "/tmp/test_close_" + time.Now().Format("20060102150405") + ".db"
	defer os.Remove(tmpFile)

	sqliteDB := NewSQLiteDB(tmpFile)

	// Test successful close
	err := sqliteDB.Close()
	if err != nil {
		t.Errorf("Expected Close() to return nil, got: %v", err)
	}

	// Test that operations fail after close
	_, err = sqliteDB.CheckIfFileProcessed("test.mp3")
	if err == nil {
		t.Error("Expected operations to fail after Close(), but they didn't")
	}
}

// TestSQLiteDB_CheckIfFileProcessed tests the CheckIfFileProcessed method
func TestSQLiteDB_CheckIfFileProcessed(t *testing.T) {
	testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

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

				id, err := sqliteDB.CheckIfFileProcessed(tt.fileName)

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

// TestSQLiteDB_RecordToDB tests the RecordToDB method
func TestSQLiteDB_RecordToDB(t *testing.T) {
	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

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
				name:          "long_transcription",
				user:          "test_user",
				inputDir:      "/test/input",
				fileName:      "long.mp3",
				mp3FileName:   "long.mp3",
				audioDuration: 3600,
				transcription: testutil.RandomTestTranscriptionText(),
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
				sqliteDB.RecordToDB(
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
				err := db.QueryRow("SELECT COUNT(*) FROM transcriptions WHERE file_name = ?", tc.fileName).Scan(&count)
				if err != nil {
					t.Fatalf("Failed to verify record insertion: %v", err)
				}

				if count != 1 {
					t.Errorf("Expected 1 record to be inserted, found %d", count)
				}

				// Verify the record content
				var storedUser, storedTranscription, storedErrorMessage string
				var storedAudioDuration, storedHasError int
				err = db.QueryRow(`
					SELECT user, audio_duration, transcription, has_error, error_message 
					FROM transcriptions 
					WHERE file_name = ?
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
					t.Errorf("Expected transcription %s, got %s", tc.transcription, storedTranscription)
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

// TestSQLiteDB_GetAllByUser tests the GetAllByUser method
func TestSQLiteDB_GetAllByUser(t *testing.T) {
	testutil.WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

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
				transcriptions, err := sqliteDB.GetAllByUser(tt.userNickname)

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

// TestSQLiteDB_ConcurrentAccess tests concurrent access to the database
func TestSQLiteDB_ConcurrentAccess(t *testing.T) {
	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

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

					sqliteDB.RecordToDB(
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

					// Also test concurrent reads
					_, err := sqliteDB.GetAllByUser(user)
					if err != nil {
						t.Errorf("Concurrent read failed: %v", err)
					}
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

// TestSQLiteDB_DataIntegrity tests data integrity constraints
func TestSQLiteDB_DataIntegrity(t *testing.T) {
	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

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
				transcription: "这是一个中文转录测试 🎵",
				duration:      180,
			},
			{
				name:          "special_characters",
				user:          "user@domain.com",
				fileName:      "file-with_special!chars.mp3",
				transcription: "Transcription with 'quotes', \"double quotes\", and symbols: !@#$%^&*()",
				duration:      300,
			},
			{
				name:          "very_long_text",
				user:          "long_text_user",
				fileName:      "long_text.mp3",
				transcription: generateLongText(5000), // 5KB of text
				duration:      7200,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				sqliteDB.RecordToDB(
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
				transcriptions, err := sqliteDB.GetAllByUser(tc.user)
				if err != nil {
					t.Fatalf("Failed to retrieve data: %v", err)
				}

				if len(transcriptions) == 0 {
					t.Fatal("No transcriptions found after insertion")
				}

				// Find our record
				var found *model.Transcription
				for _, tr := range transcriptions {
					if tr.Mp3FileName == tc.fileName {
						found = &tr
						break
					}
				}

				if found == nil {
					t.Fatal("Inserted record not found")
				}

				if found.User != tc.user {
					t.Errorf("User mismatch: expected %s, got %s", tc.user, found.User)
				}
				if found.Transcription != tc.transcription {
					t.Errorf("Transcription mismatch: lengths %d vs %d", len(tc.transcription), len(found.Transcription))
				}
				if int(found.AudioDuration) != tc.duration {
					t.Errorf("Duration mismatch: expected %d, got %f", tc.duration, found.AudioDuration)
				}
			})
		}
	})
}

// Benchmark tests for performance analysis

// BenchmarkSQLiteDB_RecordToDB benchmarks the RecordToDB method
func BenchmarkSQLiteDB_RecordToDB(b *testing.B) {
	testutil.WithTestDB(&testing.T{}, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sqliteDB.RecordToDB(
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

// BenchmarkSQLiteDB_GetAllByUser benchmarks the GetAllByUser method
func BenchmarkSQLiteDB_GetAllByUser(b *testing.B) {
	testutil.WithTestDB(&testing.T{}, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

		// Seed some data for benchmarking
		for i := 0; i < 100; i++ {
			sqliteDB.RecordToDB(
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

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sqliteDB.GetAllByUser("benchmark_user")
			if err != nil {
				b.Fatalf("Benchmark failed: %v", err)
			}
		}
	})
}

// BenchmarkSQLiteDB_CheckIfFileProcessed benchmarks the CheckIfFileProcessed method
func BenchmarkSQLiteDB_CheckIfFileProcessed(b *testing.B) {
	testutil.WithSeekedTestDB(&testing.T{}, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = sqliteDB.CheckIfFileProcessed("test_audio_1.mp3")
		}
	})
}

// Helper function to generate long text for testing
func generateLongText(length int) string {
	text := "This is a test transcription with repeated content. "
	result := ""
	for len(result) < length {
		result += text
	}
	return result[:length]
}

// TestSQLiteDB_MemoryUsage tests memory usage patterns
func TestSQLiteDB_MemoryUsage(t *testing.T) {
	benchmark := testutil.NewBenchmarkHelper("SQLiteDB_MemoryUsage")

	testutil.WithTestDB(t, func(t *testing.T, db *sql.DB) {
		sqliteDB := &SQLiteDB{db: db}

		benchmark.Start()

		// Perform various operations and measure memory
		benchmark.Measure("record_operations", func() {
			for i := 0; i < 100; i++ {
				sqliteDB.RecordToDB(
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

		benchmark.Measure("read_operations", func() {
			for i := 0; i < 50; i++ {
				_, _ = sqliteDB.GetAllByUser("memory_test_user")
			}
		})

		benchmark.Stop()

		// Assert reasonable memory usage
		benchmark.AssertMemoryUsageLessThan(t, 10*1024*1024) // 10MB
		benchmark.AssertDurationLessThan(t, 5*time.Second)   // 5 seconds

		t.Log(benchmark.Report())
	})
}
