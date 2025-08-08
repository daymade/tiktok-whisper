package ssh_whisper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/common"
)

// SSHWhisperProvider implements transcription via SSH to a remote whisper.cpp instance
type SSHWhisperProvider struct {
	common.BaseProvider
	config SSHWhisperConfig
}

// SSHWhisperConfig represents configuration for SSH remote whisper.cpp
type SSHWhisperConfig struct {
	Host       string `yaml:"host"`        // SSH host (e.g., "user@hostname")
	RemoteDir  string `yaml:"remote_dir"`  // Remote whisper.cpp directory
	BinaryPath string `yaml:"binary_path"` // Remote binary path (relative to remote_dir)
	ModelPath  string `yaml:"model_path"`  // Remote model path (relative to remote_dir)
	Language   string `yaml:"language"`    // Language code (optional)
	Prompt     string `yaml:"prompt"`      // Prompt for transcription (optional)
	Threads    int    `yaml:"threads"`     // Number of threads (optional, default 4)
}

// NewSSHWhisperProvider creates a new SSH whisper provider
func NewSSHWhisperProvider(config SSHWhisperConfig) *SSHWhisperProvider {
	// Set defaults
	if config.BinaryPath == "" {
		config.BinaryPath = "./build/bin/whisper-cli"
	}
	if config.ModelPath == "" {
		config.ModelPath = "models/ggml-base.en.bin"
	}
	if config.Threads == 0 {
		config.Threads = 4
	}

	baseProvider := common.NewBaseProvider(
		"ssh_whisper",
		"SSH Remote Whisper.cpp",
		provider.ProviderTypeRemote,
		"1.0.0",
	)
	
	// Set specific attributes for SSH whisper provider
	baseProvider.SupportedFormats = []provider.AudioFormat{
		provider.FormatWAV,
		provider.FormatMP3,
		provider.FormatM4A,
		provider.FormatFLAC,
	}
	baseProvider.MaxFileSizeMB = 1000
	baseProvider.MaxDurationSec = 3600
	baseProvider.SupportsTimestamps = true
	baseProvider.SupportsWordLevel = false
	baseProvider.SupportsConfidence = false
	baseProvider.SupportsLanguageDetection = true
	baseProvider.SupportsStreaming = false
	baseProvider.RequiresInternet = false
	baseProvider.RequiresAPIKey = false
	baseProvider.RequiresBinary = true
	baseProvider.DefaultModel = "models/ggml-base.en.bin"
	baseProvider.AvailableModels = []string{
		"models/ggml-tiny.bin",
		"models/ggml-tiny.en.bin",
		"models/ggml-base.bin",
		"models/ggml-base.en.bin",
		"models/ggml-small.bin",
		"models/ggml-small.en.bin",
		"models/ggml-medium.bin",
		"models/ggml-medium.en.bin",
		"models/ggml-large-v1.bin",
		"models/ggml-large-v2.bin",
		"models/ggml-large-v3.bin",
	}
	
	// Set config schema for SSH provider
	baseProvider.ConfigSchema = map[string]interface{}{
		"host": map[string]string{
			"type":        "string",
			"description": "SSH host (e.g., 'user@hostname')",
			"required":    "true",
		},
		"remote_dir": map[string]string{
			"type":        "string",
			"description": "Remote whisper.cpp directory",
			"required":    "true",
		},
		"binary_path": map[string]string{
			"type":        "string",
			"description": "Remote binary path (relative to remote_dir)",
			"default":     "./build/bin/whisper-cli",
		},
		"model_path": map[string]string{
			"type":        "string",
			"description": "Remote model path (relative to remote_dir)",
			"default":     "models/ggml-base.en.bin",
		},
		"language": map[string]string{
			"type":        "string",
			"description": "Language code (e.g., 'zh', 'en')",
			"default":     "",
		},
		"threads": map[string]string{
			"type":        "integer",
			"description": "Number of threads for processing",
			"default":     "4",
		},
	}

	return &SSHWhisperProvider{
		BaseProvider: baseProvider,
		config:       config,
	}
}

// NewSSHWhisperProviderFromSettings creates provider from generic settings
func NewSSHWhisperProviderFromSettings(settings map[string]interface{}) (*SSHWhisperProvider, error) {
	config := SSHWhisperConfig{}

	// Extract required settings
	if host, ok := settings["host"].(string); ok {
		config.Host = host
	} else {
		return nil, fmt.Errorf("ssh host is required")
	}

	if remoteDir, ok := settings["remote_dir"].(string); ok {
		config.RemoteDir = remoteDir
	} else {
		return nil, fmt.Errorf("remote_dir is required")
	}

	// Extract optional settings
	if binaryPath, ok := settings["binary_path"].(string); ok {
		config.BinaryPath = binaryPath
	}
	if modelPath, ok := settings["model_path"].(string); ok {
		config.ModelPath = modelPath
	}
	if language, ok := settings["language"].(string); ok {
		config.Language = language
	}
	if prompt, ok := settings["prompt"].(string); ok {
		config.Prompt = prompt
	}
	if threads, ok := settings["threads"].(float64); ok {
		config.Threads = int(threads)
	}

	return NewSSHWhisperProvider(config), nil
}

