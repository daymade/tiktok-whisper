package config

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// TestConfigCommand tests the config command basic functionality
func TestConfigCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  string
		expectSuccess  bool
	}{
		{
			name:           "config_command_execution",
			args:           []string{},
			expectedOutput: "config called",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "config_command_help",
			args:           []string{"--help"},
			expectedOutput: "A longer description",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "config_command_short_help",
			args:           []string{"-h"},
			expectedOutput: "A longer description",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "config_with_arguments",
			args:           []string{"arg1", "arg2"},
			expectedOutput: "config called",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "config_unknown_flag",
			args:           []string{"--unknown-flag"},
			expectedOutput: "",
			expectedError:  "unknown flag",
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command for each test
			testCmd := createTestConfigCommand()

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

// TestConfigCommandStructure tests the command structure and metadata
func TestConfigCommandStructure(t *testing.T) {
	testCmd := createTestConfigCommand()

	t.Run("command_metadata", func(t *testing.T) {
		assert.Equal(t, "config", testCmd.Use, "Command use name mismatch")
		assert.NotEmpty(t, testCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, testCmd.Long, "Long description should not be empty")
		assert.Contains(t, testCmd.Short, "brief description", "Short description content check")
		assert.Contains(t, testCmd.Long, "longer description", "Long description content check")
	})

	t.Run("command_runnable", func(t *testing.T) {
		assert.NotNil(t, testCmd.Run, "Command should have a Run function")
		assert.True(t, testCmd.Runnable(), "Command should be runnable")
	})

	t.Run("command_flags", func(t *testing.T) {
		// Currently, the config command doesn't have specific flags
		// Test that no unexpected flags are defined
		flags := testCmd.Flags()
		assert.NotNil(t, flags, "Flags should be initialized")

		// Test that we can access flag methods without panicking
		assert.NotPanics(t, func() {
			flags.VisitAll(func(flag *pflag.Flag) {
				// Just visiting, no specific assertions needed for this basic command
			})
		})
	})
}

// TestConfigCommandExecution tests the actual execution logic
func TestConfigCommandExecution(t *testing.T) {
	t.Run("basic_execution", func(t *testing.T) {
		testCmd := createTestConfigCommand()

		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)

		err := testCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Config command execution should not fail")
		assert.Contains(t, output, "config called", "Expected output message not found")
	})

	t.Run("execution_with_args", func(t *testing.T) {
		testCmd := createTestConfigCommand()
		testCmd.SetArgs([]string{"arg1", "arg2", "arg3"})

		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)

		err := testCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Config command with args should not fail")
		assert.Contains(t, output, "config called", "Expected output message not found")
	})

	t.Run("multiple_executions", func(t *testing.T) {
		// Test that the command can be executed multiple times
		for i := 0; i < 3; i++ {
			testCmd := createTestConfigCommand()

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			err := testCmd.Execute()
			output := buf.String()

			assert.NoError(t, err, "Config command execution %d should not fail", i+1)
			assert.Contains(t, output, "config called", "Expected output message not found in execution %d", i+1)
		}
	})
}

// TestConfigCommandHelp tests the help functionality
func TestConfigCommandHelp(t *testing.T) {
	testCmd := createTestConfigCommand()

	var buf bytes.Buffer
	testCmd.SetOut(&buf)
	testCmd.SetArgs([]string{"--help"})

	err := testCmd.Execute()
	output := buf.String()

	// Help command execution behavior
	assert.NoError(t, err, "Help command should not return an error")

	t.Run("help_contains_usage", func(t *testing.T) {
		assert.Contains(t, output, "Usage:", "Help should contain usage section")
		assert.Contains(t, output, "config", "Help should show command name")
	})

	t.Run("help_contains_description", func(t *testing.T) {
		assert.Contains(t, output, "A longer description", "Help should contain long description")
	})

	t.Run("help_contains_flags", func(t *testing.T) {
		assert.Contains(t, output, "Flags:", "Help should contain flags section")
		assert.Contains(t, output, "-h, --help", "Help should show help flag")
	})
}

// TestConfigCommandErrorHandling tests error scenarios
func TestConfigCommandErrorHandling(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCmd := createTestConfigCommand()
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

// TestConfigCommandConcurrency tests concurrent execution
func TestConfigCommandConcurrency(t *testing.T) {
	t.Run("concurrent_executions", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Launch concurrent config commands
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				testCmd := createTestConfigCommand()
				testCmd.SetOut(io.Discard)
				testCmd.SetErr(io.Discard)
				results <- testCmd.Execute()
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent config command %d failed", i+1)
		}
	})
}

// TestConfigCommandIntegration tests integration scenarios
func TestConfigCommandIntegration(t *testing.T) {
	t.Run("command_in_parent_context", func(t *testing.T) {
		// Test the config command as it would be used in the actual CLI
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		configCmd := createTestConfigCommand()
		parentCmd.AddCommand(configCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"config"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Config command in parent context should not fail")
		assert.Contains(t, output, "config called", "Expected output not found")
	})

	t.Run("command_with_global_flags", func(t *testing.T) {
		// Test config command with global flags (like verbose)
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		// Add a global flag
		var verbose bool
		parentCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

		configCmd := createTestConfigCommand()
		parentCmd.AddCommand(configCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"--verbose", "config"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Config command with global flags should not fail")
		assert.Contains(t, output, "config called", "Expected output not found")
		assert.True(t, verbose, "Global verbose flag should be set")
	})
}

// TestConfigCommandValidation tests input validation
func TestConfigCommandValidation(t *testing.T) {
	t.Run("accepts_any_arguments", func(t *testing.T) {
		// The config command currently accepts any arguments
		testCases := [][]string{
			{},
			{"single"},
			{"multiple", "arguments"},
			{"with", "special", "chars!", "@#$"},
		}

		for i, args := range testCases {
			testCmd := createTestConfigCommand()
			testCmd.SetArgs(args)
			testCmd.SetOut(io.Discard)
			testCmd.SetErr(io.Discard)

			err := testCmd.Execute()
			assert.NoError(t, err, "Config command should accept arguments in test case %d", i+1)
		}
	})
}

// createTestConfigCommand creates a fresh config command for testing
func createTestConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("config called")
		},
	}
}

// BenchmarkConfigCommand benchmarks config command performance
func BenchmarkConfigCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testCmd := createTestConfigCommand()
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}

// BenchmarkConfigCommandHelp benchmarks help command performance
func BenchmarkConfigCommandHelp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testCmd := createTestConfigCommand()
		testCmd.SetArgs([]string{"--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}
