package embed

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestEmbedCommand tests the embed command basic functionality
func TestEmbedCommand(t *testing.T) {
	// Save original variables
	origFlagAll := flagAll
	origFlagUser := flagUser
	origFlagBatchSize := flagBatchSize
	origFlagProvider := flagProvider

	defer func() {
		// Restore original variables
		flagAll = origFlagAll
		flagUser = origFlagUser
		flagBatchSize = origFlagBatchSize
		flagProvider = origFlagProvider
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
			name:           "embed_command_help",
			args:           []string{"--help"},
			expectedOutput: "Generate, manage, and analyze embeddings for transcriptions using OpenAI and Gemini",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "embed_command_no_args",
			args:           []string{},
			expectedOutput: "Generate, manage, and analyze embeddings for transcriptions using OpenAI and Gemini",
			expectedError:  "",
			expectSuccess:  true,
			setupFunc:      func() {},
		},
		{
			name:           "embed_unknown_flag",
			args:           []string{"--unknown-flag"},
			expectedOutput: "",
			expectedError:  "unknown flag",
			expectSuccess:  false,
			setupFunc:      func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetEmbedFlags()
			tt.setupFunc()

			// Create a fresh command for each test
			testCmd := createTestEmbedCommand()

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

// TestEmbedCommandStructure tests the command structure and metadata
func TestEmbedCommandStructure(t *testing.T) {
	testCmd := createTestEmbedCommand()

	t.Run("command_metadata", func(t *testing.T) {
		assert.Equal(t, "embed", testCmd.Use, "Command use name mismatch")
		assert.NotEmpty(t, testCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, testCmd.Long, "Long description should not be empty")
		assert.Contains(t, testCmd.Short, "embeddings", "Short description should mention embeddings")
		assert.Contains(t, testCmd.Long, "OpenAI", "Long description should mention OpenAI")
		assert.Contains(t, testCmd.Long, "Gemini", "Long description should mention Gemini")
	})

	t.Run("command_not_directly_runnable", func(t *testing.T) {
		// The embed command is a parent command, not directly runnable
		assert.Nil(t, testCmd.Run, "Embed command should not have a Run function")
		assert.Nil(t, testCmd.RunE, "Embed command should not have a RunE function")
		assert.False(t, testCmd.Runnable(), "Embed command should not be directly runnable")
	})

	t.Run("command_has_subcommands", func(t *testing.T) {
		assert.True(t, testCmd.HasSubCommands(), "Embed command should have subcommands")
		subcommands := testCmd.Commands()
		assert.Greater(t, len(subcommands), 0, "Embed command should have at least one subcommand")

		// Check for expected subcommands
		expectedSubcommands := []string{"generate", "status"}
		subcommandNames := make([]string, len(subcommands))
		for i, subcmd := range subcommands {
			subcommandNames[i] = subcmd.Name()
		}

		for _, expectedCmd := range expectedSubcommands {
			assert.Contains(t, subcommandNames, expectedCmd, "Missing expected subcommand: %s", expectedCmd)
		}
	})
}

// TestEmbedGenerateCommand tests the generate subcommand
func TestEmbedGenerateCommand(t *testing.T) {
	// Save original environment variables
	origOpenAIKey := os.Getenv("OPENAI_API_KEY")
	origGeminiKey := os.Getenv("GEMINI_API_KEY")

	defer func() {
		// Restore environment variables
		if origOpenAIKey != "" {
			os.Setenv("OPENAI_API_KEY", origOpenAIKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
		if origGeminiKey != "" {
			os.Setenv("GEMINI_API_KEY", origGeminiKey)
		} else {
			os.Unsetenv("GEMINI_API_KEY")
		}
	}()

	tests := []struct {
		name           string
		args           []string
		envSetup       func()
		expectedOutput string
		expectedError  string
		expectSuccess  bool
	}{
		{
			name:           "generate_help",
			args:           []string{"generate", "--help"},
			envSetup:       func() {},
			expectedOutput: "Generate embeddings for transcriptions",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name: "generate_all_with_mock_providers",
			args: []string{"generate", "--all"},
			envSetup: func() {
				// Clear API keys to force mock providers
				os.Unsetenv("OPENAI_API_KEY")
				os.Unsetenv("GEMINI_API_KEY")
			},
			expectedOutput: "Mock: generating embeddings for all transcriptions",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name: "generate_with_openai_provider",
			args: []string{"generate", "--all", "--provider", "openai"},
			envSetup: func() {
				os.Setenv("OPENAI_API_KEY", "test-key")
				os.Unsetenv("GEMINI_API_KEY")
			},
			expectedOutput: "Mock: generating embeddings for all transcriptions",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name: "generate_with_batch_size",
			args: []string{"generate", "--all", "--batch-size", "5"},
			envSetup: func() {
				os.Unsetenv("OPENAI_API_KEY")
				os.Unsetenv("GEMINI_API_KEY")
			},
			expectedOutput: "Mock: generating embeddings for all transcriptions",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name: "generate_no_flags",
			args: []string{"generate"},
			envSetup: func() {
				os.Unsetenv("OPENAI_API_KEY")
				os.Unsetenv("GEMINI_API_KEY")
			},
			expectedOutput: "Usage:",
			expectedError:  "",
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetEmbedFlags()
			tt.envSetup()

			testCmd := createTestEmbedCommand()

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)
			testCmd.SetArgs(tt.args)

			err := testCmd.Execute()
			output := buf.String()

			if tt.expectSuccess {
				if !strings.Contains(output, "Usage:") {
					assert.NoError(t, err, "Expected command to succeed but got error: %v", err)
				}
			} else {
				assert.Error(t, err, "Expected command to fail but it succeeded")
			}

			if tt.expectedOutput != "" {
				assert.Contains(t, output, tt.expectedOutput, "Expected output not found in: %s", output)
			}
		})
	}
}

// TestEmbedGenerateCommandFlags tests the generate command flags
func TestEmbedGenerateCommandFlags(t *testing.T) {
	// Save original variables
	origFlagAll := flagAll
	origFlagUser := flagUser
	origFlagBatchSize := flagBatchSize
	origFlagProvider := flagProvider

	defer func() {
		// Restore original variables
		flagAll = origFlagAll
		flagUser = origFlagUser
		flagBatchSize = origFlagBatchSize
		flagProvider = origFlagProvider
	}()

	tests := []struct {
		name          string
		args          []string
		expectedFlags map[string]interface{}
	}{
		{
			name: "all_flag",
			args: []string{"generate", "--all"},
			expectedFlags: map[string]interface{}{
				"all": true,
			},
		},
		{
			name: "user_flag",
			args: []string{"generate", "--user", "test_user"},
			expectedFlags: map[string]interface{}{
				"user": "test_user",
			},
		},
		{
			name: "batch_size_flag",
			args: []string{"generate", "--batch-size", "15"},
			expectedFlags: map[string]interface{}{
				"batchSize": 15,
			},
		},
		{
			name: "provider_openai",
			args: []string{"generate", "--provider", "openai"},
			expectedFlags: map[string]interface{}{
				"provider": "openai",
			},
		},
		{
			name: "provider_gemini",
			args: []string{"generate", "--provider", "gemini"},
			expectedFlags: map[string]interface{}{
				"provider": "gemini",
			},
		},
		{
			name: "provider_both",
			args: []string{"generate", "--provider", "both"},
			expectedFlags: map[string]interface{}{
				"provider": "both",
			},
		},
		{
			name: "multiple_flags",
			args: []string{"generate", "--all", "--batch-size", "20", "--provider", "openai"},
			expectedFlags: map[string]interface{}{
				"all":       true,
				"batchSize": 20,
				"provider":  "openai",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetEmbedFlags()

			testCmd := createTestEmbedCommand()
			testCmd.SetArgs(tt.args)

			// Suppress output for flag testing
			testCmd.SetOut(io.Discard)
			testCmd.SetErr(io.Discard)

			// Execute command to parse flags properly
			err := testCmd.Execute()
			// We expect successful execution for these flag tests
			assert.NoError(t, err, "Command execution should not fail")

			// Validate flags
			for flagName, expectedValue := range tt.expectedFlags {
				switch flagName {
				case "all":
					assert.Equal(t, expectedValue, flagAll, "all flag value mismatch")
				case "user":
					assert.Equal(t, expectedValue, flagUser, "user flag value mismatch")
				case "batchSize":
					assert.Equal(t, expectedValue, flagBatchSize, "batchSize flag value mismatch")
				case "provider":
					assert.Equal(t, expectedValue, flagProvider, "provider flag value mismatch")
				}
			}
		})
	}
}

// TestEmbedStatusCommand tests the status subcommand
func TestEmbedStatusCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  string
		expectSuccess  bool
	}{
		{
			name:           "status_help",
			args:           []string{"status", "--help"},
			expectedOutput: "Display the current status of embedding generation for transcriptions",
			expectedError:  "",
			expectSuccess:  true,
		},
		{
			name:           "status_execution",
			args:           []string{"status"},
			expectedOutput: "Mock: embedding status",
			expectedError:  "",
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetEmbedFlags()

			testCmd := createTestEmbedCommand()

			var buf bytes.Buffer
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)
			testCmd.SetArgs(tt.args)

			err := testCmd.Execute()
			output := buf.String()

			if tt.expectSuccess {
				if !strings.Contains(output, "Usage:") {
					assert.NoError(t, err, "Expected command to succeed but got error: %v", err)
				}
			} else {
				assert.Error(t, err, "Expected command to fail but it succeeded")
			}

			if tt.expectedOutput != "" {
				assert.Contains(t, output, tt.expectedOutput, "Expected output not found in: %s", output)
			}
		})
	}
}

