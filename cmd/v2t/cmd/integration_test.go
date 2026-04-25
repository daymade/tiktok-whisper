//go:build integration
// +build integration

package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestCLIIntegration tests the complete CLI integration with all commands
func TestCLIIntegration(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  string
		expectSuccess  bool
	}{
		{
			name:           "root_help",
			args:           []string{"--help"},
			expectedOutput: "An application for batch converting video to text",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "version_command",
			args:           []string{"version"},
			expectedOutput: "Mock: version v0.0.1",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "config_command",
			args:           []string{"config"},
			expectedOutput: "Mock: config command executed",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "convert_help",
			args:           []string{"convert", "--help"},
			expectedOutput: "Start converting the video files",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "download_help",
			args:           []string{"download", "--help"},
			expectedOutput: "Download podcasts from Small Universe",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "download_xiaoyuzhou_help",
			args:           []string{"download", "xiaoyuzhou", "--help"},
			expectedOutput: "Download podcasts from Small Universe",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "embed_help",
			args:           []string{"embed", "--help"},
			expectedOutput: "Generate, manage, and analyze embeddings for transcriptions using OpenAI and Gemini",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "embed_generate_help",
			args:           []string{"embed", "generate", "--help"},
			expectedOutput: "Generate embeddings for transcriptions",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "embed_status_help",
			args:           []string{"embed", "status", "--help"},
			expectedOutput: "Display the current status of embedding generation for transcriptions",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "export_help",
			args:           []string{"export", "--help"},
			expectedOutput: "Export the specified user's text to excel",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "unknown_command",
			args:           []string{"unknown"},
			expectedOutput: "Usage:", // Cobra shows help for unknown commands
			expectedError:  "",
			expectSuccess:  true, // Cobra shows help without returning error
		},
		{
			name:           "global_verbose_flag",
			args:           []string{"--verbose", "version"},
			expectedOutput: "Mock: version v0.0.1",
			expectedError:  "",
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			Verbose = false

			// Create integrated CLI
			rootCmd := createIntegratedCLI()

			// Capture output
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			// Set args
			rootCmd.SetArgs(tt.args)

			// Execute command
			err := rootCmd.Execute()

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

// TestCLIWorkflows tests complete CLI workflows
func TestCLIWorkflows(t *testing.T) {
	t.Run("download_convert_export_workflow", func(t *testing.T) {
		// Test a complete workflow: download -> convert -> export
		// This tests that commands work together in sequence

		rootCmd := createIntegratedCLI()

		// Step 1: Download (mock)
		var buf1 bytes.Buffer
		rootCmd.SetOut(&buf1)
		rootCmd.SetErr(&buf1)
		rootCmd.SetArgs([]string{"download", "xiaoyuzhou", "--podcast", "https://example.com/podcast"})

		err1 := rootCmd.Execute()
		assert.NoError(t, err1, "Download step should succeed")
		assert.Contains(t, buf1.String(), "Mock: downloading podcast", "Download should execute")

		// Step 2: Convert (mock - requires files from step 1)
		rootCmd = createIntegratedCLI() // Fresh instance
		var buf2 bytes.Buffer
		rootCmd.SetOut(&buf2)
		rootCmd.SetErr(&buf2)
		rootCmd.SetArgs([]string{"convert", "--audio", "--input", "test.mp3"})

		err2 := rootCmd.Execute()
		assert.NoError(t, err2, "Convert step should succeed")
		assert.Contains(t, buf2.String(), "Mock: converting audio", "Convert should execute")

		// Step 3: Export (mock - requires data from step 2)
		rootCmd = createIntegratedCLI() // Fresh instance
		var buf3 bytes.Buffer
		rootCmd.SetOut(&buf3)
		rootCmd.SetErr(&buf3)
		rootCmd.SetArgs([]string{"export", "--userNickname", "test_user", "--outputFilePath", "/test/output.xlsx"})

		err3 := rootCmd.Execute()
		assert.NoError(t, err3, "Export step should succeed")
		assert.Contains(t, buf3.String(), "Mock: exporting data", "Export should execute")
	})

	t.Run("convert_embed_workflow", func(t *testing.T) {
		// Test convert -> embed workflow
		rootCmd := createIntegratedCLI()

		// Step 1: Convert
		var buf1 bytes.Buffer
		rootCmd.SetOut(&buf1)
		rootCmd.SetErr(&buf1)
		rootCmd.SetArgs([]string{"convert", "--video", "--input", "test.mp4", "--userNickname", "test_user"})

		err1 := rootCmd.Execute()
		assert.NoError(t, err1, "Convert step should succeed")

		// Step 2: Generate embeddings
		rootCmd = createIntegratedCLI() // Fresh instance
		var buf2 bytes.Buffer
		rootCmd.SetOut(&buf2)
		rootCmd.SetErr(&buf2)
		rootCmd.SetArgs([]string{"embed", "generate", "--user", "test_user"})

		err2 := rootCmd.Execute()
		assert.NoError(t, err2, "Embed generation should succeed")
		assert.Contains(t, buf2.String(), "Mock: generating embeddings", "Embed should execute")
	})
}

// TestCLIGlobalFlags tests global flag behavior across commands
func TestCLIGlobalFlags(t *testing.T) {
	t.Run("verbose_flag_propagation", func(t *testing.T) {
		tests := []struct {
			name    string
			args    []string
			verbose bool
		}{
			{
				name:    "verbose_with_version",
				args:    []string{"--verbose", "version"},
				verbose: true,
			},
			{
				name:    "verbose_short_with_config",
				args:    []string{"-V", "config"},
				verbose: true,
			},
			{
				name:    "no_verbose_with_version",
				args:    []string{"version"},
				verbose: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Reset global state
				Verbose = false

				rootCmd := createIntegratedCLI()
				rootCmd.SetOut(io.Discard)
				rootCmd.SetErr(io.Discard)
				rootCmd.SetArgs(tt.args)

				err := rootCmd.Execute()
				assert.NoError(t, err, "Command should execute successfully")
				assert.Equal(t, tt.verbose, Verbose, "Verbose flag should be set correctly")
			})
		}
	})
}

// TestCLIErrorHandling tests error handling across the CLI
func TestCLIErrorHandling(t *testing.T) {
	t.Run("command_error_isolation", func(t *testing.T) {
		// Test that errors in one command don't affect others
		rootCmd := createIntegratedCLI()

		// Execute an invalid command
		var buf1 bytes.Buffer
		rootCmd.SetOut(&buf1)
		rootCmd.SetErr(&buf1)
		rootCmd.SetArgs([]string{"invalid-command"})

		err1 := rootCmd.Execute()
		// Cobra shows help for invalid commands without returning error
		assert.NoError(t, err1, "Cobra shows help without error for invalid commands")
		assert.Contains(t, buf1.String(), "Usage:", "Should show help for invalid command")

		// Execute a valid command after the error
		rootCmd = createIntegratedCLI() // Fresh instance
		var buf2 bytes.Buffer
		rootCmd.SetOut(&buf2)
		rootCmd.SetErr(&buf2)
		rootCmd.SetArgs([]string{"version"})

		err2 := rootCmd.Execute()
		assert.NoError(t, err2, "Valid command should succeed after previous error")
		assert.Contains(t, buf2.String(), "v0.0.1", "Version should be displayed")
	})

	t.Run("flag_validation_errors", func(t *testing.T) {
		tests := []struct {
			name         string
			args         []string
			errorMessage string
		}{
			{
				name:         "unknown_global_flag",
				args:         []string{"--unknown-flag", "version"},
				errorMessage: "unknown flag",
			},
			{
				name:         "convert_missing_type",
				args:         []string{"convert"},
				errorMessage: "Please specify the conversion type, -v or -a",
			},
			{
				name:         "export_missing_required_flags",
				args:         []string{"export"},
				errorMessage: "required flag",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rootCmd := createIntegratedCLI()
				var buf bytes.Buffer
				rootCmd.SetOut(&buf)
				rootCmd.SetErr(&buf)
				rootCmd.SetArgs(tt.args)

				err := rootCmd.Execute()
				output := buf.String()

				// Check if error message appears either in err or in output
				var errorFound bool
				if err != nil {
					errorFound = strings.Contains(err.Error(), tt.errorMessage) || strings.Contains(output, tt.errorMessage)
				} else {
					errorFound = strings.Contains(output, tt.errorMessage)
				}
				
				// Some commands print error messages without returning an error
				if !errorFound {
					t.Errorf("Expected error message '%s' not found in error '%v' or output '%s'", tt.errorMessage, err, output)
				}
			})
		}
	})
}

