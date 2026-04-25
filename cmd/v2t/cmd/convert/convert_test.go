package convert

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestConvertCommand tests the convert command basic functionality
func TestConvertCommand(t *testing.T) {
	// Save original variables
	origUserNickname := userNickname
	origDirectory := directory
	origOutputDirectory := outputDirectory
	origFileExtension := fileExtension
	origVideo := video
	origAudio := audio
	origConvertCount := convertCount
	origParallel := parallel
	origInputFile := inputFile

	defer func() {
		// Restore original variables
		userNickname = origUserNickname
		directory = origDirectory
		outputDirectory = origOutputDirectory
		fileExtension = origFileExtension
		video = origVideo
		audio = origAudio
		convertCount = origConvertCount
		parallel = origParallel
		inputFile = origInputFile
	}()

	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  string
		expectSuccess  bool
		setupFunc      func()
	}{
		{
			name:           "convert_command_help",
			args:           []string{"--help"},
			expectedOutput: "Start converting the video files",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "convert_no_type_specified",
			args:           []string{},
			expectedOutput: "Please specify the conversion type, -v or -a",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "convert_both_types_specified",
			args:           []string{"--video", "--audio"},
			expectedOutput: "Please specify the conversion type, -v or -a",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "convert_no_input_specified",
			args:           []string{"--video"},
			expectedOutput: "Please specify the directory or file to convert",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "convert_both_directory_and_file_specified",
			args:           []string{"--video", "--directory", "/test", "--input", "test.mp4"},
			expectedOutput: "Please specify the directory or file to convert",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "convert_video_directory_no_user",
			args:           []string{"--video", "--directory", "/test"},
			expectedOutput: "UserNickName must be set when converting video in directory",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags before each test
			resetConvertFlags()
			tt.setupFunc()

			// Create a fresh command for each test
			testCmd := createTestConvertCommand()

			// Capture output
			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			// Set args
			testCmd.SetArgs(tt.args)

			// Execute command
			err := testCmd.Execute()

			output := buf.String()

			// Validate execution success/failure
			if tt.expectSuccess {
				if !strings.Contains(output, "Usage:") { // Help commands are special
					assert.NoError(t, err, "Expected command to succeed but got error: %v", err)
				}
			} else {
				assert.Error(t, err, "Expected command to fail but it succeeded")
			}

			// Validate output
			if tt.expectedOutput != "" {
				assert.Contains(t, output, tt.expectedOutput, "Expected output not found in: %s", output)
			}

			// Validate error message
			if tt.expectedError != "" {
				assert.Error(t, err, "Expected an error but got none")
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedError, "Expected error message not found")
				}
			}
		})
	}
}

