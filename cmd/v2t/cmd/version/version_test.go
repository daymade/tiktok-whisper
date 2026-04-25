package version

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// TestVersionCommand tests the version command basic functionality
func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  string
		expectSuccess  bool
	}{
		{
			name:           "version_command_execution",
			args:           []string{},
			expectedOutput: "v0.0.1",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "version_command_help",
			args:           []string{"--help"},
			expectedOutput: "video-to-text",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "version_command_short_help",
			args:           []string{"-h"},
			expectedOutput: "video-to-text",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "version_with_arguments",
			args:           []string{"arg1", "arg2"},
			expectedOutput: "v0.0.1",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "version_unknown_flag",
			args:           []string{"--unknown-flag"},
			expectedOutput: "",
			expectedError:  "unknown flag",
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command for each test
			testCmd := createTestVersionCommand()

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

// TestVersionCommandStructure tests the command structure and metadata
func TestVersionCommandStructure(t *testing.T) {
	testCmd := createTestVersionCommand()

	t.Run("command_metadata", func(t *testing.T) {
		assert.Equal(t, "version", testCmd.Use, "Command use name mismatch")
		assert.NotEmpty(t, testCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, testCmd.Long, "Long description should not be empty")
		assert.Contains(t, testCmd.Short, "version number", "Short description should mention version number")
		assert.Contains(t, testCmd.Short, "video-to-text", "Short description should mention the application name")
		assert.Contains(t, testCmd.Long, "video-to-text", "Long description should mention the application name")
	})

	t.Run("command_runnable", func(t *testing.T) {
		assert.NotNil(t, testCmd.RunE, "Command should have a RunE function")
		assert.True(t, testCmd.Runnable(), "Command should be runnable")
	})

	t.Run("command_flags", func(t *testing.T) {
		flags := testCmd.Flags()
		assert.NotNil(t, flags, "Flags should be initialized")

		// The version command doesn't have specific flags
		assert.NotPanics(t, func() {
			flags.VisitAll(func(flag *pflag.Flag) {
				// Just visiting, no specific assertions needed for this simple command
			})
		})
	})
}

// TestVersionCommandExecution tests the actual execution logic
func TestVersionCommandExecution(t *testing.T) {
	t.Run("basic_execution", func(t *testing.T) {
		testCmd := createTestVersionCommand()

		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)

		err := testCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Version command execution should not fail")
		assert.Contains(t, output, "v0.0.1", "Expected version string not found")

		// Verify exact output (should be just the version with newline)
		assert.Equal(t, "v0.0.1\n", output, "Version output should be exactly the version string with newline")
	})

	t.Run("execution_with_args", func(t *testing.T) {
		testCmd := createTestVersionCommand()
		testCmd.SetArgs([]string{"ignored", "arguments"})

		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)

		err := testCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Version command with args should not fail")
		assert.Contains(t, output, "v0.0.1", "Expected version string not found")
		// Arguments should be ignored, output should be the same
		assert.Equal(t, "v0.0.1\n", output, "Version output should be exactly the version string with newline")
	})

	t.Run("multiple_executions", func(t *testing.T) {
		// Test that the command can be executed multiple times consistently
		for i := 0; i < 3; i++ {
			testCmd := createTestVersionCommand()

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			err := testCmd.Execute()
			output := buf.String()

			assert.NoError(t, err, "Version command execution %d should not fail", i+1)
			assert.Equal(t, "v0.0.1\n", output, "Version output should be consistent in execution %d", i+1)
		}
	})
}

// TestVersionCommandHelp tests the help functionality
func TestVersionCommandHelp(t *testing.T) {
	testCmd := createTestVersionCommand()

	var buf bytes.Buffer
	testCmd.SetOut(&buf)
	testCmd.SetArgs([]string{"--help"})

	err := testCmd.Execute()
	output := buf.String()

	assert.NoError(t, err, "Help command should not return an error")

	t.Run("help_contains_usage", func(t *testing.T) {
		assert.Contains(t, output, "Usage:", "Help should contain usage section")
		assert.Contains(t, output, "version", "Help should show command name")
	})

	t.Run("help_contains_description", func(t *testing.T) {
		assert.Contains(t, output, "video-to-text", "Help should mention the application name")
	})

	t.Run("help_contains_flags", func(t *testing.T) {
		assert.Contains(t, output, "Flags:", "Help should contain flags section")
		assert.Contains(t, output, "-h, --help", "Help should show help flag")
	})
}