// TestCLIConcurrency tests concurrent CLI usage
func TestCLIConcurrency(t *testing.T) {
	t.Run("concurrent_command_execution", func(t *testing.T) {
		const numGoroutines = 10
		var wg sync.WaitGroup
		results := make(chan error, numGoroutines)

		// Test concurrent execution of different commands
		commands := [][]string{
			{"version"},
			{"config"},
			{"convert", "--help"},
			{"download", "--help"},
			{"embed", "--help"},
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(cmdIndex int) {
				defer wg.Done()

				rootCmd := createIntegratedCLI()
				rootCmd.SetOut(io.Discard)
				rootCmd.SetErr(io.Discard)
				rootCmd.SetArgs(commands[cmdIndex%len(commands)])

				results <- rootCmd.Execute()
			}(i)
		}

		wg.Wait()
		close(results)

		// Check all results
		for err := range results {
			assert.NoError(t, err, "Concurrent command execution should not fail")
		}
	})

	t.Run("concurrent_global_flag_usage", func(t *testing.T) {
		const numGoroutines = 5
		var wg sync.WaitGroup
		results := make(chan bool, numGoroutines)

		// Test concurrent usage of global flags
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Create isolated command instance
				rootCmd := createIntegratedCLI()
				rootCmd.SetOut(io.Discard)
				rootCmd.SetErr(io.Discard)
				rootCmd.SetArgs([]string{"--verbose", "version"})

				// Use local verbose tracking to avoid race conditions
				var localVerbose bool
				rootCmd.PersistentFlags().BoolVarP(&localVerbose, "local-verbose", "", false, "local verbose")

				err := rootCmd.Execute()
				results <- err == nil
			}()
		}

		wg.Wait()
		close(results)

		// Check all results
		for success := range results {
			assert.True(t, success, "Concurrent global flag usage should succeed")
		}
	})
}

