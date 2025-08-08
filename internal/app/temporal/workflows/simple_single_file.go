package workflows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TranscriptionRequest represents a request to transcribe a file
type TranscriptionRequest struct {
	FileID       string                 `json:"file_id"`
	FilePath     string                 `json:"file_path"`
	Provider     string                 `json:"provider"`
	Language     string                 `json:"language"`
	OutputFormat string                 `json:"output_format"`
	Options      map[string]interface{} `json:"options"`
}

// TranscriptionResult represents the result of a transcription
type TranscriptionResult struct {
	FileID         string        `json:"file_id"`
	Text           string        `json:"text"`
	Provider       string        `json:"provider"`
	ProcessingTime time.Duration `json:"processing_time"`
	Error          string        `json:"error,omitempty"`
}

// SimpleSingleFileWorkflow processes a single file transcription without MinIO
func SimpleSingleFileWorkflow(ctx workflow.Context, req SingleFileWorkflowRequest) (SingleFileWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting simple single file transcription workflow", "fileId", req.FileID)

	startTime := workflow.Now(ctx)

	// Configure activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    100 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Perform transcription
	var transcriptionResult TranscriptionResult
	err := workflow.ExecuteActivity(ctx, "TranscribeFileSimple", TranscriptionRequest{
		FileID:       req.FileID,
		FilePath:     req.FilePath,
		Provider:     req.Provider,
		Language:     req.Language,
		OutputFormat: req.OutputFormat,
		Options:      req.Options,
	}).Get(ctx, &transcriptionResult)
	
	if err != nil {
		logger.Error("Failed to transcribe file", "error", err)
		return SingleFileWorkflowResult{
			FileID: req.FileID,
			Error:  fmt.Sprintf("Failed to transcribe: %v", err),
		}, err
	}

	// Save transcription to local file
	outputPath := filepath.Join(filepath.Dir(req.FilePath), 
		fmt.Sprintf("%s_transcription.txt", strings.TrimSuffix(filepath.Base(req.FilePath), filepath.Ext(req.FilePath))))
	
	err = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		return saveTranscriptionToFile(outputPath, transcriptionResult.Text)
	}).Get(&err)
	
	if err != nil {
		logger.Error("Failed to save transcription to file", "error", err)
		return SingleFileWorkflowResult{
			FileID: req.FileID,
			Error:  fmt.Sprintf("Failed to save transcription: %v", err),
		}, err
	}

	// Calculate total processing time
	processingTime := workflow.Now(ctx).Sub(startTime)

	result := SingleFileWorkflowResult{
		FileID:           req.FileID,
		TranscriptionURL: outputPath,
		Provider:         transcriptionResult.Provider,
		ProcessingTime:   processingTime,
	}

	logger.Info("Simple single file transcription completed", 
		"fileId", req.FileID,
		"provider", result.Provider,
		"duration", result.ProcessingTime)

	return result, nil
}

// Helper function remains the same
func saveTranscriptionToFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}