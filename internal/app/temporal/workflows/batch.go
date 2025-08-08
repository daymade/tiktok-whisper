package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// BatchWorkflowRequest represents the input for batch transcription workflow
type BatchWorkflowRequest struct {
	BatchID      string                 `json:"batch_id"`
	Files        []BatchFile            `json:"files"`
	Provider     string                 `json:"provider"`      // Optional, uses default or per-file override
	Language     string                 `json:"language"`
	MaxParallel  int                    `json:"max_parallel"`  // Max concurrent transcriptions
	UseMinIO     bool                   `json:"use_minio"`
	Options      map[string]interface{} `json:"options"`
}

// BatchFile represents a file in the batch
type BatchFile struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	Provider string `json:"provider,omitempty"` // Optional provider override
}

// BatchWorkflowResult represents the output of batch transcription workflow
type BatchWorkflowResult struct {
	BatchID        string                      `json:"batch_id"`
	TotalFiles     int                         `json:"total_files"`
	SuccessCount   int                         `json:"success_count"`
	FailureCount   int                         `json:"failure_count"`
	Results        []SingleFileWorkflowResult  `json:"results"`
	ProcessingTime time.Duration               `json:"processing_time"`
}

// BatchTranscriptionWorkflow processes multiple files in parallel
func BatchTranscriptionWorkflow(ctx workflow.Context, req BatchWorkflowRequest) (BatchWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting batch transcription workflow", 
		"batchId", req.BatchID, 
		"fileCount", len(req.Files),
		"maxParallel", req.MaxParallel)

	startTime := workflow.Now(ctx)

	// Default max parallel if not specified
	if req.MaxParallel <= 0 {
		req.MaxParallel = 5
	}

	// Create a buffered channel for controlling parallelism
	semaphore := workflow.NewBufferedChannel(ctx, req.MaxParallel)
	defer semaphore.Close()

	// Fill the semaphore
	for i := 0; i < req.MaxParallel; i++ {
		semaphore.Send(ctx, struct{}{})
	}

	// Results channel
	resultsChan := workflow.NewBufferedChannel(ctx, len(req.Files))
	defer resultsChan.Close()

	// Process files in parallel
	for _, file := range req.Files {
		workflow.Go(ctx, func(ctx workflow.Context) {
			// Acquire semaphore
			var token struct{}
			semaphore.Receive(ctx, &token)
			defer semaphore.Send(ctx, token)

			// Determine provider for this file
			provider := file.Provider
			if provider == "" {
				provider = req.Provider
			}

			// Configure child workflow options
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("%s-%s", req.BatchID, file.FileID),
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    100 * time.Second,
					MaximumAttempts:    2,
				},
			})

			// Execute single file workflow
			var result SingleFileWorkflowResult
			err := workflow.ExecuteChildWorkflow(childCtx, 
				SingleFileTranscriptionWorkflow,
				SingleFileWorkflowRequest{
					FileID:       file.FileID,
					FilePath:     file.FilePath,
					Provider:     provider,
					Language:     req.Language,
					OutputFormat: "text",
					Options:      req.Options,
					UseMinIO:     req.UseMinIO,
				}).Get(childCtx, &result)

			if err != nil {
				logger.Error("File transcription failed", 
					"fileId", file.FileID, 
					"error", err)
				result = SingleFileWorkflowResult{
					FileID: file.FileID,
					Error:  err.Error(),
				}
			}

			resultsChan.Send(ctx, result)
		})
	}

	// Collect results
	results := make([]SingleFileWorkflowResult, 0, len(req.Files))
	successCount := 0
	failureCount := 0

	for i := 0; i < len(req.Files); i++ {
		var result SingleFileWorkflowResult
		resultsChan.Receive(ctx, &result)
		results = append(results, result)

		if result.Error == "" {
			successCount++
		} else {
			failureCount++
		}

		// Log progress
		if (i+1)%10 == 0 || i+1 == len(req.Files) {
			logger.Info("Batch progress", 
				"completed", i+1,
				"total", len(req.Files),
				"success", successCount,
				"failed", failureCount)
		}
	}

	processingTime := workflow.Now(ctx).Sub(startTime)

	batchResult := BatchWorkflowResult{
		BatchID:        req.BatchID,
		TotalFiles:     len(req.Files),
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		Results:        results,
		ProcessingTime: processingTime,
	}

	logger.Info("Batch transcription completed",
		"batchId", req.BatchID,
		"totalFiles", batchResult.TotalFiles,
		"success", batchResult.SuccessCount,
		"failed", batchResult.FailureCount,
		"duration", batchResult.ProcessingTime)

	return batchResult, nil
}

// BatchWithRetryWorkflow processes batch with automatic retry for failed files
func BatchWithRetryWorkflow(ctx workflow.Context, req BatchWorkflowRequest) (BatchWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting batch workflow with retry", "batchId", req.BatchID)

	// First attempt
	firstResult, err := BatchTranscriptionWorkflow(ctx, req)
	if err != nil {
		return firstResult, err
	}

	// If all succeeded, return
	if firstResult.FailureCount == 0 {
		return firstResult, nil
	}

	// Collect failed files for retry
	failedFiles := make([]BatchFile, 0, firstResult.FailureCount)
	for i, result := range firstResult.Results {
		if result.Error != "" {
			failedFiles = append(failedFiles, req.Files[i])
		}
	}

	logger.Info("Retrying failed files", 
		"count", len(failedFiles),
		"batchId", req.BatchID)

	// Wait before retry
	workflow.Sleep(ctx, 30*time.Second)

	// Retry failed files with different provider if specified
	retryReq := req
	retryReq.Files = failedFiles
	if req.Provider == "whisper_cpp" {
		retryReq.Provider = "openai" // Fallback to OpenAI
	}

	retryResult, err := BatchTranscriptionWorkflow(ctx, retryReq)
	if err != nil {
		return firstResult, err
	}

	// Merge results
	finalResult := BatchWorkflowResult{
		BatchID:        req.BatchID,
		TotalFiles:     firstResult.TotalFiles,
		SuccessCount:   firstResult.SuccessCount + retryResult.SuccessCount,
		FailureCount:   retryResult.FailureCount,
		Results:        append(firstResult.Results, retryResult.Results...),
		ProcessingTime: firstResult.ProcessingTime + retryResult.ProcessingTime,
	}

	return finalResult, nil
}