package workflows

import (
	"fmt"
	"path/filepath"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"tiktok-whisper/internal/app/common"
)

// ImportVideoWorkflow processes a single Douyin video import
// Orchestrates: Download → Extract Audio → Transcribe → Update Record
func ImportVideoWorkflow(ctx workflow.Context, input common.ImportVideoWorkflowInput) (common.ImportVideoWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Douyin video import workflow",
		"userId", input.UserID,
		"videoUrl", input.VideoURL,
		"jobId", input.JobID,
		"skipTranscription", input.SkipTranscription)

	startTime := workflow.Now(ctx)

	// Configure activity options with retry policy
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute, // Long timeout for video downloads
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    100 * time.Second,
			MaximumAttempts:    3,
			// Don't retry on specific errors
			NonRetryableErrorTypes: []string{
				"VIDEO_UNAVAILABLE",
				"INSUFFICIENT_CREDITS",
			},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := common.ImportVideoWorkflowResult{
		VideoID: input.JobID,
		Success: false,
	}

	// Step 1: Update status to "downloading"
	updateInput := common.UpdateVideoRecordActivityInput{
		JobID:  input.JobID,
		Status: "downloading",
	}
	var updateResult common.UpdateVideoRecordActivityResult
	err := workflow.ExecuteActivity(ctx, "UpdateDouyinVideoRecord", updateInput).Get(ctx, &updateResult)
	if err != nil {
		logger.Warn("Failed to update status to downloading", "error", err)
		// Don't fail workflow if update fails
	}

	// Step 2: Download video and extract audio
	targetPath := filepath.Join("/tmp/whisper-audio", input.UserID, input.JobID)
	downloadInput := common.DownloadVideoActivityInput{
		VideoURL:   input.VideoURL,
		TargetPath: targetPath,
		AudioOnly:  false, // Download full video first
	}

	var downloadResult common.DownloadVideoActivityResult
	err = workflow.ExecuteActivity(ctx, "DownloadDouyinVideo", downloadInput).Get(ctx, &downloadResult)
	if err != nil || !downloadResult.Success {
		logger.Error("Failed to download video", "error", err, "downloadResult", downloadResult)
		result.Error = downloadResult.Error
		result.ErrorCode = downloadResult.ErrorCode

		// Update record with error
		_ = workflow.ExecuteActivity(ctx, "UpdateDouyinVideoRecord", common.UpdateVideoRecordActivityInput{
			JobID:        input.JobID,
			Status:       "failed",
			ErrorMessage: downloadResult.Error,
		}).Get(ctx, nil)

		// Report to DLQ (non-blocking)
		reportWorkflowFailureToDLQ(ctx, "ImportVideoWorkflow", input, downloadResult.Error, map[string]interface{}{
			"errorCode": downloadResult.ErrorCode,
			"videoUrl":  input.VideoURL,
			"step":      "download",
		})

		return result, fmt.Errorf("video download failed: %s", downloadResult.Error)
	}

	// Store metadata
	result.Metadata = downloadResult.Metadata

	// Step 3: Extract audio from video
	audioPath := filepath.Join(targetPath, "audio.wav")
	extractInput := common.ExtractAudioActivityInput{
		VideoFilePath:   downloadResult.VideoFilePath,
		OutputAudioPath: audioPath,
		Format:          "wav",
	}

	var extractResult common.ExtractAudioActivityResult
	err = workflow.ExecuteActivity(ctx, "ExtractAudioFromVideo", extractInput).Get(ctx, &extractResult)
	if err != nil || !extractResult.Success {
		logger.Error("Failed to extract audio", "error", err, "extractResult", extractResult)
		result.Error = extractResult.Error
		result.ErrorCode = "AUDIO_EXTRACTION_FAILED"

		_ = workflow.ExecuteActivity(ctx, "UpdateDouyinVideoRecord", common.UpdateVideoRecordActivityInput{
			JobID:        input.JobID,
			Status:       "failed",
			Metadata:     result.Metadata,
			ErrorMessage: extractResult.Error,
		}).Get(ctx, nil)

		return result, fmt.Errorf("audio extraction failed: %s", extractResult.Error)
	}

	// Step 4: Transcribe audio (if not skipped)
	if !input.SkipTranscription {
		// Update status to "transcribing"
		_ = workflow.ExecuteActivity(ctx, "UpdateDouyinVideoRecord", common.UpdateVideoRecordActivityInput{
			JobID:    input.JobID,
			Status:   "transcribing",
			Metadata: result.Metadata,
		}).Get(ctx, nil)

		// Determine language (default to Chinese)
		language := input.Language
		if language == "" {
			language = "zh"
		}

		// Use existing transcription activity
		transcriptionInput := common.TranscriptionRequest{
			FileID:       input.JobID,
			FilePath:     extractResult.AudioFilePath,
			Provider:     "", // Use default provider
			Language:     language,
			OutputFormat: "json", // Need segments
		}

		var transcriptionResult common.TranscriptionResult
		err = workflow.ExecuteActivity(ctx, "TranscribeFile", transcriptionInput).Get(ctx, &transcriptionResult)
		if err != nil {
			logger.Error("Failed to transcribe audio", "error", err)
			result.Error = fmt.Sprintf("Transcription failed: %v", err)
			result.ErrorCode = "TRANSCRIPTION_FAILED"

			_ = workflow.ExecuteActivity(ctx, "UpdateDouyinVideoRecord", common.UpdateVideoRecordActivityInput{
				JobID:        input.JobID,
				Status:       "failed",
				Metadata:     result.Metadata,
				ErrorMessage: result.Error,
			}).Get(ctx, nil)

			// Report to DLQ
			reportWorkflowFailureToDLQ(ctx, "ImportVideoWorkflow", input, result.Error, map[string]interface{}{
				"errorCode": result.ErrorCode,
				"videoUrl":  input.VideoURL,
				"step":      "transcription",
			})

			return result, err
		}

		result.TranscriptText = transcriptionResult.Transcription

		// Note: The existing TranscribeFile activity doesn't return segments in the format we need
		// We'll use the full text for now. If segments are needed, we can add segment parsing later
	}

	// Step 5: Update record with final result
	finalUpdateInput := common.UpdateVideoRecordActivityInput{
		JobID:          input.JobID,
		Status:         "completed",
		Metadata:       result.Metadata,
		TranscriptText: result.TranscriptText,
	}

	err = workflow.ExecuteActivity(ctx, "UpdateDouyinVideoRecord", finalUpdateInput).Get(ctx, &updateResult)
	if err != nil {
		logger.Warn("Failed to update final record", "error", err)
		// Don't fail workflow if update fails
	}

	// Calculate processing time
	processingTime := workflow.Now(ctx).Sub(startTime)
	result.DurationMs = processingTime.Milliseconds()
	result.Success = true

	logger.Info("Douyin video import completed successfully",
		"jobId", input.JobID,
		"durationMs", result.DurationMs,
		"hasTranscript", result.TranscriptText != "")

	return result, nil
}

