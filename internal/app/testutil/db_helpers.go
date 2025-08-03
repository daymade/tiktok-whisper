package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DatabaseType represents the type of database to use in tests
type DatabaseType string

const (
	SQLiteDB   DatabaseType = "sqlite"
	PostgresDB DatabaseType = "postgres"
)

// DBConfig holds database configuration for testing
type DBConfig struct {
	Type     DatabaseType
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultSQLiteConfig returns default SQLite configuration for testing
func DefaultSQLiteConfig() DBConfig {
	return DBConfig{
		Type: SQLiteDB,
	}
}

// DefaultPostgresConfig returns default PostgreSQL configuration for testing
func DefaultPostgresConfig() DBConfig {
	return DBConfig{
		Type:     PostgresDB,
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "tiktok_whisper_test",
		SSLMode:  "disable",
	}
}

// SetupTestDB creates a test database based on environment or defaults to SQLite
// It automatically determines the database type and creates appropriate test database
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Check if POSTGRES_TEST_URL is set in environment
	if pgURL := os.Getenv("POSTGRES_TEST_URL"); pgURL != "" {
		db, err := sql.Open("postgres", pgURL)
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL test database: %v", err)
		}

		if err := db.Ping(); err != nil {
			t.Fatalf("Failed to ping PostgreSQL test database: %v", err)
		}

		// Create tables
		if err := createTestTables(db); err != nil {
			t.Fatalf("Failed to create test tables: %v", err)
		}

		return db
	}

	// Default to SQLite
	return SetupTestSQLite(t)
}

// SetupTestSQLite creates a SQLite test database with a unique name
func SetupTestSQLite(t *testing.T) *sql.DB {
	t.Helper()

	// Create a unique test database file
	testDBPath := filepath.Join(os.TempDir(), fmt.Sprintf("test_db_%d.sqlite", time.Now().UnixNano()))

	db, err := sql.Open("sqlite3", testDBPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite test database: %v", err)
	}

	// Create tables
	if err := createTestTables(db); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	// Clean up database file when test completes
	t.Cleanup(func() {
		db.Close()
		os.Remove(testDBPath)
	})

	return db
}

// SetupTestPostgres creates a PostgreSQL test database
func SetupTestPostgres(t *testing.T) *sql.DB {
	t.Helper()

	config := DefaultPostgresConfig()

	// Override with environment variables if set
	if host := os.Getenv("POSTGRES_TEST_HOST"); host != "" {
		config.Host = host
	}
	if user := os.Getenv("POSTGRES_TEST_USER"); user != "" {
		config.User = user
	}
	if password := os.Getenv("POSTGRES_TEST_PASSWORD"); password != "" {
		config.Password = password
	}
	if dbname := os.Getenv("POSTGRES_TEST_DB"); dbname != "" {
		config.DBName = dbname
	}

	// Create unique test database name
	testDBName := fmt.Sprintf("%s_%d", config.DBName, time.Now().UnixNano())

	// Connect to postgres database to create test database
	adminConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.SSLMode)

	adminDB, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL admin database: %v", err)
	}
	defer adminDB.Close()

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to test database
	testConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, testDBName, config.SSLMode)

	testDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create tables
	if err := createTestTables(testDB); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	// Clean up database when test completes
	t.Cleanup(func() {
		testDB.Close()
		_, err := adminDB.Exec(fmt.Sprintf("DROP DATABASE %s", testDBName))
		if err != nil {
			t.Logf("Failed to drop test database %s: %v", testDBName, err)
		}
	})

	return testDB
}

// TeardownTestDB closes the database connection and performs cleanup
func TeardownTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	if db != nil {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close test database: %v", err)
		}
	}
}

// SeedTestData inserts test data into the test database
func SeedTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	// Clear existing data
	_, err := db.Exec("DELETE FROM transcriptions")
	if err != nil {
		t.Fatalf("Failed to clear test data: %v", err)
	}

	// Insert test transcriptions
	testData := []struct {
		user          string
		mp3FileName   string
		audioDuration int
		transcription string
		hasError      int
		errorMessage  string
	}{
		{
			user:          "test_user_1",
			mp3FileName:   "test_audio_1.mp3",
			audioDuration: 120,
			transcription: "This is a test transcription for the first audio file.",
			hasError:      0,
			errorMessage:  "",
		},
		{
			user:          "test_user_1",
			mp3FileName:   "test_audio_2.mp3",
			audioDuration: 180,
			transcription: "This is a test transcription for the second audio file.",
			hasError:      0,
			errorMessage:  "",
		},
		{
			user:          "test_user_2",
			mp3FileName:   "test_audio_3.mp3",
			audioDuration: 90,
			transcription: "This is a test transcription for the third audio file.",
			hasError:      0,
			errorMessage:  "",
		},
		{
			user:          "test_user_2",
			mp3FileName:   "test_audio_error.mp3",
			audioDuration: 0,
			transcription: "",
			hasError:      1,
			errorMessage:  "Test error message",
		},
	}

	for _, data := range testData {
		_, err := db.Exec(`
			INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, data.user, "/test/input", data.mp3FileName, data.mp3FileName, data.audioDuration, data.transcription, time.Now(), data.hasError, data.errorMessage)

		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}
}

// CleanTestData removes all test data from the database
func CleanTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec("DELETE FROM transcriptions")
	if err != nil {
		t.Fatalf("Failed to clean test data: %v", err)
	}
}

// GetTestDataCount returns the number of records in the transcriptions table
func GetTestDataCount(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM transcriptions").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to get test data count: %v", err)
	}

	return count
}

// createTestTables creates the necessary tables for testing
func createTestTables(db *sql.DB) error {
	// Create transcriptions table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS transcriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user TEXT NOT NULL,
		input_dir TEXT NOT NULL,
		file_name TEXT NOT NULL,
		mp3_file_name TEXT NOT NULL,
		audio_duration INTEGER NOT NULL,
		transcription TEXT NOT NULL,
		last_conversion_time DATETIME NOT NULL,
		has_error INTEGER NOT NULL DEFAULT 0,
		error_message TEXT
	);`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create transcriptions table: %w", err)
	}

	// Create index for faster queries
	indexSQL := `CREATE INDEX IF NOT EXISTS idx_transcriptions_user ON transcriptions(user);`
	_, err = db.Exec(indexSQL)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// WithTestDB is a helper function that provides a test database to a test function
func WithTestDB(t *testing.T, testFunc func(t *testing.T, db *sql.DB)) {
	t.Helper()

	db := SetupTestDB(t)
	// Note: cleanup is already handled by t.Cleanup() in SetupTestDB/SetupTestSQLite

	testFunc(t, db)
}

// WithSeekedTestDB is a helper function that provides a test database with test data
func WithSeekedTestDB(t *testing.T, testFunc func(t *testing.T, db *sql.DB)) {
	t.Helper()

	db := SetupTestDB(t)
	// Note: cleanup is already handled by t.Cleanup() in SetupTestDB/SetupTestSQLite

	SeedTestData(t, db)
	testFunc(t, db)
}