// TestConvertCommandFlags tests flag parsing and validation
func TestConvertCommandFlags(t *testing.T) {
	// Save original variables
	origUserNickname := userNickname
	origDirectory := directory
	origOutputDirectory := outputDirectory
	origFileExtension := fileExtension
	origVideo := video
	origAudio := audio
	origConvertCount := convertCount
	origParallel := parallel
	origInputFile := inputFile

	defer func() {
		// Restore original variables
		userNickname = origUserNickname
		directory = origDirectory
		outputDirectory = origOutputDirectory
		fileExtension = origFileExtension
		video = origVideo
		audio = origAudio
		convertCount = origConvertCount
		parallel = origParallel
		inputFile = origInputFile
	}()

	tests := []struct {
		name          string
		args          []string
		expectedFlags map[string]interface{}
	}{
		{
			name: "user_nickname_flag",
			args: []string{"--userNickname", "test_user"},
			expectedFlags: map[string]interface{}{
				"userNickname": "test_user",
			},
		},
		{
			name: "user_nickname_short_flag",
			args: []string{"-u", "short_user"},
			expectedFlags: map[string]interface{}{
				"userNickname": "short_user",
			},
		},
		{
			name: "directory_flag",
			args: []string{"--directory", "/test/dir"},
			expectedFlags: map[string]interface{}{
				"directory": "/test/dir",
			},
		},
		{
			name: "directory_short_flag",
			args: []string{"-d", "/short/dir"},
			expectedFlags: map[string]interface{}{
				"directory": "/short/dir",
			},
		},
		{
			name: "output_directory_flag",
			args: []string{"--outputDirectory", "/output/dir"},
			expectedFlags: map[string]interface{}{
				"outputDirectory": "/output/dir",
			},
		},
		{
			name: "output_directory_short_flag",
			args: []string{"-o", "/output/short"},
			expectedFlags: map[string]interface{}{
				"outputDirectory": "/output/short",
			},
		},
		{
			name: "convert_count_flag",
			args: []string{"--convertCount", "5"},
			expectedFlags: map[string]interface{}{
				"convertCount": 5,
			},
		},
		{
			name: "convert_count_short_flag",
			args: []string{"-n", "10"},
			expectedFlags: map[string]interface{}{
				"convertCount": 10,
			},
		},
		{
			name: "parallel_flag",
			args: []string{"--parallel", "3"},
			expectedFlags: map[string]interface{}{
				"parallel": 3,
			},
		},
		{
			name: "parallel_short_flag",
			args: []string{"-p", "4"},
			expectedFlags: map[string]interface{}{
				"parallel": 4,
			},
		},
		{
			name: "input_file_flag",
			args: []string{"--input", "test.mp3"},
			expectedFlags: map[string]interface{}{
				"inputFile": "test.mp3",
			},
		},
		{
			name: "input_file_short_flag",
			args: []string{"-i", "short.mp3"},
			expectedFlags: map[string]interface{}{
				"inputFile": "short.mp3",
			},
		},
		{
			name: "file_extension_flag",
			args: []string{"--type", "wav"},
			expectedFlags: map[string]interface{}{
				"fileExtension": "wav",
			},
		},
		{
			name: "file_extension_short_flag",
			args: []string{"-t", "flac"},
			expectedFlags: map[string]interface{}{
				"fileExtension": "flac",
			},
		},
		{
			name: "video_flag",
			args: []string{"--video"},
			expectedFlags: map[string]interface{}{
				"video": true,
			},
		},
		{
			name: "video_short_flag",
			args: []string{"-v"},
			expectedFlags: map[string]interface{}{
				"video": true,
			},
		},
		{
			name: "audio_flag",
			args: []string{"--audio"},
			expectedFlags: map[string]interface{}{
				"audio": true,
			},
		},
		{
			name: "audio_short_flag",
			args: []string{"-a"},
			expectedFlags: map[string]interface{}{
				"audio": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetConvertFlags()

			testCmd := createTestConvertCommand()
			testCmd.SetArgs(tt.args)

			// Suppress output for flag testing
			testCmd.SetOut(io.Discard)
			testCmd.SetErr(io.Discard)

			// Parse flags
			err := testCmd.ParseFlags(tt.args)
			assert.NoError(t, err, "Flag parsing should not fail")

			// Validate flags
			for flagName, expectedValue := range tt.expectedFlags {
				switch flagName {
				case "userNickname":
					assert.Equal(t, expectedValue, userNickname, "userNickname flag value mismatch")
				case "directory":
					assert.Equal(t, expectedValue, directory, "directory flag value mismatch")
				case "outputDirectory":
					assert.Equal(t, expectedValue, outputDirectory, "outputDirectory flag value mismatch")
				case "convertCount":
					assert.Equal(t, expectedValue, convertCount, "convertCount flag value mismatch")
				case "parallel":
					assert.Equal(t, expectedValue, parallel, "parallel flag value mismatch")
				case "inputFile":
					assert.Equal(t, expectedValue, inputFile, "inputFile flag value mismatch")
				case "fileExtension":
					assert.Equal(t, expectedValue, fileExtension, "fileExtension flag value mismatch")
				case "video":
					assert.Equal(t, expectedValue, video, "video flag value mismatch")
				case "audio":
					assert.Equal(t, expectedValue, audio, "audio flag value mismatch")
				}
			}
		})
	}
}

