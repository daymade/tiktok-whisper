package testutil

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tiktok-whisper/internal/app/model"
)

// TestTranscriptions provides sample transcription data for testing
var TestTranscriptions = []model.Transcription{
	{
		ID:                 1,
		User:               "test_user_1",
		LastConversionTime: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Mp3FileName:        "podcast_episode_001.mp3",
		AudioDuration:      1800.5, // 30 minutes
		Transcription:      "Welcome to our podcast. Today we're discussing the latest developments in artificial intelligence and machine learning. Our guest is Dr. Sarah Johnson, a leading researcher in the field of neural networks.",
		ErrorMessage:       "",
	},
	{
		ID:                 2,
		User:               "test_user_1",
		LastConversionTime: time.Date(2024, 1, 16, 14, 45, 0, 0, time.UTC),
		Mp3FileName:        "podcast_episode_002.mp3",
		AudioDuration:      2100.3, // 35 minutes
		Transcription:      "In this episode, we explore the impact of automation on modern businesses. We'll discuss how companies are adapting to technological changes and what this means for the future of work.",
		ErrorMessage:       "",
	},
	{
		ID:                 3,
		User:               "test_user_2",
		LastConversionTime: time.Date(2024, 1, 17, 9, 15, 0, 0, time.UTC),
		Mp3FileName:        "interview_ceo_tech.mp3",
		AudioDuration:      3600.0, // 1 hour
		Transcription:      "Today we have an exclusive interview with the CEO of TechCorp. We'll be discussing their latest product launches, market strategy, and vision for the next decade in technology innovation.",
		ErrorMessage:       "",
	},
	{
		ID:                 4,
		User:               "test_user_2",
		LastConversionTime: time.Date(2024, 1, 18, 16, 20, 0, 0, time.UTC),
		Mp3FileName:        "corrupted_audio.mp3",
		AudioDuration:      0,
		Transcription:      "",
		ErrorMessage:       "Failed to process audio file: corrupted format",
	},
	{
		ID:                 5,
		User:               "test_user_3",
		LastConversionTime: time.Date(2024, 1, 19, 11, 0, 0, 0, time.UTC),
		Mp3FileName:        "short_clip.mp3",
		AudioDuration:      45.2, // 45 seconds
		Transcription:      "This is a short audio clip used for testing purposes. It contains basic speech patterns and common vocabulary.",
		ErrorMessage:       "",
	},
}

// TestFileInfos provides sample file information for testing
var TestFileInfos = []model.FileInfo{
	{
		FullPath: "/test/data/audio/podcast_episode_001.mp3",
		ModTime:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Name:     "podcast_episode_001.mp3",
	},
	{
		FullPath: "/test/data/audio/podcast_episode_002.mp3",
		ModTime:  time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC),
		Name:     "podcast_episode_002.mp3",
	},
	{
		FullPath: "/test/data/audio/interview_ceo_tech.mp3",
		ModTime:  time.Date(2024, 1, 17, 9, 0, 0, 0, time.UTC),
		Name:     "interview_ceo_tech.mp3",
	},
	{
		FullPath: "/test/data/audio/corrupted_audio.mp3",
		ModTime:  time.Date(2024, 1, 18, 16, 0, 0, 0, time.UTC),
		Name:     "corrupted_audio.mp3",
	},
	{
		FullPath: "/test/data/audio/short_clip.mp3",
		ModTime:  time.Date(2024, 1, 19, 11, 0, 0, 0, time.UTC),
		Name:     "short_clip.mp3",
	},
}

// TestUsers provides sample user data for testing
var TestUsers = []string{
	"test_user_1",
	"test_user_2",
	"test_user_3",
	"podcast_host",
	"interview_channel",
}

// TestAudioFiles provides sample audio file paths for testing
var TestAudioFiles = []string{
	"/test/data/audio/podcast_episode_001.mp3",
	"/test/data/audio/podcast_episode_002.mp3",
	"/test/data/audio/interview_ceo_tech.mp3",
	"/test/data/audio/short_clip.mp3",
	"/test/data/audio/long_presentation.mp3",
}

// TestTranscriptionTexts provides sample transcription texts for testing
var TestTranscriptionTexts = []string{
	"Welcome to our podcast. Today we're discussing artificial intelligence.",
	"In this episode, we explore the impact of automation on modern businesses.",
	"Today we have an exclusive interview with the CEO of TechCorp.",
	"This is a short audio clip used for testing purposes.",
	"The presentation covers the fundamentals of machine learning algorithms.",
	"Our guest today is a renowned expert in data science and analytics.",
	"We'll be discussing the latest trends in software development.",
	"This recording contains important information about project management.",
}

