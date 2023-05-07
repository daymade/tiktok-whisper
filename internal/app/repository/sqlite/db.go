package sqlite

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"tiktok-whisper/internal/app/util/files"
)

var (
	connection *sql.DB
	once       sync.Once
)

func GetConnection() (*sql.DB, error) {
	var err error
	once.Do(func() {
		projectRoot, err := files.GetProjectRoot()
		if err != nil {
			log.Fatalf("Failed to get project root: %v\n", err)
		}

		dbPath := filepath.Join(projectRoot, "data/transcription.db")

		connection, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&mode=rwc", dbPath))

	})

	if err != nil {
		return nil, fmt.Errorf("Failed to open database: %v\n", err)
	}
	return connection, err
}

func InitDB() *sql.DB {
	db, err := GetConnection()

	projectRoot, err := files.GetProjectRoot()
	if err != nil {
		log.Fatalf("Failed to get project root: %v\n", err)
	}

	sqlPath := filepath.Join(projectRoot, "cmd/app/repository/sqlite/create_table.sql")

	createTableSQL, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		log.Fatalf("Failed to create table: %v\n", err)
	}

	_, err = db.Exec(string(createTableSQL))
	if err != nil {
		log.Fatalf("Failed to create table: %v\n", err)
	}

	return db
}
