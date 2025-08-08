package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// WhisperProxyHandler handles proxying requests to external v2t backend
type WhisperProxyHandler struct {
	backendURL string
	apiKey     string
	httpClient *http.Client
}

// NewWhisperProxyHandler creates a new proxy handler
func NewWhisperProxyHandler() *WhisperProxyHandler {
	backendURL := os.Getenv("V2T_API_URL")
	if backendURL == "" {
		backendURL = os.Getenv("WHISPER_BACKEND_URL")
	}
	if backendURL == "" {
		backendURL = "http://localhost:8085"
	}

	return &WhisperProxyHandler{
		backendURL: strings.TrimSuffix(backendURL, "/"),
		apiKey:     os.Getenv("WHISPER_BACKEND_API_KEY"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ProxyRequest handles proxying any request to the v2t backend
// @Summary Proxy to v2t backend
// @Description Proxy requests to external v2t transcription backend
// @Tags WhisperProxy
// @Accept json
// @Produce json
// @Param path path string true "API path to proxy"
// @Success 200 {object} interface{} "Proxied response"
// @Router /api/whisper/{path} [get]
// @Router /api/whisper/{path} [post]
// @Router /api/whisper/{path} [put]
// @Router /api/whisper/{path} [delete]
func (h *WhisperProxyHandler) ProxyRequest(c *gin.Context) {
	// Get the path after /api/whisper/
	path := c.Param("path")
	if path == "" || path == "/" {
		path = ""
	} else if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Build the backend URL
	targetURL := fmt.Sprintf("%s/api/v1%s", h.backendURL, path)

	// Add query parameters if any
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	// Create the proxy request
	var body io.Reader
	if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
		// Read the request body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read request body",
			})
			return
		}
		body = bytes.NewReader(bodyBytes)
	}

	// Create the request
	req, err := http.NewRequest(c.Request.Method, targetURL, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create proxy request",
		})
		return
	}

	// Copy headers from original request
	for key, values := range c.Request.Header {
		// Skip hop-by-hop headers
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add API key if configured
	if h.apiKey != "" {
		req.Header.Set("X-API-Key", h.apiKey)
	}

	// Add forwarded headers
	if clientIP := c.ClientIP(); clientIP != "" {
		req.Header.Set("X-Forwarded-For", clientIP)
	}
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Forwarded-Proto", c.Request.URL.Scheme)

	// Make the request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		// Check if it's a connection error
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error": "Request to backend timed out",
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("Backend service unavailable: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read backend response",
		})
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		// Skip hop-by-hop headers
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Try to parse as JSON
	var jsonResponse interface{}
	if err := json.Unmarshal(respBody, &jsonResponse); err == nil {
		// It's valid JSON, return as JSON
		c.JSON(resp.StatusCode, jsonResponse)
	} else {
		// Not JSON, return as raw data
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
	}
}

// isHopByHopHeader checks if a header is a hop-by-hop header that shouldn't be forwarded
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
	
	header = strings.ToLower(header)
	for _, h := range hopByHopHeaders {
		if strings.ToLower(h) == header {
			return true
		}
	}
	return false
}

// HealthCheck checks if the backend is available
func (h *WhisperProxyHandler) HealthCheck(c *gin.Context) {
	healthURL := fmt.Sprintf("%s/health", h.backendURL)
	
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"message": "Failed to create health check request",
		})
		return
	}

	if h.apiKey != "" {
		req.Header.Set("X-API-Key", h.apiKey)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"message": fmt.Sprintf("Backend unavailable: %v", err),
			"backend_url": h.backendURL,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"message": "Backend is available",
			"backend_url": h.backendURL,
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"message": fmt.Sprintf("Backend returned status %d", resp.StatusCode),
			"backend_url": h.backendURL,
		})
	}
}