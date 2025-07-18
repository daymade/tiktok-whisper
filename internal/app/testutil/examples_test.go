package testutil

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ExampleWithTestDB demonstrates how to use the database test helpers
func ExampleWithTestDB() {
	// This would typically be in a test function with *testing.T
	var t *testing.T // placeholder for example

	WithTestDB(t, func(t *testing.T, db *sql.DB) {
		// Use the database in your test
		_, err := db.Exec("INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"example_user", "/test", "example.mp3", "example.mp3", 120, "Example transcription", time.Now(), 0, "")

		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		// Database is automatically cleaned up after this function
	})
}

// ExampleWithSeekedTestDB demonstrates using pre-populated test data
func ExampleWithSeekedTestDB() {
	var t *testing.T // placeholder for example

	WithSeekedTestDB(t, func(t *testing.T, db *sql.DB) {
		// Database already contains test data from SeedTestData
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM transcriptions").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count records: %v", err)
		}

		// count should be > 0 because of seeded data
		if count == 0 {
			t.Error("Expected seeded data but found none")
		}
	})
}

// ExampleMockTranscriber demonstrates using the mock transcriber
func ExampleMockTranscriber() {
	// Create a mock transcriber with custom responses
	mockTranscriber := NewMockTranscriber().
		WithDefaultResponse("Default transcription").
		WithResponse("/path/to/specific.mp3", "Specific transcription").
		WithError("/path/to/error.mp3", errors.New("transcription failed"))

	// Test successful transcription
	result, err := mockTranscriber.Transcript("/path/to/audio.mp3")
	if err != nil {
		panic(err)
	}
	_ = result // result == "Default transcription"

	// Test specific file response
	result, err = mockTranscriber.Transcript("/path/to/specific.mp3")
	if err != nil {
		panic(err)
	}
	_ = result // result == "Specific transcription"

	// Test error case
	_, err = mockTranscriber.Transcript("/path/to/error.mp3")
	if err == nil {
		panic("Expected error but got none")
	}
	// err.Error() == "transcription failed"
}

// ExampleMockTranscriptionDAO demonstrates using the mock DAO
func ExampleMockTranscriptionDAO() {
	// Create a mock DAO with test data
	mockDAO := NewMockTranscriptionDAO().
		WithTranscriptions(TestTranscriptions).
		WithProcessedFile("processed.mp3", 1)

	// Test getting transcriptions by user
	transcriptions, err := mockDAO.GetAllByUser("test_user_1")
	if err != nil {
		panic(err)
	}
	_ = transcriptions // transcriptions contains all TestTranscriptions for "test_user_1"

	// Test checking processed file
	id, err := mockDAO.CheckIfFileProcessed("processed.mp3")
	if err != nil {
		panic(err)
	}
	_ = id // id == 1

	// Test recording new transcription
	mockDAO.RecordToDB("new_user", "/input", "new.mp3", "new.mp3", 180, "New transcription", time.Now(), 0, "")

	// Check the recorded call
	calls := mockDAO.GetRecordCalls()
	if len(calls) != 1 {
		panic("Expected 1 record call")
	}
	// calls[0].User == "new_user"
}

// ExampleBenchmarkHelper demonstrates performance testing
func ExampleBenchmarkHelper() {
	var t *testing.T // placeholder for example

	bh := NewBenchmarkHelper("Example Benchmark")
	bh.Start()

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	bh.Stop()

	// Assert performance requirements
	bh.AssertDurationLessThan(t, 100*time.Millisecond)
	bh.AssertMemoryUsageLessThan(t, 1024*1024) // 1MB

	// Generate performance report
	report := bh.Report()
	_ = report // Would typically log this: t.Log(report)
}

// ExampleBenchmarkRunner demonstrates running benchmarks
func ExampleBenchmarkRunner() {
	runner := NewBenchmarkRunner().
		WithIterations(1000).
		WithWarmup(100).
		WithParallel(false)

	results := runner.Run("Example Operation", func() {
		// Your operation to benchmark
		time.Sleep(1 * time.Microsecond)
	})

	// Analyze results
	avgDuration := results.AverageDuration()
	minDuration := results.MinDuration()
	maxDuration := results.MaxDuration()

	_ = avgDuration // Would typically assert these values
	_ = minDuration
	_ = maxDuration

	// Generate report
	report := results.Report()
	_ = report // Would typically log this
}

// TestDatabaseHelpers demonstrates actual test usage
func TestDatabaseHelpers(t *testing.T) {
	t.Run("SetupTestDB", func(t *testing.T) {
		db := SetupTestDB(t)
		defer TeardownTestDB(t, db)

		// Test that we can query the database
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM transcriptions").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count) // Should be empty initially
	})

	t.Run("SeedTestData", func(t *testing.T) {
		db := SetupTestDB(t)
		defer TeardownTestDB(t, db)

		SeedTestData(t, db)

		count := GetTestDataCount(t, db)
		assert.Greater(t, count, 0) // Should have test data
	})

	t.Run("WithTestDB", func(t *testing.T) {
		WithTestDB(t, func(t *testing.T, db *sql.DB) {
			// Database is set up and will be cleaned up automatically
			assert.NotNil(t, db)

			// Test database operations
			_, err := db.Exec("INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
				"test_user", "/test", "test.mp3", "test.mp3", 120, "Test transcription", time.Now(), 0, "")
			require.NoError(t, err)

			count := GetTestDataCount(t, db)
			assert.Equal(t, 1, count)
		})
	})
}