// Transcript implements the basic transcription interface for backward compatibility
func (ssp *SSHWhisperProvider) Transcript(inputFilePath string) (string, error) {
	ctx := context.Background()
	request := &provider.TranscriptionRequest{
		InputFilePath: inputFilePath,
	}

	response, err := ssp.TranscriptWithOptions(ctx, request)
	if err != nil {
		return "", err
	}

	return response.Text, nil
}

// TranscriptWithOptions implements the enhanced transcription interface
func (ssp *SSHWhisperProvider) TranscriptWithOptions(ctx context.Context, request *provider.TranscriptionRequest) (*provider.TranscriptionResponse, error) {
	startTime := time.Now()

	// Validate input
	if request.InputFilePath == "" {
		return nil, &provider.TranscriptionError{
			Code:      "invalid_input",
			Message:   "input file path is required",
			Provider:  "ssh_whisper",
			Retryable: false,
		}
	}

	// Check if local file exists
	if _, err := os.Stat(request.InputFilePath); os.IsNotExist(err) {
		return nil, &provider.TranscriptionError{
			Code:      "file_not_found",
			Message:   fmt.Sprintf("input file not found: %s", request.InputFilePath),
			Provider:  "ssh_whisper",
			Retryable: false,
		}
	}

	// Get absolute path for the input file
	absInputPath, err := filepath.Abs(request.InputFilePath)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "file_path_error",
			Message:   fmt.Sprintf("failed to get absolute path: %v", err),
			Provider:  "ssh_whisper",
			Retryable: false,
		}
	}

	// Generate remote file path
	fileName := filepath.Base(absInputPath)
	remoteFilePath := fmt.Sprintf("/tmp/whisper_%d_%s", time.Now().Unix(), fileName)

	// Step 1: Copy file to remote host
	err = ssp.copyFileToRemote(absInputPath, remoteFilePath)
	if err != nil {
		return nil, &provider.TranscriptionError{
			Code:      "file_transfer_failed",
			Message:   fmt.Sprintf("failed to copy file to remote: %v", err),
			Provider:  "ssh_whisper",
			Retryable: true,
		}
	}

	// Step 2: Run whisper on remote host
	transcriptionText, err := ssp.runRemoteWhisper(ctx, remoteFilePath, request)
	if err != nil {
		// Clean up remote file
		ssp.cleanupRemoteFile(remoteFilePath)
		return nil, err
	}

	// Step 3: Clean up remote file
	err = ssp.cleanupRemoteFile(remoteFilePath)
	if err != nil {
		// Log warning but don't fail the transcription
		fmt.Printf("Warning: failed to cleanup remote file %s: %v\n", remoteFilePath, err)
	}

	// Build response
	response := &provider.TranscriptionResponse{
		Text:           transcriptionText,
		Language:       ssp.getLanguage(request),
		ProcessingTime: time.Since(startTime),
		ModelUsed:      ssp.config.ModelPath,
		ProviderMetadata: map[string]interface{}{
			"ssh_host":     ssp.config.Host,
			"remote_dir":   ssp.config.RemoteDir,
			"binary_path":  ssp.config.BinaryPath,
			"model_path":   ssp.config.ModelPath,
			"threads":      ssp.config.Threads,
			"remote_file":  remoteFilePath,
		},
	}

	return response, nil
}

// copyFileToRemote copies a local file to the remote host using scp
func (ssp *SSHWhisperProvider) copyFileToRemote(localPath, remotePath string) error {
	remoteTarget := fmt.Sprintf("%s:%s", ssp.config.Host, remotePath)
	
	cmd := exec.Command("scp", localPath, remoteTarget)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp failed: %v, output: %s", err, string(output))
	}

	return nil
}