// BatchImportVideosWorkflow processes multiple Douyin videos in parallel
func BatchImportVideosWorkflow(ctx workflow.Context, input common.BatchImportVideosWorkflowInput) (common.BatchImportVideosWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting batch Douyin video import workflow",
		"batchId", input.BatchID,
		"videoCount", len(input.VideoURLs),
		"concurrency", input.Options.Concurrency)

	startTime := workflow.Now(ctx)

	result := common.BatchImportVideosWorkflowResult{
		BatchID:            input.BatchID,
		TotalCount:         len(input.VideoURLs),
		SuccessCount:       0,
		FailedCount:        0,
		SuccessfulVideoIDs: []string{},
		FailedVideos:       []common.FailedVideoInfo{},
	}

	// Default concurrency to 5
	concurrency := input.Options.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	// Create a channel to limit concurrency
	selector := workflow.NewSelector(ctx)
	pending := 0
	maxConcurrent := concurrency
	videoIndex := 0
	videoResults := make(map[string]common.ImportVideoWorkflowResult)

	// Process videos with concurrency control
	for pending > 0 || videoIndex < len(input.VideoURLs) {
		// Start new workflows if under concurrency limit
		for pending < maxConcurrent && videoIndex < len(input.VideoURLs) {
			videoURL := input.VideoURLs[videoIndex]
			videoJobID := fmt.Sprintf("%s-%d", input.BatchID, videoIndex)

			// Start child workflow for each video
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID:            videoJobID,
				WorkflowExecutionTimeout: 1 * time.Hour,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    100 * time.Second,
					MaximumAttempts:    3,
				},
			})

			childInput := common.ImportVideoWorkflowInput{
				UserID:            input.UserID,
				VideoURL:          videoURL,
				JobID:             videoJobID,
				SkipTranscription: input.Options.SkipTranscription,
				Language:          input.Options.Language,
			}

			future := workflow.ExecuteChildWorkflow(childCtx, ImportVideoWorkflow, childInput)

			// Capture variables for closure
			currentVideoURL := videoURL
			currentJobID := videoJobID

			selector.AddFuture(future, func(f workflow.Future) {
				var childResult common.ImportVideoWorkflowResult
				err := f.Get(ctx, &childResult)

				if err != nil || !childResult.Success {
					logger.Warn("Video import failed",
						"videoUrl", currentVideoURL,
						"jobId", currentJobID,
						"error", err)

					result.FailedCount++
					result.FailedVideos = append(result.FailedVideos, common.FailedVideoInfo{
						VideoURL: currentVideoURL,
						Error:    childResult.Error,
					})
				} else {
					result.SuccessCount++
					result.SuccessfulVideoIDs = append(result.SuccessfulVideoIDs, currentJobID)
				}

				videoResults[currentJobID] = childResult
				pending--
			})

			pending++
			videoIndex++
		}

		// Wait for at least one to complete if we're at max concurrency
		if pending >= maxConcurrent || (pending > 0 && videoIndex >= len(input.VideoURLs)) {
			selector.Select(ctx)
		}
	}

	// Calculate total processing time
	processingTime := workflow.Now(ctx).Sub(startTime)
	result.DurationMs = processingTime.Milliseconds()

	logger.Info("Batch Douyin video import completed",
		"batchId", input.BatchID,
		"totalCount", result.TotalCount,
		"successCount", result.SuccessCount,
		"failedCount", result.FailedCount,
		"durationMs", result.DurationMs)

	return result, nil
}

