package handlers

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestApplyCORSAllowsConfiguredOrigin(t *testing.T) {
	old := os.Getenv("WEB_ALLOWED_ORIGINS")
	t.Cleanup(func() {
		_ = os.Setenv("WEB_ALLOWED_ORIGINS", old)
	})

	if err := os.Setenv("WEB_ALLOWED_ORIGINS", "https://app.example.com, https://ops.example.com"); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/embeddings", nil)
	req.Header.Set("Origin", "https://ops.example.com")
	rec := httptest.NewRecorder()

	applyCORS(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://ops.example.com" {
		t.Fatalf("unexpected allow-origin header: %q", got)
	}
}

func TestApplyCORSDoesNothingByDefault(t *testing.T) {
	old := os.Getenv("WEB_ALLOWED_ORIGINS")
	t.Cleanup(func() {
		_ = os.Setenv("WEB_ALLOWED_ORIGINS", old)
	})
	_ = os.Unsetenv("WEB_ALLOWED_ORIGINS")

	req := httptest.NewRequest("GET", "/api/embeddings", nil)
	req.Header.Set("Origin", "https://ops.example.com")
	rec := httptest.NewRecorder()

	applyCORS(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow-origin header, got %q", got)
	}
}
