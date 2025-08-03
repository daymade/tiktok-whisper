package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderSwitching(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", "test-v2t", "./cmd/v2t/")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-v2t")

	// Create test audio file path
	testAudioPath := filepath.Join("data", "test.mp3")
	
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectError    bool
		errorContains  string
	}{
		{
			name: "List providers",
			args: []string{"providers", "list"},
			expectedOutput: "Whisper.cpp (Local)",
			expectError: false,
		},
		{
			name: "Use default provider",
			args: []string{"convert", "-i", testAudioPath, "-a"},
			expectedOutput: "Using provider:",
			expectError: false,
		},
		{
			name: "Switch to invalid provider",
			args: []string{"convert", "-i", testAudioPath, "-a", "--provider", "invalid_provider"},
			expectError: true,
			errorContains: "Provider 'invalid_provider' not found in configuration",
		},
		{
			name: "Switch to openai without API key",
			args: []string{"convert", "-i", testAudioPath, "-a", "--provider", "openai"},
			expectError: true,
			errorContains: "openai provider requires 'api_key'",
		},
		{
			name: "Show provider config",
			args: []string{"providers", "config"},
			expectedOutput: "default_provider:",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./test-v2t", tt.args...)
			output, err := cmd.CombinedOutput()
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. Output: %s", output)
				} else if tt.errorContains != "" && !strings.Contains(string(output), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorContains, output)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v. Output: %s", err, output)
				} else if tt.expectedOutput != "" && !strings.Contains(string(output), tt.expectedOutput) {
					t.Errorf("Expected output containing '%s', got: %s", tt.expectedOutput, output)
				}
			}
		})
	}
}

func TestProviderConfiguration(t *testing.T) {
	// Test configuration file handling
	tests := []struct {
		name         string
		configContent string
		expectError  bool
		errorContains string
	}{
		{
			name: "Missing required binary_path",
			configContent: `
default_provider: "whisper_cpp"
providers:
  whisper_cpp:
    type: "whisper_cpp"
    enabled: true
    settings:
      model_path: "/path/to/model"
`,
			expectError: true,
			errorContains: "whisper_cpp provider requires 'binary_path' setting",
		},
		{
			name: "Missing required model_path",
			configContent: `
default_provider: "whisper_cpp"
providers:
  whisper_cpp:
    type: "whisper_cpp"
    enabled: true
    settings:
      binary_path: "/path/to/binary"
`,
			expectError: true,
			errorContains: "whisper_cpp provider requires 'model_path' setting",
		},
		{
			name: "Valid whisper_cpp config",
			configContent: `
default_provider: "whisper_cpp"
providers:
  whisper_cpp:
    type: "whisper_cpp"
    enabled: true
    settings:
      binary_path: "/path/to/binary"
      model_path: "/path/to/model"
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpfile, err := os.CreateTemp("", "providers-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.configContent)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Test with the config
			cmd := exec.Command("./test-v2t", "providers", "config", "-c", tmpfile.Name())
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. Output: %s", output)
				} else if tt.errorContains != "" && !strings.Contains(string(output), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorContains, output)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v. Output: %s", err, output)
				}
			}
		})
	}
}