package api_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/api/openai/whisper"
	"tiktok-whisper/internal/app/api/whisper_cpp"
	"tiktok-whisper/internal/app/testutil"
)

// TranscriberTestSuite provides common test cases for all Transcriber implementations
type TranscriberTestSuite struct {
	transcriber api.Transcriber
	name        string
}

// TestTranscriberInterface verifies that implementations satisfy the interface
func TestTranscriberInterface(t *testing.T) {
	// Compile-time check that implementations satisfy the interface
	var _ api.Transcriber = (*whisper.RemoteTranscriber)(nil)
	var _ api.Transcriber = (*whisper_cpp.LocalTranscriber)(nil)
	var _ api.Transcriber = (*testutil.MockTranscriber)(nil)
}

// TestAllTranscribers runs the test suite on all available transcriber implementations
func TestAllTranscribers(t *testing.T) {
	// Create test audio files
	testDir := t.TempDir()
	validAudioFile := createTestAudioFile(t, filepath.Join(testDir, "valid.wav"))
	emptyAudioFile := createEmptyFile(t, filepath.Join(testDir, "empty.wav"))
	largeAudioFile := createLargeAudioFile(t, filepath.Join(testDir, "large.wav"))

	// Define test implementations
	testCases := []struct {
		name        string
		transcriber api.Transcriber
		skipReason  string
	}{
		{
			name:        "MockTranscriber",
			transcriber: createMockTranscriber(),
		},
		{
			name:        "LocalTranscriber",
			transcriber: createLocalTranscriberIfAvailable(t),
			skipReason:  "whisper.cpp not available",
		},
		{
			name:        "RemoteTranscriber",
			transcriber: createRemoteTranscriberIfConfigured(t),
			skipReason:  "OpenAI API key not configured",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.transcriber == nil {
				t.Skip(tc.skipReason)
			}

			suite := &TranscriberTestSuite{
				transcriber: tc.transcriber,
				name:        tc.name,
			}

			// Run common tests
			t.Run("ValidAudioFile", func(t *testing.T) {
				suite.testValidAudioFile(t, validAudioFile)
			})

			t.Run("NonExistentFile", func(t *testing.T) {
				suite.testNonExistentFile(t)
			})

			t.Run("EmptyFile", func(t *testing.T) {
				suite.testEmptyFile(t, emptyAudioFile)
			})

			t.Run("LargeFile", func(t *testing.T) {
				if tc.name == "RemoteTranscriber" {
					t.Skip("Skipping large file test for remote API to save costs")
				}
				suite.testLargeFile(t, largeAudioFile)
			})

			t.Run("ConcurrentRequests", func(t *testing.T) {
				suite.testConcurrentRequests(t, validAudioFile)
			})

			t.Run("SpecialCharactersPath", func(t *testing.T) {
				suite.testSpecialCharactersPath(t, testDir)
			})
		})
	}
}

// Test methods for TranscriberTestSuite

func (s *TranscriberTestSuite) testValidAudioFile(t *testing.T, audioFile string) {
	result, err := s.transcriber.Transcript(audioFile)
	if err != nil {
		t.Errorf("%s: Failed to transcribe valid audio file: %v", s.name, err)
	}
	if result == "" {
		t.Errorf("%s: Expected non-empty transcription result", s.name)
	}
	t.Logf("%s: Transcription result: %s", s.name, result)
}

func (s *TranscriberTestSuite) testNonExistentFile(t *testing.T) {
	_, err := s.transcriber.Transcript("/non/existent/file.mp3")
	if err == nil {
		t.Errorf("%s: Expected error for non-existent file, got none", s.name)
	}
}

func (s *TranscriberTestSuite) testEmptyFile(t *testing.T, emptyFile string) {
	result, err := s.transcriber.Transcript(emptyFile)
	// Empty files might either error or return empty/minimal transcription
	if err != nil {
		t.Logf("%s: Empty file produced error (acceptable): %v", s.name, err)
	} else {
		t.Logf("%s: Empty file transcription: '%s'", s.name, result)
	}
}

