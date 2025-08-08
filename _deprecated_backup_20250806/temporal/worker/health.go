package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HealthStatus represents the health status of the worker
type HealthStatus struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	WorkerID     string            `json:"worker_id"`
	TaskQueue    string            `json:"task_queue"`
	Providers    []ProviderStatus  `json:"providers"`
	MinIO        ConnectionStatus  `json:"minio"`
	Temporal     ConnectionStatus  `json:"temporal"`
}

// ProviderStatus represents the status of a transcription provider
type ProviderStatus struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

// ConnectionStatus represents the status of an external connection
type ConnectionStatus struct {
	Connected bool   `json:"connected"`
	Endpoint  string `json:"endpoint"`
	Error     string `json:"error,omitempty"`
}

// startHealthServer starts the health check HTTP server
func startHealthServer(port string, status *HealthStatus) {
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status.Timestamp = time.Now()
		
		// Determine overall status
		if status.Temporal.Connected && status.MinIO.Connected && len(status.Providers) > 0 {
			status.Status = "healthy"
		} else if status.Temporal.Connected {
			status.Status = "degraded"
		} else {
			status.Status = "unhealthy"
		}
		
		w.Header().Set("Content-Type", "application/json")
		if status.Status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if status.Status == "degraded" {
			w.WriteHeader(http.StatusPartialContent)
		}
		
		json.NewEncoder(w).Encode(status)
	})
	
	// Liveness probe (always returns 200 if service is running)
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	
	// Readiness probe (returns 200 only if ready to process)
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if status.Temporal.Connected && len(status.Providers) > 0 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
		}
	})
	
	// Metrics endpoint (Prometheus format)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(`# HELP v2t_worker_up Worker is up and running
# TYPE v2t_worker_up gauge
v2t_worker_up 1

# HELP v2t_worker_providers_total Total number of available providers
# TYPE v2t_worker_providers_total gauge
`))
		
		availableProviders := 0
		for _, p := range status.Providers {
			if p.Available {
				availableProviders++
			}
		}
		w.Write([]byte(fmt.Sprintf("v2t_worker_providers_total %d\n", availableProviders)))
		
		w.Write([]byte(`
# HELP v2t_worker_temporal_connected Temporal connection status
# TYPE v2t_worker_temporal_connected gauge
`))
		if status.Temporal.Connected {
			w.Write([]byte("v2t_worker_temporal_connected 1\n"))
		} else {
			w.Write([]byte("v2t_worker_temporal_connected 0\n"))
		}
		
		w.Write([]byte(`
# HELP v2t_worker_minio_connected MinIO connection status
# TYPE v2t_worker_minio_connected gauge
`))
		if status.MinIO.Connected {
			w.Write([]byte("v2t_worker_minio_connected 1\n"))
		} else {
			w.Write([]byte("v2t_worker_minio_connected 0\n"))
		}
	})
	
	server := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()
}