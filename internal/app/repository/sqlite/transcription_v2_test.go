package sqlite

import (
	"database/sql"
	"testing"
	"time"

	"tiktok-whisper/internal/app/model"

	_ "github.com/mattn/go-sqlite3"
)

func setupTranscriptionV2SQLite(t *testing.T) *SQLiteDB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE transcriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user TEXT,
		input_dir TEXT,
		file_name TEXT,
		mp3_file_name TEXT,
		audio_duration INTEGER,
		transcription TEXT,
		last_conversion_time DATETIME,
		has_error INTEGER,
		error_message TEXT,
		file_hash TEXT,
		file_size INTEGER,
		provider_type TEXT,
		language TEXT,
		model_name TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return &SQLiteDB{db: db}
}

func TestRecordToDBV2AssignsInsertedID(t *testing.T) {
	dao := setupTranscriptionV2SQLite(t)
	defer dao.Close()

	transcription := &model.TranscriptionFull{
		User:               "tester",
		InputDir:           "/tmp",
		FileName:           "sample.mp3",
		Mp3FileName:        "sample.mp3",
		AudioDuration:      42,
		Transcription:      "hello",
		LastConversionTime: time.Now(),
		HasError:           0,
		FileHash:           "hash-1",
		FileSize:           123,
		ProviderType:       "whisper_cpp",
		Language:           "zh",
		ModelName:          "model-a",
	}

	if err := dao.RecordToDBV2(transcription); err != nil {
		t.Fatalf("RecordToDBV2 insert failed: %v", err)
	}

	if transcription.ID == 0 {
		t.Fatalf("expected inserted transcription ID to be assigned")
	}
}

func TestRecordToDBV2UpdatesExistingRowByID(t *testing.T) {
	dao := setupTranscriptionV2SQLite(t)
	defer dao.Close()

	transcription := &model.TranscriptionFull{
		User:               "tester",
		InputDir:           "/tmp",
		FileName:           "sample.mp3",
		Mp3FileName:        "sample.mp3",
		AudioDuration:      42,
		Transcription:      "hello",
		LastConversionTime: time.Now(),
		HasError:           0,
		FileHash:           "hash-1",
		FileSize:           123,
		ProviderType:       "whisper_cpp",
		Language:           "zh",
		ModelName:          "model-a",
	}

	if err := dao.RecordToDBV2(transcription); err != nil {
		t.Fatalf("initial RecordToDBV2 failed: %v", err)
	}

	originalID := transcription.ID
	transcription.Transcription = "updated"
	transcription.AudioDuration = 84
	transcription.LastConversionTime = time.Now().Add(time.Minute)

	if err := dao.RecordToDBV2(transcription); err != nil {
		t.Fatalf("update RecordToDBV2 failed: %v", err)
	}

	var count int
	if err := dao.db.QueryRow("SELECT COUNT(*) FROM transcriptions WHERE file_hash = ?", "hash-1").Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one row after update, got %d", count)
	}

	var storedID, duration int
	var text string
	if err := dao.db.QueryRow("SELECT id, audio_duration, transcription FROM transcriptions WHERE file_hash = ?", "hash-1").Scan(&storedID, &duration, &text); err != nil {
		t.Fatalf("read updated row: %v", err)
	}
	if storedID != originalID {
		t.Fatalf("expected updated row to keep ID %d, got %d", originalID, storedID)
	}
	if duration != 84 || text != "updated" {
		t.Fatalf("expected updated row values to persist, got duration=%d text=%q", duration, text)
	}
}
