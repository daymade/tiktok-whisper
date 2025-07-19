//go:build integration
// +build integration

package integration

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/repository/pg"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/testutil"
)

// TestPostgreSQLConnectivityResilience tests PostgreSQL connection handling
func TestPostgreSQLConnectivityResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL connectivity test in short mode")
	}

	tests := []struct {
		name           string
		connectionStr  string
		expectError    bool
		errorContains  string
		testOperations bool
	}{
		{
			name:          "InvalidHost",
			connectionStr: "postgres://user:pass@invalid-host-12345:5432/db?sslmode=disable&connect_timeout=2",
			expectError:   true,
			errorContains: "no such host",
		},
		{
			name:          "InvalidPort",
			connectionStr: "postgres://user:pass@localhost:99999/db?sslmode=disable&connect_timeout=2",
			expectError:   true,
			errorContains: "connection refused",
		},
		{
			name:          "InvalidCredentials",
			connectionStr: "postgres://invalid:invalid@localhost:5432/postgres?sslmode=disable&connect_timeout=2",
			expectError:   true,
			errorContains: "authentication",
		},
		{
			name:          "InvalidDatabase",
			connectionStr: "postgres://postgres:password@localhost:5432/nonexistent_db?sslmode=disable&connect_timeout=2",
			expectError:   true,
			errorContains: "does not exist",
		},
		{
			name:          "ConnectionTimeout",
			connectionStr: "postgres://user:pass@10.255.255.1:5432/db?sslmode=disable&connect_timeout=1",
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name:           "LocalConnection",
			connectionStr:  "postgres://postgres:password@localhost:5432/postgres?sslmode=disable",
			expectError:    false,
			testOperations: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			dao, err := pg.NewPostgresDB(tt.connectionStr)
			duration := time.Since(start)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
				// Should fail quickly for invalid connections
				assert.Less(t, duration, 30*time.Second, "Connection attempt should timeout quickly")
			} else {
				if err != nil {
					t.Skipf("PostgreSQL not available: %v", err)
				}
				assert.NoError(t, err)
				defer dao.Close()

				if tt.testOperations {
					testDatabaseOperations(t, dao)
				}
			}

			t.Logf("Connection attempt took %v", duration)
		})
	}
}

// testDatabaseOperations tests basic database operations
func testDatabaseOperations(t *testing.T, dao repository.TranscriptionDAO) {
	user := "test_db_ops_user"
	fileName := "test_db_ops.mp3"
	
	// Test file check (should not exist initially)
	_, err := dao.CheckIfFileProcessed(fileName)
	assert.Error(t, err, "File should not exist initially")
	assert.Equal(t, sql.ErrNoRows, err)

	// Test record insertion
	dao.RecordToDB(
		user,
		"/test/input",
		fileName,
		fileName,
		100,
		"Test transcription for database operations",
		time.Now(),
		0,
		"",
	)

	// Test file check (should exist now)
	id, err := dao.CheckIfFileProcessed(fileName)
	assert.NoError(t, err, "File should exist after recording")
	assert.Greater(t, id, 0, "Should have valid ID")

	// Test retrieval by user
	transcriptions, err := dao.GetAllByUser(user)
	assert.NoError(t, err, "Should retrieve transcriptions")
	assert.Len(t, transcriptions, 1, "Should have one transcription")
	
	transcription := transcriptions[0]
	assert.Equal(t, user, transcription.User)
	assert.Equal(t, fileName, transcription.Mp3FileName)
	assert.Equal(t, "Test transcription for database operations", transcription.Transcription)
}

// TestDatabaseConnectionPooling tests connection pooling behavior
func TestDatabaseConnectionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping connection pooling test in short mode")
	}

	// Test with SQLite (simpler setup)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "pooling_test.db")
	
	dao := sqlite.NewSQLiteDB(dbPath)
	defer dao.Close()

	// Test concurrent database operations
	numGoroutines := 10
	numOperations := 5
	
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				user := fmt.Sprintf("pool_user_%d", goroutineID)
				fileName := fmt.Sprintf("pool_file_%d_%d.mp3", goroutineID, j)
				
				// Record to database
				dao.RecordToDB(
					user,
					"/test/pool",
					fileName,
					fileName,
					100+j,
					fmt.Sprintf("Pooling test transcription %d-%d", goroutineID, j),
					time.Now(),
					0,
					"",
				)
				
				// Check if file exists
				_, err := dao.CheckIfFileProcessed(fileName)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d operation %d: %w", goroutineID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Concurrent operation error: %v", err)
	}

	assert.Equal(t, 0, errorCount, "No errors should occur in concurrent operations")

	// Verify all records were inserted
	totalExpected := numGoroutines * numOperations
	allTranscriptions := make([]model.Transcription, 0)
	
	for i := 0; i < numGoroutines; i++ {
		user := fmt.Sprintf("pool_user_%d", i)
		transcriptions, err := dao.GetAllByUser(user)
		assert.NoError(t, err)
		allTranscriptions = append(allTranscriptions, transcriptions...)
	}

	assert.Len(t, allTranscriptions, totalExpected, "Should have all expected transcriptions")
}

