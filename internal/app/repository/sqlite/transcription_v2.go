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

// RecordToDBV2 inserts a transcription with all new fields
func (sdb *SQLiteDB) RecordToDBV2(t *model.TranscriptionFull) error {
	insertSQL := `
		INSERT INTO transcriptions (
			user, input_dir, file_name, mp3_file_name, audio_duration, 
			transcription, last_conversion_time, has_error, error_message,
			file_hash, file_size, provider_type, language, model_name,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := sdb.db.Exec(insertSQL,
		t.User, t.InputDir, t.FileName, t.Mp3FileName, t.AudioDuration,
		t.Transcription, t.LastConversionTime, t.HasError, t.ErrorMessage,
		t.FileHash, t.FileSize, t.ProviderType, t.Language, t.ModelName,
		time.Now(), time.Now())
	
	if err != nil {
		return fmt.Errorf("failed to insert transcription: %w", err)
	}
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
		var t model.TranscriptionFull
		var deletedAt sql.NullTime
		
		err = rows.Scan(
			&t.ID, &t.User, &t.InputDir, &t.FileName, &t.Mp3FileName,
			&t.AudioDuration, &t.Transcription, &t.LastConversionTime,
			&t.HasError, &t.ErrorMessage, &t.FileHash, &t.FileSize,
			&t.ProviderType, &t.Language, &t.ModelName,
			&t.CreatedAt, &t.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		
		transcriptions = append(transcriptions, t)
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
	
	var t model.TranscriptionFull
	var deletedAt sql.NullTime
	
	err := sdb.db.QueryRow(query, fileHash).Scan(
		&t.ID, &t.User, &t.InputDir, &t.FileName, &t.Mp3FileName,
		&t.AudioDuration, &t.Transcription, &t.LastConversionTime,
		&t.HasError, &t.ErrorMessage, &t.FileHash, &t.FileSize,
		&t.ProviderType, &t.Language, &t.ModelName,
		&t.CreatedAt, &t.UpdatedAt, &deletedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	
	if deletedAt.Valid {
		t.DeletedAt = &deletedAt.Time
	}
	
	return &t, nil
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
		var t model.TranscriptionFull
		var deletedAt sql.NullTime
		
		err = rows.Scan(
			&t.ID, &t.User, &t.InputDir, &t.FileName, &t.Mp3FileName,
			&t.AudioDuration, &t.Transcription, &t.LastConversionTime,
			&t.HasError, &t.ErrorMessage, &t.FileHash, &t.FileSize,
			&t.ProviderType, &t.Language, &t.ModelName,
			&t.CreatedAt, &t.UpdatedAt, &deletedAt,
		)
		if err != nil {
			log.Printf("scan error: %v", err)
			continue
		}
		
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		
		transcriptions = append(transcriptions, t)
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
		var t model.TranscriptionFull
		var deletedAt sql.NullTime
		
		err = rows.Scan(
			&t.ID, &t.User, &t.InputDir, &t.FileName, &t.Mp3FileName,
			&t.AudioDuration, &t.Transcription, &t.LastConversionTime,
			&t.HasError, &t.ErrorMessage, &t.FileHash, &t.FileSize,
			&t.ProviderType, &t.Language, &t.ModelName,
			&t.CreatedAt, &t.UpdatedAt, &deletedAt,
		)
		if err != nil {
			log.Printf("scan error: %v", err)
			continue
		}
		
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		
		transcriptions = append(transcriptions, t)
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
	
	var t model.TranscriptionFull
	var deletedAt sql.NullTime
	
	err := sdb.db.QueryRow(query, id).Scan(
		&t.ID, &t.User, &t.InputDir, &t.FileName, &t.Mp3FileName,
		&t.AudioDuration, &t.Transcription, &t.LastConversionTime,
		&t.HasError, &t.ErrorMessage, &t.FileHash, &t.FileSize,
		&t.ProviderType, &t.Language, &t.ModelName,
		&t.CreatedAt, &t.UpdatedAt, &deletedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	
	if deletedAt.Valid {
		t.DeletedAt = &deletedAt.Time
	}
	
	return &t, nil
}