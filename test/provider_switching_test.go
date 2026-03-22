package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tiktok-whisper/internal/app/api/provider"
)

func buildProviderTestBinary(t *testing.T) string {
	t.Helper()

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	binaryPath := filepath.Join(t.TempDir(), "test-v2t")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/v2t/")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	return binaryPath
}

func TestProviderSwitching(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	binaryPath := buildProviderTestBinary(t)

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
				cmd := exec.Command(binaryPath, tt.args...)
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

				_, err = provider.NewConfigManager(tmpfile.Name()).LoadConfig()

				if tt.expectError {
					if err == nil {
						t.Errorf("Expected error but got none")
					} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
						t.Errorf("Expected error containing '%s', got: %s", tt.errorContains, err)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
				}
			})
	}
}