// ========================================
// Engagement & Comments Scraping Workflows
// ========================================

// ScrapeEngagementWorkflow scrapes engagement data for a Douyin video
func ScrapeEngagementWorkflow(ctx workflow.Context, input common.ScrapeEngagementWorkflowInput) (common.ScrapeEngagementWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting engagement scraping workflow",
		"userId", input.UserID,
		"videoId", input.VideoID,
		"jobId", input.JobID)

	startTime := workflow.Now(ctx)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := common.ScrapeEngagementWorkflowResult{
		VideoID: input.VideoID,
		Success: false,
	}

	// Step 1: Scrape engagement data from Douyin
	scrapeInput := common.ScrapeEngagementActivityInput{
		VideoID: input.VideoID,
	}

	var scrapeResult common.ScrapeEngagementActivityResult
	err := workflow.ExecuteActivity(ctx, "ScrapeDouyinEngagement", scrapeInput).Get(ctx, &scrapeResult)
	if err != nil || !scrapeResult.Success {
		logger.Error("Failed to scrape engagement data", "error", err, "scrapeResult", scrapeResult)
		result.Error = scrapeResult.Error
		result.ErrorCode = scrapeResult.ErrorCode
		return result, fmt.Errorf("engagement scraping failed: %s", scrapeResult.Error)
	}

	result.Engagement = scrapeResult.Engagement

	// Step 2: Update database record
	updateInput := common.UpdateEngagementRecordActivityInput{
		VideoID:    input.VideoID,
		Engagement: scrapeResult.Engagement,
	}

	var updateResult common.UpdateEngagementRecordActivityResult
	err = workflow.ExecuteActivity(ctx, "UpdateEngagementRecord", updateInput).Get(ctx, &updateResult)
	if err != nil {
		logger.Warn("Failed to update engagement record", "error", err)
		// Don't fail workflow if database update fails
	}

	processingTime := workflow.Now(ctx).Sub(startTime)
	result.DurationMs = processingTime.Milliseconds()
	result.Success = true

	logger.Info("Engagement scraping completed successfully",
		"videoId", input.VideoID,
		"durationMs", result.DurationMs)

	return result, nil
}

// ScrapeCommentsWorkflow scrapes comments for a Douyin video
func ScrapeCommentsWorkflow(ctx workflow.Context, input common.ScrapeCommentsWorkflowInput) (common.ScrapeCommentsWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting comments scraping workflow",
		"userId", input.UserID,
		"videoId", input.VideoID,
		"jobId", input.JobID,
		"limit", input.Limit)

	startTime := workflow.Now(ctx)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute, // Comments can take longer
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := common.ScrapeCommentsWorkflowResult{
		VideoID: input.VideoID,
		Success: false,
	}

	// Step 1: Scrape comments from Douyin
	scrapeInput := common.ScrapeCommentsActivityInput{
		VideoID:        input.VideoID,
		Limit:          input.Limit,
		IncludeReplies: input.IncludeReplies,
	}

	var scrapeResult common.ScrapeCommentsActivityResult
	err := workflow.ExecuteActivity(ctx, "ScrapeDouyinComments", scrapeInput).Get(ctx, &scrapeResult)
	if err != nil || !scrapeResult.Success {
		logger.Error("Failed to scrape comments", "error", err, "scrapeResult", scrapeResult)
		result.Error = scrapeResult.Error
		result.ErrorCode = scrapeResult.ErrorCode
		return result, fmt.Errorf("comment scraping failed: %s", scrapeResult.Error)
	}

	result.Comments = scrapeResult.Comments
	result.CommentCount = len(scrapeResult.Comments)

	// Step 2: Update database record
	updateInput := common.UpdateCommentsRecordActivityInput{
		VideoID:  input.VideoID,
		Comments: scrapeResult.Comments,
	}

	var updateResult common.UpdateCommentsRecordActivityResult
	err = workflow.ExecuteActivity(ctx, "UpdateCommentsRecord", updateInput).Get(ctx, &updateResult)
	if err != nil {
		logger.Warn("Failed to update comments record", "error", err)
		// Don't fail workflow if database update fails
	}

	processingTime := workflow.Now(ctx).Sub(startTime)
	result.DurationMs = processingTime.Milliseconds()
	result.Success = true

	logger.Info("Comments scraping completed successfully",
		"videoId", input.VideoID,
		"commentCount", result.CommentCount,
		"durationMs", result.DurationMs)

	return result, nil
}