// runRemoteWhisper executes whisper.cpp on the remote host
func (ssp *SSHWhisperProvider) runRemoteWhisper(ctx context.Context, remoteFilePath string, request *provider.TranscriptionRequest) (string, error) {
	// Build whisper command
	args := []string{
		ssp.config.Host,
		fmt.Sprintf("cd %s && %s", ssp.config.RemoteDir, ssp.buildWhisperCommand(remoteFilePath, request)),
	}

	// Create command with context for timeout support
	cmd := exec.CommandContext(ctx, "ssh", args...)
	
	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &provider.TranscriptionError{
			Code:      "transcription_failed",
			Message:   fmt.Sprintf("remote whisper execution failed: %v, output: %s", err, string(output)),
			Provider:  "ssh_whisper",
			Retryable: true,
		}
	}

	// Parse output to extract transcription text
	transcription := ssp.parseWhisperOutput(string(output))
	if transcription == "" {
		return "", &provider.TranscriptionError{
			Code:      "empty_transcription",
			Message:   "no transcription text found in output",
			Provider:  "ssh_whisper",
			Retryable: false,
			Suggestions: []string{"Check audio file format", "Verify model compatibility"},
		}
	}

	return transcription, nil
}

// buildWhisperCommand builds the whisper.cpp command line
func (ssp *SSHWhisperProvider) buildWhisperCommand(remoteFilePath string, request *provider.TranscriptionRequest) string {
	args := []string{
		ssp.config.BinaryPath,
		"-m", ssp.config.ModelPath,
		"-f", remoteFilePath,
		"-nt", // no timestamps in output
		fmt.Sprintf("-t %d", ssp.config.Threads),
	}

	// Add language if specified
	language := ssp.getLanguage(request)
	if language != "" {
		args = append(args, "-l", language)
	}

	// Add prompt if specified
	prompt := ssp.getPrompt(request)
	if prompt != "" {
		args = append(args, "--prompt", fmt.Sprintf("'%s'", prompt))
	}

	return strings.Join(args, " ")
}

// parseWhisperOutput extracts the transcription text from whisper output
func (ssp *SSHWhisperProvider) parseWhisperOutput(output string) string {
	lines := strings.Split(output, "\n")
	
	// Look for the transcription text (usually the first line before detailed logs)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and lines that look like debug/timing info
		if line != "" && 
		   !strings.Contains(line, "whisper_") && 
		   !strings.Contains(line, "system_info") &&
		   !strings.Contains(line, "ggml_") &&
		   !strings.Contains(line, "main:") &&
		   !strings.Contains(line, "load time") &&
		   !strings.Contains(line, "Metal") {
			return line
		}
	}
	
	return ""
}

// cleanupRemoteFile removes the temporary file from remote host
func (ssp *SSHWhisperProvider) cleanupRemoteFile(remoteFilePath string) error {
	cmd := exec.Command("ssh", ssp.config.Host, "rm", "-f", remoteFilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cleanup failed: %v, output: %s", err, string(output))
	}
	return nil
}

// Helper methods
func (ssp *SSHWhisperProvider) getLanguage(request *provider.TranscriptionRequest) string {
	if request.Language != "" {
		return request.Language
	}
	return ssp.config.Language
}

func (ssp *SSHWhisperProvider) getPrompt(request *provider.TranscriptionRequest) string {
	if request.Prompt != "" {
		return request.Prompt
	}
	return ssp.config.Prompt
}

// GetProviderInfo method is now inherited from BaseProvider

// ValidateConfiguration validates the provider configuration
func (ssp *SSHWhisperProvider) ValidateConfiguration() error {
	// Check required fields
	if ssp.config.Host == "" {
		return fmt.Errorf("SSH host is required")
	}
	if ssp.config.RemoteDir == "" {
		return fmt.Errorf("remote directory is required")
	}

	// Validate threads
	if ssp.config.Threads < 1 || ssp.config.Threads > 32 {
		return fmt.Errorf("threads must be between 1 and 32")
	}

	// Test SSH connectivity (basic check)
	cmd := exec.Command("ssh", ssp.config.Host, "echo", "test")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SSH connection test failed: %v (check SSH keys and host connectivity)", err)
	}

	return nil
}

// HealthCheck performs a health check on the provider
func (ssp *SSHWhisperProvider) HealthCheck(ctx context.Context) error {
	// Basic configuration validation
	if err := ssp.ValidateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Test remote directory and binary existence
	remoteTestCmd := fmt.Sprintf("cd %s && test -f %s", ssp.config.RemoteDir, ssp.config.BinaryPath)
	cmd := exec.CommandContext(ctx, "ssh", ssp.config.Host, remoteTestCmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("remote binary not found: %s/%s", ssp.config.RemoteDir, ssp.config.BinaryPath)
	}

	// Test model file existence
	modelTestCmd := fmt.Sprintf("cd %s && test -f %s", ssp.config.RemoteDir, ssp.config.ModelPath)
	cmd = exec.CommandContext(ctx, "ssh", ssp.config.Host, modelTestCmd)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("remote model not found: %s/%s", ssp.config.RemoteDir, ssp.config.ModelPath)
	}

	return nil
}