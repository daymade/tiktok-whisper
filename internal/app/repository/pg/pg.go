package pg

import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
)

func GetConnection() (*sql.DB, error) {
	postgresDB, err := sql.Open("postgres", "user=postgres password=passwd dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to open database: %v\n", err)
	}
	return postgresDB, nil
}