func (s *TranscriberTestSuite) testLargeFile(t *testing.T, largeFile string) {
	start := time.Now()
	result, err := s.transcriber.Transcript(largeFile)
	duration := time.Since(start)
	
	if err != nil {
		t.Errorf("%s: Failed to transcribe large file: %v", s.name, err)
	}
	
	t.Logf("%s: Large file transcription took %v", s.name, duration)
	t.Logf("%s: Result length: %d characters", s.name, len(result))
}

func (s *TranscriberTestSuite) testConcurrentRequests(t *testing.T, audioFile string) {
	numRequests := 3
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)
	results := make(chan string, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := s.transcriber.Transcript(audioFile)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(results)

	errorCount := len(errors)
	resultCount := len(results)

	if errorCount > 0 {
		t.Logf("%s: %d/%d concurrent requests failed", s.name, errorCount, numRequests)
	}
	if resultCount != numRequests-errorCount {
		t.Errorf("%s: Result count mismatch", s.name)
	}
}

func (s *TranscriberTestSuite) testSpecialCharactersPath(t *testing.T, testDir string) {
	specialPath := filepath.Join(testDir, "audio with spaces & special.wav")
	createTestAudioFile(t, specialPath)
	
	result, err := s.transcriber.Transcript(specialPath)
	if err != nil {
		t.Errorf("%s: Failed with special characters in path: %v", s.name, err)
	}
	if result == "" {
		t.Errorf("%s: Expected non-empty result for special path", s.name)
	}
}

// TestTranscriberComparison compares outputs from different transcriber implementations
func TestTranscriberComparison(t *testing.T) {
	// Skip if not all implementations are available
	localTranscriber := createLocalTranscriberIfAvailable(t)
	remoteTranscriber := createRemoteTranscriberIfConfigured(t)
	
	if localTranscriber == nil || remoteTranscriber == nil {
		t.Skip("Both local and remote transcribers must be available for comparison")
	}

	testFile := createTestAudioFile(t, filepath.Join(t.TempDir(), "comparison.wav"))

	// Transcribe with both implementations
	localResult, localErr := localTranscriber.Transcript(testFile)
	remoteResult, remoteErr := remoteTranscriber.Transcript(testFile)

	if localErr != nil {
		t.Errorf("Local transcription failed: %v", localErr)
	}
	if remoteErr != nil {
		t.Errorf("Remote transcription failed: %v", remoteErr)
	}

	// Compare results (they won't be identical but should be similar)
	t.Logf("Local result: %s", localResult)
	t.Logf("Remote result: %s", remoteResult)

	// Calculate similarity (simple word overlap)
	localWords := strings.Fields(strings.ToLower(localResult))
	remoteWords := strings.Fields(strings.ToLower(remoteResult))
	
	wordSet := make(map[string]bool)
	for _, word := range localWords {
		wordSet[word] = true
	}
	
	overlap := 0
	for _, word := range remoteWords {
		if wordSet[word] {
			overlap++
		}
	}
	
	similarity := float64(overlap) / float64(len(remoteWords))
	t.Logf("Word overlap similarity: %.2f%%", similarity*100)
}

// TestTranscriberPerformance benchmarks different implementations
func TestTranscriberPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testFile := createTestAudioFile(t, filepath.Join(t.TempDir(), "performance.wav"))
	
	implementations := []struct {
		name        string
		transcriber api.Transcriber
	}{
		{"Mock", createMockTranscriber()},
		{"Local", createLocalTranscriberIfAvailable(t)},
		{"Remote", createRemoteTranscriberIfConfigured(t)},
	}

	for _, impl := range implementations {
		if impl.transcriber == nil {
			continue
		}

		t.Run(impl.name, func(t *testing.T) {
			iterations := 5
			var totalDuration time.Duration

			for i := 0; i < iterations; i++ {
				start := time.Now()
				_, err := impl.transcriber.Transcript(testFile)
				duration := time.Since(start)
				
				if err != nil {
					t.Errorf("Transcription failed: %v", err)
					continue
				}
				
				totalDuration += duration
				t.Logf("Iteration %d: %v", i+1, duration)
			}

			avgDuration := totalDuration / time.Duration(iterations)
			t.Logf("Average duration: %v", avgDuration)
		})
	}
}