// ========================================
// Report Generation Workflows
// ========================================

// GenerateReportWorkflow generates an AI analysis report for a Douyin video
func GenerateReportWorkflow(ctx workflow.Context, input common.GenerateReportWorkflowInput) (common.GenerateReportWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting report generation workflow",
		"userId", input.UserID,
		"videoId", input.VideoID,
		"jobId", input.JobID)

	startTime := workflow.Now(ctx)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute, // AI generation can be slow
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    60 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := common.GenerateReportWorkflowResult{
		VideoID: input.VideoID,
		Success: false,
	}

	// Step 1: Fetch video data (metadata, transcript, engagement, comments)
	// This will be done by calling Next.js API to get the data
	var metadata *common.DouyinVideoMetadata
	var transcript string
	var engagement *common.DouyinEngagement
	var comments []common.DouyinComment

	// For now, we'll rely on the data being available in the database
	// The activity will fetch it

	// Step 2: Generate AI report
	reportInput := common.GenerateAIReportActivityInput{
		VideoID:    input.VideoID,
		Metadata:   metadata,
		Transcript: transcript,
		Engagement: engagement,
		Comments:   comments,
		PromptID:   input.PromptID,
	}

	var reportResult common.GenerateAIReportActivityResult
	err := workflow.ExecuteActivity(ctx, "GenerateAIReport", reportInput).Get(ctx, &reportResult)
	if err != nil || !reportResult.Success {
		logger.Error("Failed to generate report", "error", err, "reportResult", reportResult)
		result.Error = reportResult.Error
		result.ErrorCode = reportResult.ErrorCode

		// Update session with error
		if input.SessionID != "" {
			_ = workflow.ExecuteActivity(ctx, "UpdateReportRecord", common.UpdateReportRecordActivityInput{
				SessionID: input.SessionID,
				Status:    "failed",
				Error:     reportResult.Error,
			}).Get(ctx, nil)
		}

		return result, fmt.Errorf("report generation failed: %s", reportResult.Error)
	}

	result.ReportText = reportResult.ReportText
	result.SessionID = input.SessionID

	// Step 3: Update session record
	if input.SessionID != "" {
		updateInput := common.UpdateReportRecordActivityInput{
			SessionID:  input.SessionID,
			ReportText: reportResult.ReportText,
			Status:     "completed",
		}

		var updateResult common.UpdateReportRecordActivityResult
		err = workflow.ExecuteActivity(ctx, "UpdateReportRecord", updateInput).Get(ctx, &updateResult)
		if err != nil {
			logger.Warn("Failed to update report record", "error", err)
		}
	}

	processingTime := workflow.Now(ctx).Sub(startTime)
	result.DurationMs = processingTime.Milliseconds()
	result.Success = true

	logger.Info("Report generation completed successfully",
		"videoId", input.VideoID,
		"sessionId", result.SessionID,
		"durationMs", result.DurationMs)

	return result, nil
}

