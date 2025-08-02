package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// StaticHandler handles static file serving
type StaticHandler struct {
	staticDir string
}

// NewStaticHandler creates a new static file handler
func NewStaticHandler() *StaticHandler {
	// Get the current working directory
	wd, _ := os.Getwd()
	staticDir := filepath.Join(wd, "web", "static")

	return &StaticHandler{
		staticDir: staticDir,
	}
}

// ServeStatic serves static files and the main HTML page
func (h *StaticHandler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	path := filepath.Clean(r.URL.Path)

	// If requesting root, serve index.html
	if path == "/" || path == "/index.html" {
		h.serveFile(w, r, "index.html", "text/html")
		return
	}

	// Remove leading slash
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// Serve static files
	fullPath := filepath.Join(h.staticDir, path)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Determine content type based on file extension
	contentType := getContentType(path)
	h.serveFile(w, r, path, contentType)
}

// serveFile serves a specific file with the given content type
func (h *StaticHandler) serveFile(w http.ResponseWriter, r *http.Request, filename, contentType string) {
	fullPath := filepath.Join(h.staticDir, filename)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", contentType)

	// Set caching headers for static assets
	if filename != "index.html" {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}

	// Serve the file
	http.ServeFile(w, r, fullPath)
}

// getContentType returns the appropriate content type for a file
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}
