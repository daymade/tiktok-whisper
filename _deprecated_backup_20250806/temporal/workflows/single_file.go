package workflows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"tiktok-whisper/temporal/activities"
)

// SingleFileWorkflowRequest represents the input for single file transcription workflow
type SingleFileWorkflowRequest struct {
	FileID       string                 `json:"file_id"`
	FilePath     string                 `json:"file_path"`     // Can be local path or MinIO URL
	Provider     string                 `json:"provider"`      // Optional, uses default if empty
	Language     string                 `json:"language"`
	OutputFormat string                 `json:"output_format"`
	Options      map[string]interface{} `json:"options"`
	UseMinIO     bool                   `json:"use_minio"`    // Whether to use MinIO for storage
}

// SingleFileWorkflowResult represents the output of single file transcription workflow
type SingleFileWorkflowResult struct {
	FileID           string        `json:"file_id"`
	TranscriptionURL string        `json:"transcription_url"`
	Provider         string        `json:"provider"`
	ProcessingTime   time.Duration `json:"processing_time"`
	Error            string        `json:"error,omitempty"`
}

// SingleFileTranscriptionWorkflow processes a single file transcription
func SingleFileTranscriptionWorkflow(ctx workflow.Context, req SingleFileWorkflowRequest) (SingleFileWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting single file transcription workflow", "fileId", req.FileID)

	startTime := workflow.Now(ctx)

	// Configure activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute, // Long timeout for large files
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    100 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var localFilePath string
	var err error

	// Step 1: Handle file location (download from MinIO if needed)
	if req.UseMinIO && strings.HasPrefix(req.FilePath, "minio://") {
		// Parse MinIO URL
		objectKey := strings.TrimPrefix(req.FilePath, "minio://")
		parts := strings.SplitN(objectKey, "/", 2)
		if len(parts) == 2 {
			objectKey = parts[1]
		}

		// Download file from MinIO
		var downloadResult activities.FileDownloadResult
		err = workflow.ExecuteActivity(ctx, "DownloadFile", activities.FileDownloadRequest{
			ObjectKey: objectKey,
		}).Get(ctx, &downloadResult)
		if err != nil {
			logger.Error("Failed to download file from MinIO", "error", err)
			return SingleFileWorkflowResult{
				FileID: req.FileID,
				Error:  fmt.Sprintf("Failed to download file: %v", err),
			}, err
		}
		localFilePath = downloadResult.LocalPath

		// Schedule cleanup
		defer func() {
			// Use a separate context for cleanup to ensure it runs
			cleanupCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute,
			})
			_ = workflow.ExecuteActivity(cleanupCtx, "CleanupTempFile", localFilePath).Get(cleanupCtx, nil)
		}()
	} else {
		localFilePath = req.FilePath
	}

	// Step 2: Perform transcription
	var transcriptionResult activities.TranscriptionResult
	err = workflow.ExecuteActivity(ctx, "TranscribeFile", activities.TranscriptionRequest{
		FileID:       req.FileID,
		FilePath:     localFilePath,
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

	var transcriptionURL string

	// Step 3: Store transcription result
	if req.UseMinIO {
		// Save transcription to MinIO
		transcriptionKey := fmt.Sprintf("transcriptions/%s/%s.txt",
			time.Now().Format("2006-01-02"),
			req.FileID)

		// Create temp file for transcription
		tempPath := fmt.Sprintf("/tmp/v2t-temporal/%s.txt", req.FileID)
		err = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
			return saveTranscriptionToFile(tempPath, transcriptionResult.Text)
		}).Get(&err)
		if err != nil {
			logger.Error("Failed to save transcription to temp file", "error", err)
			return SingleFileWorkflowResult{
				FileID: req.FileID,
				Error:  fmt.Sprintf("Failed to save transcription: %v", err),
			}, err
		}

		// Upload to MinIO
		var uploadResult activities.FileUploadResult
		err = workflow.ExecuteActivity(ctx, "UploadFile", activities.FileUploadRequest{
			LocalPath: tempPath,
			ObjectKey: transcriptionKey,
			Metadata: map[string]string{
				"file_id":  req.FileID,
				"provider": transcriptionResult.Provider,
				"language": req.Language,
			},
		}).Get(ctx, &uploadResult)
		if err != nil {
			logger.Error("Failed to upload transcription to MinIO", "error", err)
			return SingleFileWorkflowResult{
				FileID: req.FileID,
				Error:  fmt.Sprintf("Failed to upload transcription: %v", err),
			}, err
		}

		transcriptionURL = uploadResult.URL

		// Cleanup temp transcription file
		cleanupCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		})
		_ = workflow.ExecuteActivity(cleanupCtx, "CleanupTempFile", tempPath).Get(cleanupCtx, nil)
	} else {
		// Save to local file
		outputPath := filepath.Join(filepath.Dir(localFilePath), 
			fmt.Sprintf("%s_transcription.txt", strings.TrimSuffix(filepath.Base(localFilePath), filepath.Ext(localFilePath))))
		
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
		transcriptionURL = outputPath
	}

	// Calculate total processing time
	processingTime := workflow.Now(ctx).Sub(startTime)

	result := SingleFileWorkflowResult{
		FileID:           req.FileID,
		TranscriptionURL: transcriptionURL,
		Provider:         transcriptionResult.Provider,
		ProcessingTime:   processingTime,
	}

	logger.Info("Single file transcription completed", 
		"fileId", req.FileID,
		"provider", result.Provider,
		"duration", result.ProcessingTime)

	return result, nil
}

// saveTranscriptionToFile is a helper to save transcription text to a file
func saveTranscriptionToFile(path string, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write transcription to file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}