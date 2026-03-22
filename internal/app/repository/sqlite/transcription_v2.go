package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
)

// Ensure SQLiteDB implements TranscriptionDAOV2
var _ repository.TranscriptionDAOV2 = (*SQLiteDB)(nil)

func normalizeTimestamps(t *model.TranscriptionFull) {
	now := time.Now()

	if t.CreatedAt.IsZero() {
		if !t.LastConversionTime.IsZero() {
			t.CreatedAt = t.LastConversionTime
		} else {
			t.CreatedAt = now
		}
	}

	if t.UpdatedAt.IsZero() {
		if !t.LastConversionTime.IsZero() {
			t.UpdatedAt = t.LastConversionTime
		} else {
			t.UpdatedAt = now
		}
	}
}

func parseSQLiteTime(value sql.NullString) time.Time {
	if !value.Valid || value.String == "" {
		return time.Time{}
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	} {
		if parsed, err := time.Parse(layout, value.String); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func scanSQLiteTranscription(scanner interface{ Scan(dest ...interface{}) error }) (*model.TranscriptionFull, error) {
	var t model.TranscriptionFull
	var deletedAt sql.NullString
	var lastConversionTime sql.NullString
	var createdAt sql.NullString
	var updatedAt sql.NullString

	err := scanner.Scan(
		&t.ID, &t.User, &t.InputDir, &t.FileName, &t.Mp3FileName,
		&t.AudioDuration, &t.Transcription, &lastConversionTime,
		&t.HasError, &t.ErrorMessage, &t.FileHash, &t.FileSize,
		&t.ProviderType, &t.Language, &t.ModelName,
		&createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	t.LastConversionTime = parseSQLiteTime(lastConversionTime)
	t.CreatedAt = parseSQLiteTime(createdAt)
	t.UpdatedAt = parseSQLiteTime(updatedAt)
	if deleted := parseSQLiteTime(deletedAt); !deleted.IsZero() {
		t.DeletedAt = &deleted
	}

	return &t, nil
}

// RecordToDBV2 inserts a transcription with all new fields
func (sdb *SQLiteDB) RecordToDBV2(t *model.TranscriptionFull) error {
	if t == nil {
		return fmt.Errorf("transcription is nil")
	}

	if t.ID == 0 && t.FileHash != "" {
		existing, err := sdb.GetByFileHash(t.FileHash)
		if err != nil {
			return err
		}
		if existing != nil {
			t.ID = existing.ID
			if t.CreatedAt.IsZero() {
				t.CreatedAt = existing.CreatedAt
			}
		}
	}

	normalizeTimestamps(t)

	if t.ID > 0 {
		updateSQL := `
			UPDATE transcriptions SET
				user = ?, input_dir = ?, file_name = ?, mp3_file_name = ?, audio_duration = ?,
				transcription = ?, last_conversion_time = ?, has_error = ?, error_message = ?,
				file_hash = ?, file_size = ?, provider_type = ?, language = ?, model_name = ?,
				updated_at = ?
			WHERE id = ?`

		_, err := sdb.db.Exec(updateSQL,
			t.User, t.InputDir, t.FileName, t.Mp3FileName, t.AudioDuration,
			t.Transcription, t.LastConversionTime, t.HasError, t.ErrorMessage,
			t.FileHash, t.FileSize, t.ProviderType, t.Language, t.ModelName,
			t.UpdatedAt, t.ID)
		if err != nil {
			return fmt.Errorf("failed to update transcription %d: %w", t.ID, err)
		}
		return nil
	}

	insertSQL := `
		INSERT INTO transcriptions (
			user, input_dir, file_name, mp3_file_name, audio_duration, 
			transcription, last_conversion_time, has_error, error_message,
			file_hash, file_size, provider_type, language, model_name,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	result, err := sdb.db.Exec(insertSQL,
		t.User, t.InputDir, t.FileName, t.Mp3FileName, t.AudioDuration,
		t.Transcription, t.LastConversionTime, t.HasError, t.ErrorMessage,
		t.FileHash, t.FileSize, t.ProviderType, t.Language, t.ModelName,
		t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert transcription: %w", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to read inserted transcription id: %w", err)
	}

	t.ID = int(lastInsertID)
	return nil
}

// GetAllByUserV2 retrieves all transcriptions for a user with full field support
func (sdb *SQLiteDB) GetAllByUserV2(userNickname string) ([]model.TranscriptionFull, error) {
	query := `
		SELECT 
			id, user, input_dir, file_name, mp3_file_name, audio_duration,
			transcription, last_conversion_time, has_error, error_message,
			COALESCE(file_hash, '') as file_hash,
			COALESCE(file_size, 0) as file_size,
			COALESCE(provider_type, 'whisper_cpp') as provider_type,
			COALESCE(language, 'zh') as language,
			COALESCE(model_name, '') as model_name,
			COALESCE(created_at, last_conversion_time) as created_at,
			COALESCE(updated_at, last_conversion_time) as updated_at,
			deleted_at
		FROM transcriptions
		WHERE has_error = 0 
			AND user = ?
			AND (deleted_at IS NULL OR deleted_at = '')
		ORDER BY last_conversion_time DESC`
	
	rows, err := sdb.db.Query(query, userNickname)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	var transcriptions []model.TranscriptionFull
	for rows.Next() {
		parsed, err := scanSQLiteTranscription(rows)
		if err != nil {
			return nil, err
		}
		transcriptions = append(transcriptions, *parsed)
	}
	
	return transcriptions, nil
}

// GetByFileHash retrieves a transcription by its file hash
func (sdb *SQLiteDB) GetByFileHash(fileHash string) (*model.TranscriptionFull, error) {
	query := `
		SELECT 
			id, user, input_dir, file_name, mp3_file_name, audio_duration,
			transcription, last_conversion_time, has_error, error_message,
			COALESCE(file_hash, '') as file_hash,
			COALESCE(file_size, 0) as file_size,
			COALESCE(provider_type, 'whisper_cpp') as provider_type,
			COALESCE(language, 'zh') as language,
			COALESCE(model_name, '') as model_name,
			COALESCE(created_at, last_conversion_time) as created_at,
			COALESCE(updated_at, last_conversion_time) as updated_at,
			deleted_at
		FROM transcriptions
		WHERE file_hash = ? 
			AND (deleted_at IS NULL OR deleted_at = '')
		LIMIT 1`
	
	rows, err := sdb.db.Query(query, fileHash)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	return scanSQLiteTranscription(rows)
}

// GetByProvider retrieves transcriptions by provider type
func (sdb *SQLiteDB) GetByProvider(providerType string, limit int) ([]model.TranscriptionFull, error) {
	query := `
		SELECT 
			id, user, input_dir, file_name, mp3_file_name, audio_duration,
			transcription, last_conversion_time, has_error, error_message,
			COALESCE(file_hash, '') as file_hash,
			COALESCE(file_size, 0) as file_size,
			COALESCE(provider_type, 'whisper_cpp') as provider_type,
			COALESCE(language, 'zh') as language,
			COALESCE(model_name, '') as model_name,
			COALESCE(created_at, last_conversion_time) as created_at,
			COALESCE(updated_at, last_conversion_time) as updated_at,
			deleted_at
		FROM transcriptions
		WHERE provider_type = ?
			AND (deleted_at IS NULL OR deleted_at = '')
		ORDER BY last_conversion_time DESC
		LIMIT ?`
	
	rows, err := sdb.db.Query(query, providerType, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	var transcriptions []model.TranscriptionFull
	for rows.Next() {
		parsed, err := scanSQLiteTranscription(rows)
		if err != nil {
			log.Printf("scan error: %v", err)
			continue
		}

		transcriptions = append(transcriptions, *parsed)
	}
	
	return transcriptions, nil
}

// UpdateFileMetadata updates file hash and size for an existing record
func (sdb *SQLiteDB) UpdateFileMetadata(id int, fileHash string, fileSize int64) error {
	updateSQL := `
		UPDATE transcriptions 
		SET file_hash = ?, file_size = ?, updated_at = ?
		WHERE id = ?`
	
	_, err := sdb.db.Exec(updateSQL, fileHash, fileSize, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update file metadata: %w", err)
	}
	return nil
}

// SoftDelete marks a record as deleted without removing it
func (sdb *SQLiteDB) SoftDelete(id int) error {
	updateSQL := `
		UPDATE transcriptions 
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`
	
	result, err := sdb.db.Exec(updateSQL, time.Now(), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to soft delete: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("record not found or already deleted")
	}
	
	return nil
}

// GetActiveTranscriptions retrieves non-deleted transcriptions
func (sdb *SQLiteDB) GetActiveTranscriptions(limit int) ([]model.TranscriptionFull, error) {
	query := `
		SELECT 
			id, user, input_dir, file_name, mp3_file_name, audio_duration,
			transcription, last_conversion_time, has_error, error_message,
			COALESCE(file_hash, '') as file_hash,
			COALESCE(file_size, 0) as file_size,
			COALESCE(provider_type, 'whisper_cpp') as provider_type,
			COALESCE(language, 'zh') as language,
			COALESCE(model_name, '') as model_name,
			COALESCE(created_at, last_conversion_time) as created_at,
			COALESCE(updated_at, last_conversion_time) as updated_at,
			deleted_at
		FROM transcriptions
		WHERE (deleted_at IS NULL OR deleted_at = '')
			AND has_error = 0
		ORDER BY last_conversion_time DESC
		LIMIT ?`
	
	rows, err := sdb.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	var transcriptions []model.TranscriptionFull
	for rows.Next() {
		parsed, err := scanSQLiteTranscription(rows)
		if err != nil {
			log.Printf("scan error: %v", err)
			continue
		}

		transcriptions = append(transcriptions, *parsed)
	}
	
	return transcriptions, nil
}

// GetTranscriptionByID retrieves a single transcription by ID
func (sdb *SQLiteDB) GetTranscriptionByID(id int) (*model.TranscriptionFull, error) {
	query := `
		SELECT 
			id, user, input_dir, file_name, mp3_file_name, audio_duration,
			transcription, last_conversion_time, has_error, error_message,
			COALESCE(file_hash, '') as file_hash,
			COALESCE(file_size, 0) as file_size,
			COALESCE(provider_type, 'whisper_cpp') as provider_type,
			COALESCE(language, 'zh') as language,
			COALESCE(model_name, '') as model_name,
			COALESCE(created_at, last_conversion_time) as created_at,
			COALESCE(updated_at, last_conversion_time) as updated_at,
			deleted_at
		FROM transcriptions
		WHERE id = ?`
	
	rows, err := sdb.db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	return scanSQLiteTranscription(rows)
}
