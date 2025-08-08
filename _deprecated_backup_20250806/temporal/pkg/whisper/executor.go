package whisper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	
	"tiktok-whisper/temporal/pkg/common"
)

// Config holds whisper execution configuration
type Config struct {
	BinaryPath string
	ModelPath  string
}

// DefaultConfig returns default whisper configuration
func DefaultConfig() Config {
	return Config{
		BinaryPath: common.GetEnv("WHISPER_BINARY_PATH", common.DefaultWhisperBinary),
		ModelPath:  common.GetEnv("WHISPER_MODEL_PATH", common.DefaultWhisperModel),
	}
}

// ExecuteWhisper runs whisper.cpp and returns the transcription
func ExecuteWhisper(ctx context.Context, config Config, filePath, fileID, language string) (string, error) {
	// Validate configuration
	if _, err := os.Stat(config.BinaryPath); os.IsNotExist(err) {
		return "", fmt.Errorf("whisper binary not found at %s", config.BinaryPath)
	}
	
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		return "", fmt.Errorf("whisper model not found at %s", config.ModelPath)
	}
	
	// Create output file path
	outputPath := filepath.Join(os.TempDir(), fmt.Sprintf("transcription_%s.txt", fileID))
	
	// Build whisper command
	args := []string{
		"-m", config.ModelPath,
		"-f", filePath,
		"-otxt",
		"-of", strings.TrimSuffix(outputPath, ".txt"),
	}
	
	// Add language if specified
	if language != "" && language != "auto" {
		args = append(args, "-l", language)
	}
	
	// Run whisper
	cmd := exec.CommandContext(ctx, config.BinaryPath, args...)
	
	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper execution failed: %w, output: %s", err, string(output))
	}
	
	// Read transcription result
	transcriptionBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription output: %w", err)
	}
	
	// Clean up output file
	os.Remove(outputPath)
	
	return string(transcriptionBytes), nil
}