package pg

import (
	"database/sql"
	"errors"
	"log"
	"tiktok-whisper/internal/app/model"
	"time"

	_ "github.com/lib/pq"
)

type PostgresDB struct {
	db *sql.DB
}

func NewPostgresDB(connectionString string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}
	return &PostgresDB{db: db}, nil
}

func (pdb *PostgresDB) Close() error {
	return pdb.db.Close()
}

func (pdb *PostgresDB) CheckIfFileProcessed(fileName string) (int, error) {
	query := `SELECT id FROM transcriptions WHERE file_name = $1 AND has_error = 0`
	row := pdb.db.QueryRow(query, fileName)
	var id int
	err := row.Scan(&id)
	return id, err
}

func (pdb *PostgresDB) RecordToDB(user, inputDir, fileName, mp3FileName string, audioDuration int, transcription string,
	lastConversionTime time.Time, hasError int, errorMessage string) {
	insertSQL := `INSERT INTO transcriptions (user, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`
	_, err := pdb.db.Exec(insertSQL, user, inputDir, fileName, mp3FileName, audioDuration, transcription, lastConversionTime, hasError, errorMessage)
	if err != nil {
		log.Fatalf("Failed to insert data into database: %v\n", err)
	}
}

func (pdb *PostgresDB) GetAllByUser(userNickname string) ([]model.Transcription, error) {
	return nil, errors.New("not implemented")
}
