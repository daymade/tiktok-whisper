package pg

import (
	"fmt"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
)

// Ensure PostgresDB implements TranscriptionDAOV2
var _ repository.TranscriptionDAOV2 = (*PostgresDB)(nil)

// RecordToDBV2 inserts a transcription with all new fields
func (pdb *PostgresDB) RecordToDBV2(t *model.TranscriptionFull) error {
	// First check if there's already a record with this file hash
	if t.FileHash != "" {
		existing, _ := pdb.GetByFileHash(t.FileHash)
		if existing != nil {
			// Update the existing record
			t.ID = existing.ID
		}
	}

	if t.ID > 0 {
		// Update existing record
		updateSQL := `
			UPDATE transcriptions SET
				user_nickname = $1,
				input_dir = $2,
				file_name = $3,
				mp3_file_name = $4,
				audio_duration = $5,
				transcription = $6,
				last_conversion_time = $7,
				has_error = $8,
				error_message = $9,
				"user" = $10
			WHERE id = $11`
		
		_, err := pdb.db.Exec(updateSQL,
			t.User, t.InputDir, t.FileName, t.Mp3FileName, t.AudioDuration,
			t.Transcription, t.LastConversionTime, t.HasError, t.ErrorMessage,
			t.User, t.ID)
		return err
	}

	// Insert new record
	insertSQL := `
		INSERT INTO transcriptions (
			"user", input_dir, file_name, mp3_file_name, audio_duration, 
			transcription, last_conversion_time, has_error, error_message, user_nickname
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	
	err := pdb.db.QueryRow(insertSQL,
		t.User, t.InputDir, t.FileName, t.Mp3FileName, t.AudioDuration,
		t.Transcription, t.LastConversionTime, t.HasError, t.ErrorMessage,
		t.User).Scan(&t.ID)
	
	return err
}

// GetAllByUserV2 retrieves all non-errored transcriptions for a user with full fields
func (pdb *PostgresDB) GetAllByUserV2(userNickname string) ([]model.TranscriptionFull, error) {
	query := `
		SELECT id, COALESCE("user", ''), COALESCE(user_nickname, ''), 
		       input_dir, file_name, mp3_file_name, audio_duration, 
		       transcription, last_conversion_time, has_error, COALESCE(error_message, '')
		FROM transcriptions
		WHERE has_error = 0 AND (user_nickname = $1 OR "user" = $1)
		ORDER BY last_conversion_time DESC`

	rows, err := pdb.db.Query(query, userNickname)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	var transcriptions []model.TranscriptionFull
	for rows.Next() {
		var t model.TranscriptionFull
		var userNickname string
		err = rows.Scan(&t.ID, &t.User, &userNickname, &t.InputDir, &t.FileName, 
			&t.Mp3FileName, &t.AudioDuration, &t.Transcription, 
			&t.LastConversionTime, &t.HasError, &t.ErrorMessage)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %v", err)
		}
		transcriptions = append(transcriptions, t)
	}

	return transcriptions, rows.Err()
}

// GetByFileHash retrieves a transcription by file hash
func (pdb *PostgresDB) GetByFileHash(fileHash string) (*model.TranscriptionFull, error) {
	// Since the current PostgreSQL schema doesn't have file_hash column,
	// we'll return nil for now
	return nil, nil
}

// GetByProvider retrieves transcriptions by provider type
func (pdb *PostgresDB) GetByProvider(providerType string, limit int) ([]model.TranscriptionFull, error) {
	// Since the current PostgreSQL schema doesn't have provider_type column,
	// we'll return empty for now
	return []model.TranscriptionFull{}, nil
}

// UpdateFileMetadata updates file metadata for a transcription
func (pdb *PostgresDB) UpdateFileMetadata(id int, fileHash string, fileSize int64) error {
	// Since the current PostgreSQL schema doesn't have these columns,
	// we'll skip for now
	return nil
}

// SoftDelete performs a soft delete on a transcription
func (pdb *PostgresDB) SoftDelete(id int) error {
	// Mark as error to hide from normal queries
	updateSQL := `UPDATE transcriptions SET has_error = 1 WHERE id = $1`
	_, err := pdb.db.Exec(updateSQL, id)
	return err
}

// GetActiveTranscriptions retrieves active (non-errored) transcriptions
func (pdb *PostgresDB) GetActiveTranscriptions(limit int) ([]model.TranscriptionFull, error) {
	query := `
		SELECT id, COALESCE("user", ''), COALESCE(user_nickname, ''),
		       input_dir, file_name, mp3_file_name, audio_duration,
		       transcription, last_conversion_time, has_error, COALESCE(error_message, '')
		FROM transcriptions
		WHERE has_error = 0
		ORDER BY last_conversion_time DESC
		LIMIT $1`

	rows, err := pdb.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	var transcriptions []model.TranscriptionFull
	for rows.Next() {
		var t model.TranscriptionFull
		var userNickname string
		err = rows.Scan(&t.ID, &t.User, &userNickname, &t.InputDir, &t.FileName,
			&t.Mp3FileName, &t.AudioDuration, &t.Transcription,
			&t.LastConversionTime, &t.HasError, &t.ErrorMessage)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %v", err)
		}
		transcriptions = append(transcriptions, t)
	}

	return transcriptions, rows.Err()
}

// GetTranscriptionByID retrieves a transcription by ID
func (pdb *PostgresDB) GetTranscriptionByID(id int) (*model.TranscriptionFull, error) {
	query := `
		SELECT id, COALESCE("user", ''), COALESCE(user_nickname, ''),
		       input_dir, file_name, mp3_file_name, audio_duration,
		       transcription, last_conversion_time, has_error, COALESCE(error_message, '')
		FROM transcriptions
		WHERE id = $1`

	var t model.TranscriptionFull
	var userNickname string
	err := pdb.db.QueryRow(query, id).Scan(
		&t.ID, &t.User, &userNickname, &t.InputDir, &t.FileName,
		&t.Mp3FileName, &t.AudioDuration, &t.Transcription,
		&t.LastConversionTime, &t.HasError, &t.ErrorMessage)
	
	if err != nil {
		return nil, err
	}
	
	return &t, nil
}