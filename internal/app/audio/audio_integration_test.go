//go:build integration
// +build integration

package audio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These are integration tests that can be run when FFmpeg is available
// Run with: go test -tags=integration ./internal/app/audio/

// TestGetAudioDurationIntegration tests duration extraction with actual audio files
func TestGetAudioDurationIntegration(t *testing.T) {
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping integration tests")
	}

	// Test with actual audio files from test/data/audio/
	testFiles := []struct {
		name                string
		relativePath        string
		expectedMinDuration int
		expectedMaxDuration int
	}{
		{
			name:                "short sine wave",
			relativePath:        "test/data/audio/short_sine_16khz.wav",
			expectedMinDuration: 1,
			expectedMaxDuration: 10,
		},
		{
			name:                "medium sine wave",
			relativePath:        "test/data/audio/medium_sine_16khz.wav",
			expectedMinDuration: 5,
			expectedMaxDuration: 60,
		},
		{
			name:                "silence file",
			relativePath:        "test/data/audio/silence_5s.wav",
			expectedMinDuration: 4,
			expectedMaxDuration: 6,
		},
	}

	for _, tt := range testFiles {
		t.Run(tt.name, func(t *testing.T) {
			// Get absolute path
			absPath, err := filepath.Abs(tt.relativePath)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			// Check if file exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("test file %s does not exist", absPath)
			}

			// Get duration
			duration, err := GetAudioDuration(absPath)
			if err != nil {
				t.Errorf("GetAudioDuration failed: %v", err)
				return
			}

			// Validate duration is within expected range
			if duration < tt.expectedMinDuration || duration > tt.expectedMaxDuration {
				t.Errorf("duration %d not in expected range [%d, %d]",
					duration, tt.expectedMinDuration, tt.expectedMaxDuration)
			}

			t.Logf("File: %s, Duration: %d seconds", tt.name, duration)
		})
	}
}

// TestIs16kHzWavFileIntegration tests WAV file detection with actual files
func TestIs16kHzWavFileIntegration(t *testing.T) {
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping integration tests")
	}

	testFiles := []struct {
		name          string
		relativePath  string
		expected16kHz bool
	}{
		{
			name:          "16kHz WAV file",
			relativePath:  "test/data/audio/short_sine_16khz.wav",
			expected16kHz: true,
		},
		{
			name:          "44kHz WAV file",
			relativePath:  "test/data/audio/short_sine_44khz.wav",
			expected16kHz: false,
		},
		{
			name:          "MP3 file",
			relativePath:  "test/data/audio/short_sine_22khz.mp3",
			expected16kHz: false,
		},
	}

	for _, tt := range testFiles {
		t.Run(tt.name, func(t *testing.T) {
			// Get absolute path
			absPath, err := filepath.Abs(tt.relativePath)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			// Check if file exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("test file %s does not exist", absPath)
			}

			// Check if it's 16kHz WAV
			is16kHz, err := Is16kHzWavFile(absPath)
			if err != nil {
				t.Errorf("Is16kHzWavFile failed: %v", err)
				return
			}

			if is16kHz != tt.expected16kHz {
				t.Errorf("expected is16kHz=%v, got %v", tt.expected16kHz, is16kHz)
			}

			t.Logf("File: %s, Is16kHz: %v", tt.name, is16kHz)
		})
	}
}

// TestConvertTo16kHzWavIntegration tests audio conversion with actual files
func TestConvertTo16kHzWavIntegration(t *testing.T) {
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping integration tests")
	}

	testFiles := []struct {
		name         string
		relativePath string
	}{
		{
			name:         "MP3 conversion",
			relativePath: "test/data/audio/short_sine_22khz.mp3",
		},
		{
			name:         "M4A conversion",
			relativePath: "test/data/audio/short_sine_48khz.m4a",
		},
	}

	for _, tt := range testFiles {
		t.Run(tt.name, func(t *testing.T) {
			// Get absolute path
			absPath, err := filepath.Abs(tt.relativePath)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			// Check if file exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("test file %s does not exist", absPath)
			}

			// Convert to 16kHz WAV
			outputPath, err := ConvertTo16kHzWav(absPath)
			if err != nil {
				t.Errorf("ConvertTo16kHzWav failed: %v", err)
				return
			}

			// Verify output file was created
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("output file %s was not created", outputPath)
				return
			}

			// Verify it's actually 16kHz WAV
			is16kHz, err := Is16kHzWavFile(outputPath)
			if err != nil {
				t.Errorf("failed to check converted file: %v", err)
			} else if !is16kHz {
				t.Errorf("converted file is not 16kHz WAV")
			}

			// Clean up
			defer func() {
				if err := os.Remove(outputPath); err != nil {
					t.Logf("failed to clean up %s: %v", outputPath, err)
				}
			}()

			t.Logf("Successfully converted %s to %s", tt.name, outputPath)
		})
	}
}

