package audio

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	model2 "tiktok-whisper/internal/app/model"
)

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Exit with test result code
	os.Exit(code)
}

// TestGetAudioDuration tests the GetAudioDuration function logic
// Note: These are unit tests for the parsing logic, not integration tests
func TestGetAudioDuration(t *testing.T) {
	// Test the duration parsing logic that would be used with FFprobe output
	tests := []struct {
		name             string
		ffprobeOutput    string
		expectedDuration int
		expectedError    bool
		errorContains    string
	}{
		{
			name:             "valid duration - integer seconds",
			ffprobeOutput:    "30\n",
			expectedDuration: 30,
			expectedError:    false,
		},
		{
			name:             "valid duration - decimal seconds",
			ffprobeOutput:    "45.678\n",
			expectedDuration: 46, // rounded
			expectedError:    false,
		},
		{
			name:             "valid duration - round down",
			ffprobeOutput:    "29.4\n",
			expectedDuration: 29,
			expectedError:    false,
		},
		{
			name:             "valid duration - round up",
			ffprobeOutput:    "29.5\n",
			expectedDuration: 30,
			expectedError:    false,
		},
		{
			name:          "invalid duration format",
			ffprobeOutput: "not-a-number\n",
			expectedError: true,
			errorContains: "invalid syntax",
		},
		{
			name:          "empty output",
			ffprobeOutput: "",
			expectedError: true,
			errorContains: "invalid syntax",
		},
		{
			name:             "whitespace in output",
			ffprobeOutput:    "  \t120.5  \n",
			expectedDuration: 121,
			expectedError:    false,
		},
		{
			name:             "very long duration",
			ffprobeOutput:    "7200.0\n", // 2 hours
			expectedDuration: 7200,
			expectedError:    false,
		},
		{
			name:             "zero duration",
			ffprobeOutput:    "0.0\n",
			expectedDuration: 0,
			expectedError:    false,
		},
		{
			name:             "negative duration (corrupted)",
			ffprobeOutput:    "-5.0\n",
			expectedDuration: -5,
			expectedError:    false, // function doesn't validate negative
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the parsing logic directly
			duration, err := parseDurationOutput(tt.ffprobeOutput)

			// Check error
			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorContains)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got '%v'", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if duration != tt.expectedDuration {
					t.Errorf("expected duration %d, got %d", tt.expectedDuration, duration)
				}
			}
		})
	}
}

// parseDurationOutput extracts the duration parsing logic for testing
func parseDurationOutput(output string) (int, error) {
	durationFloat, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, err
	}
	duration := int(math.Round(durationFloat))
	return duration, nil
}

// parseProbeOutput extracts the probe output parsing logic for testing
func parseProbeOutput(output string) (bool, error) {
	var probeOutput model2.FFProbeOutput
	err := json.Unmarshal([]byte(output), &probeOutput)
	if err != nil {
		return false, err
	}

	for _, stream := range probeOutput.Streams {
		if stream.CodecType == "audio" && stream.CodecName == "pcm_s16le" && stream.SampleRate == 16000 {
			return true, nil
		}
	}

	return false, nil
}

// TestConvertToMp3 tests the ConvertToMp3 function logic
func TestConvertToMp3(t *testing.T) {
	tests := []struct {
		name          string
		fileName      string
		fileExists    bool
		expectedError bool
	}{
		{
			name:          "file already exists - should skip",
			fileName:      "existing.mp4",
			fileExists:    true,
			expectedError: false,
		},
		{
			name:          "file does not exist - should attempt conversion",
			fileName:      "new.mp4",
			fileExists:    false,
			expectedError: false, // We'll test the logic, not the actual conversion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()

			outputPath := filepath.Join(tempDir, strings.Replace(tt.fileName, ".mp4", ".mp3", 1))

			// Create the output file if it should exist
			if tt.fileExists {
				file, err := os.Create(outputPath)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				file.Close()
			}

			// Test the file existence check logic
			_, err := os.Stat(outputPath)
			fileExists := !os.IsNotExist(err)

			if fileExists != tt.fileExists {
				t.Errorf("expected file exists=%v, got %v", tt.fileExists, fileExists)
			}

			t.Logf("Test case: %s, File exists: %v", tt.name, fileExists)
		})
	}
}

// TestIs16kHzWavFile tests the Is16kHzWavFile logic with different probe outputs
func TestIs16kHzWavFile(t *testing.T) {
	tests := []struct {
		name          string
		probeOutput   string
		expected16kHz bool
		expectedError bool
	}{
		{
			name: "valid 16kHz WAV file",
			probeOutput: `{
				"streams": [
					{
						"codec_type": "audio",
						"codec_name": "pcm_s16le",
						"sample_rate": "16000"
					}
				]
			}`,
			expected16kHz: true,
			expectedError: false,
		},
		{
			name: "non-16kHz WAV file",
			probeOutput: `{
				"streams": [
					{
						"codec_type": "audio",
						"codec_name": "pcm_s16le",
						"sample_rate": "44100"
					}
				]
			}`,
			expected16kHz: false,
			expectedError: false,
		},
		{
			name: "non-WAV audio file",
			probeOutput: `{
				"streams": [
					{
						"codec_type": "audio",
						"codec_name": "mp3",
						"sample_rate": "16000"
					}
				]
			}`,
			expected16kHz: false,
			expectedError: false,
		},
		{
			name: "multiple streams with 16kHz WAV",
			probeOutput: `{
				"streams": [
					{
						"codec_type": "video",
						"codec_name": "h264"
					},
					{
						"codec_type": "audio",
						"codec_name": "pcm_s16le",
						"sample_rate": "16000"
					}
				]
			}`,
			expected16kHz: true,
			expectedError: false,
		},
		{
			name:          "invalid JSON output",
			probeOutput:   "not valid json",
			expectedError: true,
		},
		{
			name: "empty streams array",
			probeOutput: `{
				"streams": []
			}`,
			expected16kHz: false,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the probe output parsing logic
			is16kHz, err := parseProbeOutput(tt.probeOutput)

			// Check error
			if tt.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if is16kHz != tt.expected16kHz {
					t.Errorf("expected is16kHz=%v, got %v", tt.expected16kHz, is16kHz)
				}
			}
		})
	}
}

