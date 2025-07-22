package web

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"tiktok-whisper/web/handlers"

	_ "github.com/lib/pq"
)

// Server represents the web server
type Server struct {
	db   *sql.DB
	addr string
}

// NewServer creates a new web server instance
func NewServer(addr string) (*Server, error) {
	// Connect to PostgreSQL database (pgvector)
	// Use environment variables or defaults for database connection
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "5432")
	user := getEnvOrDefault("DB_USER", "postgres")
	password := getEnvOrDefault("DB_PASSWORD", "")
	dbname := getEnvOrDefault("DB_NAME", "postgres")
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", 
		host, port, user, password, dbname)
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	// Test database connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	// Log connection info without sensitive data
	log.Printf("✅ Connected to PostgreSQL database (pgvector): host=%s port=%s user=%s dbname=%s", 
		host, port, user, dbname)

	return &Server{
		db:   db,
		addr: addr,
	}, nil
}

// Start starts the web server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Create handlers
	apiHandler := handlers.NewAPIHandler(s.db)
	staticHandler := handlers.NewStaticHandler()

	// API routes
	mux.HandleFunc("/api/embeddings", apiHandler.GetEmbeddings)
	mux.HandleFunc("/api/embeddings/search", apiHandler.SearchEmbeddings)
	mux.HandleFunc("/api/embeddings/cluster", apiHandler.GetClusters)
	mux.HandleFunc("/api/users", apiHandler.GetUsers)
	mux.HandleFunc("/api/stats", apiHandler.GetStats)

	// Static file serving
	mux.HandleFunc("/", staticHandler.ServeStatic)

	log.Printf("🚀 Starting embedding visualization server on %s", s.addr)
	log.Printf("🌐 Visit http://localhost%s to view the visualization", s.addr)
	
	return http.ListenAndServe(s.addr, mux)
}

// Close closes the database connection
func (s *Server) Close() error {
	return s.db.Close()
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}