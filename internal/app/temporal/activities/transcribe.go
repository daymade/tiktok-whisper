package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/common"
)

// TranscribeActivities provides transcription activities using v2t providers
type TranscribeActivities struct {
	registry provider.ProviderRegistry
}

// NewTranscribeActivities creates a new instance of transcription activities
func NewTranscribeActivities(registry provider.ProviderRegistry) *TranscribeActivities {
	return &TranscribeActivities{
		registry: registry,
	}
}

// Use types from common package
type TranscriptionRequest = common.TranscriptionRequest
type TranscriptionResult = common.TranscriptionResult

// TranscribeFile performs transcription using the v2t provider framework
func (a *TranscribeActivities) TranscribeFile(ctx context.Context, req TranscriptionRequest) (TranscriptionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting transcription", "fileId", req.FileID, "provider", req.Provider)

	// Record heartbeat for long-running activities
	activity.RecordHeartbeat(ctx, fmt.Sprintf("Processing file: %s", req.FileID))

	startTime := time.Now()

	// Get provider (use default if not specified)
	var transcriber provider.TranscriptionProvider
	var err error
	
	if req.Provider != "" {
		transcriber, err = a.registry.GetProvider(req.Provider)
	} else {
		transcriber, err = a.registry.GetDefaultProvider()
	}
	
	if err != nil {
		logger.Error("Failed to get provider", "error", err)
		return TranscriptionResult{
			FileID: req.FileID,
			Error:  err.Error(),
		}, err
	}

	// Create transcription request for provider
	providerReq := &provider.TranscriptionRequest{
		InputFilePath:   req.FilePath,
		Language:        req.Language,
		ResponseFormat:  req.OutputFormat,
		ProviderOptions: req.Options,
		Context:         ctx,
	}

	// Perform transcription with heartbeats
	heartbeatTicker := time.NewTicker(10 * time.Second)
	defer heartbeatTicker.Stop()

	done := make(chan struct{})
	var response *provider.TranscriptionResponse
	var transcribeErr error

	go func() {
		response, transcribeErr = transcriber.TranscriptWithOptions(ctx, providerReq)
		close(done)
	}()

	// Send heartbeats while transcription is running
	for {
		select {
		case <-done:
			// Transcription completed
			if transcribeErr != nil {
				logger.Error("Transcription failed", "error", transcribeErr)
				return TranscriptionResult{
					FileID: req.FileID,
					Error:  transcribeErr.Error(),
				}, transcribeErr
			}

			result := TranscriptionResult{
				FileID:         req.FileID,
				Transcription:  response.Text,
				Provider:       transcriber.GetProviderInfo().Name,
				ProcessingTime: time.Since(startTime),
			}

			logger.Info("Transcription completed", 
				"fileId", req.FileID, 
				"provider", result.Provider,
				"duration", result.ProcessingTime)

			return result, nil

		case <-heartbeatTicker.C:
			activity.RecordHeartbeat(ctx, fmt.Sprintf("Still processing file: %s", req.FileID))

		case <-ctx.Done():
			// Activity cancelled
			return TranscriptionResult{
				FileID: req.FileID,
				Error:  "Activity cancelled",
			}, ctx.Err()
		}
	}
}

// GetProviderStatus checks the health status of a specific provider
func (a *TranscribeActivities) GetProviderStatus(ctx context.Context, providerName string) (provider.ProviderHealthStatus, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Checking provider status", "provider", providerName)

	p, err := a.registry.GetProvider(providerName)
	if err != nil {
		return provider.ProviderHealthStatus{
			ProviderName: providerName,
			IsHealthy:    false,
			LastError:    err.Error(),
		}, err
	}

	err = p.HealthCheck(ctx)
	isHealthy := err == nil
	
	status := provider.ProviderHealthStatus{
		ProviderName: providerName,
		IsHealthy:    isHealthy,
		LastChecked:  time.Now(),
	}
	
	if err != nil {
		status.LastError = err.Error()
	}

	return status, nil
}

// ListAvailableProviders returns all registered providers
func (a *TranscribeActivities) ListAvailableProviders(ctx context.Context) ([]provider.ProviderInfo, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Listing available providers")

	providerNames := a.registry.ListProviders()
	infos := make([]provider.ProviderInfo, 0, len(providerNames))
	
	for _, name := range providerNames {
		p, err := a.registry.GetProvider(name)
		if err != nil {
			continue
		}
		infos = append(infos, p.GetProviderInfo())
	}

	return infos, nil
}

// GetRecommendedProvider suggests the best provider for a given file
func (a *TranscribeActivities) GetRecommendedProvider(ctx context.Context, filePath string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting recommended provider", "file", filePath)

	// Use orchestrator if available
	if orchestrator, ok := a.registry.(provider.TranscriptionOrchestrator); ok {
		req := &provider.TranscriptionRequest{
			InputFilePath: filePath,
		}
		
		recommended, err := orchestrator.RecommendProvider(req)
		if err == nil && len(recommended) > 0 {
			return recommended[0], nil
		}
	}

	// Fallback to default provider
	defaultProvider, err := a.registry.GetDefaultProvider()
	if err != nil {
		return "", err
	}

	return defaultProvider.GetProviderInfo().Name, nil
}

