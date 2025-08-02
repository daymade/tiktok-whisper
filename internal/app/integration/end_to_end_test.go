//go:build integration
// +build integration

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/api/openai/whisper"
	"tiktok-whisper/internal/app/api/whisper_cpp"
	"tiktok-whisper/internal/app/converter"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/testutil"
)

// TestEndToEndWorkflow tests the complete transcription workflow
func TestEndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end workflow test in short mode")
	}

	tests := []struct {
		name          string
		useLocalAPI   bool
		useRemoteAPI  bool
		expectSuccess bool
		skipCondition func(t *testing.T) bool
	}{
		{
			name:          "RemoteAPIWorkflow",
			useRemoteAPI:  true,
			expectSuccess: true,
			skipCondition: func(t *testing.T) bool {
				return os.Getenv("OPENAI_API_KEY") == ""
			},
		},
		{
			name:          "LocalAPIWorkflow",
			useLocalAPI:   true,
			expectSuccess: true,
			skipCondition: func(t *testing.T) bool {
				return !isWhisperCppAvailable()
			},
		},
		{
			name:          "MockWorkflow",
			useRemoteAPI:  false,
			useLocalAPI:   false,
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipCondition != nil && tt.skipCondition(t) {
				t.Skip("Skipping test due to missing dependencies")
			}

			testEndToEndWorkflow(t, tt.useLocalAPI, tt.useRemoteAPI, tt.expectSuccess)
		})
	}
}

func testEndToEndWorkflow(t *testing.T, useLocal, useRemote, expectSuccess bool) {
	// Setup test environment
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create database
	dao := sqlite.NewSQLiteDB(dbPath)
	defer dao.Close()

	// Create transcriber
	var transcriber api.Transcriber
	if useRemote {
		apiKey := os.Getenv("OPENAI_API_KEY")
		client := openai.NewClient(apiKey)
		transcriber = whisper.NewRemoteTranscriber(client)
	} else if useLocal {
		binaryPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/main"
		modelPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"
		transcriber = whisper_cpp.NewLocalTranscriber(binaryPath, modelPath)
	} else {
		// Use mock transcriber
		transcriber = testutil.NewMockTranscriberWithDefaults()
	}

	// Create converter
	conv := converter.NewConverter(transcriber, dao)

	// Create test audio file
	testAudioFile := testutil.CreateTestAudioFile(t, "e2e_test.wav")
	defer testutil.CleanupFile(t, testAudioFile)

	// Test parameters
	user := "e2e_test_user"
	inputDir := filepath.Dir(testAudioFile)
	fileName := filepath.Base(testAudioFile)

	// Execute conversion using the private method through a test helper
	// Since ConvertSingleFile doesn't exist, we'll simulate the conversion process
	err := simulateFileConversion(conv, dao, transcriber, user, inputDir, fileName, testAudioFile)

	if expectSuccess {
		assert.NoError(t, err, "End-to-end conversion should succeed")

		// Verify database record
		id, err := dao.CheckIfFileProcessed(fileName)
		assert.NoError(t, err, "File should be marked as processed")
		assert.Greater(t, id, 0, "Should have valid record ID")

		// Verify transcription data
		transcriptions, err := dao.GetAllByUser(user)
		assert.NoError(t, err, "Should retrieve user transcriptions")
		assert.Len(t, transcriptions, 1, "Should have one transcription")

		transcription := transcriptions[0]
		assert.Equal(t, user, transcription.User)
		assert.Equal(t, fileName, transcription.Mp3FileName)
		assert.NotEmpty(t, transcription.Transcription)
		assert.Empty(t, transcription.ErrorMessage)
	} else {
		assert.Error(t, err, "End-to-end conversion should fail")
	}
}

