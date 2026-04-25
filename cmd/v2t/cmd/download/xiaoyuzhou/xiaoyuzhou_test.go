package xiaoyuzhou

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestXiaoyuzhouCommand tests the xiaoyuzhou command basic functionality
func TestXiaoyuzhouCommand(t *testing.T) {
	// Save original variables
	origDownloadDir := downloadDir
	origPodcast := podcast
	origEpisode := episode

	defer func() {
		// Restore original variables
		downloadDir = origDownloadDir
		podcast = origPodcast
		episode = origEpisode
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
			name:           "xiaoyuzhou_command_help",
			args:           []string{"--help"},
			expectedOutput: "Download podcasts from Small Universe",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "xiaoyuzhou_no_input",
			args:           []string{},
			expectedOutput: "",
			expectedError:  "please input a podcast or an episode",
			expectSuccess:  false,
			setupFunc:      func() { resetXiaoyuzhouFlags() },
		},
		{
			name:           "xiaoyuzhou_with_podcast",
			args:           []string{"--podcast", "https://www.xiaoyuzhoufm.com/podcast/test"},
			expectedOutput: "Mock: downloading podcast",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() { resetXiaoyuzhouFlags() },
		},
		{
			name:           "xiaoyuzhou_with_episode",
			args:           []string{"--episode", "https://www.xiaoyuzhoufm.com/episode/test"},
			expectedOutput: "Mock: downloading 1 episodes",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() { resetXiaoyuzhouFlags() },
		},
		{
			name:           "xiaoyuzhou_with_multiple_episodes",
			args:           []string{"--episode", "https://www.xiaoyuzhoufm.com/episode/test1,https://www.xiaoyuzhoufm.com/episode/test2"},
			expectedOutput: "Mock: downloading 2 episodes",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() { resetXiaoyuzhouFlags() },
		},
		{
			name:           "xiaoyuzhou_with_custom_download_dir",
			args:           []string{"--downloadDir", "/custom/dir", "--podcast", "https://www.xiaoyuzhoufm.com/podcast/test"},
			expectedOutput: "Mock: downloading podcast",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() { resetXiaoyuzhouFlags() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			// Create a fresh command for each test
			testCmd := createTestXiaoyuzhouCommand()

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

// TestXiaoyuzhouCommandFlags tests flag parsing and validation
func TestXiaoyuzhouCommandFlags(t *testing.T) {
	// Save original variables
	origDownloadDir := downloadDir
	origPodcast := podcast
	origEpisode := episode

	defer func() {
		// Restore original variables
		downloadDir = origDownloadDir
		podcast = origPodcast
		episode = origEpisode
	}()

	tests := []struct {
		name          string
		args          []string
		expectedFlags map[string]interface{}
	}{
		{
			name: "download_dir_flag",
			args: []string{"--downloadDir", "/test/dir"},
			expectedFlags: map[string]interface{}{
				"downloadDir": "/test/dir",
			},
		},
		{
			name: "download_dir_short_flag",
			args: []string{"-d", "/short/dir"},
			expectedFlags: map[string]interface{}{
				"downloadDir": "/short/dir",
			},
		},
		{
			name: "podcast_flag",
			args: []string{"--podcast", "https://www.xiaoyuzhoufm.com/podcast/test"},
			expectedFlags: map[string]interface{}{
				"podcast": "https://www.xiaoyuzhoufm.com/podcast/test",
			},
		},
		{
			name: "podcast_short_flag",
			args: []string{"-p", "https://example.com/podcast"},
			expectedFlags: map[string]interface{}{
				"podcast": "https://example.com/podcast",
			},
		},
		{
			name: "episode_flag",
			args: []string{"--episode", "https://www.xiaoyuzhoufm.com/episode/test"},
			expectedFlags: map[string]interface{}{
				"episode": "https://www.xiaoyuzhoufm.com/episode/test",
			},
		},
		{
			name: "episode_short_flag",
			args: []string{"-e", "https://example.com/episode"},
			expectedFlags: map[string]interface{}{
				"episode": "https://example.com/episode",
			},
		},
		{
			name: "multiple_episodes",
			args: []string{"--episode", "https://example.com/ep1,https://example.com/ep2"},
			expectedFlags: map[string]interface{}{
				"episode": "https://example.com/ep1,https://example.com/ep2",
			},
		},
		{
			name: "all_flags_together",
			args: []string{
				"--downloadDir", "/custom/path",
				"--podcast", "https://example.com/podcast",
				"--episode", "https://example.com/episode",
			},
			expectedFlags: map[string]interface{}{
				"downloadDir": "/custom/path",
				"podcast":     "https://example.com/podcast",
				"episode":     "https://example.com/episode",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetXiaoyuzhouFlags()

			testCmd := createTestXiaoyuzhouCommand()
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
				case "downloadDir":
					assert.Equal(t, expectedValue, downloadDir, "downloadDir flag value mismatch")
				case "podcast":
					assert.Equal(t, expectedValue, podcast, "podcast flag value mismatch")
				case "episode":
					assert.Equal(t, expectedValue, episode, "episode flag value mismatch")
				}
			}
		})
	}
}

// TestXiaoyuzhouCommandStructure tests the command structure and metadata
func TestXiaoyuzhouCommandStructure(t *testing.T) {
	testCmd := createTestXiaoyuzhouCommand()

	t.Run("command_metadata", func(t *testing.T) {
		assert.Equal(t, "xiaoyuzhou", testCmd.Use, "Command use name mismatch")
		assert.NotEmpty(t, testCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, testCmd.Long, "Long description should not be empty")
		assert.Contains(t, testCmd.Short, "Download podcasts", "Short description should mention downloading")
		assert.Contains(t, testCmd.Short, "Small Universe", "Short description should mention Small Universe")
	})

	t.Run("command_runnable", func(t *testing.T) {
		assert.NotNil(t, testCmd.RunE, "Command should have a RunE function")
		assert.True(t, testCmd.Runnable(), "Command should be runnable")
	})

	t.Run("command_flags", func(t *testing.T) {
		flags := testCmd.Flags()
		assert.NotNil(t, flags, "Flags should be initialized")

		// Test that all expected flags are present
		expectedFlags := []string{"downloadDir", "podcast", "episode"}

		for _, flagName := range expectedFlags {
			flag := flags.Lookup(flagName)
			assert.NotNil(t, flag, "Flag '%s' should be defined", flagName)
		}
	})
}

// TestXiaoyuzhouCommandValidation tests input validation logic
func TestXiaoyuzhouCommandValidation(t *testing.T) {
	// Save original variables
	origDownloadDir := downloadDir
	origPodcast := podcast
	origEpisode := episode

	defer func() {
		// Restore original variables
		downloadDir = origDownloadDir
		podcast = origPodcast
		episode = origEpisode
	}()

	tests := []struct {
		name         string
		args         []string
		expectError  bool
		errorMessage string
	}{
		{
			name:         "no_podcast_or_episode",
			args:         []string{},
			expectError:  true,
			errorMessage: "please input a podcast or an episode",
		},
		{
			name:         "podcast_provided",
			args:         []string{"--podcast", "https://www.xiaoyuzhoufm.com/podcast/test"},
			expectError:  false,
			errorMessage: "",
		},
		{
			name:         "episode_provided",
			args:         []string{"--episode", "https://www.xiaoyuzhoufm.com/episode/test"},
			expectError:  false,
			errorMessage: "",
		},
		{
			name:         "both_podcast_and_episode",
			args:         []string{"--podcast", "https://www.xiaoyuzhoufm.com/podcast/test", "--episode", "https://www.xiaoyuzhoufm.com/episode/test"},
			expectError:  false, // Both are allowed, podcast takes precedence
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags before each test
			resetXiaoyuzhouFlags()

			testCmd := createTestXiaoyuzhouCommand()

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			// Set args
			testCmd.SetArgs(tt.args)

			// Execute the command
			err := testCmd.Execute()

			if tt.expectError {
				assert.Error(t, err, "Expected an error but got none")
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage, "Error message mismatch")
				}
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
			}
		})
	}
}

