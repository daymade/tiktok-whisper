package export

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportCommand tests the export command basic functionality
func TestExportCommand(t *testing.T) {
	// Save original variables
	origUserNickname := userNickname
	origOutputFilePath := outputFilePath

	defer func() {
		// Restore original variables
		userNickname = origUserNickname
		outputFilePath = origOutputFilePath
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
			name:           "export_command_help",
			args:           []string{"--help"},
			expectedOutput: "Export the specified user's text to excel",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "export_missing_required_flags",
			args:           []string{},
			expectedOutput: "",
			expectedError:  "required flag",
			expectSuccess:  false,
			setupFunc:      func() { resetExportFlags() },
		},
		{
			name:           "export_missing_user_nickname",
			args:           []string{"--outputFilePath", "/test/output.xlsx"},
			expectedOutput: "",
			expectedError:  "required flag",
			expectSuccess:  false,
			setupFunc:      func() { resetExportFlags() },
		},
		{
			name:           "export_missing_output_file_path",
			args:           []string{"--userNickname", "test_user"},
			expectedOutput: "",
			expectedError:  "required flag",
			expectSuccess:  false,
			setupFunc:      func() { resetExportFlags() },
		},
		{
			name:           "export_with_all_required_flags",
			args:           []string{"--userNickname", "test_user", "--outputFilePath", "/test/output.xlsx"},
			expectedOutput: "Mock: exporting data for user test_user to /test/output.xlsx",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() { resetExportFlags() },
		},
		{
			name:           "export_with_short_flags",
			args:           []string{"-n", "short_user", "-o", "/short/output.xlsx"},
			expectedOutput: "Mock: exporting data for user short_user to /short/output.xlsx",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() { resetExportFlags() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			// Create a fresh command for each test
			testCmd := createTestExportCommand()

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

// TestExportCommandFlags tests flag parsing and validation
func TestExportCommandFlags(t *testing.T) {
	// Save original variables
	origUserNickname := userNickname
	origOutputFilePath := outputFilePath

	defer func() {
		// Restore original variables
		userNickname = origUserNickname
		outputFilePath = origOutputFilePath
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
			args: []string{"-n", "short_user"},
			expectedFlags: map[string]interface{}{
				"userNickname": "short_user",
			},
		},
		{
			name: "output_file_path_flag",
			args: []string{"--outputFilePath", "/test/output.xlsx"},
			expectedFlags: map[string]interface{}{
				"outputFilePath": "/test/output.xlsx",
			},
		},
		{
			name: "output_file_path_short_flag",
			args: []string{"-o", "/short/output.xlsx"},
			expectedFlags: map[string]interface{}{
				"outputFilePath": "/short/output.xlsx",
			},
		},
		{
			name: "both_flags_long",
			args: []string{"--userNickname", "full_user", "--outputFilePath", "/full/path/output.xlsx"},
			expectedFlags: map[string]interface{}{
				"userNickname":   "full_user",
				"outputFilePath": "/full/path/output.xlsx",
			},
		},
		{
			name: "both_flags_short",
			args: []string{"-n", "short_user", "-o", "/short/output.xlsx"},
			expectedFlags: map[string]interface{}{
				"userNickname":   "short_user",
				"outputFilePath": "/short/output.xlsx",
			},
		},
		{
			name: "unicode_user_name",
			args: []string{"--userNickname", "用户名测试", "--outputFilePath", "/test/unicode.xlsx"},
			expectedFlags: map[string]interface{}{
				"userNickname":   "用户名测试",
				"outputFilePath": "/test/unicode.xlsx",
			},
		},
		{
			name: "special_characters_in_path",
			args: []string{"--userNickname", "test", "--outputFilePath", "/test/path with spaces/output-file_v2.xlsx"},
			expectedFlags: map[string]interface{}{
				"userNickname":   "test",
				"outputFilePath": "/test/path with spaces/output-file_v2.xlsx",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetExportFlags()

			testCmd := createTestExportCommand()
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
				case "outputFilePath":
					assert.Equal(t, expectedValue, outputFilePath, "outputFilePath flag value mismatch")
				}
			}
		})
	}
}