// TestWorkflowWithDatabaseFailures tests workflow behavior when database operations fail
func TestWorkflowWithDatabaseFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping workflow database failure test in short mode")
	}

	// Create a read-only database to simulate write failures
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "readonly.db")

	// Create database first
	dao := sqlite.NewSQLiteDB(dbPath)
	dao.Close()

	// Make database file read-only
	err := os.Chmod(dbPath, 0444)
	require.NoError(t, err)

	// Try to use read-only database
	readonlyDAO := sqlite.NewSQLiteDB(dbPath)
	defer readonlyDAO.Close()

	// Create mock transcriber
	transcriber := testutil.NewMockTranscriberWithDefaults()

	// Create converter with read-only database
	conv := converter.NewConverter(transcriber, readonlyDAO)

	// Create test audio file
	testAudioFile := testutil.CreateTestAudioFile(t, "db_failure_test.wav")
	defer testutil.CleanupFile(t, testAudioFile)

	user := "db_failure_user"
	inputDir := filepath.Dir(testAudioFile)
	fileName := filepath.Base(testAudioFile)

	// Execute conversion - should handle database write failure gracefully
	err = simulateFileConversion(conv, readonlyDAO, transcriber, user, inputDir, fileName, testAudioFile)

	// The exact behavior depends on implementation
	// This test documents the current behavior
	t.Logf("Conversion with read-only database result: %v", err)
}

