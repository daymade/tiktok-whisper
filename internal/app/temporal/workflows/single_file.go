package workflows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"tiktok-whisper/internal/app/audio"
	"tiktok-whisper/internal/app/common"
	"tiktok-whisper/internal/app/temporal/activities"
)

// Use types from common package
type SingleFileWorkflowRequest = common.SingleFileWorkflowRequest
type SingleFileWorkflowResult = common.SingleFileWorkflowResult

func buildTranscriptionObjectKey(now time.Time, fileID string) string {
	return fmt.Sprintf("transcriptions/%s/%s.txt", now.Format("2006-01-02"), fileID)
}

func buildTempTranscriptionPath(tempDir, fileID string) string {
	return fmt.Sprintf("%s/%s.txt", tempDir, fileID)
}

func getRequiredWorkflowEnv(ctx workflow.Context, key string) (string, error) {
	var value string
	if err := workflow.SideEffect(ctx, func(workflow.Context) interface{} {
		return os.Getenv(key)
	}).Get(&value); err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%s environment variable is required", key)
	}
	return value, nil
}

// SingleFileTranscriptionWorkflow processes a single file transcription
func SingleFileTranscriptionWorkflow(ctx workflow.Context, req SingleFileWorkflowRequest) (SingleFileWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting single file transcription workflow", 
		"fileId", req.FileID,
		"filePath", req.FilePath,
		"provider", req.Provider,
		"language", req.Language,
		"outputFormat", req.OutputFormat,
		"useMinIO", req.UseMinIO)

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

	// Step 1: Handle file location (download from MinIO or HTTP URL if needed)
	if req.UseMinIO {
		// Handle MinIO URLs (both minio:// and HTTP pre-signed URLs)
		var objectKey string
		
		if strings.HasPrefix(req.FilePath, "minio://") {
			// Parse minio:// URL
			objectKey = strings.TrimPrefix(req.FilePath, "minio://")
			parts := strings.SplitN(objectKey, "/", 2)
			if len(parts) == 2 {
				objectKey = parts[1]
			}
		} else if strings.HasPrefix(req.FilePath, "http://") || strings.HasPrefix(req.FilePath, "https://") {
			// For HTTP URLs from MinIO, extract the object key from the URL path
			// Example: http://minio:9000/pod0-storage/whisper/user/file.mp3?signature=...
			// We need to extract "whisper/user/file.mp3"
			
			// Parse URL to get path
			urlParts := strings.SplitN(req.FilePath, "?", 2) // Remove query parameters
			urlPath := urlParts[0]
			
			// Extract object key from path (assuming format: /bucket-name/object-key)
			pathParts := strings.Split(urlPath, "/")
			if len(pathParts) >= 5 { // http://host:port/bucket/path...
				// Skip protocol, host, port, and bucket to get object key
				objectKey = strings.Join(pathParts[4:], "/")
			} else {
				logger.Error("Invalid MinIO URL format", "url", req.FilePath)
				return SingleFileWorkflowResult{
					FileID: req.FileID,
					Error:  fmt.Sprintf("Invalid MinIO URL format: %s", req.FilePath),
				}, fmt.Errorf("invalid MinIO URL format")
			}
		} else {
			logger.Error("Unsupported file path format with UseMinIO=true", "path", req.FilePath)
			return SingleFileWorkflowResult{
				FileID: req.FileID,
				Error:  fmt.Sprintf("Unsupported file path format: %s", req.FilePath),
			}, fmt.Errorf("unsupported file path format")
		}

		// Download file from MinIO
		var downloadResult activities.FileDownloadResult
		err = workflow.ExecuteActivity(ctx, "DownloadFile", activities.FileDownloadRequest{
			ObjectKey: objectKey,
		}).Get(ctx, &downloadResult)
		if err != nil {
			logger.Error("Failed to download file from MinIO", "error", err, "objectKey", objectKey)
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
	} else if strings.HasPrefix(req.FilePath, "http://") || strings.HasPrefix(req.FilePath, "https://") {
		// Handle HTTP URLs when UseMinIO is false
		// This shouldn't normally happen, but handle it gracefully
		logger.Error("HTTP URL provided but UseMinIO is false", "url", req.FilePath)
		return SingleFileWorkflowResult{
			FileID: req.FileID,
			Error:  "HTTP URLs require UseMinIO flag to be set",
		}, fmt.Errorf("HTTP URLs require UseMinIO flag")
	} else {
		// Local file path
		localFilePath = req.FilePath
	}

	// Step 2: Calculate audio duration
	var audioDuration int
	// Use SideEffect to execute the audio duration calculation deterministically
	err = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		duration, err := audio.GetAudioDuration(localFilePath)
		if err != nil {
			logger.Warn("Failed to get audio duration, using 0", "error", err)
			return 0
		}
		return duration
	}).Get(&audioDuration)
	if err != nil {
		logger.Warn("Failed to get audio duration from side effect", "error", err)
		audioDuration = 0
	}
	logger.Info("Audio duration calculated", "fileId", req.FileID, "duration", audioDuration)

	// Step 3: Perform transcription
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
		transcriptionKey := buildTranscriptionObjectKey(workflow.Now(ctx), req.FileID)

		// Create temp file for transcription
		// V2T_TEMP_DIR is captured via SideEffect so workflow replay stays deterministic.
		tempDir, err := getRequiredWorkflowEnv(ctx, "V2T_TEMP_DIR")
		if err != nil {
			logger.Error("Missing required environment variable", "error", err)
			return SingleFileWorkflowResult{
				FileID: req.FileID,
				Error:  err.Error(),
			}, err
		}
		tempPath := buildTempTranscriptionPath(tempDir, req.FileID)
		err = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
			return saveTranscriptionContent(tempPath, transcriptionResult.Transcription)
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
			return saveTranscriptionContent(outputPath, transcriptionResult.Transcription)
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
		Transcription:    transcriptionResult.Transcription,  // Include the actual transcription text
		TranscriptionURL: transcriptionURL,
		Provider:         transcriptionResult.Provider,
		ProcessingTime:   processingTime,
		AudioDuration:    audioDuration,
	}

	logger.Info("Single file transcription completed", 
		"fileId", req.FileID,
		"provider", result.Provider,
		"processingTime", result.ProcessingTime,
		"audioDuration", result.AudioDuration)

	return result, nil
}

// saveTranscriptionContent is a helper to save transcription text to a file
func saveTranscriptionContent(path string, content string) error {
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