// TestTranscriberErrorHandling tests error scenarios across implementations
func TestTranscriberErrorHandling(t *testing.T) {
	testCases := []struct {
		name          string
		setupFunc     func(t *testing.T) (api.Transcriber, string)
		expectError   bool
		errorContains string
	}{
		{
			name: "CorruptedAudioFile",
			setupFunc: func(t *testing.T) (api.Transcriber, string) {
				corruptFile := createCorruptedAudioFile(t)
				return createMockTranscriber().WithError(corruptFile, errors.New("corrupted audio")), corruptFile
			},
			expectError:   true,
			errorContains: "corrupted",
		},
		{
			name: "UnsupportedFormat",
			setupFunc: func(t *testing.T) (api.Transcriber, string) {
				unsupportedFile := filepath.Join(t.TempDir(), "audio.xyz")
				os.WriteFile(unsupportedFile, []byte("not audio"), 0644)
				return createMockTranscriber().WithError(unsupportedFile, errors.New("unsupported format")), unsupportedFile
			},
			expectError:   true,
			errorContains: "unsupported",
		},
		{
			name: "PermissionDenied",
			setupFunc: func(t *testing.T) (api.Transcriber, string) {
				restrictedFile := filepath.Join(t.TempDir(), "restricted.wav")
				createTestAudioFile(t, restrictedFile)
				os.Chmod(restrictedFile, 0000) // Remove all permissions
				return createMockTranscriber(), restrictedFile
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transcriber, testFile := tc.setupFunc(t)
			defer os.RemoveAll(filepath.Dir(testFile))

			_, err := transcriber.Transcript(testFile)
			
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tc.errorContains != "" && err != nil && !strings.Contains(err.Error(), tc.errorContains) {
				t.Errorf("Expected error containing '%s', got '%v'", tc.errorContains, err)
			}
		})
	}
}

// Helper functions

func createMockTranscriber() *testutil.MockTranscriber {
	mock := testutil.NewMockTranscriber()
	
	// Set up custom behavior that checks file existence
	mock.TranscriptFunc = func(inputFilePath string) (string, error) {
		// Check if file exists
		if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", inputFilePath)
		}
		
		// Return default response for existing files
		return "This is a mock transcription of the audio file.", nil
	}
	
	return mock
}

func createLocalTranscriberIfAvailable(t *testing.T) api.Transcriber {
	// Check if whisper.cpp is available
	binaryPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/main"
	modelPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"
	
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil
	}
	
	return whisper_cpp.NewLocalTranscriber(binaryPath, modelPath)
}

func createRemoteTranscriberIfConfigured(t *testing.T) api.Transcriber {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil
	}
	
	client := openai.NewClient(apiKey)
	return whisper.NewRemoteTranscriber(client)
}

func createTestAudioFile(t *testing.T, path string) string {
	t.Helper()
	
	// Create a minimal valid WAV file
	wavHeader := []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x24, 0x08, 0x00, 0x00, // File size (2084 bytes)
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Chunk size
		0x01, 0x00,             // Audio format (PCM)
		0x01, 0x00,             // Channels (mono)
		0x80, 0x3E, 0x00, 0x00, // Sample rate (16000)
		0x00, 0x7D, 0x00, 0x00, // Byte rate
		0x02, 0x00,             // Block align
		0x10, 0x00,             // Bits per sample
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x08, 0x00, 0x00, // Data size (2048 bytes)
	}
	
	// Add some audio data (silence)
	audioData := make([]byte, 2048)
	
	data := append(wavHeader, audioData...)
	err := os.WriteFile(path, data, 0644)
	if err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}
	
	return path
}

func createEmptyFile(t *testing.T, path string) string {
	t.Helper()
	err := os.WriteFile(path, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}
	return path
}

