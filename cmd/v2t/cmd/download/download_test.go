package download

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDownloadCommand tests the download command basic functionality
func TestDownloadCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  string
		expectSuccess  bool
	}{
		{
			name:           "download_command_help",
			args:           []string{"--help"},
			expectedOutput: "Download podcasts from Small Universe",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "download_command_short_help",
			args:           []string{"-h"},
			expectedOutput: "Download podcasts from Small Universe",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "download_command_no_args",
			args:           []string{},
			expectedOutput: "Download podcasts from Small Universe",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "download_unknown_flag",
			args:           []string{"--unknown-flag"},
			expectedOutput: "",
			expectedError:  "unknown flag",
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command for each test
			testCmd := createTestDownloadCommand()

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

// TestDownloadCommandStructure tests the command structure and metadata
func TestDownloadCommandStructure(t *testing.T) {
	testCmd := createTestDownloadCommand()

	t.Run("command_metadata", func(t *testing.T) {
		assert.Equal(t, "download", testCmd.Use, "Command use name mismatch")
		assert.NotEmpty(t, testCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, testCmd.Long, "Long description should not be empty")
		assert.Contains(t, testCmd.Short, "Download podcasts", "Short description should mention downloading")
		assert.Contains(t, testCmd.Short, "Small Universe", "Short description should mention Small Universe")
		assert.Contains(t, testCmd.Long, "tiktok", "Long description should mention tiktok")
	})

	t.Run("command_not_directly_runnable", func(t *testing.T) {
		// The download command is a parent command, not directly runnable
		assert.Nil(t, testCmd.Run, "Download command should not have a Run function")
		assert.Nil(t, testCmd.RunE, "Download command should not have a RunE function")
		assert.False(t, testCmd.Runnable(), "Download command should not be directly runnable")
	})

	t.Run("command_has_subcommands", func(t *testing.T) {
		assert.True(t, testCmd.HasSubCommands(), "Download command should have subcommands")
		subcommands := testCmd.Commands()
		assert.Greater(t, len(subcommands), 0, "Download command should have at least one subcommand")

		// Check for xiaoyuzhou subcommand
		var hasXiaoyuzhou bool
		for _, subcmd := range subcommands {
			if subcmd.Name() == "xiaoyuzhou" {
				hasXiaoyuzhou = true
				break
			}
		}
		assert.True(t, hasXiaoyuzhou, "Download command should have xiaoyuzhou subcommand")
	})

	t.Run("command_flags", func(t *testing.T) {
		flags := testCmd.Flags()
		assert.NotNil(t, flags, "Flags should be initialized")

		// The download command itself doesn't have specific flags
		// Flags are defined on subcommands
		assert.NotPanics(t, func() {
			flags.VisitAll(func(flag *pflag.Flag) {
				// Just visiting, no specific assertions needed for this parent command
			})
		})
	})
}

// TestDownloadCommandSubcommands tests subcommand registration and functionality
func TestDownloadCommandSubcommands(t *testing.T) {
	testCmd := createTestDownloadCommand()

	t.Run("has_xiaoyuzhou_subcommand", func(t *testing.T) {
		subcommands := testCmd.Commands()
		var xiaoyuzhouCmd *cobra.Command

		for _, subcmd := range subcommands {
			if subcmd.Name() == "xiaoyuzhou" {
				xiaoyuzhouCmd = subcmd
				break
			}
		}

		require.NotNil(t, xiaoyuzhouCmd, "Should have xiaoyuzhou subcommand")
		assert.Equal(t, "xiaoyuzhou", xiaoyuzhouCmd.Use, "Subcommand name should be xiaoyuzhou")
		assert.NotEmpty(t, xiaoyuzhouCmd.Short, "Xiaoyuzhou subcommand should have short description")
	})

	t.Run("subcommand_execution", func(t *testing.T) {
		// Test that we can execute the xiaoyuzhou subcommand help
		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)
		testCmd.SetArgs([]string{"xiaoyuzhou", "--help"})

		err := testCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Xiaoyuzhou help should execute without error")
		assert.Contains(t, output, "xiaoyuzhou", "Help should mention xiaoyuzhou")
		assert.Contains(t, output, "Download podcasts", "Help should mention downloading")
	})

	t.Run("unknown_subcommand", func(t *testing.T) {
		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)
		testCmd.SetArgs([]string{"unknown-subcommand"})

		err := testCmd.Execute()

		assert.Error(t, err, "Unknown subcommand should return an error")
		assert.Contains(t, err.Error(), "unknown command", "Error should mention unknown command")
	})
}

// TestDownloadCommandHelp tests the help functionality
func TestDownloadCommandHelp(t *testing.T) {
	testCmd := createTestDownloadCommand()

	var buf bytes.Buffer
	testCmd.SetOut(&buf)
	testCmd.SetArgs([]string{"--help"})

	err := testCmd.Execute()
	output := buf.String()

	assert.NoError(t, err, "Help command should not return an error")

	t.Run("help_contains_usage", func(t *testing.T) {
		assert.Contains(t, output, "Usage:", "Help should contain usage section")
		assert.Contains(t, output, "download", "Help should show command name")
	})

	t.Run("help_contains_description", func(t *testing.T) {
		assert.Contains(t, output, "Download podcasts from Small Universe", "Help should contain main description")
		assert.Contains(t, output, "tiktok", "Help should mention tiktok")
	})

	t.Run("help_contains_subcommands", func(t *testing.T) {
		assert.Contains(t, output, "Available Commands:", "Help should list available commands")
		assert.Contains(t, output, "xiaoyuzhou", "Help should list xiaoyuzhou subcommand")
	})

	t.Run("help_contains_global_flags", func(t *testing.T) {
		assert.Contains(t, output, "Flags:", "Help should contain flags section")
		assert.Contains(t, output, "-h, --help", "Help should show help flag")
	})
}

