// +build integration

package test

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"tiktok-whisper/internal/app"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/utils"

	_ "github.com/mattn/go-sqlite3"
)

const (
	testUser = "integration_test_user"
	testDB   = "./data/transcription.db"
)

// TestDatabaseSchema verifies the database has all expected columns and indexes
func TestDatabaseSchema(t *testing.T) {
	db, err := sql.Open("sqlite3", testDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check columns
	expectedColumns := []string{
		"id", "user", "input_dir", "file_name", "mp3_file_name",
		"audio_duration", "transcription", "last_conversion_time",
		"has_error", "error_message", "file_hash", "file_size",
		"provider_type", "language", "model_name", "created_at",
		"updated_at", "deleted_at",
	}

	rows, err := db.Query("PRAGMA table_info(transcriptions)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	actualColumns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue sql.NullString
		
		err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk)
		if err != nil {
			t.Errorf("Failed to scan row: %v", err)
			continue
		}
		actualColumns[name] = true
	}

	for _, col := range expectedColumns {
		if !actualColumns[col] {
			t.Errorf("Missing expected column: %s", col)
		}
	}

	// Check indexes
	rows, err = db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='transcriptions'")
	if err != nil {
		t.Fatalf("Failed to query indexes: %v", err)
	}
	defer rows.Close()

	indexCount := 0
	for rows.Next() {
		indexCount++
	}

	if indexCount < 7 {
		t.Errorf("Expected at least 7 indexes, found %d", indexCount)
	}
}

// TestQueryPerformance verifies that queries use indexes and perform well
func TestQueryPerformance(t *testing.T) {
	db, err := sql.Open("sqlite3", testDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test query performance
	start := time.Now()
	_, err = db.Exec("SELECT COUNT(*) FROM transcriptions WHERE user = ? AND has_error = 0", "test_user")
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("Query too slow: %v (expected < 100ms)", duration)
	}

	// Verify index usage
	rows, err := db.Query("EXPLAIN QUERY PLAN SELECT * FROM transcriptions WHERE file_name = ? AND has_error = 0", "test.mp3")
	if err != nil {
		t.Fatalf("Failed to explain query: %v", err)
	}
	defer rows.Close()

	var foundIndex bool
	for rows.Next() {
		var id, parent, notused int
		var detail string
		err := rows.Scan(&id, &parent, &notused, &detail)
		if err != nil {
			continue
		}
		if contains(detail, "USING INDEX") {
			foundIndex = true
			break
		}
	}

	if !foundIndex {
		t.Error("Query not using index")
	}
}

// TestDAOV2Implementation tests the enhanced DAO functionality
func TestDAOV2Implementation(t *testing.T) {
	dao := sqlite.NewSQLiteDB(testDB)
	defer dao.Close()

	// Clean up any existing test data
	cleanupTestData(t, dao)

	// Test RecordToDBV2
	transcription := &model.TranscriptionFull{
		User:               testUser,
		InputDir:           "/test/input",
		FileName:           "test_integration.mp3",
		Mp3FileName:        "test_integration.mp3",
		AudioDuration:      120,
		Transcription:      "Test transcription content",
		LastConversionTime: time.Now(),
		HasError:           0,
		ErrorMessage:       "",
		FileHash:           "testhash123",
		FileSize:           1024,
		ProviderType:       "whisper_cpp",
		Language:           "zh",
		ModelName:          "test-model",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	err := dao.RecordToDBV2(transcription)
	if err != nil {
		t.Fatalf("Failed to record to DB: %v", err)
	}

	// Test GetByFileHash
	result, err := dao.GetByFileHash("testhash123")
	if err != nil {
		t.Errorf("Failed to get by file hash: %v", err)
	}
	if result == nil {
		t.Error("Expected to find record by file hash")
	} else if result.User != testUser {
		t.Errorf("Expected user %s, got %s", testUser, result.User)
	}

	// Test GetAllByUserV2
	results, err := dao.GetAllByUserV2(testUser)
	if err != nil {
		t.Errorf("Failed to get by user: %v", err)
	}
	if len(results) == 0 {
		t.Error("Expected to find records for test user")
	}

	// Clean up
	cleanupTestData(t, dao)
}

// TestFileHashCalculation tests the file hash utility
func TestFileHashCalculation(t *testing.T) {
	// Create a test file
	testFile := filepath.Join(os.TempDir(), "test_hash.txt")
	content := []byte("test content for hash calculation")
	err := os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Calculate hash
	hash, err := utils.CalculateFileHash(testFile)
	if err != nil {
		t.Errorf("Failed to calculate hash: %v", err)
	}

	if len(hash) != 64 { // SHA256 produces 64 character hex string
		t.Errorf("Invalid hash length: expected 64, got %d", len(hash))
	}

	// Verify consistency
	hash2, err := utils.CalculateFileHash(testFile)
	if err != nil {
		t.Errorf("Failed to calculate hash again: %v", err)
	}

	if hash != hash2 {
		t.Error("Hash calculation not consistent")
	}
}

// TestProviderFramework tests basic provider framework functionality
func TestProviderFramework(t *testing.T) {
	// This is a basic test - full provider testing would require mocking
	
	// Check if provider config exists
	configPath := filepath.Join(os.Getenv("HOME"), ".tiktok-whisper", "providers.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Provider configuration not found, skipping provider tests")
	}

	// Test provider list command
	cmd := exec.Command("./v2t", "providers", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Provider list command failed: %v\nOutput: %s", err, output)
	}
}

// TestBuildAndBasicCommands tests that the application builds and basic commands work
func TestBuildAndBasicCommands(t *testing.T) {
	// Build the application
	cmd := exec.Command("go", "build", "-o", "v2t_test", "./cmd/v2t/main.go")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build application: %v", err)
	}
	defer os.Remove("v2t_test")

	// Test help command
	cmd = exec.Command("./v2t_test", "--help")
	if err := cmd.Run(); err != nil {
		t.Errorf("Help command failed: %v", err)
	}

	// Test version command
	cmd = exec.Command("./v2t_test", "version")
	if err := cmd.Run(); err != nil {
		t.Errorf("Version command failed: %v", err)
	}
}

// Helper functions

func cleanupTestData(t *testing.T, dao *sqlite.SQLiteDB) {
	db, err := sql.Open("sqlite3", testDB)
	if err != nil {
		t.Logf("Failed to open database for cleanup: %v", err)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM transcriptions WHERE user = ?", testUser)
	if err != nil {
		t.Logf("Failed to cleanup test data: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr)))
}