// TestExportCommandStructure tests the command structure and metadata
func TestExportCommandStructure(t *testing.T) {
	testCmd := createTestExportCommand()

	t.Run("command_metadata", func(t *testing.T) {
		assert.Equal(t, "export", testCmd.Use, "Command use name mismatch")
		assert.NotEmpty(t, testCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, testCmd.Long, "Long description should not be empty")
		assert.Contains(t, testCmd.Short, "Export", "Short description should mention exporting")
		assert.Contains(t, testCmd.Short, "excel", "Short description should mention excel")
		assert.Contains(t, testCmd.Long, "Export all", "Long description should mention exporting all")
	})

	t.Run("command_runnable", func(t *testing.T) {
		assert.NotNil(t, testCmd.Run, "Command should have a Run function")
		assert.True(t, testCmd.Runnable(), "Command should be runnable")
	})

	t.Run("command_flags", func(t *testing.T) {
		flags := testCmd.Flags()
		assert.NotNil(t, flags, "Flags should be initialized")

		// Test that all expected flags are present
		expectedFlags := []string{"userNickname", "outputFilePath"}

		for _, flagName := range expectedFlags {
			flag := flags.Lookup(flagName)
			assert.NotNil(t, flag, "Flag '%s' should be defined", flagName)
		}
	})

	t.Run("required_flags", func(t *testing.T) {
		// Test that required flags are marked as required
		testCmd := createTestExportCommand()

		// This tests the flag setup, not execution
		userFlag := testCmd.Flags().Lookup("userNickname")
		outputFlag := testCmd.Flags().Lookup("outputFilePath")

		require.NotNil(t, userFlag, "userNickname flag should exist")
		require.NotNil(t, outputFlag, "outputFilePath flag should exist")

		// Note: Testing required flag enforcement requires executing the command
		// which is covered in other test cases
	})
}