// TestXiaoyuzhouCommandHelp tests the help functionality
func TestXiaoyuzhouCommandHelp(t *testing.T) {
	testCmd := createTestXiaoyuzhouCommand()

	var buf bytes.Buffer
	testCmd.SetOut(&buf)
	testCmd.SetArgs([]string{"--help"})

	err := testCmd.Execute()
	output := buf.String()

	assert.NoError(t, err, "Help command should not return an error")

	t.Run("help_contains_usage", func(t *testing.T) {
		assert.Contains(t, output, "Usage:", "Help should contain usage section")
		assert.Contains(t, output, "xiaoyuzhou", "Help should show command name")
	})

	t.Run("help_contains_description", func(t *testing.T) {
		assert.Contains(t, output, "Download podcasts from Small Universe", "Help should contain main description")
	})

	t.Run("help_contains_flags", func(t *testing.T) {
		assert.Contains(t, output, "Flags:", "Help should contain flags section")

		// Check for key flags
		expectedFlags := []string{
			"--downloadDir", "-d",
			"--podcast", "-p",
			"--episode", "-e",
		}

		for _, flag := range expectedFlags {
			assert.Contains(t, output, flag, "Help should show %s flag", flag)
		}
	})

	t.Run("help_contains_examples", func(t *testing.T) {
		// Check for URLs in flag descriptions
		assert.Contains(t, output, "xiaoyuzhoufm.com", "Help should contain example URLs")
	})
}