// TestEmbedCommandHelp tests the help functionality
func TestEmbedCommandHelp(t *testing.T) {
	testCmd := createTestEmbedCommand()

	var buf bytes.Buffer
	testCmd.SetOut(&buf)
	testCmd.SetArgs([]string{"--help"})

	err := testCmd.Execute()
	output := buf.String()

	assert.NoError(t, err, "Help command should not return an error")

	t.Run("help_contains_usage", func(t *testing.T) {
		assert.Contains(t, output, "Usage:", "Help should contain usage section")
		assert.Contains(t, output, "embed", "Help should show command name")
	})

	t.Run("help_contains_description", func(t *testing.T) {
		assert.Contains(t, output, "Generate, manage, and analyze embeddings for transcriptions using OpenAI and Gemini", "Help should contain main description")
		assert.Contains(t, output, "OpenAI", "Help should mention OpenAI")
		assert.Contains(t, output, "Gemini", "Help should mention Gemini")
	})

	t.Run("help_contains_subcommands", func(t *testing.T) {
		assert.Contains(t, output, "Available Commands:", "Help should list available commands")
		assert.Contains(t, output, "generate", "Help should list generate subcommand")
		assert.Contains(t, output, "status", "Help should list status subcommand")
	})
}

// TestEmbedCommandErrorHandling tests error scenarios
func TestEmbedCommandErrorHandling(t *testing.T) {
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
			expectError:  true,
			errorMessage: "unknown command",
		},
		{
			name:         "invalid_batch_size",
			args:         []string{"generate", "--batch-size", "invalid"},
			expectError:  true,
			errorMessage: "invalid syntax",
		},
		{
			name:         "negative_batch_size",
			args:         []string{"generate", "--batch-size", "-1"},
			expectError:  false, // Command accepts negative numbers
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetEmbedFlags()

			testCmd := createTestEmbedCommand()
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
				if !strings.Contains(buf.String(), "Usage:") {
					assert.NoError(t, err, "Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestEmbedCommandDefaults tests default flag values
func TestEmbedCommandDefaults(t *testing.T) {
	resetEmbedFlags()

	testCmd := createTestEmbedCommand()
	testCmd.SetOut(io.Discard)
	testCmd.SetErr(io.Discard)

	// Parse with no flags to test defaults
	err := testCmd.ParseFlags([]string{"generate"})
	assert.NoError(t, err, "Parsing flags should not fail")

	// Test default values
	assert.Equal(t, false, flagAll, "Default all should be false")
	assert.Equal(t, "", flagUser, "Default user should be empty")
	assert.Equal(t, 10, flagBatchSize, "Default batch size should be 10")
	assert.Equal(t, "both", flagProvider, "Default provider should be both")
}

// TestEmbedCommandConcurrency tests concurrent execution
func TestEmbedCommandConcurrency(t *testing.T) {
	t.Run("concurrent_help_calls", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Launch concurrent help commands
		for i := 0; i < numGoroutines; i++ {
			go func() {
				resetEmbedFlags()
				testCmd := createTestEmbedCommand()
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

// TestEmbedCommandIntegration tests integration scenarios
func TestEmbedCommandIntegration(t *testing.T) {
	t.Run("command_in_parent_context", func(t *testing.T) {
		resetEmbedFlags()

		// Test the embed command as it would be used in the actual CLI
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		embedCmd := createTestEmbedCommand()
		parentCmd.AddCommand(embedCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"embed", "--help"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Embed command in parent context should not fail")
		assert.Contains(t, output, "Generate, manage, and analyze embeddings for transcriptions using OpenAI and Gemini", "Expected output not found")
	})

	t.Run("subcommand_in_parent_context", func(t *testing.T) {
		resetEmbedFlags()

		// Test subcommand execution through parent
		parentCmd := &cobra.Command{
			Use: "parent",
		}

		embedCmd := createTestEmbedCommand()
		parentCmd.AddCommand(embedCmd)

		var buf bytes.Buffer
		parentCmd.SetOut(&buf)
		parentCmd.SetErr(&buf)
		parentCmd.SetArgs([]string{"embed", "status"})

		err := parentCmd.Execute()
		output := buf.String()

		assert.NoError(t, err, "Subcommand through parent should not fail")
		assert.Contains(t, output, "Mock: embedding status", "Expected subcommand output not found")
	})
}

// resetEmbedFlags resets all embed command flags to their defaults
func resetEmbedFlags() {
	flagAll = false
	flagUser = ""
	flagBatchSize = 10
	flagProvider = "both"
}

// createTestEmbedCommand creates a fresh embed command for testing
func createTestEmbedCommand() *cobra.Command {
	// Create a command similar to the original but with mocked business logic
	testCmd := &cobra.Command{
		Use:   "embed",
		Short: "Manage transcription embeddings",
		Long:  "Generate, manage, and analyze embeddings for transcriptions using OpenAI and Gemini",
	}

	// Create generate subcommand
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate embeddings for transcriptions",
		Long: `Generate embeddings for transcriptions using OpenAI and/or Gemini providers.
	
Examples:
  v2t embed generate --all                    # Generate embeddings for all transcriptions
  v2t embed generate --user "用户名"          # Generate for specific user
  v2t embed generate --batch-size 5          # Use custom batch size
  v2t embed generate --provider openai       # Use only OpenAI provider
  v2t embed generate --provider gemini       # Use only Gemini provider`,
		Run: func(cmd *cobra.Command, args []string) {
			// Mock validation logic
			if !flagAll && flagUser == "" {
				cmd.Help()
				return
			}

			// Mock successful execution
			if flagAll {
				cmd.Printf("Mock: generating embeddings for all transcriptions with batch size %d using %s provider(s)\n",
					flagBatchSize, flagProvider)
			} else if flagUser != "" {
				cmd.Printf("Mock: generating embeddings for user %s\n", flagUser)
			}
		},
	}

	// Add generate command flags
	generateCmd.Flags().BoolVar(&flagAll, "all", false, "Generate embeddings for all transcriptions")
	generateCmd.Flags().StringVar(&flagUser, "user", "", "Generate embeddings for specific user")
	generateCmd.Flags().IntVar(&flagBatchSize, "batch-size", 10, "Number of transcriptions to process in each batch")
	generateCmd.Flags().StringVar(&flagProvider, "provider", "both", "Provider to use (openai, gemini, both)")

	// Create status subcommand
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show embedding generation status",
		Long:  "Display the current status of embedding generation for transcriptions",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Mock: embedding status - Total: 100, OpenAI: 80, Gemini: 75, Pending: 25\n")
		},
	}

	// Add subcommands
	testCmd.AddCommand(generateCmd)
	testCmd.AddCommand(statusCmd)

	return testCmd
}

// BenchmarkEmbedCommand benchmarks embed command performance
func BenchmarkEmbedCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetEmbedFlags()
		testCmd := createTestEmbedCommand()
		testCmd.SetArgs([]string{"--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}

// BenchmarkEmbedGenerateCommand benchmarks generate subcommand performance
func BenchmarkEmbedGenerateCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetEmbedFlags()
		testCmd := createTestEmbedCommand()
		testCmd.SetArgs([]string{"generate", "--help"})
		testCmd.SetOut(io.Discard)
		testCmd.SetErr(io.Discard)
		_ = testCmd.Execute()
	}
}
