package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeStaticBlocksDebugPages(t *testing.T) {
	tmpDir := t.TempDir()
	staticDir := filepath.Join(tmpDir, "web", "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "debug.html"), []byte("debug"), 0644); err != nil {
		t.Fatalf("write debug: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("ok"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	h := &StaticHandler{staticDir: staticDir}

	req := httptest.NewRequest(http.MethodGet, "/debug.html", nil)
	rec := httptest.NewRecorder()
	h.ServeStatic(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for debug page, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	h.ServeStatic(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for index, got %d", rec.Code)
	}
}
