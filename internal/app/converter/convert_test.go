package converter

import (
	"log"
	"path/filepath"
	"testing"
	"tiktok-whisper/internal/app/api/whisper_cpp"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/util/files"
)

func TestDo(t *testing.T) {
	type args struct {
		user         string
		filePath     string
		convertCount int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "testUser",
			args: args{
				user:         "testUser",
				filePath:     "test/data/mp4",
				convertCount: 1,
			},
		},
	}

	projectRoot, err := files.GetProjectRoot()
	if err != nil {
		log.Fatalf("Failed to get project root: %v\n", err)
	}

	dbPath := filepath.Join(projectRoot, "data/transcription.db")

	binaryPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/main"
	modelPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"

	converter := NewConverter(whisper_cpp.NewLocalTranscriber(binaryPath, modelPath), sqlite.NewSQLiteDB(dbPath))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter.ConvertVideoDir(tt.args.user,
				filepath.Join(projectRoot, tt.args.filePath),
				"mp4",
				tt.args.convertCount)
		})
	}
}