// TestErrorMessages provides sample error messages for testing
var TestErrorMessages = []string{
	"Failed to process audio file: corrupted format",
	"Audio file not found at specified path",
	"Insufficient permissions to read audio file",
	"Audio format not supported",
	"Network timeout while processing remote file",
	"Database connection failed during transcription save",
	"Audio file too large for processing",
	"Invalid audio encoding detected",
}

// Configuration fixtures for testing different scenarios

// TestConfigLocal provides a test configuration for local whisper.cpp
var TestConfigLocal = struct {
	BinaryPath string
	ModelPath  string
	Language   string
}{
	BinaryPath: "/usr/local/bin/whisper",
	ModelPath:  "/usr/local/share/whisper/models/ggml-large-v2.bin",
	Language:   "en",
}

// TestConfigRemote provides a test configuration for remote OpenAI API
var TestConfigRemote = struct {
	APIKey      string
	Model       string
	Temperature float32
	Language    string
}{
	APIKey:      "sk-test-api-key-for-testing-purposes",
	Model:       "whisper-1",
	Temperature: 0.0,
	Language:    "en",
}

// TestConfigDatabase provides database configurations for testing
var TestConfigDatabase = struct {
	SQLite   string
	Postgres string
}{
	SQLite:   ":memory:",
	Postgres: "postgres://test:test@localhost/test_db?sslmode=disable",
}

// MockAPIResponses provides sample API responses for testing
var MockAPIResponses = map[string]interface{}{
	"whisper_success": map[string]interface{}{
		"text": "This is a successful transcription result from the API.",
	},
	"whisper_error": map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Invalid file format",
			"type":    "invalid_request_error",
			"code":    "invalid_file_format",
		},
	},
	"whisper_with_segments": map[string]interface{}{
		"text": "This is a transcription with segments.",
		"segments": []map[string]interface{}{
			{
				"id":                0,
				"seek":              0,
				"start":             0.0,
				"end":               5.0,
				"text":              "This is a transcription",
				"tokens":            []int{1, 2, 3, 4, 5},
				"temperature":       0.0,
				"avg_logprob":       -0.5,
				"compression_ratio": 1.2,
				"no_speech_prob":    0.1,
			},
			{
				"id":                1,
				"seek":              500,
				"start":             5.0,
				"end":               10.0,
				"text":              " with segments.",
				"tokens":            []int{6, 7, 8},
				"temperature":       0.0,
				"avg_logprob":       -0.3,
				"compression_ratio": 1.1,
				"no_speech_prob":    0.05,
			},
		},
	},
}

// TestFilePaths provides commonly used file paths for testing
var TestFilePaths = struct {
	AudioFiles   []string
	OutputDir    string
	ConfigFile   string
	DatabaseFile string
	LogFile      string
	TempDir      string
}{
	AudioFiles: []string{
		"/test/data/audio/sample1.mp3",
		"/test/data/audio/sample2.wav",
		"/test/data/audio/sample3.m4a",
		"/test/data/audio/sample4.mp4",
	},
	OutputDir:    "/test/output",
	ConfigFile:   "/test/config/app.json",
	DatabaseFile: "/test/data/transcriptions.db",
	LogFile:      "/test/logs/app.log",
	TempDir:      "/test/temp",
}

// TestScenarios provides different test scenarios with their expected outcomes
var TestScenarios = []struct {
	Name        string
	Description string
	Input       string
	Expected    string
	ShouldError bool
}{
	{
		Name:        "successful_transcription",
		Description: "Test successful transcription of a valid audio file",
		Input:       "/test/data/audio/valid_sample.mp3",
		Expected:    "This is a successful transcription.",
		ShouldError: false,
	},
	{
		Name:        "file_not_found",
		Description: "Test handling of non-existent audio file",
		Input:       "/test/data/audio/nonexistent.mp3",
		Expected:    "",
		ShouldError: true,
	},
	{
		Name:        "corrupted_file",
		Description: "Test handling of corrupted audio file",
		Input:       "/test/data/audio/corrupted.mp3",
		Expected:    "",
		ShouldError: true,
	},
	{
		Name:        "empty_file",
		Description: "Test handling of empty audio file",
		Input:       "/test/data/audio/empty.mp3",
		Expected:    "",
		ShouldError: true,
	},
	{
		Name:        "long_audio",
		Description: "Test transcription of long audio file",
		Input:       "/test/data/audio/long_presentation.mp3",
		Expected:    "This is a very long transcription that spans multiple minutes...",
		ShouldError: false,
	},
}

