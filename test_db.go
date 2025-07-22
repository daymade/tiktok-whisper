package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    _ "github.com/lib/pq"
)

func main() {
    // Use environment variables for database connection
    host := getEnvOrDefault("DB_HOST", "localhost")
    port := getEnvOrDefault("DB_PORT", "5432")
    user := getEnvOrDefault("DB_USER", "postgres")
    password := getEnvOrDefault("DB_PASSWORD", "")
    dbname := getEnvOrDefault("DB_NAME", "postgres")
    
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", 
        host, port, user, password, dbname)
    
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer db.Close()

    if err := db.Ping(); err \!= nil {
        log.Fatalf("Failed to ping: %v", err)
    }

    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM transcriptions WHERE embedding_gemini IS NOT NULL").Scan(&count)
    if err \!= nil {
        log.Fatalf("Query failed: %v", err)
    }

    fmt.Printf("✅ 数据库连接成功！找到 %d 个Gemini embeddings\n", count)
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
