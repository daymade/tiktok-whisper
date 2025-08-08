package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"tiktok-whisper/internal/app/temporal/activities"
)

// FallbackWorkflowRequest represents the input for fallback transcription workflow
type FallbackWorkflowRequest struct {
	FileID       string                 `json:"file_id"`
	FilePath     string                 `json:"file_path"`
	Providers    []string               `json:"providers"`     // Ordered list of providers to try
	Language     string                 `json:"language"`
	OutputFormat string                 `json:"output_format"`
	Options      map[string]interface{} `json:"options"`
	UseMinIO     bool                   `json:"use_minio"`
}

// FallbackWorkflowResult represents the output with provider fallback information
type FallbackWorkflowResult struct {
	FileID             string        `json:"file_id"`
	TranscriptionURL   string        `json:"transcription_url"`
	SuccessfulProvider string        `json:"successful_provider"`
	AttemptedProviders []string      `json:"attempted_providers"`
	ProcessingTime     time.Duration `json:"processing_time"`
	Error              string        `json:"error,omitempty"`
}

// TranscriptionWithFallbackWorkflow attempts transcription with multiple providers
func TranscriptionWithFallbackWorkflow(ctx workflow.Context, req FallbackWorkflowRequest) (FallbackWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting transcription with fallback workflow", 
		"fileId", req.FileID,
		"providers", req.Providers)

	startTime := workflow.Now(ctx)

	// Default providers if none specified
	if len(req.Providers) == 0 {
		req.Providers = []string{"whisper_cpp", "openai", "elevenlabs"}
	}

	var lastError error
	attemptedProviders := make([]string, 0, len(req.Providers))

	// Try each provider in order
	for _, provider := range req.Providers {
		logger.Info("Attempting transcription with provider", 
			"provider", provider,
			"fileId", req.FileID)

		// Check provider health first
		healthCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 1, // No retry for health check
			},
		})

		var healthStatus activities.ProviderHealthStatus
		err := workflow.ExecuteActivity(healthCtx, "GetProviderStatus", provider).Get(healthCtx, &healthStatus)
		
		if err != nil || !healthStatus.IsHealthy {
			logger.Warn("Provider not healthy, skipping", 
				"provider", provider,
				"error", err)
			attemptedProviders = append(attemptedProviders, provider)
			if err != nil {
				lastError = err
			} else {
				lastError = fmt.Errorf("provider %s is not healthy: %s", provider, healthStatus.LastError)
			}
			continue
		}

		// Attempt transcription with this provider
		transcriptionReq := SingleFileWorkflowRequest{
			FileID:       req.FileID,
			FilePath:     req.FilePath,
			Provider:     provider,
			Language:     req.Language,
			OutputFormat: req.OutputFormat,
			Options:      req.Options,
			UseMinIO:     req.UseMinIO,
		}

		// Use child workflow with specific timeout for this provider
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s-%s-%d", req.FileID, provider, workflow.Now(ctx).Unix()),
			WorkflowExecutionTimeout: 20 * time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 1, // No retry at workflow level, we handle it here
			},
		})

		var result SingleFileWorkflowResult
		err = workflow.ExecuteChildWorkflow(childCtx, 
			SingleFileTranscriptionWorkflow,
			transcriptionReq).Get(childCtx, &result)

		attemptedProviders = append(attemptedProviders, provider)

		if err == nil && result.Error == "" {
			// Success!
			processingTime := workflow.Now(ctx).Sub(startTime)
			
			return FallbackWorkflowResult{
				FileID:             req.FileID,
				TranscriptionURL:   result.TranscriptionURL,
				SuccessfulProvider: provider,
				AttemptedProviders: attemptedProviders,
				ProcessingTime:     processingTime,
			}, nil
		}

		// Log failure and continue to next provider
		if err != nil {
			logger.Error("Provider failed with error", 
				"provider", provider,
				"error", err)
			lastError = err
		} else {
			logger.Error("Provider returned error", 
				"provider", provider,
				"error", result.Error)
			lastError = fmt.Errorf("provider error: %s", result.Error)
		}

		// Wait before trying next provider
		if len(attemptedProviders) < len(req.Providers) {
			workflow.Sleep(ctx, 10*time.Second)
		}
	}

	// All providers failed
	processingTime := workflow.Now(ctx).Sub(startTime)
	
	return FallbackWorkflowResult{
		FileID:             req.FileID,
		AttemptedProviders: attemptedProviders,
		ProcessingTime:     processingTime,
		Error:              fmt.Sprintf("all providers failed, last error: %v", lastError),
	}, lastError
}

// SmartFallbackWorkflow uses intelligent provider selection based on file characteristics
func SmartFallbackWorkflow(ctx workflow.Context, req FallbackWorkflowRequest) (FallbackWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting smart fallback workflow", "fileId", req.FileID)

	// Get recommended provider first
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	var recommendedProvider string
	err := workflow.ExecuteActivity(activityCtx, "GetRecommendedProvider", req.FilePath).Get(activityCtx, &recommendedProvider)
	
	if err == nil && recommendedProvider != "" {
		// Build provider list with recommended first
		providers := []string{recommendedProvider}
		for _, p := range req.Providers {
			if p != recommendedProvider {
				providers = append(providers, p)
			}
		}
		req.Providers = providers
		
		logger.Info("Using recommended provider order", 
			"providers", providers,
			"recommended", recommendedProvider)
	}

	// Execute fallback workflow with optimized order
	return TranscriptionWithFallbackWorkflow(ctx, req)
}