package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// HealthStatus represents the worker health status
type HealthStatus struct {
	WorkerID    string             `json:"worker_id"`
	TaskQueue   string             `json:"task_queue"`
	Status      string             `json:"status"`
	Uptime      time.Duration      `json:"uptime"`
	StartedAt   time.Time          `json:"started_at"`
	Temporal    ConnectionStatus   `json:"temporal"`
	MinIO       ConnectionStatus   `json:"minio,omitempty"`
	Providers   []ProviderStatus   `json:"providers,omitempty"`
}

// ConnectionStatus represents a connection status
type ConnectionStatus struct {
	Connected bool   `json:"connected"`
	Endpoint  string `json:"endpoint"`
	Error     string `json:"error,omitempty"`
}

// ProviderStatus represents a provider status
type ProviderStatus struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

// startHealthServer starts the health check HTTP server
func startHealthServer(port string, status *HealthStatus) {
	mux := http.NewServeMux()
	
	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Update uptime
		status.Uptime = time.Since(status.StartedAt)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})
	
	// Liveness probe
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Readiness probe
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if status.Temporal.Connected {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("READY"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
		}
	})
	
	// Start server in background
	go func() {
		if err := http.ListenAndServe(port, mux); err != nil {
			// Log error but don't fail the worker
			log.Printf("Health server failed to start on %s: %v", port, err)
		}
	}()
}

// getEnv gets environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}