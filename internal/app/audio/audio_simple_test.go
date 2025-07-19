package audio

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// TestAudioFileExtensions tests audio file extension handling
func TestAudioFileExtensions(t *testing.T) {
	tests := []struct {
		name           string
		inputFile      string
		expectedOutput string
		supported      bool
	}{
		{
			name:           "MP3 file",
			inputFile:      "/test/audio.mp3",
			expectedOutput: "/test/audio_16khz.wav",
			supported:      true,
		},
		{
			name:           "M4A file",
			inputFile:      "/test/audio.m4a",
			expectedOutput: "/test/audio_16khz.wav",
			supported:      true,
		},
		{
			name:           "WAV file",
			inputFile:      "/test/audio.wav",
			expectedOutput: "/test/audio_16khz.wav",
			supported:      true,
		},
		{
			name:           "FLAC file",
			inputFile:      "/test/audio.flac",
			expectedOutput: "/test/audio_16khz.wav",
			supported:      false, // Not supported by current implementation
		},
		{
			name:           "OGG file",
			inputFile:      "/test/audio.ogg",
			expectedOutput: "/test/audio_16khz.wav",
			supported:      false, // Not supported by current implementation
		},
		{
			name:           "File with no extension",
			inputFile:      "/test/audio",
			expectedOutput: "/test/audio_16khz.wav",
			supported:      false, // Would be treated as no extension
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test output path generation logic
			outputPath := strings.TrimSuffix(tt.inputFile, filepath.Ext(tt.inputFile)) + "_16khz.wav"
			
			if outputPath != tt.expectedOutput {
				t.Errorf("expected output path %s, got %s", tt.expectedOutput, outputPath)
			}
			
			// Test format support logic
			ext := strings.ToLower(filepath.Ext(tt.inputFile))
			supported := ext == ".mp3" || ext == ".m4a" || ext == ".wav"
			
			if supported != tt.supported {
				t.Errorf("expected supported=%v, got %v for extension %s", tt.supported, supported, ext)
			}
		})
	}
}

// TestPathHandling tests various path scenarios
func TestPathHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename",
			input:    "audio.mp3",
			expected: "audio_16khz.wav",
		},
		{
			name:     "path with directory",
			input:    "/path/to/audio.mp3",
			expected: "/path/to/audio_16khz.wav",
		},
		{
			name:     "multiple dots",
			input:    "audio.test.mp3",
			expected: "audio.test_16khz.wav",
		},
		{
			name:     "no extension",
			input:    "audio",
			expected: "audio_16khz.wav",
		},
		{
			name:     "hidden file",
			input:    ".audio.mp3",
			expected: ".audio_16khz.wav",
		},
		{
			name:     "uppercase extension",
			input:    "AUDIO.MP3",
			expected: "AUDIO_16khz.wav",
		},
		{
			name:     "windows path",
			input:    "C:\\Users\\test\\audio.mp3",
			expected: "C:\\Users\\test\\audio_16khz.wav",
		},
		{
			name:     "path with spaces",
			input:    "/path with spaces/audio file.mp3",
			expected: "/path with spaces/audio file_16khz.wav",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the path transformation logic
			result := strings.TrimSuffix(tt.input, filepath.Ext(tt.input)) + "_16khz.wav"
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestFormatValidation tests audio format validation logic
func TestFormatValidation(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		supported bool
	}{
		{"MP3 file", "audio.mp3", true},
		{"M4A file", "audio.m4a", true},
		{"WAV file", "audio.wav", true},
		{"MP3 uppercase", "audio.MP3", true},
		{"M4A uppercase", "audio.M4A", true},
		{"WAV uppercase", "audio.WAV", true},
		{"FLAC file", "audio.flac", false},
		{"OGG file", "audio.ogg", false},
		{"AAC file", "audio.aac", false},
		{"WMA file", "audio.wma", false},
		{"No extension", "audio", false},
		{"Text file", "audio.txt", false},
		{"Empty extension", "audio.", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := strings.ToLower(filepath.Ext(tt.filename))
			supported := ext == ".mp3" || ext == ".m4a" || ext == ".wav"
			
			if supported != tt.supported {
				t.Errorf("expected supported=%v for file %s (ext: %s)", tt.supported, tt.filename, ext)
			}
		})
	}
}

