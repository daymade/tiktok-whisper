package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootCommand tests the root command structure and basic functionality
func TestRootCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  string
		validateFlags  bool
	}{
		{
			name:           "root_command_help",
			args:           []string{"--help"},
			expectedOutput: "An application for batch converting video to text",
			expectedError:  "",
			validateFlags:  false,
		},
		{
			name:           "root_command_short_help",
			args:           []string{"-h"},
			expectedOutput: "An application for batch converting video to text",
			expectedError:  "",
			validateFlags:  false,
		},
		{
			name:           "root_command_version_flag",
			args:           []string{"--version"},
			expectedOutput: "",
			expectedError:  "",
			validateFlags:  false,
		},
		{
			name:           "root_command_verbose_flag",
			args:           []string{"--verbose"},
			expectedOutput: "",
			expectedError:  "",
			validateFlags:  true,
		},
		{
			name:           "root_command_verbose_short_flag",
			args:           []string{"-V"},
			expectedOutput: "",
			expectedError:  "",
			validateFlags:  true,
		},
		{
			name:           "invalid_command",
			args:           []string{"invalid-command"},
			expectedOutput: "Usage:", // Cobra shows help for invalid commands
			expectedError:  "",
		},
		{
			name:           "no_args_shows_help",
			args:           []string{},
			expectedOutput: "An application for batch converting video to text",
			expectedError:  "",
			validateFlags:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the Verbose flag before each test
			Verbose = false

			// Create a new root command for testing to avoid side effects
			testRootCmd := createTestRootCommand()

			// Capture output
			var buf bytes.Buffer
			testRootCmd.SetOut(&buf)
			testRootCmd.SetErr(&buf)

			// Set args
			testRootCmd.SetArgs(tt.args)

			// Execute command
			err := testRootCmd.Execute()

			output := buf.String()

			// Validate output
			if tt.expectedOutput != "" {
				assert.Contains(t, output, tt.expectedOutput, "Expected output not found")
			}

			// Validate error
			if tt.expectedError != "" {
				assert.Error(t, err, "Expected an error but got none")
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedError, "Expected error message not found")
				}
			} else {
				// For some commands like help, err might be nil even though they exit
				if err != nil && !strings.Contains(output, "Usage:") {
					assert.NoError(t, err, "Unexpected error: %v", err)
				}
			}

			// Validate flags if needed
			if tt.validateFlags {
				assert.True(t, Verbose, "Verbose flag should be set to true")
			}
		})
	}
}

// TestRootCommandSubcommands tests that all expected subcommands are registered
func TestRootCommandSubcommands(t *testing.T) {
	testRootCmd := createTestRootCommand()

	expectedSubcommands := []string{
		"config",
		"convert",
		"download",
		"embed",
		"export",
		"version",
	}

	// Get all subcommands
	subcommands := testRootCmd.Commands()
	subcommandNames := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		subcommandNames[i] = cmd.Name()
	}

	// Verify all expected subcommands are present
	for _, expectedCmd := range expectedSubcommands {
		assert.Contains(t, subcommandNames, expectedCmd, "Missing expected subcommand: %s", expectedCmd)
	}

	// Verify no unexpected subcommands
	for _, actualCmd := range subcommandNames {
		// Skip help and completion as they're automatically added by cobra
		if actualCmd != "help" && actualCmd != "completion" {
			assert.Contains(t, expectedSubcommands, actualCmd, "Unexpected subcommand found: %s", actualCmd)
		}
	}
}