// GetTestTranscriptionByID returns a test transcription by ID
func GetTestTranscriptionByID(id int) (model.Transcription, bool) {
	for _, t := range TestTranscriptions {
		if t.ID == id {
			return t, true
		}
	}
	return model.Transcription{}, false
}

// GetTestTranscriptionsByUser returns test transcriptions for a specific user
func GetTestTranscriptionsByUser(user string) []model.Transcription {
	var result []model.Transcription
	for _, t := range TestTranscriptions {
		if t.User == user {
			result = append(result, t)
		}
	}
	return result
}

// GetTestFileInfoByName returns test file info by name
func GetTestFileInfoByName(name string) (model.FileInfo, bool) {
	for _, f := range TestFileInfos {
		if f.Name == name {
			return f, true
		}
	}
	return model.FileInfo{}, false
}

// GetMockAPIResponse returns a mock API response by key
func GetMockAPIResponse(key string) (interface{}, bool) {
	response, exists := MockAPIResponses[key]
	return response, exists
}

// GetMockAPIResponseJSON returns a mock API response as JSON bytes
func GetMockAPIResponseJSON(key string) ([]byte, error) {
	response, exists := MockAPIResponses[key]
	if !exists {
		return nil, errors.New("mock response not found")
	}
	return json.Marshal(response)
}

// GenerateTestTranscription creates a test transcription with custom parameters
func GenerateTestTranscription(id int, user string, fileName string, duration float64, text string) model.Transcription {
	return model.Transcription{
		ID:                 id,
		User:               user,
		LastConversionTime: time.Now(),
		Mp3FileName:        fileName,
		AudioDuration:      duration,
		Transcription:      text,
		ErrorMessage:       "",
	}
}

// GenerateTestFileInfo creates a test file info with custom parameters
func GenerateTestFileInfo(fullPath string, name string, modTime time.Time) model.FileInfo {
	return model.FileInfo{
		FullPath: fullPath,
		ModTime:  modTime,
		Name:     name,
	}
}

// RandomTestTranscription returns a random test transcription
func RandomTestTranscription() model.Transcription {
	if len(TestTranscriptions) == 0 {
		return model.Transcription{}
	}
	return TestTranscriptions[time.Now().UnixNano()%int64(len(TestTranscriptions))]
}

// RandomTestUser returns a random test user
func RandomTestUser() string {
	if len(TestUsers) == 0 {
		return "default_user"
	}
	return TestUsers[time.Now().UnixNano()%int64(len(TestUsers))]
}

// RandomTestAudioFile returns a random test audio file path
func RandomTestAudioFile() string {
	if len(TestAudioFiles) == 0 {
		return "/test/default.mp3"
	}
	return TestAudioFiles[time.Now().UnixNano()%int64(len(TestAudioFiles))]
}

// RandomTestTranscriptionText returns a random test transcription text
func RandomTestTranscriptionText() string {
	if len(TestTranscriptionTexts) == 0 {
		return "Default test transcription text."
	}
	return TestTranscriptionTexts[time.Now().UnixNano()%int64(len(TestTranscriptionTexts))]
}

// RandomTestErrorMessage returns a random test error message
func RandomTestErrorMessage() string {
	if len(TestErrorMessages) == 0 {
		return "Default test error message"
	}
	return TestErrorMessages[time.Now().UnixNano()%int64(len(TestErrorMessages))]
}

// CreateTestAudioFile creates a minimal valid WAV file for testing
func CreateTestAudioFile(t *testing.T, filename string) string {
	t.Helper()
	
	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}
	
	// Create absolute path in temp directory
	tempDir := t.TempDir()
	fullPath := filepath.Join(tempDir, filepath.Base(filename))
	
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
	err := os.WriteFile(fullPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}
	
	return fullPath
}

// CleanupFile removes a test file
func CleanupFile(t *testing.T, filepath string) {
	t.Helper()
	if filepath != "" {
		os.Remove(filepath)
	}
}

// CreateCorruptedAudioFile creates a file with invalid audio data for testing
func CreateCorruptedAudioFile(t *testing.T, filename string) string {
	t.Helper()
	
	tempDir := t.TempDir()
	fullPath := filepath.Join(tempDir, filepath.Base(filename))
	
	// Create a file with invalid WAV header
	corruptedData := []byte("This is not a valid audio file!")
	err := os.WriteFile(fullPath, corruptedData, 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}
	
	return fullPath
}

// CreateEmptyFile creates an empty file for testing
func CreateEmptyFile(t *testing.T, filename string) string {
	t.Helper()
	
	tempDir := t.TempDir()
	fullPath := filepath.Join(tempDir, filepath.Base(filename))
	
	err := os.WriteFile(fullPath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}
	
	return fullPath
}
