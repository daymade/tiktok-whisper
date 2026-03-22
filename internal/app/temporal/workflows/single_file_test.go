package workflows

import (
	"testing"
	"time"
)

func TestBuildTranscriptionObjectKeyUsesProvidedTimestamp(t *testing.T) {
	now := time.Date(2026, time.March, 22, 20, 0, 0, 0, time.UTC)
	key := buildTranscriptionObjectKey(now, "file-123")

	if key != "transcriptions/2026-03-22/file-123.txt" {
		t.Fatalf("unexpected key: %s", key)
	}
}

func TestBuildTempTranscriptionPath(t *testing.T) {
	path := buildTempTranscriptionPath("/tmp/workflow", "file-123")
	if path != "/tmp/workflow/file-123.txt" {
		t.Fatalf("unexpected temp path: %s", path)
	}
}