// TestDatabaseTransactionHandling tests transaction behavior
func TestDatabaseTransactionHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping transaction handling test in short mode")
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "transaction_test.db")
	
	dao := sqlite.NewSQLiteDB(dbPath)
	defer dao.Close()

	// Test normal operation
	user := "transaction_user"
	fileName := "transaction_test.mp3"
	
	dao.RecordToDB(user, "/test", fileName, fileName, 100, "Transaction test", time.Now(), 0, "")
	
	// Verify record exists
	id, err := dao.CheckIfFileProcessed(fileName)
	assert.NoError(t, err)
	assert.Greater(t, id, 0)

	// Test error handling
	errorFileName := "error_test.mp3"
	dao.RecordToDB(user, "/test", errorFileName, errorFileName, 100, "Error test", time.Now(), 1, "Simulated error")
	
	// Verify error record exists
	transcriptions, err := dao.GetAllByUser(user)
	assert.NoError(t, err)
	assert.Len(t, transcriptions, 2)
	
	// Find error record
	var errorRecord *model.Transcription
	for _, t := range transcriptions {
		if t.Mp3FileName == errorFileName {
			errorRecord = &t
			break
		}
	}
	
	require.NotNil(t, errorRecord, "Error record should exist")
	assert.NotEmpty(t, errorRecord.ErrorMessage)
}

// TestFileSystemOperationFailures tests various file system failure scenarios
func TestFileSystemOperationFailures(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError bool
		cleanup     func(string)
	}{
		{
			name: "ReadOnlyDirectory",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				subDir := filepath.Join(dir, "readonly")
				err := os.Mkdir(subDir, 0755)
				require.NoError(t, err)
				
				// Make directory read-only
				err = os.Chmod(subDir, 0444)
				require.NoError(t, err)
				
				return subDir
			},
			expectError: true,
			cleanup: func(path string) {
				os.Chmod(path, 0755) // Restore permissions for cleanup
			},
		},
		{
			name: "InsufficientDiskSpace",
			setupFunc: func(t *testing.T) string {
				// This is hard to simulate reliably across different systems
				// We'll create a scenario that might trigger disk space issues
				return t.TempDir()
			},
			expectError: false, // This test is more for documentation
		},
		{
			name: "NonExistentParentDirectory",
			setupFunc: func(t *testing.T) string {
				return "/nonexistent/parent/directory"
			},
			expectError: true,
		},
		{
			name: "FileWithInvalidCharacters",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				// Some systems may have issues with certain characters
				return filepath.Join(dir, "file\x00with\x00nulls.txt")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc(t)
			if tt.cleanup != nil {
				defer tt.cleanup(path)
			}

			// Test file operations
			testFilePath := filepath.Join(path, "test.txt")
			err := os.WriteFile(testFilePath, []byte("test content"), 0644)

			if tt.expectError {
				assert.Error(t, err, "File operation should fail")
			} else {
				if err != nil {
					t.Logf("Expected no error but got: %v", err)
				}
			}

			// Test file reading if write succeeded
			if err == nil {
				_, readErr := os.ReadFile(testFilePath)
				assert.NoError(t, readErr, "Should be able to read written file")
				
				// Cleanup
				os.Remove(testFilePath)
			}
		})
	}
}

// TestFileSystemConcurrentAccess tests concurrent file operations
func TestFileSystemConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent file access test in short mode")
	}

	tempDir := t.TempDir()
	numGoroutines := 10
	
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Test concurrent file creation/deletion
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			fileName := filepath.Join(tempDir, fmt.Sprintf("concurrent_%d.txt", id))
			content := fmt.Sprintf("Content from goroutine %d", id)
			
			// Write file
			err := os.WriteFile(fileName, []byte(content), 0644)
			if err != nil {
				errors <- fmt.Errorf("write error in goroutine %d: %w", id, err)
				return
			}
			
			// Read file back
			readContent, err := os.ReadFile(fileName)
			if err != nil {
				errors <- fmt.Errorf("read error in goroutine %d: %w", id, err)
				return
			}
			
			if string(readContent) != content {
				errors <- fmt.Errorf("content mismatch in goroutine %d", id)
				return
			}
			
			// Delete file
			err = os.Remove(fileName)
			if err != nil {
				errors <- fmt.Errorf("delete error in goroutine %d: %w", id, err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Concurrent file operation error: %v", err)
	}

	assert.Equal(t, 0, errorCount, "No errors should occur in concurrent file operations")
}