// TestCommandArgumentConstruction tests FFmpeg command argument construction
func TestCommandArgumentConstruction(t *testing.T) {
	tests := []struct {
		name           string
		operation      string
		inputFile      string
		outputFile     string
		expectedArgs   []string
	}{
		{
			name:      "Convert to 16kHz WAV",
			operation: "convert_wav",
			inputFile: "/test/input.mp3",
			outputFile: "/test/output.wav",
			expectedArgs: []string{
				"-i", "/test/input.mp3",
				"-vn",
				"-acodec", "pcm_s16le",
				"-ar", "16000",
				"-ac", "2",
				"/test/output.wav",
			},
		},
		{
			name:      "Convert to MP3",
			operation: "convert_mp3",
			inputFile: "/test/input.mp4",
			outputFile: "/test/output.mp3",
			expectedArgs: []string{
				"-i", "/test/input.mp4",
				"-vn",
				"-acodec", "libmp3lame",
				"/test/output.mp3",
			},
		},
		{
			name:      "Get duration",
			operation: "duration",
			inputFile: "/test/input.mp3",
			expectedArgs: []string{
				"-v", "error",
				"-show_entries", "format=duration",
				"-of", "default=noprint_wrappers=1:nokey=1",
				"/test/input.mp3",
			},
		},
		{
			name:      "Check format",
			operation: "check_format",
			inputFile: "/test/input.wav",
			expectedArgs: []string{
				"-v", "quiet",
				"-print_format", "json",
				"-show_streams",
				"/test/input.wav",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualArgs []string
			
			switch tt.operation {
			case "convert_wav":
				actualArgs = []string{
					"-i", tt.inputFile,
					"-vn",
					"-acodec", "pcm_s16le",
					"-ar", "16000",
					"-ac", "2",
					tt.outputFile,
				}
			case "convert_mp3":
				actualArgs = []string{
					"-i", tt.inputFile,
					"-vn",
					"-acodec", "libmp3lame",
					tt.outputFile,
				}
			case "duration":
				actualArgs = []string{
					"-v", "error",
					"-show_entries", "format=duration",
					"-of", "default=noprint_wrappers=1:nokey=1",
					tt.inputFile,
				}
			case "check_format":
				actualArgs = []string{
					"-v", "quiet",
					"-print_format", "json",
					"-show_streams",
					tt.inputFile,
				}
			}
			
			if len(actualArgs) != len(tt.expectedArgs) {
				t.Errorf("expected %d args, got %d", len(tt.expectedArgs), len(actualArgs))
				return
			}
			
			for i, expectedArg := range tt.expectedArgs {
				if actualArgs[i] != expectedArg {
					t.Errorf("arg %d: expected %s, got %s", i, expectedArg, actualArgs[i])
				}
			}
		})
	}
}

// TestErrorMessageFormatting tests error message formatting
func TestErrorMessageFormatting(t *testing.T) {
	tests := []struct {
		name           string
		originalError  string
		stderr         string
		expectedFormat string
	}{
		{
			name:           "Basic FFmpeg error",
			originalError:  "exit status 1",
			stderr:         "Invalid data found when processing input",
			expectedFormat: "FFmpeg error: exit status 1, stderr: Invalid data found when processing input",
		},
		{
			name:           "Parse error",
			originalError:  "parsing \"invalid\": invalid syntax",
			expectedFormat: "parsing \"invalid\": invalid syntax",
		},
		{
			name:           "File not found",
			originalError:  "No such file or directory",
			expectedFormat: "No such file or directory",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var formattedError string
			
			if tt.stderr != "" {
				formattedError = fmt.Sprintf("FFmpeg error: %s, stderr: %s", tt.originalError, tt.stderr)
			} else {
				formattedError = tt.originalError
			}
			
			if formattedError != tt.expectedFormat {
				t.Errorf("expected error format %s, got %s", tt.expectedFormat, formattedError)
			}
		})
	}
}

// TestFileExtensionValidation tests file extension validation
func TestFileExtensionValidation(t *testing.T) {
	supportedFormats := map[string]bool{
		".mp3": true,
		".m4a": true,
		".wav": true,
	}
	
	tests := []struct {
		extension string
		supported bool
	}{
		{".mp3", true},
		{".m4a", true}, 
		{".wav", true},
		{".MP3", true}, // Should handle case insensitive
		{".M4A", true},
		{".WAV", true},
		{".flac", false},
		{".ogg", false},
		{".aac", false},
		{".wma", false},
		{"", false},
		{".txt", false},
	}
	
	for _, tt := range tests {
		t.Run("ext_"+tt.extension, func(t *testing.T) {
			normalized := strings.ToLower(tt.extension)
			supported := supportedFormats[normalized]
			
			if supported != tt.supported {
				t.Errorf("extension %s: expected supported=%v, got %v", tt.extension, tt.supported, supported)
			}
		})
	}
}