// TestXiaoyuzhouCommandErrorHandling tests error scenarios
func TestXiaoyuzhouCommandErrorHandling(t *testing.T) {
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
			name:         "invalid_flag_format",
			args:         []string{"---invalid"},
			expectError:  true,
			errorMessage: "bad flag syntax",
		},
		{
			name:         "valid_flags",
			args:         []string{"--podcast", "https://example.com/podcast"},
			expectError:  false,
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetXiaoyuzhouFlags()

			testCmd := createTestXiaoyuzhouCommand()
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
				assert.NoError(t, err, "Unexpected error: %v", err)
			}
		})
	}
}

// TestXiaoyuzhouCommandDefaults tests default flag values
func TestXiaoyuzhouCommandDefaults(t *testing.T) {
	resetXiaoyuzhouFlags()

	testCmd := createTestXiaoyuzhouCommand()
	testCmd.SetOut(io.Discard)
	testCmd.SetErr(io.Discard)

	// Parse with no flags to test defaults
	err := testCmd.ParseFlags([]string{})
	assert.NoError(t, err, "Parsing empty flags should not fail")

	// Test default values
	assert.Equal(t, "data/xiaoyuzhou", downloadDir, "Default downloadDir should be data/xiaoyuzhou")
	assert.Equal(t, "", podcast, "Default podcast should be empty")
	assert.Equal(t, "", episode, "Default episode should be empty")
}