// TestCLIPerformance tests CLI performance characteristics
func TestCLIPerformance(t *testing.T) {
	t.Run("command_startup_time", func(t *testing.T) {
		// Test that commands start up quickly
		start := time.Now()

		rootCmd := createIntegratedCLI()
		rootCmd.SetOut(io.Discard)
		rootCmd.SetErr(io.Discard)
		rootCmd.SetArgs([]string{"version"})

		err := rootCmd.Execute()
		elapsed := time.Since(start)

		assert.NoError(t, err, "Command should execute successfully")
		assert.Less(t, elapsed, 100*time.Millisecond, "Command should start up quickly")
	})

	t.Run("help_generation_performance", func(t *testing.T) {
		// Test that help generation is fast
		commands := [][]string{
			{"--help"},
			{"convert", "--help"},
			{"download", "--help"},
			{"embed", "--help"},
			{"export", "--help"},
		}

		for _, args := range commands {
			start := time.Now()

			rootCmd := createIntegratedCLI()
			rootCmd.SetOut(io.Discard)
			rootCmd.SetErr(io.Discard)
			rootCmd.SetArgs(args)

			err := rootCmd.Execute()
			elapsed := time.Since(start)

			assert.NoError(t, err, "Help command should execute successfully")
			assert.Less(t, elapsed, 50*time.Millisecond, "Help generation should be fast for %v", args)
		}
	})
}