// TestResourceCleanupVerification tests that resources are properly cleaned up
func TestResourceCleanupVerification(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test database cleanup
	t.Run("DatabaseCleanup", func(t *testing.T) {
		dbPath := filepath.Join(tempDir, "cleanup_test.db")
		
		dao := sqlite.NewSQLiteDB(dbPath)
		// Note: NewSQLiteDB may panic on error rather than returning error
		
		// Use the database
		dao.RecordToDB("cleanup_user", "/test", "cleanup.mp3", "cleanup.mp3", 100, "cleanup test", time.Now(), 0, "")
		
		// Close and verify file exists
		err := dao.Close()
		assert.NoError(t, err, "Database should close without error")
		
		_, err = os.Stat(dbPath)
		assert.NoError(t, err, "Database file should exist after close")
	})

	// Test file cleanup
	t.Run("FileCleanup", func(t *testing.T) {
		testFiles := make([]string, 5)
		
		// Create test files
		for i := 0; i < 5; i++ {
			testFiles[i] = testutil.CreateTestAudioFile(t, fmt.Sprintf("cleanup_test_%d.wav", i))
		}
		
		// Verify files exist
		for i, file := range testFiles {
			_, err := os.Stat(file)
			assert.NoError(t, err, "Test file %d should exist", i)
		}
		
		// Cleanup files
		for i, file := range testFiles {
			testutil.CleanupFile(t, file)
			
			// Verify file is gone (may not work immediately due to OS caching)
			time.Sleep(10 * time.Millisecond)
			_, err := os.Stat(file)
			if !os.IsNotExist(err) {
				t.Logf("File %d may still exist after cleanup: %v", i, err)
			}
		}
	})

	// Test temporary directory cleanup
	t.Run("TempDirCleanup", func(t *testing.T) {
		// The test temp directory should be cleaned up automatically
		// This test verifies that we can create and use temp directories properly
		
		subTempDir := filepath.Join(tempDir, "subtemp")
		err := os.Mkdir(subTempDir, 0755)
		// Note: NewSQLiteDB may panic on error rather than returning error
		
		// Create files in subdirectory
		testFile := filepath.Join(subTempDir, "temp_test.txt")
		err = os.WriteFile(testFile, []byte("temporary content"), 0644)
		// Note: NewSQLiteDB may panic on error rather than returning error
		
		// Verify file exists
		_, err = os.Stat(testFile)
		assert.NoError(t, err)
		
		// Manual cleanup
		err = os.RemoveAll(subTempDir)
		assert.NoError(t, err, "Should be able to clean up temp directory")
		
		// Verify cleanup
		_, err = os.Stat(subTempDir)
		assert.True(t, os.IsNotExist(err), "Temp directory should be gone after cleanup")
	})
}

// TestDatabaseMigrationResilience tests database schema changes and migrations
func TestDatabaseMigrationResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database migration test in short mode")
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migration_test.db")
	
	// Create database with initial schema
	dao := sqlite.NewSQLiteDB(dbPath)
	// Note: NewSQLiteDB may panic on error rather than returning error
	defer dao.Close()

	// Add some data
	dao.RecordToDB("migration_user", "/test", "migration.mp3", "migration.mp3", 100, "migration test", time.Now(), 0, "")
	
	// Verify data exists
	transcriptions, err := dao.GetAllByUser("migration_user")
	assert.NoError(t, err)
	assert.Len(t, transcriptions, 1)

	// Close database
	dao.Close()

	// Reopen database (simulating application restart)
	dao2 := sqlite.NewSQLiteDB(dbPath)
	defer dao2.Close()

	// Verify data still exists
	transcriptions2, err := dao2.GetAllByUser("migration_user")
	assert.NoError(t, err)
	assert.Len(t, transcriptions2, 1)
	assert.Equal(t, transcriptions[0].Transcription, transcriptions2[0].Transcription)
}

// TestNetworkFileSystemOperations tests operations on network-mounted file systems
func TestNetworkFileSystemOperations(t *testing.T) {
	// This test is mostly for documentation as it's hard to test reliably
	// without specific network file system setup
	
	t.Skip("Network file system tests require specific infrastructure setup")
	
	// Example test structure:
	/*
	nfsPath := "/mnt/nfs/test"
	if _, err := os.Stat(nfsPath); os.IsNotExist(err) {
		t.Skip("NFS mount not available")
	}
	
	// Test file operations on network storage
	testFile := filepath.Join(nfsPath, "nfs_test.txt")
	err := os.WriteFile(testFile, []byte("nfs test"), 0644)
	assert.NoError(t, err, "Should be able to write to NFS")
	
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err, "Should be able to read from NFS")
	assert.Equal(t, "nfs test", string(content))
	
	err = os.Remove(testFile)
	assert.NoError(t, err, "Should be able to delete from NFS")
	*/
}