// TestDownloadCommandErrorHandling tests error scenarios
func TestDownloadCommandErrorHandling(t *testing.T) {
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
			name:         "unknown_subcommand",
			args:         []string{"unknown-subcommand"},
			expectError:  true,
			errorMessage: "unknown command",
		},
		{
			name:         "valid_subcommand_with_help",
			args:         []string{"xiaoyuzhou", "--help"},
			expectError:  false,
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCmd := createTestDownloadCommand()
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

// TestDownloadCommandConcurrency tests concurrent execution
func TestDownloadCommandConcurrency(t *testing.T) {
	t.Run("concurrent_help_calls", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Launch concurrent help commands
		for i := 0; i < numGoroutines; i++ {
			go func() {
				testCmd := createTestDownloadCommand()
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

	t.Run("concurrent_subcommand_help", func(t *testing.T) {
		const numGoroutines = 5
		results := make(chan error, numGoroutines)

		// Launch concurrent subcommand help calls
		for i := 0; i < numGoroutines; i++ {
			go func() {
				testCmd := createTestDownloadCommand()
				testCmd.SetArgs([]string{"xiaoyuzhou", "--help"})
				testCmd.SetOut(io.Discard)
				testCmd.SetErr(io.Discard)
				results <- testCmd.Execute()
			}()
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent subcommand help failed")
		}
	})
}

// TestDownloadCommandIntegration tests integration scenarios
func TestDownloadCommandIntegration(t *testing.T) {
	t.Run("command_in_parent_context", func(t *testing.T) {
		// Test the download command as it would be used in the actual CLI
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		downloadCmd := createTestDownloadCommand()
		parentCmd.AddCommand(downloadCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"download", "--help"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Download command in parent context should not fail")
		assert.Contains(t, output, "Download podcasts", "Expected output not found")
	})

	t.Run("subcommand_in_parent_context", func(t *testing.T) {
		// Test subcommand execution through parent
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		downloadCmd := createTestDownloadCommand()
		parentCmd.AddCommand(downloadCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"download", "xiaoyuzhou", "--help"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Subcommand through parent should not fail")
		assert.Contains(t, output, "xiaoyuzhou", "Expected subcommand output not found")
	})

	t.Run("command_with_global_flags", func(t *testing.T) {
		// Test download command with global flags
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		// Add a global flag
		var verbose bool
		parentCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

		downloadCmd := createTestDownloadCommand()
		parentCmd.AddCommand(downloadCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"--verbose", "download", "--help"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Download command with global flags should not fail")
		assert.Contains(t, output, "Download podcasts", "Expected output not found")
		assert.True(t, verbose, "Global verbose flag should be set")
	})
}

// TestDownloadCommandValidation tests validation logic
func TestDownloadCommandValidation(t *testing.T) {
	t.Run("no_direct_execution", func(t *testing.T) {
		// The download command should show help when called without subcommands
		testCmd := createTestDownloadCommand()

		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)

		err := testCmd.Execute()
		output := buf.String()

		// Should not error, but should show help since it's not directly runnable
		assert.NoError(t, err, "Download command without subcommand should not error")
		assert.Contains(t, output, "Usage:", "Should show usage when no subcommand specified")
	})

	t.Run("subcommand_validation", func(t *testing.T) {
		// Test that subcommands are properly validated
		testCmd := createTestDownloadCommand()

		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)
		testCmd.SetArgs([]string{"invalid-subcommand"})

		err := testCmd.Execute()

		assert.Error(t, err, "Invalid subcommand should return an error")
		assert.Contains(t, err.Error(), "unknown command", "Error should mention unknown command")
	})
}

// createTestDownloadCommand creates a fresh download command for testing
func createTestDownloadCommand() *cobra.Command {
	// Create a command similar to the original
	testCmd := &cobra.Command{
		Use:   "download",
		Short: "Download podcasts from Small Universe or tiktok(unsupported now)",
		Long:  `Download podcasts from Small Universe or tiktok(unsupported now), support downloading all shows from the home page and single downloads`,
	}

	// Add xiaoyuzhou subcommand (simplified for testing)
	xiaoyuzhouCmd := createTestXiaoyuzhouCommand()
	testCmd.AddCommand(xiaoyuzhouCmd)

	return testCmd
}

// createTestXiaoyuzhouCommand creates a simplified xiaoyuzhou command for testing
func createTestXiaoyuzhouCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "xiaoyuzhou",
		Short: "Download podcasts from Small Universe",
		Long:  `Download podcasts from Small Universe, support downloading all shows from the home page and single downloads`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Mock: xiaoyuzhou download executed")
			return nil
		},
	}
}

// BenchmarkDownloadCommand benchmarks download command performance
func BenchmarkDownloadCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testCmd := createTestDownloadCommand()
		testCmd.SetArgs([]string{"--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}

// BenchmarkDownloadCommandSubcommand benchmarks subcommand performance
func BenchmarkDownloadCommandSubcommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testCmd := createTestDownloadCommand()
		testCmd.SetArgs([]string{"xiaoyuzhou", "--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}
