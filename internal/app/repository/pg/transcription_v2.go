package pg

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
)

// Ensure PostgresDB implements TranscriptionDAOV2
var _ repository.TranscriptionDAOV2 = (*PostgresDB)(nil)

func (pdb *PostgresDB) transcriptionColumns() (map[string]bool, error) {
	rows, err := pdb.db.Query(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'transcriptions'`)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect transcriptions schema: %w", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan transcriptions schema: %w", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate transcriptions schema: %w", err)
	}
	return columns, nil
}

func quoteIdentifier(column string) string {
	if column == "user" {
		return `"user"`
	}
	return column
}

func addInsertColumn(columns []string, placeholders []string, args []interface{}, column string, value interface{}) ([]string, []string, []interface{}) {
	columns = append(columns, column)
	placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
	args = append(args, value)
	return columns, placeholders, args
}

func addUpdateAssignment(assignments []string, args []interface{}, column string, value interface{}) ([]string, []interface{}) {
	assignments = append(assignments, fmt.Sprintf("%s = $%d", quoteIdentifier(column), len(args)+1))
	args = append(args, value)
	return assignments, args
}

func normalizePGTimes(t *model.TranscriptionFull) {
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

func missingColumnError(column string) error {
	return fmt.Errorf("transcriptions schema does not include required column %q", column)
}

func sqlString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func sqlInt(value interface{}) int {
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case []byte:
		i, _ := strconv.Atoi(string(v))
		return i
	default:
		return 0
	}
}

func sqlInt64(value interface{}) int64 {
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case []byte:
		i, _ := strconv.ParseInt(string(v), 10, 64)
		return i
	default:
		return 0
	}
}

func sqlTime(value interface{}) time.Time {
	switch v := value.(type) {
	case nil:
		return time.Time{}
	case time.Time:
		return v
	case []byte:
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05-07", "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, string(v)); err == nil {
				return parsed
			}
		}
	case string:
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05-07", "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, v); err == nil {
				return parsed
			}
		}
	}
	return time.Time{}
}

func scanTranscriptionRow(rows *sql.Rows) (*model.TranscriptionFull, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to list columns: %w", err)
	}

	raw := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range raw {
		ptrs[i] = &raw[i]
	}

	if err := rows.Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("failed to scan transcription row: %w", err)
	}

	t := &model.TranscriptionFull{}
	for i, col := range cols {
		switch col {
		case "id":
			t.ID = sqlInt(raw[i])
		case "user", "user_nickname":
			if t.User == "" {
				t.User = sqlString(raw[i])
			}
		case "input_dir":
			t.InputDir = sqlString(raw[i])
		case "file_name":
			t.FileName = sqlString(raw[i])
		case "mp3_file_name":
			t.Mp3FileName = sqlString(raw[i])
		case "audio_duration":
			t.AudioDuration = sqlInt(raw[i])
		case "transcription":
			t.Transcription = sqlString(raw[i])
		case "last_conversion_time":
			t.LastConversionTime = sqlTime(raw[i])
		case "has_error":
			t.HasError = sqlInt(raw[i])
		case "error_message":
			t.ErrorMessage = sqlString(raw[i])
		case "file_hash":
			t.FileHash = sqlString(raw[i])
		case "file_size":
			t.FileSize = sqlInt64(raw[i])
		case "provider_type":
			t.ProviderType = sqlString(raw[i])
		case "language":
			t.Language = sqlString(raw[i])
		case "model_name":
			t.ModelName = sqlString(raw[i])
		case "created_at":
			t.CreatedAt = sqlTime(raw[i])
		case "updated_at":
			t.UpdatedAt = sqlTime(raw[i])
		case "deleted_at":
			deletedAt := sqlTime(raw[i])
			if !deletedAt.IsZero() {
				t.DeletedAt = &deletedAt
			}
		}
	}
	return t, nil
}

// RecordToDBV2 inserts or updates a transcription with all supported fields.
func (pdb *PostgresDB) RecordToDBV2(t *model.TranscriptionFull) error {
	if t == nil {
		return fmt.Errorf("transcription is nil")
	}

	columns, err := pdb.transcriptionColumns()
	if err != nil {
		return err
	}

	normalizePGTimes(t)

	if t.ID == 0 && t.FileHash != "" && columns["file_hash"] {
		existing, err := pdb.GetByFileHash(t.FileHash)
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

	if t.ID > 0 {
		assignments := make([]string, 0, 16)
		args := make([]interface{}, 0, 16)

		if columns["user_nickname"] {
			assignments, args = addUpdateAssignment(assignments, args, "user_nickname", t.User)
		}
		if columns["user"] {
			assignments, args = addUpdateAssignment(assignments, args, "user", t.User)
		}
		if columns["input_dir"] {
			assignments, args = addUpdateAssignment(assignments, args, "input_dir", t.InputDir)
		}
		if columns["file_name"] {
			assignments, args = addUpdateAssignment(assignments, args, "file_name", t.FileName)
		}
		if columns["mp3_file_name"] {
			assignments, args = addUpdateAssignment(assignments, args, "mp3_file_name", t.Mp3FileName)
		}
		if columns["audio_duration"] {
			assignments, args = addUpdateAssignment(assignments, args, "audio_duration", t.AudioDuration)
		}
		if columns["transcription"] {
			assignments, args = addUpdateAssignment(assignments, args, "transcription", t.Transcription)
		}
		if columns["last_conversion_time"] {
			assignments, args = addUpdateAssignment(assignments, args, "last_conversion_time", t.LastConversionTime)
		}
		if columns["has_error"] {
			assignments, args = addUpdateAssignment(assignments, args, "has_error", t.HasError)
		}
		if columns["error_message"] {
			assignments, args = addUpdateAssignment(assignments, args, "error_message", t.ErrorMessage)
		}
		if columns["file_hash"] {
			assignments, args = addUpdateAssignment(assignments, args, "file_hash", t.FileHash)
		}
		if columns["file_size"] {
			assignments, args = addUpdateAssignment(assignments, args, "file_size", t.FileSize)
		}
		if columns["provider_type"] {
			assignments, args = addUpdateAssignment(assignments, args, "provider_type", t.ProviderType)
		}
		if columns["language"] {
			assignments, args = addUpdateAssignment(assignments, args, "language", t.Language)
		}
		if columns["model_name"] {
			assignments, args = addUpdateAssignment(assignments, args, "model_name", t.ModelName)
		}
		if columns["updated_at"] {
			assignments, args = addUpdateAssignment(assignments, args, "updated_at", t.UpdatedAt)
		}

		if len(assignments) == 0 {
			return fmt.Errorf("no updatable transcription columns available")
		}

		args = append(args, t.ID)
		updateSQL := fmt.Sprintf(
			`UPDATE transcriptions SET %s WHERE id = $%d`,
			strings.Join(assignments, ", "),
			len(args),
		)
		_, err := pdb.db.Exec(updateSQL, args...)
		return err
	}

	insertColumns := make([]string, 0, 16)
	placeholders := make([]string, 0, 16)
	args := make([]interface{}, 0, 16)

	if columns["user"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, `"user"`, t.User)
	}
	if columns["user_nickname"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "user_nickname", t.User)
	}
	if columns["input_dir"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "input_dir", t.InputDir)
	}
	if columns["file_name"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "file_name", t.FileName)
	}
	if columns["mp3_file_name"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "mp3_file_name", t.Mp3FileName)
	}
	if columns["audio_duration"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "audio_duration", t.AudioDuration)
	}
	if columns["transcription"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "transcription", t.Transcription)
	}
	if columns["last_conversion_time"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "last_conversion_time", t.LastConversionTime)
	}
	if columns["has_error"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "has_error", t.HasError)
	}
	if columns["error_message"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "error_message", t.ErrorMessage)
	}
	if columns["file_hash"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "file_hash", t.FileHash)
	}
	if columns["file_size"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "file_size", t.FileSize)
	}
	if columns["provider_type"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "provider_type", t.ProviderType)
	}
	if columns["language"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "language", t.Language)
	}
	if columns["model_name"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "model_name", t.ModelName)
	}
	if columns["created_at"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "created_at", t.CreatedAt)
	}
	if columns["updated_at"] {
		insertColumns, placeholders, args = addInsertColumn(insertColumns, placeholders, args, "updated_at", t.UpdatedAt)
	}

	insertSQL := fmt.Sprintf(
		`INSERT INTO transcriptions (%s) VALUES (%s) RETURNING id`,
		strings.Join(insertColumns, ", "),
		strings.Join(placeholders, ", "),
	)
	return pdb.db.QueryRow(insertSQL, args...).Scan(&t.ID)
}

// GetAllByUserV2 retrieves all non-errored transcriptions for a user with full fields.
func (pdb *PostgresDB) GetAllByUserV2(userNickname string) ([]model.TranscriptionFull, error) {
	columns, err := pdb.transcriptionColumns()
	if err != nil {
		return nil, err
	}

	whereParts := []string{"has_error = 0"}
	switch {
	case columns["user"] && columns["user_nickname"]:
		whereParts = append(whereParts, `("user" = $1 OR user_nickname = $1)`)
	case columns["user"]:
		whereParts = append(whereParts, `"user" = $1`)
	case columns["user_nickname"]:
		whereParts = append(whereParts, `user_nickname = $1`)
	default:
		return nil, fmt.Errorf("transcriptions schema does not include a user column")
	}

	query := fmt.Sprintf(`
		SELECT *
		FROM transcriptions
		WHERE %s
		ORDER BY last_conversion_time DESC`, strings.Join(whereParts, " AND "))

	rows, err := pdb.db.Query(query, userNickname)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var transcriptions []model.TranscriptionFull
	for rows.Next() {
		t, err := scanTranscriptionRow(rows)
		if err != nil {
			return nil, err
		}
		transcriptions = append(transcriptions, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return transcriptions, nil
}

// GetByFileHash retrieves a transcription by file hash.
func (pdb *PostgresDB) GetByFileHash(fileHash string) (*model.TranscriptionFull, error) {
	columns, err := pdb.transcriptionColumns()
	if err != nil {
		return nil, err
	}
	if !columns["file_hash"] {
		return nil, missingColumnError("file_hash")
	}

	rows, err := pdb.db.Query(`SELECT * FROM transcriptions WHERE file_hash = $1 LIMIT 1`, fileHash)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	return scanTranscriptionRow(rows)
}

// GetByProvider retrieves transcriptions by provider type.
func (pdb *PostgresDB) GetByProvider(providerType string, limit int) ([]model.TranscriptionFull, error) {
	columns, err := pdb.transcriptionColumns()
	if err != nil {
		return nil, err
	}
	if !columns["provider_type"] {
		return nil, missingColumnError("provider_type")
	}

	rows, err := pdb.db.Query(`SELECT * FROM transcriptions WHERE provider_type = $1 ORDER BY id DESC LIMIT $2`, providerType, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var transcriptions []model.TranscriptionFull
	for rows.Next() {
		t, err := scanTranscriptionRow(rows)
		if err != nil {
			return nil, err
		}
		transcriptions = append(transcriptions, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return transcriptions, nil
}

// UpdateFileMetadata updates file metadata for a transcription.
func (pdb *PostgresDB) UpdateFileMetadata(id int, fileHash string, fileSize int64) error {
	columns, err := pdb.transcriptionColumns()
	if err != nil {
		return err
	}
	if !columns["file_hash"] {
		return missingColumnError("file_hash")
	}
	if !columns["file_size"] {
		return missingColumnError("file_size")
	}

	query := `UPDATE transcriptions SET file_hash = $1, file_size = $2`
	args := []interface{}{fileHash, fileSize}
	if columns["updated_at"] {
		query += fmt.Sprintf(", updated_at = $%d", len(args)+1)
		args = append(args, time.Now())
	}
	query += fmt.Sprintf(" WHERE id = $%d", len(args)+1)
	args = append(args, id)

	_, err = pdb.db.Exec(query, args...)
	return err
}

// SoftDelete performs a soft delete on a transcription.
func (pdb *PostgresDB) SoftDelete(id int) error {
	if columns, err := pdb.transcriptionColumns(); err == nil && columns["deleted_at"] {
		query := `UPDATE transcriptions SET deleted_at = $1`
		args := []interface{}{time.Now()}
		if columns["updated_at"] {
			query += fmt.Sprintf(", updated_at = $%d", len(args)+1)
			args = append(args, time.Now())
		}
		query += fmt.Sprintf(" WHERE id = $%d", len(args)+1)
		args = append(args, id)
		_, err = pdb.db.Exec(query, args...)
		return err
	}

	updateSQL := `UPDATE transcriptions SET has_error = 1 WHERE id = $1`
	_, err := pdb.db.Exec(updateSQL, id)
	return err
}

// GetActiveTranscriptions retrieves active (non-errored) transcriptions.
func (pdb *PostgresDB) GetActiveTranscriptions(limit int) ([]model.TranscriptionFull, error) {
	rows, err := pdb.db.Query(`
		SELECT *
		FROM transcriptions
		WHERE has_error = 0
		ORDER BY last_conversion_time DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var transcriptions []model.TranscriptionFull
	for rows.Next() {
		t, err := scanTranscriptionRow(rows)
		if err != nil {
			return nil, err
		}
		transcriptions = append(transcriptions, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}
	return transcriptions, nil
}

// GetTranscriptionByID retrieves a transcription by ID.
func (pdb *PostgresDB) GetTranscriptionByID(id int) (*model.TranscriptionFull, error) {
	rows, err := pdb.db.Query(`SELECT * FROM transcriptions WHERE id = $1 LIMIT 1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	return scanTranscriptionRow(rows)
}
