package main

import (
	"log"
	"os"
	"tiktok-whisper/internal/app/repository/migrate"
)

func main() {
	// Set DB_PASSWORD for PostgreSQL connection
	os.Setenv("DB_PASSWORD", "passwd")

	log.Println("Re-running data migration from SQLite to PostgreSQL...")
	migrate.MigrateToPostgres()
	log.Println("Migration completed!")
}