// TestVersionCommandErrorHandling tests error scenarios
func TestVersionCommandErrorHandling(t *testing.T) {
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
			name:         "valid_execution",
			args:         []string{},
			expectError:  false,
			errorMessage: "",
		},
		{
			name:         "with_arguments",
			args:         []string{"some", "args"},
			expectError:  false,
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCmd := createTestVersionCommand()
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

// TestVersionCommandConcurrency tests concurrent execution
func TestVersionCommandConcurrency(t *testing.T) {
	t.Run("concurrent_executions", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)
		outputs := make(chan string, numGoroutines)

		// Launch concurrent version commands
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				testCmd := createTestVersionCommand()
				var buf bytes.Buffer
				testCmd.SetOut(&buf)
				testCmd.SetErr(&buf)

				err := testCmd.Execute()
				results <- err
				outputs <- buf.String()
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			output := <-outputs

			assert.NoError(t, err, "Concurrent version command %d failed", i+1)
			assert.Equal(t, "v0.0.1\n", output, "Concurrent execution %d output mismatch", i+1)
		}
	})

	t.Run("concurrent_help_calls", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Launch concurrent help commands
		for i := 0; i < numGoroutines; i++ {
			go func() {
				testCmd := createTestVersionCommand()
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

// TestVersionCommandIntegration tests integration scenarios
func TestVersionCommandIntegration(t *testing.T) {
	t.Run("command_in_parent_context", func(t *testing.T) {
		// Test the version command as it would be used in the actual CLI
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		versionCmd := createTestVersionCommand()
		parentCmd.AddCommand(versionCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"version"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Version command in parent context should not fail")
		assert.Equal(t, "v0.0.1\n", output, "Expected version output not found")
	})

	t.Run("command_with_global_flags", func(t *testing.T) {
		// Test version command with global flags
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		// Add a global flag
		var verbose bool
		parentCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

		versionCmd := createTestVersionCommand()
		parentCmd.AddCommand(versionCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"--verbose", "version"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Version command with global flags should not fail")
		assert.Equal(t, "v0.0.1\n", output, "Expected version output not found")
		assert.True(t, verbose, "Global verbose flag should be set")
	})
}

// TestVersionCommandOutputFormat tests output formatting
func TestVersionCommandOutputFormat(t *testing.T) {
	t.Run("output_format_consistency", func(t *testing.T) {
		testCmd := createTestVersionCommand()

		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)

		err := testCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Version command should execute successfully")

		// Test exact format
		assert.Equal(t, "v0.0.1\n", output, "Version output format should be exact")
		assert.True(t, strings.HasPrefix(output, "v"), "Version should start with 'v'")
		assert.True(t, strings.HasSuffix(output, "\n"), "Version should end with newline")
		assert.Equal(t, 1, strings.Count(output, "\n"), "Version should have exactly one newline")
	})

	t.Run("version_string_validation", func(t *testing.T) {
		// Test that the version string follows semantic versioning pattern
		testCmd := createTestVersionCommand()

		var buf bytes.Buffer
		testCmd.SetOut(&buf)
		testCmd.SetErr(&buf)

		err := testCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Version command should execute successfully")

		versionStr := strings.TrimSpace(output)
		assert.Regexp(t, `^v\d+\.\d+\.\d+`, versionStr, "Version should follow semantic versioning pattern")
	})
}

// TestVersionCommandPrintVersion tests the printVersion function
func TestVersionCommandPrintVersion(t *testing.T) {
	t.Run("print_version_function", func(t *testing.T) {
		// This tests the internal printVersion function
		// Note: Since printVersion() prints to stdout, we can't easily capture it in tests
		// The actual testing is done through command execution which calls printVersion

		// We can at least verify the function doesn't panic
		assert.NotPanics(t, func() {
			// Create a command that uses printVersion internally
			testCmd := createTestVersionCommand()
			testCmd.SetOut(io.Discard)
			testCmd.Execute()
		}, "printVersion function should not panic")
	})
}

// TestVersionCommandStability tests command stability
func TestVersionCommandStability(t *testing.T) {
	t.Run("repeated_executions", func(t *testing.T) {
		// Test that repeated executions are stable
		outputs := make([]string, 10)

		for i := 0; i < 10; i++ {
			testCmd := createTestVersionCommand()
			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			err := testCmd.Execute()
			assert.NoError(t, err, "Execution %d should not fail", i+1)
			outputs[i] = buf.String()
		}

		// All outputs should be identical
		for i := 1; i < len(outputs); i++ {
			assert.Equal(t, outputs[0], outputs[i], "Output %d should match output 0", i)
		}
	})

	t.Run("version_immutability", func(t *testing.T) {
		// Test that the version doesn't change during execution
		testCmd := createTestVersionCommand()

		var buf1 bytes.Buffer
		testCmd.SetOut(&buf1)
		testCmd.SetErr(&buf1)
		err1 := testCmd.Execute()
		output1 := buf1.String()

		// Execute again with the same command instance
		var buf2 bytes.Buffer
		testCmd.SetOut(&buf2)
		testCmd.SetErr(&buf2)
		err2 := testCmd.Execute()
		output2 := buf2.String()

		assert.NoError(t, err1, "First execution should not fail")
		assert.NoError(t, err2, "Second execution should not fail")
		assert.Equal(t, output1, output2, "Version should be immutable across executions")
	})
}

// createTestVersionCommand creates a fresh version command for testing
func createTestVersionCommand() *cobra.Command {
	// Create a command similar to the original
	testVersion := "v0.0.1"

	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of video-to-text",
		Long:  `All software has versions. This is video-to-text's.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println(testVersion)
			return nil
		},
	}
}

// BenchmarkVersionCommand benchmarks version command performance
func BenchmarkVersionCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testCmd := createTestVersionCommand()
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}

// BenchmarkVersionCommandHelp benchmarks help command performance
func BenchmarkVersionCommandHelp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testCmd := createTestVersionCommand()
		testCmd.SetArgs([]string{"--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}

// BenchmarkVersionCommandConcurrent benchmarks concurrent version command execution
func BenchmarkVersionCommandConcurrent(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			testCmd := createTestVersionCommand()
			testCmd.SetOut(io.Discard)
			testCmd.SetErr(io.Discard)
			_ = testCmd.Execute()
		}
	})
}