// TestExportCommandValidation tests input validation logic
func TestExportCommandValidation(t *testing.T) {
	// Save original variables
	origUserNickname := userNickname
	origOutputFilePath := outputFilePath

	defer func() {
		// Restore original variables
		userNickname = origUserNickname
		outputFilePath = origOutputFilePath
	}()

	tests := []struct {
		name         string
		setupFlags   func()
		expectError  bool
		errorMessage string
	}{
		{
			name: "valid_flags",
			setupFlags: func() {
				resetExportFlags()
				userNickname = "valid_user"
				outputFilePath = "/valid/path.xlsx"
			},
			expectError:  false,
			errorMessage: "",
		},
		{
			name: "empty_user_nickname",
			setupFlags: func() {
				resetExportFlags()
				userNickname = ""
				outputFilePath = "/valid/path.xlsx"
			},
			expectError:  true,
			errorMessage: "required flag",
		},
		{
			name: "empty_output_file_path",
			setupFlags: func() {
				resetExportFlags()
				userNickname = "valid_user"
				outputFilePath = ""
			},
			expectError:  true,
			errorMessage: "required flag",
		},
		{
			name: "both_empty",
			setupFlags: func() {
				resetExportFlags()
				userNickname = ""
				outputFilePath = ""
			},
			expectError:  true,
			errorMessage: "required flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFlags()

			testCmd := createTestExportCommand()

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			// For valid_flags test, we need to pass the values as command-line args
			// since the command requires them to be set via flags
			if tt.name == "valid_flags" {
				testCmd.SetArgs([]string{"--userNickname", "valid_user", "--outputFilePath", "/valid/path.xlsx"})
			} else {
				// For other tests, build args based on the flag values
				args := []string{}
				if userNickname != "" {
					args = append(args, "--userNickname", userNickname)
				}
				if outputFilePath != "" {
					args = append(args, "--outputFilePath", outputFilePath)
				}
				testCmd.SetArgs(args)
			}

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

// TestExportCommandHelp tests the help functionality
func TestExportCommandHelp(t *testing.T) {
	testCmd := createTestExportCommand()

	var buf bytes.Buffer
	testCmd.SetOut(&buf)
	testCmd.SetArgs([]string{"--help"})

	err := testCmd.Execute()
	output := buf.String()

	assert.NoError(t, err, "Help command should not return an error")

	t.Run("help_contains_usage", func(t *testing.T) {
		assert.Contains(t, output, "Usage:", "Help should contain usage section")
		assert.Contains(t, output, "export", "Help should show command name")
	})

	t.Run("help_contains_description", func(t *testing.T) {
		assert.Contains(t, output, "Export the specified user's text to excel", "Help should contain main description")
		assert.Contains(t, output, "Export all the user's text", "Help should contain detailed description")
	})

	t.Run("help_contains_flags", func(t *testing.T) {
		assert.Contains(t, output, "Flags:", "Help should contain flags section")

		// Check for required flags
		expectedFlags := []string{
			"--userNickname", "-n",
			"--outputFilePath", "-o",
		}

		for _, flag := range expectedFlags {
			assert.Contains(t, output, flag, "Help should show %s flag", flag)
		}
	})

	t.Run("help_shows_required_flags", func(t *testing.T) {
		// Both flags should be marked as required in help
		assert.Contains(t, output, "userNickname", "Help should show userNickname flag")
		assert.Contains(t, output, "outputFilePath", "Help should show outputFilePath flag")
	})
}

// TestExportCommandErrorHandling tests error scenarios
func TestExportCommandErrorHandling(t *testing.T) {
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
			name:         "missing_required_userNickname",
			args:         []string{"--outputFilePath", "/test/output.xlsx"},
			expectError:  true,
			errorMessage: "required flag",
		},
		{
			name:         "missing_required_outputFilePath",
			args:         []string{"--userNickname", "test_user"},
			expectError:  true,
			errorMessage: "required flag",
		},
		{
			name:         "valid_arguments",
			args:         []string{"--userNickname", "test_user", "--outputFilePath", "/test/output.xlsx"},
			expectError:  false,
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExportFlags()

			testCmd := createTestExportCommand()
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

// TestExportCommandDefaults tests default flag values
func TestExportCommandDefaults(t *testing.T) {
	resetExportFlags()

	testCmd := createTestExportCommand()
	testCmd.SetOut(io.Discard)
	testCmd.SetErr(io.Discard)

	// Parse with no flags to test defaults
	err := testCmd.ParseFlags([]string{})
	assert.NoError(t, err, "Parsing empty flags should not fail")

	// Test default values
	assert.Equal(t, "", userNickname, "Default userNickname should be empty")
	assert.Equal(t, "", outputFilePath, "Default outputFilePath should be empty")
}

// TestExportCommandConcurrency tests concurrent execution
func TestExportCommandConcurrency(t *testing.T) {
	t.Run("concurrent_help_calls", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Launch concurrent help commands
		for i := 0; i < numGoroutines; i++ {
			go func() {
				resetExportFlags()
				testCmd := createTestExportCommand()
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

// TestExportCommandIntegration tests integration scenarios
func TestExportCommandIntegration(t *testing.T) {
	// Save original variables
	origUserNickname := userNickname
	origOutputFilePath := outputFilePath

	defer func() {
		// Restore original variables
		userNickname = origUserNickname
		outputFilePath = origOutputFilePath
	}()

	t.Run("command_in_parent_context", func(t *testing.T) {
		resetExportFlags()

		// Test the export command as it would be used in the actual CLI
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		exportCmd := createTestExportCommand()
		parentCmd.AddCommand(exportCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"export", "--userNickname", "test_user", "--outputFilePath", "/test/output.xlsx"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Export command in parent context should not fail")
		assert.Contains(t, output, "Mock: exporting data", "Expected output not found")
	})

	t.Run("command_with_global_flags", func(t *testing.T) {
		resetExportFlags()

		// Test export command with global flags
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		// Add a global flag
		var verbose bool
		parentCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

		exportCmd := createTestExportCommand()
		parentCmd.AddCommand(exportCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"--verbose", "export", "--userNickname", "test_user", "--outputFilePath", "/test/output.xlsx"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Export command with global flags should not fail")
		assert.Contains(t, output, "Mock: exporting data", "Expected output not found")
		assert.True(t, verbose, "Global verbose flag should be set")
	})
}

// TestExportCommandFilePathValidation tests file path handling
func TestExportCommandFilePathValidation(t *testing.T) {
	// Save original variables
	origUserNickname := userNickname
	origOutputFilePath := outputFilePath

	defer func() {
		// Restore original variables
		userNickname = origUserNickname
		outputFilePath = origOutputFilePath
	}()

	tests := []struct {
		name           string
		userNickname   string
		outputFilePath string
		expectSuccess  bool
	}{
		{
			name:           "xlsx_extension",
			userNickname:   "test_user",
			outputFilePath: "/test/output.xlsx",
			expectSuccess:  true,
		},
		{
			name:           "xls_extension",
			userNickname:   "test_user",
			outputFilePath: "/test/output.xls",
			expectSuccess:  true,
		},
		{
			name:           "no_extension",
			userNickname:   "test_user",
			outputFilePath: "/test/output",
			expectSuccess:  true,
		},
		{
			name:           "absolute_path",
			userNickname:   "test_user",
			outputFilePath: "/absolute/path/to/output.xlsx",
			expectSuccess:  true,
		},
		{
			name:           "relative_path",
			userNickname:   "test_user",
			outputFilePath: "./relative/output.xlsx",
			expectSuccess:  true,
		},
		{
			name:           "path_with_spaces",
			userNickname:   "test_user",
			outputFilePath: "/path with spaces/output file.xlsx",
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExportFlags()

			testCmd := createTestExportCommand()
			testCmd.SetArgs([]string{"--userNickname", tt.userNickname, "--outputFilePath", tt.outputFilePath})

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)

			err := testCmd.Execute()

			if tt.expectSuccess {
				assert.NoError(t, err, "Export command should succeed with valid paths")
			} else {
				assert.Error(t, err, "Export command should fail with invalid paths")
			}
		})
	}
}

// resetExportFlags resets all export command flags to their defaults
func resetExportFlags() {
	userNickname = ""
	outputFilePath = ""
}

// createTestExportCommand creates a fresh export command for testing
func createTestExportCommand() *cobra.Command {
	// Create a command similar to the original but with mocked business logic
	testCmd := &cobra.Command{
		Use:   "export",
		Short: "Export the specified user's text to excel",
		Long: `Export the specified user's text to excel

- Export all the user's text to excel, currently does not support a limited number`,
		Run: func(cmd *cobra.Command, args []string) {
			// Mock the business logic from the original command
			cmd.Printf("Mock: exporting data for user %s to %s\n", userNickname, outputFilePath)
		},
	}

	// Add all the flags as in the original
	testCmd.Flags().StringVarP(&userNickname, "userNickname", "n", "", "set userNickname")
	testCmd.Flags().StringVarP(&outputFilePath, "outputFilePath", "o", "", "set outputFilePath")

	// Mark flags as required
	testCmd.MarkFlagRequired("userNickname")
	testCmd.MarkFlagRequired("outputFilePath")

	return testCmd
}

// BenchmarkExportCommand benchmarks export command performance
func BenchmarkExportCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetExportFlags()
		testCmd := createTestExportCommand()
		testCmd.SetArgs([]string{"--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}

// BenchmarkExportCommandFlagParsing benchmarks flag parsing performance
func BenchmarkExportCommandFlagParsing(b *testing.B) {
	args := []string{
		"--userNickname", "benchmark_user",
		"--outputFilePath", "/benchmark/output.xlsx",
	}

	for i := 0; i < b.N; i++ {
		resetExportFlags()
		testCmd := createTestExportCommand()
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.ParseFlags(args)
	}
}