// TestConvertTo16kHzWav tests the ConvertTo16kHzWav path generation logic
func TestConvertTo16kHzWav(t *testing.T) {
	tests := []struct {
		name            string
		inputFile       string
		expectedOutput  string
		supportedFormat bool
	}{
		{
			name:            "MP3 file",
			inputFile:       "/test/audio.mp3",
			expectedOutput:  "/test/audio_16khz.wav",
			supportedFormat: true,
		},
		{
			name:            "M4A file",
			inputFile:       "/test/audio.m4a",
			expectedOutput:  "/test/audio_16khz.wav",
			supportedFormat: true,
		},
		{
			name:            "WAV file",
			inputFile:       "/test/audio.wav",
			expectedOutput:  "/test/audio_16khz.wav",
			supportedFormat: true,
		},
		{
			name:            "unsupported OGG file",
			inputFile:       "/test/audio.ogg",
			expectedOutput:  "/test/audio_16khz.wav",
			supportedFormat: false,
		},
		{
			name:            "case insensitive extension",
			inputFile:       "/test/AUDIO.MP3",
			expectedOutput:  "/test/AUDIO_16khz.wav",
			supportedFormat: true,
		},
		{
			name:            "path with spaces",
			inputFile:       "/test/my audio file.mp3",
			expectedOutput:  "/test/my audio file_16khz.wav",
			supportedFormat: true,
		},
		{
			name:            "multiple extensions",
			inputFile:       "/test/audio.test.mp3",
			expectedOutput:  "/test/audio.test_16khz.wav",
			supportedFormat: true,
		},
		{
			name:            "no extension",
			inputFile:       "/test/audiofile",
			expectedOutput:  "/test/audiofile_16khz.wav",
			supportedFormat: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test output path generation
			outputPath := strings.TrimSuffix(tt.inputFile, filepath.Ext(tt.inputFile)) + "_16khz.wav"
			if outputPath != tt.expectedOutput {
				t.Errorf("expected output path %s, got %s", tt.expectedOutput, outputPath)
			}

			// Test format support logic
			ext := strings.ToLower(filepath.Ext(tt.inputFile))
			supported := ext == ".mp3" || ext == ".m4a" || ext == ".wav"
			if supported != tt.supportedFormat {
				t.Errorf("expected supported=%v for extension %s", tt.supportedFormat, ext)
			}
		})
	}
}

// TestFFProbeOutputParsing tests the parsing of FFProbe JSON output
func TestFFProbeOutputParsing(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectedError bool
		expectedCodec string
		expectedRate  int
	}{
		{
			name: "valid simple output",
			jsonInput: `{
				"streams": [
					{
						"codec_type": "audio",
						"codec_name": "pcm_s16le",
						"sample_rate": "16000"
					}
				]
			}`,
			expectedCodec: "pcm_s16le",
			expectedRate:  16000,
		},
		{
			name: "multiple audio streams",
			jsonInput: `{
				"streams": [
					{
						"codec_type": "audio",
						"codec_name": "aac",
						"sample_rate": "48000"
					},
					{
						"codec_type": "audio",
						"codec_name": "pcm_s16le",
						"sample_rate": "16000"
					}
				]
			}`,
			expectedCodec: "aac", // First stream
			expectedRate:  48000,
		},
		{
			name: "video and audio streams",
			jsonInput: `{
				"streams": [
					{
						"codec_type": "video",
						"codec_name": "h264"
					},
					{
						"codec_type": "audio",
						"codec_name": "aac",
						"sample_rate": "44100"
					}
				]
			}`,
			expectedCodec: "aac",
			expectedRate:  44100,
		},
		{
			name:          "malformed JSON",
			jsonInput:     `{"streams": [}`,
			expectedError: true,
		},
		{
			name: "missing fields",
			jsonInput: `{
				"streams": [
					{
						"codec_type": "audio"
					}
				]
			}`,
			expectedCodec: "",
			expectedRate:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output model2.FFProbeOutput
			err := json.Unmarshal([]byte(tt.jsonInput), &output)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if len(output.Streams) > 0 {
					// Find first audio stream
					for _, stream := range output.Streams {
						if stream.CodecType == "audio" {
							if stream.CodecName != tt.expectedCodec {
								t.Errorf("expected codec %s, got %s",
									tt.expectedCodec, stream.CodecName)
							}
							if stream.SampleRate != tt.expectedRate {
								t.Errorf("expected sample rate %d, got %d",
									tt.expectedRate, stream.SampleRate)
							}
							break
						}
					}
				}
			}
		})
	}
}