// TestCLIEnvironmentIsolation tests environment isolation
func TestCLIEnvironmentIsolation(t *testing.T) {
	t.Run("environment_variable_isolation", func(t *testing.T) {
		// Save original environment
		origOpenAI := os.Getenv("OPENAI_API_KEY")
		origGemini := os.Getenv("GEMINI_API_KEY")

		defer func() {
			// Restore environment
			if origOpenAI != "" {
				os.Setenv("OPENAI_API_KEY", origOpenAI)
			} else {
				os.Unsetenv("OPENAI_API_KEY")
			}
			if origGemini != "" {
				os.Setenv("GEMINI_API_KEY", origGemini)
			} else {
				os.Unsetenv("GEMINI_API_KEY")
			}
		}()

		// Test with different environment setups
		tests := []struct {
			name     string
			envSetup func()
			args     []string
		}{
			{
				name: "with_openai_key",
				envSetup: func() {
					os.Setenv("OPENAI_API_KEY", "test-key")
					os.Unsetenv("GEMINI_API_KEY")
				},
				args: []string{"embed", "generate", "--help"},
			},
			{
				name: "with_gemini_key",
				envSetup: func() {
					os.Unsetenv("OPENAI_API_KEY")
					os.Setenv("GEMINI_API_KEY", "test-key")
				},
				args: []string{"embed", "generate", "--help"},
			},
			{
				name: "no_keys",
				envSetup: func() {
					os.Unsetenv("OPENAI_API_KEY")
					os.Unsetenv("GEMINI_API_KEY")
				},
				args: []string{"embed", "generate", "--help"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tt.envSetup()

				rootCmd := createIntegratedCLI()
				rootCmd.SetOut(io.Discard)
				rootCmd.SetErr(io.Discard)
				rootCmd.SetArgs(tt.args)

				err := rootCmd.Execute()
				assert.NoError(t, err, "Command should work with different environment setups")
			})
		}
	})
}

// TestCLIUsabilityFeatures tests usability features
func TestCLIUsabilityFeatures(t *testing.T) {
	t.Run("help_text_quality", func(t *testing.T) {
		// Test that help text is informative and well-formatted
		rootCmd := createIntegratedCLI()
		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetArgs([]string{"--help"})

		err := rootCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Help should execute successfully")

		// Check for important help sections
		assert.Contains(t, output, "Usage:", "Help should contain usage")
		assert.Contains(t, output, "Available Commands:", "Help should list commands")
		assert.Contains(t, output, "Flags:", "Help should list flags")

		// Check for all expected commands
		expectedCommands := []string{"config", "convert", "download", "embed", "export", "version"}
		for _, cmd := range expectedCommands {
			assert.Contains(t, output, cmd, "Help should mention %s command", cmd)
		}
	})

	t.Run("command_discovery", func(t *testing.T) {
		// Test that users can discover commands easily
		rootCmd := createIntegratedCLI()

		// Test command listing
		commands := rootCmd.Commands()
		assert.Greater(t, len(commands), 5, "Should have multiple commands available")

		// Test that each command has proper metadata
		for _, cmd := range commands {
			if cmd.Name() != "help" && cmd.Name() != "completion" {
				assert.NotEmpty(t, cmd.Short, "Command %s should have short description", cmd.Name())
				assert.True(t, cmd.Runnable() || cmd.HasSubCommands(), "Command %s should be runnable or have subcommands", cmd.Name())
			}
		}
	})

	t.Run("error_message_quality", func(t *testing.T) {
		// Test that error messages are helpful
		tests := []struct {
			name     string
			args     []string
			expected string
		}{
			{
				name:     "unknown_command",
				args:     []string{"nonexistent"},
				expected: "Usage:", // Cobra shows help for unknown commands
			},
			{
				name:     "unknown_flag",
				args:     []string{"--nonexistent"},
				expected: "unknown flag",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rootCmd := createIntegratedCLI()
				var buf bytes.Buffer
				rootCmd.SetOut(&buf)
				rootCmd.SetErr(&buf)
				rootCmd.SetArgs(tt.args)

				err := rootCmd.Execute()
				output := buf.String()
				
				// Check if expected message appears in error or output
				var messageFound bool
				if err != nil {
					messageFound = strings.Contains(err.Error(), tt.expected) || strings.Contains(output, tt.expected)
				} else {
					// Some commands show help without error
					messageFound = strings.Contains(output, tt.expected)
				}
				assert.True(t, messageFound, "Expected message '%s' not found in error '%v' or output", tt.expected, err)
			})
		}
	})
}