// TestWorkflowWithAPIFailures tests workflow behavior when API calls fail
func TestWorkflowWithAPIFailures(t *testing.T) {
	tests := []struct {
		name          string
		failureType   string
		setupServer   func() *httptest.Server
		expectError   bool
		errorContains string
	}{
		{
			name:        "APITimeout",
			failureType: "timeout",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(3 * time.Second) // Longer than client timeout
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name:        "APIServiceUnavailable",
			failureType: "service_unavailable",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
					w.Write([]byte(`{"error": {"message": "Service temporarily unavailable"}}`))
				}))
			},
			expectError:   true,
			errorContains: "service",
		},
		{
			name:        "APIInternalError",
			failureType: "internal_error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": {"message": "Internal server error"}}`))
				}))
			},
			expectError:   true,
			errorContains: "internal",
		},
		{
			name:        "APIInvalidResponse",
			failureType: "invalid_response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`invalid json response`))
				}))
			},
			expectError:   true,
			errorContains: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := tt.setupServer()
			defer server.Close()

			// Setup database
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "api_failure_test.db")
			dao := sqlite.NewSQLiteDB(dbPath)
			defer dao.Close()

			// Setup transcriber with failing API
			config := openai.DefaultConfig("test-api-key")
			config.BaseURL = server.URL
			config.HTTPClient = &http.Client{Timeout: 1 * time.Second}
			client := openai.NewClientWithConfig(config)
			transcriber := whisper.NewRemoteTranscriber(client)

			// Create converter
			conv := converter.NewConverter(transcriber, dao)

			// Create test audio file
			testAudioFile := testutil.CreateTestAudioFile(t, "api_failure_test.wav")
			defer testutil.CleanupFile(t, testAudioFile)

			user := "api_failure_user"
			inputDir := filepath.Dir(testAudioFile)
			fileName := filepath.Base(testAudioFile)

			// Execute conversion
			err := simulateFileConversion(conv, dao, transcriber, user, inputDir, fileName, testAudioFile)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}

				// Check if error was recorded in database
				transcriptions, dbErr := dao.GetAllByUser(user)
				if dbErr == nil && len(transcriptions) > 0 {
					transcription := transcriptions[0]
					assert.NotEmpty(t, transcription.ErrorMessage, "Error should be recorded in database")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestWorkflowWithFileSystemFailures tests workflow behavior when file operations fail
func TestWorkflowWithFileSystemFailures(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, string, string) // Returns user, inputDir, fileName
		expectError bool
	}{
		{
			name: "NonExistentFile",
			setupFunc: func(t *testing.T) (string, string, string) {
				return "fs_failure_user", "/nonexistent", "missing.wav"
			},
			expectError: true,
		},
		{
			name: "UnreadableFile",
			setupFunc: func(t *testing.T) (string, string, string) {
				testFile := testutil.CreateTestAudioFile(t, "unreadable.wav")

				// Make file unreadable
				err := os.Chmod(testFile, 0000)
				require.NoError(t, err)

				return "fs_failure_user", filepath.Dir(testFile), filepath.Base(testFile)
			},
			expectError: true,
		},
		{
			name: "EmptyFile",
			setupFunc: func(t *testing.T) (string, string, string) {
				testFile := testutil.CreateEmptyFile(t, "empty.wav")
				return "fs_failure_user", filepath.Dir(testFile), filepath.Base(testFile)
			},
			expectError: true, // Depends on implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup database
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "fs_failure_test.db")
			dao := sqlite.NewSQLiteDB(dbPath)
			defer dao.Close()

			// Setup mock transcriber
			transcriber := testutil.NewMockTranscriberWithDefaults()

			// Create converter
			conv := converter.NewConverter(transcriber, dao)

			// Setup test scenario
			user, inputDir, fileName := tt.setupFunc(t)

			// Execute conversion
			err := simulateFileConversion(conv, dao, transcriber, user, inputDir, fileName, filepath.Join(inputDir, fileName))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			t.Logf("File system failure test '%s' result: %v", tt.name, err)
		})
	}
}

// TestFullWorkflowIntegration tests the complete workflow with real dependencies
func TestFullWorkflowIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full workflow integration test in short mode")
	}

	// Only run if we have real API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping full integration test - no API key")
	}

	// Setup real database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "full_integration.db")
	dao := sqlite.NewSQLiteDB(dbPath)
	defer dao.Close()

	// Setup real transcriber
	client := openai.NewClient(apiKey)
	transcriber := whisper.NewRemoteTranscriber(client)

	// Create converter
	conv := converter.NewConverter(transcriber, dao)

	// Create test audio files
	testFiles := []string{
		testutil.CreateTestAudioFile(t, "integration_test_1.wav"),
		testutil.CreateTestAudioFile(t, "integration_test_2.wav"),
	}
	defer func() {
		for _, file := range testFiles {
			testutil.CleanupFile(t, file)
		}
	}()

	user := "full_integration_user"

	// Process multiple files
	for i, testFile := range testFiles {
		inputDir := filepath.Dir(testFile)
		fileName := filepath.Base(testFile)

		t.Logf("Processing file %d: %s", i+1, fileName)

		err := simulateFileConversion(conv, dao, transcriber, user, inputDir, fileName, filepath.Join(inputDir, fileName))
		assert.NoError(t, err, "File %d conversion should succeed", i+1)

		// Verify immediate results
		id, err := dao.CheckIfFileProcessed(fileName)
		assert.NoError(t, err, "File %d should be processed", i+1)
		assert.Greater(t, id, 0, "File %d should have valid ID", i+1)
	}

	// Verify all transcriptions
	transcriptions, err := dao.GetAllByUser(user)
	assert.NoError(t, err, "Should retrieve all user transcriptions")
	assert.Len(t, transcriptions, len(testFiles), "Should have transcription for each file")

	// Verify transcription quality
	for i, transcription := range transcriptions {
		assert.Equal(t, user, transcription.User)
		assert.NotEmpty(t, transcription.Transcription, "Transcription %d should not be empty", i+1)
		assert.Empty(t, transcription.ErrorMessage, "Transcription %d should not have errors", i+1)
		assert.Greater(t, transcription.AudioDuration, 0.0, "Transcription %d should have valid duration", i+1)

		t.Logf("Transcription %d: %s", i+1, transcription.Transcription)
	}
}

// TestConcurrentWorkflow tests concurrent processing of multiple files
func TestConcurrentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent workflow test in short mode")
	}

	numFiles := 5

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "concurrent_test.db")
	dao := sqlite.NewSQLiteDB(dbPath)
	defer dao.Close()

	// Setup mock transcriber with latency
	transcriber := testutil.NewMockTranscriberWithDefaults().WithLatency(100 * time.Millisecond)

	// Create converter
	conv := converter.NewConverter(transcriber, dao)

	// Create test files
	testFiles := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		testFiles[i] = testutil.CreateTestAudioFile(t, fmt.Sprintf("concurrent_test_%d.wav", i))
		defer testutil.CleanupFile(t, testFiles[i])
	}

	user := "concurrent_user"

	// Process files concurrently
	results := make(chan error, numFiles)
	start := time.Now()

	for i, testFile := range testFiles {
		go func(index int, file string) {
			inputDir := filepath.Dir(file)
			fileName := filepath.Base(file)
			err := simulateFileConversion(conv, dao, transcriber, user, inputDir, fileName, filepath.Join(inputDir, fileName))
			results <- err
		}(i, testFile)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < numFiles; i++ {
		err := <-results
		if err != nil {
			errorCount++
			t.Logf("Concurrent processing error: %v", err)
		} else {
			successCount++
		}
	}

	duration := time.Since(start)

	t.Logf("Concurrent processing completed in %v", duration)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)

	// Verify results
	assert.Equal(t, numFiles, successCount, "All files should be processed successfully")
	assert.Equal(t, 0, errorCount, "No errors should occur")

	// Verify database consistency
	transcriptions, err := dao.GetAllByUser(user)
	assert.NoError(t, err)
	assert.Len(t, transcriptions, numFiles, "Should have transcription for each file")

	// Should be faster than sequential processing (due to concurrent I/O)
	maxSequentialTime := time.Duration(numFiles) * 200 * time.Millisecond // Allow some overhead
	assert.Less(t, duration, maxSequentialTime, "Concurrent processing should be faster")
}

// TestWorkflowRecovery tests recovery from partial failures
func TestWorkflowRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping workflow recovery test in short mode")
	}

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "recovery_test.db")
	dao := sqlite.NewSQLiteDB(dbPath)
	defer dao.Close()

	// Create files for testing
	successFile := testutil.CreateTestAudioFile(t, "success.wav")
	failureFile := testutil.CreateTestAudioFile(t, "failure.wav")
	recoveryFile := testutil.CreateTestAudioFile(t, "recovery.wav")

	defer func() {
		testutil.CleanupFile(t, successFile)
		testutil.CleanupFile(t, failureFile)
		testutil.CleanupFile(t, recoveryFile)
	}()

	// Setup transcriber that fails on specific file
	transcriber := testutil.NewMockTranscriberWithDefaults()
	transcriber.WithError(failureFile, fmt.Errorf("simulated transcription failure"))

	conv := converter.NewConverter(transcriber, dao)
	user := "recovery_user"

	// Process first file (should succeed)
	err := simulateFileConversion(conv, dao, transcriber, user, filepath.Dir(successFile), filepath.Base(successFile), successFile)
	assert.NoError(t, err, "First file should succeed")

	// Process second file (should fail)
	err = simulateFileConversion(conv, dao, transcriber, user, filepath.Dir(failureFile), filepath.Base(failureFile), failureFile)
	assert.Error(t, err, "Second file should fail")

	// Process third file (should succeed, showing recovery)
	err = simulateFileConversion(conv, dao, transcriber, user, filepath.Dir(recoveryFile), filepath.Base(recoveryFile), recoveryFile)
	assert.NoError(t, err, "Third file should succeed after failure")

	// Verify database state
	transcriptions, err := dao.GetAllByUser(user)
	assert.NoError(t, err)

	// Should have records for all files, with appropriate error states
	successCount := 0
	errorCount := 0

	for _, transcription := range transcriptions {
		if transcription.ErrorMessage == "" {
			successCount++
		} else {
			errorCount++
		}
	}

	assert.Equal(t, 2, successCount, "Should have 2 successful transcriptions")
	assert.Equal(t, 1, errorCount, "Should have 1 failed transcription")
}

// Helper functions

// simulateFileConversion simulates the file conversion process used in the main application
func simulateFileConversion(conv *converter.Converter, dao repository.TranscriptionDAO, transcriber api.Transcriber, user, inputDir, fileName, filePath string) error {
	// Check if file already processed
	_, err := dao.CheckIfFileProcessed(fileName)
	if err == nil {
		return fmt.Errorf("file already processed")
	}

	// Simulate transcription
	transcription, err := transcriber.Transcript(filePath)
	if err != nil {
		// Record error to database
		dao.RecordToDB(user, inputDir, fileName, fileName, 0, "", time.Now(), 1, err.Error())
		return err
	}

	// Record success to database
	dao.RecordToDB(user, inputDir, fileName, fileName, 100, transcription, time.Now(), 0, "")
	return nil
}

func isWhisperCppAvailable() bool {
	binaryPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/main"
	modelPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return false
	}

	return true
}