// TestConvertToMp3Integration tests MP3 conversion with actual files
func TestConvertToMp3Integration(t *testing.T) {
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping integration tests")
	}

	// Use a temporary file for testing
	tempDir := t.TempDir()
	testFileName := "test.mp4"
	testFilePath := filepath.Join(tempDir, testFileName)
	outputPath := filepath.Join(tempDir, "test.mp3")

	// Create a dummy MP4 file (this won't actually work for conversion)
	// In a real test, you'd use an actual video file
	if err := os.WriteFile(testFilePath, []byte("dummy mp4 content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test that the function handles missing files properly
	err := ConvertToMp3(testFileName, testFilePath, outputPath)

	// We expect this to fail since it's not a real MP4 file
	if err == nil {
		t.Error("expected error for dummy file, got nil")
	} else if !strings.Contains(err.Error(), "FFmpeg error") {
		t.Errorf("expected FFmpeg error, got: %v", err)
	}

	t.Logf("ConvertToMp3 correctly failed with dummy file: %v", err)
}

// TestPerformanceIntegration tests performance with actual files
func TestPerformanceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping integration tests")
	}

	testFile := "test/data/audio/medium_sine_16khz.wav"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skipf("test file %s does not exist", absPath)
	}

	// Test GetAudioDuration performance
	start := time.Now()
	_, err = GetAudioDuration(absPath)
	durDuration := time.Since(start)

	if err != nil {
		t.Errorf("GetAudioDuration failed: %v", err)
	} else {
		t.Logf("GetAudioDuration took: %v", durDuration)
	}

	// Test Is16kHzWavFile performance
	start = time.Now()
	_, err = Is16kHzWavFile(absPath)
	checkDuration := time.Since(start)

	if err != nil {
		t.Errorf("Is16kHzWavFile failed: %v", err)
	} else {
		t.Logf("Is16kHzWavFile took: %v", checkDuration)
	}

	// Performance should be reasonable (under 1 second for small files)
	if durDuration > time.Second {
		t.Errorf("GetAudioDuration took too long: %v", durDuration)
	}
	if checkDuration > time.Second {
		t.Errorf("Is16kHzWavFile took too long: %v", checkDuration)
	}
}

// TestErrorHandlingIntegration tests error handling with actual commands
func TestErrorHandlingIntegration(t *testing.T) {
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping integration tests")
	}

	// Test with non-existent file
	nonExistentFile := "/path/that/does/not/exist/audio.mp3"

	_, err := GetAudioDuration(nonExistentFile)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}

	_, err = Is16kHzWavFile(nonExistentFile)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}

	_, err = ConvertTo16kHzWav(nonExistentFile)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}

	// Test with unsupported format
	unsupportedFile := "/test/file.xyz"
	_, err = ConvertTo16kHzWav(unsupportedFile)
	if err == nil {
		t.Error("expected error for unsupported format, got nil")
	} else if !strings.Contains(err.Error(), "unsupported audio format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

// TestConcurrentOperationsIntegration tests concurrent operations
func TestConcurrentOperationsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping integration tests")
	}

	testFile := "test/data/audio/short_sine_16khz.wav"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skipf("test file %s does not exist", absPath)
	}

	// Run multiple operations concurrently
	const numGoroutines = 5
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Test GetAudioDuration
			if _, err := GetAudioDuration(absPath); err != nil {
				errors <- err
				return
			}

			// Test Is16kHzWavFile
			if _, err := Is16kHzWavFile(absPath); err != nil {
				errors <- err
				return
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

// isFFmpegAvailable checks if FFmpeg is available on the system
func isFFmpegAvailable() bool {
	// Try to run ffmpeg -version
	if _, err := os.Stat("/usr/bin/ffmpeg"); err == nil {
		return true
	}
	if _, err := os.Stat("/usr/local/bin/ffmpeg"); err == nil {
		return true
	}
	if _, err := os.Stat("/opt/homebrew/bin/ffmpeg"); err == nil {
		return true
	}

	// Check PATH
	paths := strings.Split(os.Getenv("PATH"), ":")
	for _, path := range paths {
		if _, err := os.Stat(filepath.Join(path, "ffmpeg")); err == nil {
			return true
		}
	}

	return false
}