// createIntegratedCLI creates a complete CLI with all commands for integration testing
func createIntegratedCLI() *cobra.Command {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "v2t",
		Short: "An application for batch converting video to text, supports tiktok and other video sites",
		Long: `An application for batch converting video to text, supports tiktok and other video sites or local video.
- First download all videos to local machine
- Call v2t to batch process the videos with local folder path
- The processed records will be saved to sqlite.`,
		TraverseChildren: true,
	}

	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "V", false, "verbose output")

	// Create and add all subcommands
	rootCmd.AddCommand(createIntegratedConfigCommand())
	rootCmd.AddCommand(createIntegratedConvertCommand())
	rootCmd.AddCommand(createIntegratedDownloadCommand())
	rootCmd.AddCommand(createIntegratedEmbedCommand())
	rootCmd.AddCommand(createIntegratedExportCommand())
	rootCmd.AddCommand(createIntegratedVersionCommand())

	return rootCmd
}

// Integrated command creators with mock implementations
func createIntegratedConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "A brief description of your command",
		Long:  `A longer description of the config command with examples.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Mock: config command executed")
		},
	}
}

func createIntegratedConvertCommand() *cobra.Command {
	var userNickname, directory, outputDirectory, fileExtension, inputFile string
	var video, audio bool
	var convertCount, parallel int

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Start converting the video files in the specified directory to text",
		Long:  `Start converting the video files in the specified directory to text`,
		Run: func(cmd *cobra.Command, args []string) {
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
			if video && directory != "" && userNickname == "" {
				cmd.PrintErrf("UserNickName must be set when converting video in directory\n")
				cmd.Help()
				return
			}

			if video {
				cmd.Printf("Mock: converting video files for user %s\n", userNickname)
			} else {
				cmd.Printf("Mock: converting audio files\n")
			}
		},
	}

	cmd.Flags().StringVarP(&userNickname, "userNickname", "u", "", "user nickname")
	cmd.Flags().StringVarP(&directory, "directory", "d", "", "directory path")
	cmd.Flags().StringVarP(&outputDirectory, "outputDirectory", "o", "./data/transcription", "output directory")
	cmd.Flags().IntVarP(&convertCount, "convertCount", "n", 1, "convert count")
	cmd.Flags().IntVarP(&parallel, "parallel", "p", 1, "parallel count")
	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "input file")
	cmd.Flags().StringVarP(&fileExtension, "type", "t", "", "file extension")
	cmd.Flags().BoolVarP(&video, "video", "v", false, "convert video")
	cmd.Flags().BoolVarP(&audio, "audio", "a", false, "convert audio")

	return cmd
}

func createIntegratedDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download podcasts from Small Universe or tiktok(unsupported now)",
		Long:  `Download podcasts from Small Universe or tiktok(unsupported now)`,
	}

	// Add xiaoyuzhou subcommand
	xiaoyuzhouCmd := createIntegratedXiaoyuzhouCommand()
	cmd.AddCommand(xiaoyuzhouCmd)

	return cmd
}

func createIntegratedXiaoyuzhouCommand() *cobra.Command {
	var downloadDir, podcast, episode string

	cmd := &cobra.Command{
		Use:   "xiaoyuzhou",
		Short: "Download podcasts from Small Universe",
		Long:  `Download podcasts from Small Universe`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if podcast == "" && episode == "" {
				return errors.New("please input a podcast or an episode")
			}
			if podcast != "" {
				cmd.Printf("Mock: downloading podcast from %s\n", podcast)
			} else {
				cmd.Printf("Mock: downloading episodes\n")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&downloadDir, "downloadDir", "d", "data/xiaoyuzhou", "download directory")
	cmd.Flags().StringVarP(&podcast, "podcast", "p", "", "podcast URL")
	cmd.Flags().StringVarP(&episode, "episode", "e", "", "episode URL")

	return cmd
}

func createIntegratedEmbedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Manage transcription embeddings",
		Long:  "Generate, manage, and analyze embeddings for transcriptions using OpenAI and Gemini",
	}

	// Add generate subcommand
	generateCmd := createIntegratedEmbedGenerateCommand()
	cmd.AddCommand(generateCmd)

	// Add status subcommand
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show embedding generation status",
		Long:  "Display the current status of embedding generation for transcriptions",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Mock: embedding status\n")
		},
	}
	cmd.AddCommand(statusCmd)

	return cmd
}

func createIntegratedEmbedGenerateCommand() *cobra.Command {
	var flagAll bool
	var flagUser string
	var flagBatchSize int
	var flagProvider string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate embeddings for transcriptions",
		Long:  `Generate embeddings for transcriptions using OpenAI and/or Gemini providers.`,
		Run: func(cmd *cobra.Command, args []string) {
			if !flagAll && flagUser == "" {
				cmd.Help()
				return
			}
			if flagAll {
				cmd.Printf("Mock: generating embeddings for all transcriptions\n")
			} else {
				cmd.Printf("Mock: generating embeddings for user %s\n", flagUser)
			}
		},
	}

	cmd.Flags().BoolVar(&flagAll, "all", false, "generate for all")
	cmd.Flags().StringVar(&flagUser, "user", "", "user to generate for")
	cmd.Flags().IntVar(&flagBatchSize, "batch-size", 10, "batch size")
	cmd.Flags().StringVar(&flagProvider, "provider", "both", "provider to use")

	return cmd
}

func createIntegratedExportCommand() *cobra.Command {
	var userNickname, outputFilePath string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the specified user's text to excel",
		Long:  `Export the specified user's text to excel`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Mock: exporting data for user %s to %s\n", userNickname, outputFilePath)
		},
	}

	cmd.Flags().StringVarP(&userNickname, "userNickname", "n", "", "user nickname")
	cmd.Flags().StringVarP(&outputFilePath, "outputFilePath", "o", "", "output file path")
	cmd.MarkFlagRequired("userNickname")
	cmd.MarkFlagRequired("outputFilePath")

	return cmd
}

func createIntegratedVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of video-to-text",
		Long:  `All software has versions. This is video-to-text's.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Mock: version v0.0.1")
			return nil
		},
	}
}

// BenchmarkCLIIntegration benchmarks integrated CLI performance
func BenchmarkCLIIntegration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rootCmd := createIntegratedCLI()
		rootCmd.SetArgs([]string{"version"})
		rootCmd.SetOut(io.Discard)
		rootCmd.SetErr(io.Discard)
		_ = rootCmd.Execute()
	}
}

// BenchmarkCLIHelpGeneration benchmarks help generation performance
func BenchmarkCLIHelpGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rootCmd := createIntegratedCLI()
		rootCmd.SetArgs([]string{"--help"})
		rootCmd.SetOut(io.Discard)
		rootCmd.SetErr(io.Discard)
		_ = rootCmd.Execute()
	}
}
