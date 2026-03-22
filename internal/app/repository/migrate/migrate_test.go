package migrate

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/mattn/go-sqlite3"
)

func setupSQLiteMigrationSource(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE transcriptions (
		id INTEGER PRIMARY KEY,
		input_dir TEXT,
		file_name TEXT,
		mp3_file_name TEXT,
		audio_duration INTEGER,
		transcription TEXT,
		last_conversion_time TEXT,
		has_error INTEGER,
		error_message TEXT,
		user TEXT
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	insert := `
	INSERT INTO transcriptions (
		id, input_dir, file_name, mp3_file_name, audio_duration, transcription,
		last_conversion_time, has_error, error_message, user
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	rows := [][]interface{}{
		{1, "/in/1", "a.mp3", "a.mp3", 10, "a", "2026-01-01T00:00:00Z", 0, "", "u1"},
		{2, "/in/2", "b.mp3", "b.mp3", 11, "b", "2026-01-01T00:01:00Z", 0, "", "u2"},
	}
	for _, row := range rows {
		if _, err := db.Exec(insert, row...); err != nil {
			t.Fatalf("insert row: %v", err)
		}
	}

	return db
}

func TestMigrateBatchStopsOnFirstFailedInsert(t *testing.T) {
	sqliteDB := setupSQLiteMigrationSource(t)
	defer sqliteDB.Close()

	postgresDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer postgresDB.Close()

	mock.ExpectBegin()
	mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO transcriptions (id, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message, user_nickname) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO transcriptions (id, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message, user_nickname) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`)).
		WithArgs(1, "/in/1", "a.mp3", "a.mp3", 10, "a", "2026-01-01T00:00:00Z", 0, "", "u1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO transcriptions (id, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message, user_nickname) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`)).
		WithArgs(2, "/in/2", "b.mp3", "b.mp3", 11, "b", "2026-01-01T00:01:00Z", 0, "", "u2").
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	lastID, err := migrateBatch(sqliteDB, postgresDB, 0)
	if err == nil {
		t.Fatalf("expected migrateBatch to fail")
	}
	if lastID != 1 {
		t.Fatalf("expected lastID to remain on last successful row, got %d", lastID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}