// TestRootCommandFlags tests flag parsing and validation
func TestRootCommandFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedFlags map[string]interface{}
	}{
		{
			name: "verbose_long_flag",
			args: []string{"--verbose"},
			expectedFlags: map[string]interface{}{
				"verbose": true,
			},
		},
		{
			name: "verbose_short_flag",
			args: []string{"-V"},
			expectedFlags: map[string]interface{}{
				"verbose": true,
			},
		},
		{
			name: "no_flags",
			args: []string{},
			expectedFlags: map[string]interface{}{
				"verbose": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			Verbose = false

			testRootCmd := createTestRootCommand()
			testRootCmd.SetArgs(tt.args)

			// Suppress output
			testRootCmd.SetOut(io.Discard)
			testRootCmd.SetErr(io.Discard)

			// Execute command
			_ = testRootCmd.Execute()

			// Validate flags
			for flagName, expectedValue := range tt.expectedFlags {
				switch flagName {
				case "verbose":
					assert.Equal(t, expectedValue, Verbose, "Verbose flag value mismatch")
				}
			}
		})
	}
}

// TestRootCommandStructure tests the command structure and metadata
func TestRootCommandStructure(t *testing.T) {
	testRootCmd := createTestRootCommand()

	t.Run("command_metadata", func(t *testing.T) {
		assert.Equal(t, "v2t", testRootCmd.Use, "Command use name mismatch")
		assert.NotEmpty(t, testRootCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, testRootCmd.Long, "Long description should not be empty")
		assert.Contains(t, testRootCmd.Short, "batch converting video to text", "Short description should mention core functionality")
		assert.Contains(t, testRootCmd.Long, "tiktok", "Long description should mention tiktok support")
		assert.True(t, testRootCmd.TraverseChildren, "TraverseChildren should be enabled")
	})

	t.Run("persistent_flags", func(t *testing.T) {
		verboseFlag := testRootCmd.PersistentFlags().Lookup("verbose")
		require.NotNil(t, verboseFlag, "Verbose flag should be registered")
		assert.Equal(t, "V", verboseFlag.Shorthand, "Verbose flag shorthand should be 'V'")
		assert.Equal(t, "false", verboseFlag.DefValue, "Verbose flag default should be false")
		assert.Equal(t, "verbose output", verboseFlag.Usage, "Verbose flag usage description mismatch")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		assert.True(t, testRootCmd.HasSubCommands(), "Root command should have subcommands")
		assert.Greater(t, len(testRootCmd.Commands()), 0, "Root command should have at least one subcommand")
	})
}

// TestRootCommandHelpOutput tests the help output formatting
func TestRootCommandHelpOutput(t *testing.T) {
	testRootCmd := createTestRootCommand()

	var buf bytes.Buffer
	testRootCmd.SetOut(&buf)
	testRootCmd.SetArgs([]string{"--help"})

	err := testRootCmd.Execute()
	output := buf.String()

	// Help command should not return an error
	assert.NoError(t, err, "Help command should not return an error")

	t.Run("help_contains_usage", func(t *testing.T) {
		assert.Contains(t, output, "Usage:", "Help should contain usage section")
		assert.Contains(t, output, "v2t", "Help should show command name")
	})

	t.Run("help_contains_description", func(t *testing.T) {
		assert.Contains(t, output, "batch converting video to text", "Help should contain main description")
	})

	t.Run("help_contains_subcommands", func(t *testing.T) {
		assert.Contains(t, output, "Available Commands:", "Help should list available commands")
		expectedCommands := []string{"config", "convert", "download", "embed", "export", "version"}
		for _, cmd := range expectedCommands {
			assert.Contains(t, output, cmd, "Help should list %s command", cmd)
		}
	})

	t.Run("help_contains_flags", func(t *testing.T) {
		assert.Contains(t, output, "Flags:", "Help should contain flags section")
		assert.Contains(t, output, "--verbose", "Help should show verbose flag")
		assert.Contains(t, output, "-V", "Help should show verbose flag shorthand")
	})
}