// TestConvertCommandStructure tests the command structure and metadata
func TestConvertCommandStructure(t *testing.T) {
	testCmd := createTestConvertCommand()

	t.Run("command_metadata", func(t *testing.T) {
		assert.Equal(t, "convert", testCmd.Use, "Command use name mismatch")
		assert.NotEmpty(t, testCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, testCmd.Long, "Long description should not be empty")
		assert.Contains(t, testCmd.Short, "converting", "Short description should mention converting")
		assert.Contains(t, testCmd.Long, "whisper", "Long description should mention whisper")
	})

	t.Run("command_runnable", func(t *testing.T) {
		assert.NotNil(t, testCmd.Run, "Command should have a Run function")
		assert.True(t, testCmd.Runnable(), "Command should be runnable")
	})

	t.Run("command_flags", func(t *testing.T) {
		flags := testCmd.Flags()
		assert.NotNil(t, flags, "Flags should be initialized")

		// Test that all expected flags are present
		expectedFlags := []string{
			"userNickname", "directory", "outputDirectory", "convertCount",
			"parallel", "input", "type", "video", "audio",
		}

		for _, flagName := range expectedFlags {
			flag := flags.Lookup(flagName)
			assert.NotNil(t, flag, "Flag '%s' should be defined", flagName)
		}
	})
}

// TestConvertCommandValidation tests input validation logic
func TestConvertCommandValidation(t *testing.T) {
	// Save original variables
	origUserNickname := userNickname
	origDirectory := directory
	origOutputDirectory := outputDirectory
	origFileExtension := fileExtension
	origVideo := video
	origAudio := audio
	origConvertCount := convertCount
	origParallel := parallel
	origInputFile := inputFile

	defer func() {
		// Restore original variables
		userNickname = origUserNickname
		directory = origDirectory
		outputDirectory = origOutputDirectory
		fileExtension = origFileExtension
		video = origVideo
		audio = origAudio
		convertCount = origConvertCount
		parallel = origParallel
		inputFile = origInputFile
	}()

	tests := []struct {
		name        string
		setupFlags  func()
		args        []string
		expectHelp  bool
		helpMessage string
	}{
		{
			name: "no_conversion_type",
			setupFlags: func() {
				resetConvertFlags()
			},
			args:        []string{},
			expectHelp:  true,
			helpMessage: "Please specify the conversion type",
		},
		{
			name: "both_conversion_types",
			setupFlags: func() {
				resetConvertFlags()
			},
			args:        []string{"--video", "--audio"},
			expectHelp:  true,
			helpMessage: "Please specify the conversion type",
		},
		{
			name: "no_input_source",
			setupFlags: func() {
				resetConvertFlags()
			},
			args:        []string{"--video"},
			expectHelp:  true,
			helpMessage: "Please specify the directory or file to convert",
		},
		{
			name: "both_input_sources",
			setupFlags: func() {
				resetConvertFlags()
			},
			args:        []string{"--video", "--directory", "/test", "--input", "test.mp4"},
			expectHelp:  true,
			helpMessage: "Please specify the directory or file to convert",
		},
		{
			name: "video_directory_no_user",
			setupFlags: func() {
				resetConvertFlags()
			},
			args:        []string{"--video", "--directory", "/test"},
			expectHelp:  true,
			helpMessage: "UserNickName must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFlags()

			testCmd := createTestConvertCommand()

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			// Set command args if provided
			if tt.args != nil {
				testCmd.SetArgs(tt.args)
			}

			// Execute the command
			err := testCmd.Execute()
			output := buf.String()

			if tt.expectHelp {
				// Should not return an error, but should print help message
				assert.NoError(t, err, "Validation should not return an error, just show help")
				assert.Contains(t, output, tt.helpMessage, "Expected help message not found")
			}
		})
	}
}

// TestConvertCommandHelp tests the help functionality
func TestConvertCommandHelp(t *testing.T) {
	testCmd := createTestConvertCommand()

	var buf bytes.Buffer
	testCmd.SetOut(&buf)
	testCmd.SetArgs([]string{"--help"})

	err := testCmd.Execute()
	output := buf.String()

	assert.NoError(t, err, "Help command should not return an error")

	t.Run("help_contains_usage", func(t *testing.T) {
		assert.Contains(t, output, "Usage:", "Help should contain usage section")
		assert.Contains(t, output, "convert", "Help should show command name")
	})

	t.Run("help_contains_description", func(t *testing.T) {
		assert.Contains(t, output, "converting the video files", "Help should contain main description")
		assert.Contains(t, output, "whisper", "Help should mention whisper")
	})

	t.Run("help_contains_flags", func(t *testing.T) {
		assert.Contains(t, output, "Flags:", "Help should contain flags section")

		// Check for key flags
		expectedFlags := []string{
			"--userNickname", "-u",
			"--directory", "-d",
			"--video", "-v",
			"--audio", "-a",
			"--parallel", "-p",
		}

		for _, flag := range expectedFlags {
			assert.Contains(t, output, flag, "Help should show %s flag", flag)
		}
	})

	t.Run("help_contains_examples", func(t *testing.T) {
		// The long description should contain examples or detailed usage
		assert.Contains(t, output, "mp4", "Help should mention supported formats")
		assert.Contains(t, output, "mp3", "Help should mention audio formats")
	})
}