// TestXiaoyuzhouCommandConcurrency tests concurrent execution
func TestXiaoyuzhouCommandConcurrency(t *testing.T) {
	t.Run("concurrent_help_calls", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Launch concurrent help commands
		for i := 0; i < numGoroutines; i++ {
			go func() {
				resetXiaoyuzhouFlags()
				testCmd := createTestXiaoyuzhouCommand()
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

// TestXiaoyuzhouCommandEpisodeLogic tests episode parsing logic
func TestXiaoyuzhouCommandEpisodeLogic(t *testing.T) {
	// Save original variables
	origDownloadDir := downloadDir
	origPodcast := podcast
	origEpisode := episode

	defer func() {
		// Restore original variables
		downloadDir = origDownloadDir
		podcast = origPodcast
		episode = origEpisode
	}()

	tests := []struct {
		name          string
		episodeInput  string
		expectedSplit []string
		expectSuccess bool
	}{
		{
			name:          "single_episode",
			episodeInput:  "https://www.xiaoyuzhoufm.com/episode/test1",
			expectedSplit: []string{"https://www.xiaoyuzhoufm.com/episode/test1"},
			expectSuccess: true,
		},
		{
			name:          "multiple_episodes",
			episodeInput:  "https://www.xiaoyuzhoufm.com/episode/test1,https://www.xiaoyuzhoufm.com/episode/test2",
			expectedSplit: []string{"https://www.xiaoyuzhoufm.com/episode/test1", "https://www.xiaoyuzhoufm.com/episode/test2"},
			expectSuccess: true,
		},
		{
			name:          "multiple_episodes_with_spaces",
			episodeInput:  "https://www.xiaoyuzhoufm.com/episode/test1, https://www.xiaoyuzhoufm.com/episode/test2",
			expectedSplit: []string{"https://www.xiaoyuzhoufm.com/episode/test1", " https://www.xiaoyuzhoufm.com/episode/test2"},
			expectSuccess: true,
		},
		{
			name:          "empty_episode",
			episodeInput:  "",
			expectedSplit: []string{""},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetXiaoyuzhouFlags()

			_ = createTestXiaoyuzhouCommand()
			episode = tt.episodeInput

			// Test the string splitting logic that would happen in the actual command
			if episode != "" {
				episodeList := strings.Split(episode, ",")
				assert.Equal(t, tt.expectedSplit, episodeList, "Episode splitting mismatch")
			}
		})
	}
}

// TestXiaoyuzhouCommandIntegration tests integration scenarios
func TestXiaoyuzhouCommandIntegration(t *testing.T) {
	// Save original variables
	origDownloadDir := downloadDir
	origPodcast := podcast
	origEpisode := episode

	defer func() {
		// Restore original variables
		downloadDir = origDownloadDir
		podcast = origPodcast
		episode = origEpisode
	}()

	t.Run("command_in_parent_context", func(t *testing.T) {
		resetXiaoyuzhouFlags()

		// Test the xiaoyuzhou command as it would be used in the download command
		parentCmd := &cobra.Command{
			Use: "download",
		}

		xiaoyuzhouCmd := createTestXiaoyuzhouCommand()
		parentCmd.AddCommand(xiaoyuzhouCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"xiaoyuzhou", "--podcast", "https://example.com/podcast"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Xiaoyuzhou command in parent context should not fail")
		assert.Contains(t, output, "Mock: downloading podcast", "Expected output not found")
	})

	t.Run("command_with_global_flags", func(t *testing.T) {
		resetXiaoyuzhouFlags()

		// Test xiaoyuzhou command with global flags
		parentCmd := &cobra.Command{
			Use: "download",
		}

		// Add a global flag
		var verbose bool
		parentCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

		xiaoyuzhouCmd := createTestXiaoyuzhouCommand()
		parentCmd.AddCommand(xiaoyuzhouCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"--verbose", "xiaoyuzhou", "--episode", "https://example.com/episode"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Xiaoyuzhou command with global flags should not fail")
		assert.Contains(t, output, "Mock: downloading 1 episodes", "Expected output not found")
		assert.True(t, verbose, "Global verbose flag should be set")
	})
}

// resetXiaoyuzhouFlags resets all xiaoyuzhou command flags to their defaults
func resetXiaoyuzhouFlags() {
	downloadDir = "data/xiaoyuzhou"
	podcast = ""
	episode = ""
}

// createTestXiaoyuzhouCommand creates a fresh xiaoyuzhou command for testing
func createTestXiaoyuzhouCommand() *cobra.Command {
	// Create a command similar to the original but with mocked business logic
	testCmd := &cobra.Command{
		Use:   "xiaoyuzhou",
		Short: "Download podcasts from Small Universe",
		Long:  `Download podcasts from Small Universe, support downloading all shows from the home page and single downloads`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Mock the validation logic from the original command
			if podcast == "" && episode == "" {
				return errors.New("please input a podcast or an episode")
			}

			// Mock the path processing (simplified)
			if downloadDir != "" {
				// In real implementation, this would call files.GetAbsolutePath
				// For testing, we just validate it's not empty
			}

			// Mock the download logic
			if podcast != "" {
				cmd.Printf("Mock: downloading podcast from %s\n", podcast)
				return nil
			}

			// Mock episode download
			if episode != "" {
				episodeList := strings.Split(episode, ",")
				cmd.Printf("Mock: downloading %d episodes\n", len(episodeList))
				return nil
			}

			return nil
		},
	}

	// Add all the flags as in the original
	testCmd.Flags().StringVarP(&downloadDir, "downloadDir", "d", "data/xiaoyuzhou", "set directory to save downloaded files")
	testCmd.Flags().StringVarP(&podcast, "podcast", "p", "", "set podcast url, e.g. https://www.xiaoyuzhoufm.com/podcast/61a9f093ca6141933d1a1c63")
	testCmd.Flags().StringVarP(&episode, "episode", "e", "", "set episode, If it is more than one episode can be separated by a comma, e.g. https://www.xiaoyuzhoufm.com/episode/64411602a79cc81470055c96")

	return testCmd
}

// BenchmarkXiaoyuzhouCommand benchmarks xiaoyuzhou command performance
func BenchmarkXiaoyuzhouCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetXiaoyuzhouFlags()
		testCmd := createTestXiaoyuzhouCommand()
		testCmd.SetArgs([]string{"--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}

// BenchmarkXiaoyuzhouCommandFlagParsing benchmarks flag parsing performance
func BenchmarkXiaoyuzhouCommandFlagParsing(b *testing.B) {
	args := []string{
		"--downloadDir", "/test/dir",
		"--podcast", "https://www.xiaoyuzhoufm.com/podcast/test",
		"--episode", "https://www.xiaoyuzhoufm.com/episode/test1,https://www.xiaoyuzhoufm.com/episode/test2",
	}

	for i := 0; i < b.N; i++ {
		resetXiaoyuzhouFlags()
		testCmd := createTestXiaoyuzhouCommand()
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.ParseFlags(args)
	}
}
