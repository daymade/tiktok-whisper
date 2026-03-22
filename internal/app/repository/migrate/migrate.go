package migrate

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"tiktok-whisper/internal/app/repository/pg"
	"tiktok-whisper/internal/app/repository/sqlite"
)

func getLastID() int {
	data, err := ioutil.ReadFile("last_id.txt")
	if err != nil {
		return 0
	}

	lastID, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}

	return lastID
}

func saveLastID(lastID int) error {
	return ioutil.WriteFile("last_id.txt", []byte(strconv.Itoa(lastID)), 0644)
}

func migrateBatch(sqliteDB, postgresDB *sql.DB, lastID int) (int, error) {
	rows, err := sqliteDB.Query(`SELECT id, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message, "user" FROM transcriptions WHERE id > ? ORDER BY id LIMIT 1000`, lastID)
	if err != nil {
		return lastID, err
	}
	defer rows.Close()

	tx, err := postgresDB.Begin()
	if err != nil {
		return lastID, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.Prepare(`INSERT INTO transcriptions (id, input_dir, file_name, mp3_file_name, audio_duration, transcription, last_conversion_time, has_error, error_message, user_nickname) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`)
	if err != nil {
		return lastID, err
	}
	defer stmt.Close()

	for rows.Next() {
		var id, audioDuration, hasError int
		var inputDir, fileName, mp3FileName, transcription, errorMessage, user string
		var lastConversionTime string

		err = rows.Scan(&id, &inputDir, &fileName, &mp3FileName, &audioDuration, &transcription, &lastConversionTime, &hasError, &errorMessage, &user)
		if err != nil {
			return lastID, fmt.Errorf("failed to read row with ID %d: %w", id, err)
		}

		if strings.TrimSpace(inputDir) == "" || strings.TrimSpace(fileName) == "" {
			return lastID, fmt.Errorf("validation failed for row with ID %d: input_dir or file_name is empty", id)
		}

		if _, err = stmt.Exec(id, inputDir, fileName, mp3FileName, audioDuration, transcription, lastConversionTime, hasError, errorMessage, user); err != nil {
			return lastID, fmt.Errorf("failed to insert row with ID %d: %w", id, err)
		}

		lastID = id
	}

	if err := rows.Err(); err != nil {
		return lastID, err
	}

	if err := tx.Commit(); err != nil {
		return lastID, err
	}

	return lastID, nil
}

func MigrateToPostgres() {
	sqliteDB, err := sqlite.GetConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteDB.Close()

	postgresDB, err := pg.GetConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer postgresDB.Close()

	lastID := getLastID()

	lastID, err = migrateBatch(sqliteDB, postgresDB, lastID)
	if err != nil {
		log.Fatal(err)
	}

	err = saveLastID(lastID)
	if err != nil {
		log.Fatalf("Failed to save lastID: %v", err)
	}

	fmt.Println("Data migration completed.")
}
