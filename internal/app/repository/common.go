package repository

import (
	"database/sql"
	"fmt"
	"tiktok-whisper/internal/app/model"
	"time"
)

// CommonDB provides shared database functionality
type CommonDB struct {
	db           *sql.DB
	driverName   string
	placeholders PlaceholderFunc
}

// PlaceholderFunc generates parameter placeholders for different SQL dialects
type PlaceholderFunc func(n int) string

// NewCommonDB creates a new CommonDB instance
func NewCommonDB(db *sql.DB, driverName string) *CommonDB {
	var placeholders PlaceholderFunc
	
	switch driverName {
	case "sqlite3":
		placeholders = func(n int) string { return "?" }
	case "postgres":
		placeholders = func(n int) string { return fmt.Sprintf("$%d", n) }
	default:
		placeholders = func(n int) string { return "?" }
	}
	
	return &CommonDB{
		db:           db,
		driverName:   driverName,
		placeholders: placeholders,
	}
}

// CheckIfFileProcessed checks if a file has been processed
func (c *CommonDB) CheckIfFileProcessed(fileName string) (int, error) {
	var count int
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM transcriptions WHERE file_name = %s",
		c.placeholders(1),
	)
	
	err := c.db.QueryRow(query, fileName).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}
	
	return count, nil
}

// GetAllByUser retrieves all transcriptions for a user
func (c *CommonDB) GetAllByUser(userNickname string) ([]model.Transcription, error) {
	query := fmt.Sprintf(
		`SELECT id, user, last_conversion_time, mp3_file_name, 
		        audio_duration, transcription, error_message 
		 FROM transcriptions 
		 WHERE user = %s 
		 ORDER BY last_conversion_time DESC`,
		c.placeholders(1),
	)
	
	rows, err := c.db.Query(query, userNickname)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	var transcriptions []model.Transcription
	for rows.Next() {
		var t model.Transcription
		err := rows.Scan(
			&t.ID,
			&t.User,
			&t.LastConversionTime,
			&t.Mp3FileName,
			&t.AudioDuration,
			&t.Transcription,
			&t.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		transcriptions = append(transcriptions, t)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return transcriptions, nil
}

// RecordToDB records a transcription to the database
func (c *CommonDB) RecordToDB(
	userNickname string,
	inputDir string,
	fileName string,
	mp3FileName string,
	audioDuration float64,
	transcription string,
	lastConversionTime time.Time,
	hasError int,
	errorMessage string,
) error {
	// Build placeholders
	params := make([]string, 9)
	for i := 0; i < 9; i++ {
		params[i] = c.placeholders(i + 1)
	}
	
	query := fmt.Sprintf(
		`INSERT INTO transcriptions (
			user, input_dir, file_name, mp3_file_name, 
			audio_duration, transcription, last_conversion_time, 
			has_error, error_message
		) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		params[0], params[1], params[2], params[3],
		params[4], params[5], params[6], params[7], params[8],
	)
	
	_, err := c.db.Exec(
		query,
		userNickname, inputDir, fileName, mp3FileName,
		audioDuration, transcription, lastConversionTime.Unix(),
		hasError, errorMessage,
	)
	
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	
	return nil
}

// Close closes the database connection
func (c *CommonDB) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// DB returns the underlying database connection
func (c *CommonDB) DB() *sql.DB {
	return c.db
}