package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"tiktok-whisper/internal/app/model"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLiteDB(dbFilePath string) *SQLiteDB {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		log.Fatal(err)
	}
	return &SQLiteDB{db: db}
}

func (sdb *SQLiteDB) Close() error {
	return sdb.db.Close()
}

func (sdb *SQLiteDB) CheckIfFileProcessed(fileName string) (int, error) {
	query := `SELECT id FROM transcriptions WHERE file_name = ? AND has_error = 0`
	row := sdb.db.QueryRow(query, fileName)
	var id int
	err := row.Scan(&id)
	return id, err
}

func (sdb *SQLiteDB) RecordToDB(user, inputDir, fileName, mp3FileName string, audioDuration int, transcription string,
	lastConversionTime time.Time, hasError int, errorMessage string) {
	insertSQL := `INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`
	_, err := sdb.db.Exec(insertSQL, user, inputDir, fileName, mp3FileName, audioDuration, transcription, lastConversionTime, hasError, errorMessage)
	if err != nil {
		log.Fatalf("Failed to insert data into database: %v\n", err)
	}
}

func (sdb *SQLiteDB) GetAllByUser(userNickname string) ([]model.Transcription, error) {
	sqlStr := `
		SELECT id, user, last_conversion_time, mp3_file_name, audio_duration, transcription, error_message
		FROM transcriptions
		WHERE has_error = 0
		  AND "user" = ?
		ORDER BY last_conversion_time DESC;`
	rows, err := sdb.db.Query(sqlStr, userNickname)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	transcriptions := make([]model.Transcription, 0)

	for rows.Next() {
		var t model.Transcription
		err = rows.Scan(&t.ID, &t.User, &t.LastConversionTime, &t.Mp3FileName, &t.AudioDuration, &t.Transcription, &t.ErrorMessage)
		if err != nil {
			return nil, fmt.Errorf("db scan failed: %v", err)
		}

		transcriptions = append(transcriptions, t)
	}
	return transcriptions, nil
}