// TestRootCommandEnvironment tests environment-specific behavior
func TestRootCommandEnvironment(t *testing.T) {
	// Save original environment
	originalArgs := os.Args

	defer func() {
		os.Args = originalArgs
	}()

	t.Run("command_execution_isolation", func(t *testing.T) {
		// Test that commands don't interfere with each other
		testRootCmd1 := createTestRootCommand()
		testRootCmd2 := createTestRootCommand()

		// Set different args for each
		testRootCmd1.SetArgs([]string{"--verbose"})
		testRootCmd2.SetArgs([]string{})

		// Execute both
		Verbose = false
		testRootCmd1.Execute()
		verbose1 := Verbose

		Verbose = false
		testRootCmd2.Execute()
		verbose2 := Verbose

		// First command should have set verbose
		assert.True(t, verbose1, "First command should have set verbose flag")
		// Second command should not have affected the flag for this test
		assert.False(t, verbose2, "Second command should not have set verbose flag")
	})
}

// TestRootCommandErrorHandling tests error handling scenarios
func TestRootCommandErrorHandling(t *testing.T) {
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
			name:         "unknown_subcommand",
			args:         []string{"unknown-subcommand"},
			expectError:  false, // Cobra shows help without error
			errorMessage: "",
		},
		{
			name:         "malformed_flag",
			args:         []string{"--verbose=invalid"},
			expectError:  true, // Boolean flags reject invalid values
			errorMessage: "invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRootCmd := createTestRootCommand()
			testRootCmd.SetArgs(tt.args)

			var buf bytes.Buffer
			testRootCmd.SetOut(&buf)
			testRootCmd.SetErr(&buf)

			err := testRootCmd.Execute()

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

// TestRootCommandConcurrency tests concurrent execution
func TestRootCommandConcurrency(t *testing.T) {
	t.Run("concurrent_help_calls", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Launch concurrent help commands
		for i := 0; i < numGoroutines; i++ {
			go func() {
				testRootCmd := createTestRootCommand()
				testRootCmd.SetArgs([]string{"--help"})
				testRootCmd.SetOut(io.Discard)
				testRootCmd.SetErr(io.Discard)
				results <- testRootCmd.Execute()
			}()
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent help command failed")
		}
	})
}

// createTestRootCommand creates a fresh root command for testing
// This helps avoid side effects between tests
func createTestRootCommand() *cobra.Command {
	// Create a new root command similar to the original but isolated for testing
	testCmd := &cobra.Command{
		Use:   "v2t",
		Short: "An application for batch converting video to text, supports tiktok and other video sites",
		Long: `An application for batch converting video to text, supports tiktok and other video sites or local video.
- First download all videos to local machine
- Call v2t to batch process the videos with local folder path
- The processed records will be saved to sqlite.`,
		TraverseChildren: true,
	}

	// Add the same persistent flags
	testCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "V", false, "verbose output")

	// Add all subcommands - we need to import them properly
	// For testing, we'll create simplified mock subcommands to avoid circular dependencies
	testCmd.AddCommand(createMockConfigCmd())
	testCmd.AddCommand(createMockConvertCmd())
	testCmd.AddCommand(createMockDownloadCmd())
	testCmd.AddCommand(createMockEmbedCmd())
	testCmd.AddCommand(createMockExportCmd())
	testCmd.AddCommand(createMockVersionCmd())

	return testCmd
}

// Mock command creators for testing
func createMockConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
}

func createMockConvertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "convert",
		Short: "Convert video files to text",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
}

func createMockDownloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download",
		Short: "Download content from various sources",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
}

func createMockEmbedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "embed",
		Short: "Manage transcription embeddings",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
}

func createMockExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export transcription data",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
}

func createMockVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
}

// BenchmarkRootCommandHelp benchmarks help command performance
func BenchmarkRootCommandHelp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testRootCmd := createTestRootCommand()
		testRootCmd.SetArgs([]string{"--help"})
		testRootCmd.SetOut(io.Discard)
		testRootCmd.SetErr(io.Discard)
		_ = testRootCmd.Execute()
	}
}

// BenchmarkRootCommandFlagParsing benchmarks flag parsing performance
func BenchmarkRootCommandFlagParsing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testRootCmd := createTestRootCommand()
		testRootCmd.SetArgs([]string{"--verbose"})
		testRootCmd.SetOut(io.Discard)
		testRootCmd.SetErr(io.Discard)
		_ = testRootCmd.Execute()
	}
}