func createLargeAudioFile(t *testing.T, path string) string {
	t.Helper()
	
	// Create a WAV file with 30 seconds of audio
	sampleRate := 16000
	duration := 30 // seconds
	dataSize := sampleRate * 2 * duration // 16-bit mono
	
	wavHeader := []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		byte(dataSize + 36), byte((dataSize + 36) >> 8), byte((dataSize + 36) >> 16), byte((dataSize + 36) >> 24), // File size
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Chunk size
		0x01, 0x00,             // Audio format (PCM)
		0x01, 0x00,             // Channels (mono)
		0x80, 0x3E, 0x00, 0x00, // Sample rate (16000)
		0x00, 0x7D, 0x00, 0x00, // Byte rate
		0x02, 0x00,             // Block align
		0x10, 0x00,             // Bits per sample
		0x64, 0x61, 0x74, 0x61, // "data"
		byte(dataSize), byte(dataSize >> 8), byte(dataSize >> 16), byte(dataSize >> 24), // Data size
	}
	
	// Generate simple sine wave audio
	audioData := make([]byte, dataSize)
	for i := 0; i < len(audioData); i += 2 {
		// Simple pattern instead of actual sine wave for testing
		sample := int16(i % 1000)
		audioData[i] = byte(sample)
		audioData[i+1] = byte(sample >> 8)
	}
	
	data := append(wavHeader, audioData...)
	err := os.WriteFile(path, data, 0644)
	if err != nil {
		t.Fatalf("Failed to create large audio file: %v", err)
	}
	
	return path
}

func createCorruptedAudioFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "corrupted.wav")
	
	// Create a file with invalid WAV header
	corruptedData := []byte("This is not a valid audio file!")
	err := os.WriteFile(path, corruptedData, 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}
	
	return path
}

// Benchmark functions

func BenchmarkTranscribers(b *testing.B) {
	testFile := createTestAudioFileForBenchmark(b, filepath.Join(b.TempDir(), "benchmark.wav"))
	
	implementations := []struct {
		name        string
		transcriber api.Transcriber
	}{
		{"Mock", createMockTranscriber()},
		{"Local", createLocalTranscriberForBenchmark(b)},
		{"Remote", createRemoteTranscriberForBenchmark(b)},
	}

	for _, impl := range implementations {
		if impl.transcriber == nil {
			continue
		}

		b.Run(impl.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := impl.transcriber.Transcript(testFile)
				if err != nil {
					b.Fatalf("Transcription failed: %v", err)
				}
			}
		})
	}
}

// createLocalTranscriberForBenchmark for benchmarks
func createLocalTranscriberForBenchmark(b *testing.B) api.Transcriber {
	binaryPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/main"
	modelPath := "/Users/tiansheng/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"
	
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil
	}
	
	return whisper_cpp.NewLocalTranscriber(binaryPath, modelPath)
}

// createRemoteTranscriberForBenchmark for benchmarks
func createRemoteTranscriberForBenchmark(b *testing.B) api.Transcriber {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil
	}
	
	client := openai.NewClient(apiKey)
	return whisper.NewRemoteTranscriber(client)
}

// createTestAudioFileForBenchmark for benchmarks
func createTestAudioFileForBenchmark(b *testing.B, path string) string {
	b.Helper()
	
	// Same implementation as in tests
	wavHeader := []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x24, 0x08, 0x00, 0x00, // File size
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Chunk size
		0x01, 0x00,             // Audio format (PCM)
		0x01, 0x00,             // Channels (mono)
		0x80, 0x3E, 0x00, 0x00, // Sample rate (16000)
		0x00, 0x7D, 0x00, 0x00, // Byte rate
		0x02, 0x00,             // Block align
		0x10, 0x00,             // Bits per sample
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x08, 0x00, 0x00, // Data size
	}
	
	audioData := make([]byte, 2048)
	data := append(wavHeader, audioData...)
	
	err := os.WriteFile(path, data, 0644)
	if err != nil {
		b.Fatalf("Failed to create test audio file: %v", err)
	}
	
	return path
}