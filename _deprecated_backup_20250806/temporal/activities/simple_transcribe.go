package activities

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"go.temporal.io/sdk/activity"
	"tiktok-whisper/temporal/pkg/common"
	"tiktok-whisper/temporal/pkg/whisper"
)

// SimpleTranscribeActivities provides transcription activities without external dependencies
type SimpleTranscribeActivities struct {
	whisperConfig whisper.Config
}

// NewSimpleTranscribeActivities creates transcription activities without provider registry
func NewSimpleTranscribeActivities() *SimpleTranscribeActivities {
	return &SimpleTranscribeActivities{
		whisperConfig: whisper.Config{
			BinaryPath: common.GetEnv("WHISPER_BINARY", common.DefaultWhisperBinary),
			ModelPath:  common.GetEnv("WHISPER_MODEL", "/models/ggml-large-v3.bin"),
			Language:   "auto",
		},
	}
}

// TranscribeFileSimple performs transcription using local whisper.cpp
func (a *SimpleTranscribeActivities) TranscribeFileSimple(ctx context.Context, req TranscriptionRequest) (TranscriptionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting simple transcription", "fileId", req.FileID, "file", req.FilePath)

	startTime := time.Now()

	// Configure whisper based on request
	config := a.whisperConfig
	if req.Language != "" {
		config.Language = req.Language
	}

	// Execute whisper
	output, err := whisper.ExecuteWhisper(ctx, config, req.FilePath, req.FileID, req.Language)
	if err != nil {
		logger.Error("Whisper execution failed", "error", err)
		return TranscriptionResult{
			FileID: req.FileID,
			Error:  err.Error(),
		}, err
	}

	processingTime := time.Since(startTime)

	result := TranscriptionResult{
		FileID:         req.FileID,
		Text:           output,
		Provider:       "whisper_cpp",
		ProcessingTime: processingTime,
	}

	logger.Info("Transcription completed", 
		"fileId", req.FileID, 
		"duration", processingTime,
		"textLength", len(output))

	return result, nil
}

// GetProviderStatus returns the status of a provider (simplified version)
func (a *SimpleTranscribeActivities) GetProviderStatus(ctx context.Context, providerName string) (ProviderHealthStatus, error) {
	// For simple version, only check whisper_cpp
	if providerName != "whisper_cpp" {
		return ProviderHealthStatus{
			ProviderName: providerName,
			IsHealthy:    false,
			LastError:    "Provider not available in simple mode",
		}, nil
	}

	// Check if whisper binary exists
	if _, err := exec.LookPath(a.whisperConfig.BinaryPath); err != nil {
		return ProviderHealthStatus{
			ProviderName: providerName,
			IsHealthy:    false,
			LastError:    "Whisper binary not found",
		}, nil
	}

	return ProviderHealthStatus{
		ProviderName: providerName,
		IsHealthy:    true,
		LastChecked:  time.Now(),
	}, nil
}

// ProviderHealthStatus represents the health status of a provider
type ProviderHealthStatus struct {
	ProviderName string    `json:"provider_name"`
	IsHealthy    bool      `json:"is_healthy"`
	LastChecked  time.Time `json:"last_checked"`
	LastError    string    `json:"last_error,omitempty"`
}