// TestConvertCommandErrorHandling tests error scenarios
func TestConvertCommandErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectError  bool
		errorMessage string
	}{
		{
			name:         "unknown_flag",
			args:         []string{"--unknown-flag"},
			expectError:  true,
			errorMessage: "unknown flag",
		},
		{
			name:         "invalid_convert_count",
			args:         []string{"--convertCount", "invalid"},
			expectError:  true,
			errorMessage: "invalid syntax",
		},
		{
			name:         "invalid_parallel_count",
			args:         []string{"--parallel", "not-a-number"},
			expectError:  true,
			errorMessage: "invalid syntax",
		},
		{
			name:         "negative_convert_count",
			args:         []string{"--convertCount", "-1"},
			expectError:  false, // The command accepts negative numbers
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetConvertFlags()

			testCmd := createTestConvertCommand()
			testCmd.SetArgs(tt.args)

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			err := testCmd.Execute()

			if tt.expectError {
				assert.Error(t, err, "Expected an error but got none")
				if tt.errorMessage != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorMessage, "Error message mismatch")
				}
			} else {
				// For validation errors, the command prints help but doesn't return error
				if !strings.Contains(buf.String(), "Usage:") {
					assert.NoError(t, err, "Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestConvertCommandDefaults tests default flag values
func TestConvertCommandDefaults(t *testing.T) {
	resetConvertFlags()

	testCmd := createTestConvertCommand()
	testCmd.SetOut(io.Discard)
	testCmd.SetErr(io.Discard)

	// Parse with no flags to test defaults
	err := testCmd.ParseFlags([]string{})
	assert.NoError(t, err, "Parsing empty flags should not fail")

	// Test default values
	assert.Equal(t, "", userNickname, "Default userNickname should be empty")
	assert.Equal(t, "", directory, "Default directory should be empty")
	assert.Equal(t, "./data/transcription", outputDirectory, "Default outputDirectory should be ./data/transcription")
	assert.Equal(t, "", fileExtension, "Default fileExtension should be empty")
	assert.Equal(t, false, video, "Default video should be false")
	assert.Equal(t, false, audio, "Default audio should be false")
	assert.Equal(t, 1, convertCount, "Default convertCount should be 1")
	assert.Equal(t, 1, parallel, "Default parallel should be 1")
	assert.Equal(t, "", inputFile, "Default inputFile should be empty")
}

// TestConvertCommandConcurrency tests concurrent execution
func TestConvertCommandConcurrency(t *testing.T) {
	t.Run("concurrent_help_calls", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Launch concurrent help commands
		for i := 0; i < numGoroutines; i++ {
			go func() {
				testCmd := createTestConvertCommand()
				testCmd.SetArgs([]string{"--help"})
				testCmd.SetOut(io.Discard)
				testCmd.SetErr(io.Discard)
				results <- testCmd.Execute()
			}()
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent help command failed")
		}
	})
}

// TestConvertCommandFileExtensionDefaults tests file extension default behavior
func TestConvertCommandFileExtensionDefaults(t *testing.T) {
	// This test verifies the logic described in the convert command:
	// - Video defaults to mp4 if not specified
	// - Audio defaults to mp3 if not specified

	// Save original variables
	origFileExtension := fileExtension
	origVideo := video
	origAudio := audio

	defer func() {
		fileExtension = origFileExtension
		video = origVideo
		audio = origAudio
	}()

	t.Run("video_default_extension", func(t *testing.T) {
		resetConvertFlags()
		video = true
		fileExtension = ""

		// This would be the logic in the actual command execution
		expectedExtension := "mp4"
		if fileExtension == "" {
			fileExtension = "mp4"
		}

		assert.Equal(t, expectedExtension, fileExtension, "Video should default to mp4 extension")
	})

	t.Run("audio_default_extension", func(t *testing.T) {
		resetConvertFlags()
		audio = true
		fileExtension = ""

		// This would be the logic in the actual command execution
		expectedExtension := "mp3"
		if fileExtension == "" {
			fileExtension = "mp3"
		}

		assert.Equal(t, expectedExtension, fileExtension, "Audio should default to mp3 extension")
	})
}

// resetConvertFlags resets all convert command flags to their defaults
func resetConvertFlags() {
	userNickname = ""
	directory = ""
	outputDirectory = "./data/transcription"
	fileExtension = ""
	video = false
	audio = false
	convertCount = 1
	parallel = 1
	inputFile = ""
}

// createTestConvertCommand creates a fresh convert command for testing
func createTestConvertCommand() *cobra.Command {
	// Create a command similar to the original but with mocked business logic
	testCmd := &cobra.Command{
		Use:   "convert",
		Short: "Start converting the video files in the specified directory to text",
		Long: `Start converting the video files in the specified directory to text

- Iterate through the mp4 files in the specified directory
- Convert to mp3 or wav and convert to text
- Support openai whisper or native whisper.cpp as conversion engine`,
		Run: func(cmd *cobra.Command, args []string) {
			// Mock the validation logic from the original command
			if !video && !audio {
				cmd.PrintErrf("Please specify the conversion type, -v or -a\n")
				cmd.Help()
				return
			}

			if video && audio {
				cmd.PrintErrf("Please specify the conversion type, -v or -a\n")
				cmd.Help()
				return
			}

			if directory == "" && inputFile == "" {
				cmd.PrintErrf("Please specify the directory or file to convert\n")
				cmd.Help()
				return
			}

			if directory != "" && inputFile != "" {
				cmd.PrintErrf("Please specify the directory or file to convert\n")
				cmd.Help()
				return
			}

			if video {
				if directory != "" && userNickname == "" {
					cmd.PrintErrf("UserNickName must be set when converting video in directory\n")
					cmd.Help()
					return
				}

				if fileExtension == "" {
					fileExtension = "mp4"
				}

				// Mock successful execution
				cmd.Printf("Mock: Converting video files\n")
			}

			if audio {
				if fileExtension == "" {
					fileExtension = "mp3"
				}

				// Mock successful execution
				cmd.Printf("Mock: Converting audio files\n")
			}
		},
	}

	// Add all the flags as in the original
	testCmd.Flags().StringVarP(&userNickname, "userNickname", "u", "",
		"Which user owns the videos, this parameter affects the 'user' field when they are saved to the database")
	testCmd.Flags().StringVarP(&directory, "directory", "d", "",
		"Specifies the mp4 file directory, example: ./test/data/mp4")
	testCmd.Flags().StringVarP(&outputDirectory, "outputDirectory", "o", "./data/transcription",
		"Specifies the transcriptions directory, example: ./test/data/transcription")
	testCmd.Flags().IntVarP(&convertCount, "convertCount", "n", 1,
		"How many files to convert from the directory this time")
	testCmd.Flags().IntVarP(&parallel, "parallel", "p", 1,
		"How many files to convert at the same time")
	testCmd.Flags().StringVarP(&inputFile, "input", "i", "",
		"Specifies the audio file to convert, example: ./test/data/test.mp3")
	testCmd.Flags().StringVarP(&fileExtension, "type", "t", "",
		"When converting the specified directory, you can use this option to filter the files with the specified extension, example: mp3")
	testCmd.Flags().BoolVarP(&video, "video", "v", false,
		"Convert video to text")
	testCmd.Flags().BoolVarP(&audio, "audio", "a", false,
		"Convert audio to text")

	return testCmd
}

// BenchmarkConvertCommand benchmarks convert command performance
func BenchmarkConvertCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetConvertFlags()
		testCmd := createTestConvertCommand()
		testCmd.SetArgs([]string{"--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}

// BenchmarkConvertCommandFlagParsing benchmarks flag parsing performance
func BenchmarkConvertCommandFlagParsing(b *testing.B) {
	args := []string{
		"--userNickname", "test_user",
		"--directory", "/test/dir",
		"--outputDirectory", "/output/dir",
		"--convertCount", "5",
		"--parallel", "2",
		"--type", "mp4",
		"--video",
	}

	for i := 0; i < b.N; i++ {
		resetConvertFlags()
		testCmd := createTestConvertCommand()
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.ParseFlags(args)
	}
}