// TestMockFactories demonstrates mock usage
func TestMockFactories(t *testing.T) {
	t.Run("MockTranscriber", func(t *testing.T) {
		mock := NewMockTranscriber().
			WithDefaultResponse("Default response").
			WithResponse("/specific/path.mp3", "Specific response").
			WithError("/error/path.mp3", errors.New("mock error"))

		// Test default response
		result, err := mock.Transcript("/some/path.mp3")
		require.NoError(t, err)
		assert.Equal(t, "Default response", result)

		// Test specific response
		result, err = mock.Transcript("/specific/path.mp3")
		require.NoError(t, err)
		assert.Equal(t, "Specific response", result)

		// Test error case
		_, err = mock.Transcript("/error/path.mp3")
		require.Error(t, err)
		assert.Equal(t, "mock error", err.Error())

		// Test call tracking
		assert.Equal(t, 3, mock.GetCallCount())
		assert.Equal(t, "/error/path.mp3", mock.GetLastFilePath())
	})

	t.Run("MockTranscriptionDAO", func(t *testing.T) {
		mock := NewMockTranscriptionDAO().
			WithTranscriptions(TestTranscriptions[:2]). // Use first 2 test transcriptions
			WithProcessedFile("processed.mp3", 100)

		// Test GetAllByUser
		transcriptions, err := mock.GetAllByUser("test_user_1")
		require.NoError(t, err)
		assert.Greater(t, len(transcriptions), 0)

		// Test CheckIfFileProcessed
		id, err := mock.CheckIfFileProcessed("processed.mp3")
		require.NoError(t, err)
		assert.Equal(t, 100, id)

		// Test RecordToDB
		mock.RecordToDB("new_user", "/input", "new.mp3", "new.mp3", 180, "New transcription", time.Now(), 0, "")

		calls := mock.GetRecordCalls()
		assert.Equal(t, 1, len(calls))
		assert.Equal(t, "new_user", calls[0].User)
		assert.Equal(t, "new.mp3", calls[0].FileName)
	})

	t.Run("MockLogger", func(t *testing.T) {
		mock := NewMockLogger()

		mock.Info("Test info message")
		mock.Error("Test error message")
		mock.Debug("Test debug message")

		logs := mock.GetLogs()
		assert.Equal(t, 3, len(logs))

		errorLogs := mock.GetLogsByLevel(LogLevelError)
		assert.Equal(t, 1, len(errorLogs))
		assert.Equal(t, "Test error message", errorLogs[0].Message)

		assert.True(t, mock.ContainsMessage("Test info"))
		assert.False(t, mock.ContainsMessage("Nonexistent message"))
	})
}

// TestBenchmarkUtilities demonstrates benchmark usage
func TestBenchmarkUtilities(t *testing.T) {
	t.Run("BenchmarkHelper", func(t *testing.T) {
		bh := NewBenchmarkHelper("Test Benchmark")
		bh.Start()

		// Simulate some work
		time.Sleep(1 * time.Millisecond)

		bh.Stop()

		duration := bh.Duration()
		assert.Greater(t, duration, time.Duration(0))

		memStats := bh.MemoryUsage()
		assert.NotNil(t, memStats)

		report := bh.Report()
		assert.Contains(t, report, "Test Benchmark")
		assert.Contains(t, report, "Total Duration")
	})

	t.Run("BenchmarkRunner", func(t *testing.T) {
		runner := NewBenchmarkRunner().
			WithIterations(10).
			WithWarmup(2)

		results := runner.Run("Test Operation", func() {
			// Simple operation
			_ = time.Now()
		})

		assert.Equal(t, "Test Operation", results.Name)
		assert.Equal(t, 10, results.Iterations)
		assert.Equal(t, 2, results.Warmup)
		assert.Equal(t, 10, len(results.Durations))

		avgDuration := results.AverageDuration()
		assert.Greater(t, avgDuration, time.Duration(0))

		report := results.Report()
		assert.Contains(t, report, "Test Operation")
		assert.Contains(t, report, "Iterations: 10")
	})
}

// TestFixtures demonstrates fixture usage
func TestFixtures(t *testing.T) {
	t.Run("TestTranscriptions", func(t *testing.T) {
		assert.Greater(t, len(TestTranscriptions), 0)

		// Test first transcription
		first := TestTranscriptions[0]
		assert.NotEmpty(t, first.User)
		assert.NotEmpty(t, first.Mp3FileName)
		assert.Greater(t, first.AudioDuration, float64(0))
	})

	t.Run("GetTestTranscriptionsByUser", func(t *testing.T) {
		userTranscriptions := GetTestTranscriptionsByUser("test_user_1")
		assert.Greater(t, len(userTranscriptions), 0)

		// All should be for the same user
		for _, transcription := range userTranscriptions {
			assert.Equal(t, "test_user_1", transcription.User)
		}
	})

	t.Run("GenerateTestTranscription", func(t *testing.T) {
		custom := GenerateTestTranscription(999, "custom_user", "custom.mp3", 300.0, "Custom transcription")

		assert.Equal(t, 999, custom.ID)
		assert.Equal(t, "custom_user", custom.User)
		assert.Equal(t, "custom.mp3", custom.Mp3FileName)
		assert.Equal(t, 300.0, custom.AudioDuration)
		assert.Equal(t, "Custom transcription", custom.Transcription)
	})

	t.Run("RandomHelpers", func(t *testing.T) {
		user := RandomTestUser()
		assert.NotEmpty(t, user)

		audioFile := RandomTestAudioFile()
		assert.NotEmpty(t, audioFile)

		transcriptionText := RandomTestTranscriptionText()
		assert.NotEmpty(t, transcriptionText)

		errorMessage := RandomTestErrorMessage()
		assert.NotEmpty(t, errorMessage)
	})
}