// BatchGenerateReportsWorkflow generates AI reports for multiple videos
func BatchGenerateReportsWorkflow(ctx workflow.Context, input common.BatchGenerateReportsWorkflowInput) (common.BatchGenerateReportsWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting batch report generation workflow",
		"batchId", input.BatchID,
		"videoCount", len(input.VideoIDs),
		"concurrency", input.Options.Concurrency)

	startTime := workflow.Now(ctx)

	result := common.BatchGenerateReportsWorkflowResult{
		BatchID:           input.BatchID,
		TotalCount:        len(input.VideoIDs),
		SuccessCount:      0,
		FailedCount:       0,
		SuccessfulReports: []common.GeneratedReportInfo{},
		FailedReports:     []common.FailedReportInfo{},
	}

	// Default concurrency to 3 (AI is expensive)
	concurrency := input.Options.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}

	// Process videos with concurrency control
	selector := workflow.NewSelector(ctx)
	pending := 0
	maxConcurrent := concurrency
	videoIndex := 0

	for pending > 0 || videoIndex < len(input.VideoIDs) {
		// Start new workflows if under concurrency limit
		for pending < maxConcurrent && videoIndex < len(input.VideoIDs) {
			videoID := input.VideoIDs[videoIndex]
			jobID := fmt.Sprintf("%s-%d", input.BatchID, videoIndex)

			// Start child workflow for each video
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID:               jobID,
				WorkflowExecutionTimeout: 30 * time.Minute,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    60 * time.Second,
					MaximumAttempts:    2, // Limited retries for AI
				},
			})

			childInput := common.GenerateReportWorkflowInput{
				UserID:   input.UserID,
				VideoID:  videoID,
				JobID:    jobID,
				PromptID: input.PromptID,
			}

			future := workflow.ExecuteChildWorkflow(childCtx, GenerateReportWorkflow, childInput)

			// Capture variables for closure
			currentVideoID := videoID

			selector.AddFuture(future, func(f workflow.Future) {
				var childResult common.GenerateReportWorkflowResult
				err := f.Get(ctx, &childResult)

				if err != nil || !childResult.Success {
					logger.Warn("Report generation failed",
						"videoId", currentVideoID,
						"error", err)

					result.FailedCount++
					result.FailedReports = append(result.FailedReports, common.FailedReportInfo{
						VideoID: currentVideoID,
						Error:   childResult.Error,
					})
				} else {
					result.SuccessCount++
					result.SuccessfulReports = append(result.SuccessfulReports, common.GeneratedReportInfo{
						VideoID:   currentVideoID,
						SessionID: childResult.SessionID,
					})
				}

				pending--
			})

			pending++
			videoIndex++
		}

		// Wait for at least one to complete
		if pending >= maxConcurrent || (pending > 0 && videoIndex >= len(input.VideoIDs)) {
			selector.Select(ctx)
		}
	}

	// Calculate total processing time
	processingTime := workflow.Now(ctx).Sub(startTime)
	result.DurationMs = processingTime.Milliseconds()

	logger.Info("Batch report generation completed",
		"batchId", input.BatchID,
		"totalCount", result.TotalCount,
		"successCount", result.SuccessCount,
		"failedCount", result.FailedCount,
		"durationMs", result.DurationMs)

	return result, nil
}


// reportWorkflowFailureToDLQ is a helper function to report workflow failures to DLQ
// It executes the ReportFailedWorkflow activity with best-effort semantics
func reportWorkflowFailureToDLQ(
	ctx workflow.Context,
	workflowType string,
	workflowInput interface{},
	errorMsg string,
	metadata map[string]interface{},
) {
	logger := workflow.GetLogger(ctx)

	// Get workflow ID from context
	workflowInfo := workflow.GetInfo(ctx)
	workflowID := workflowInfo.WorkflowExecution.ID

	// Configure activity with short timeout (don't delay workflow completion)
	dlqActivityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2, // Only retry once
		},
	}
	dlqCtx := workflow.WithActivityOptions(ctx, dlqActivityOptions)

	// Import DLQ activity type
	type ReportFailedWorkflowInput struct {
		WorkflowID   string                 `json:"workflowId"`
		WorkflowType string                 `json:"workflowType"`
		Input        interface{}            `json:"input"`
		Error        string                 `json:"error"`
		ErrorStack   string                 `json:"errorStack,omitempty"`
		Metadata     map[string]interface{} `json:"metadata,omitempty"`
	}

	type ReportFailedWorkflowResult struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	dlqInput := ReportFailedWorkflowInput{
		WorkflowID:   workflowID,
		WorkflowType: workflowType,
		Input:        workflowInput,
		Error:        errorMsg,
		Metadata:     metadata,
	}

	var dlqResult ReportFailedWorkflowResult
	err := workflow.ExecuteActivity(dlqCtx, "ReportFailedWorkflow", dlqInput).Get(dlqCtx, &dlqResult)
	if err != nil {
		logger.Warn("Failed to report workflow to DLQ (non-critical)", "error", err)
	} else if !dlqResult.Success {
		logger.Warn("DLQ reporting returned failure (non-critical)", "dlqError", dlqResult.Error)
	} else {
		logger.Info("Successfully reported failed workflow to DLQ", "workflowId", workflowID)
	}
}
