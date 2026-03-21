package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.temporal.io/sdk/activity"
	"tiktok-whisper/internal/app/audio"
	"tiktok-whisper/internal/app/common"
	"tiktok-whisper/internal/app/temporal/pkg/metrics"
)

// DouyinActivities provides activities for Douyin video processing
type DouyinActivities struct {
	douyinAPIEndpoint string // External Douyin scraping service endpoint
	douyinAPIKey      string
	httpClient        *http.Client
	tempDir           string
	whisperProvider   string // Provider name for transcription
}

// NewDouyinActivities creates a new instance of Douyin activities
func NewDouyinActivities(douyinAPIEndpoint, douyinAPIKey, tempDir, whisperProvider string) (*DouyinActivities, error) {
	if douyinAPIEndpoint == "" {
		return nil, fmt.Errorf("DOUYIN_API_ENDPOINT environment variable is required")
	}
	if douyinAPIKey == "" {
		return nil, fmt.Errorf("DOUYIN_API_KEY environment variable is required")
	}
	if tempDir == "" {
		return nil, fmt.Errorf("V2T_TEMP_DIR environment variable is required")
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &DouyinActivities{
		douyinAPIEndpoint: douyinAPIEndpoint,
		douyinAPIKey:      douyinAPIKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for video downloads
		},
		tempDir:         tempDir,
		whisperProvider: whisperProvider,
	}, nil
}

// DownloadDouyinVideo downloads a Douyin video and extracts audio
// This calls the external Douyin scraping service instead of implementing scraping logic
func (a *DouyinActivities) DownloadDouyinVideo(ctx context.Context, input common.DownloadVideoActivityInput) (common.DownloadVideoActivityResult, error) {
	startTime := time.Now()
	logger := activity.GetLogger(ctx)
	logger.Info("Starting Douyin video download", "videoUrl", input.VideoURL)

	// Record heartbeat
	activity.RecordHeartbeat(ctx, "Downloading video from external service")

	// Step 1: Call external Douyin API to download video
	downloadReq := map[string]interface{}{
		"video_url":  input.VideoURL,
		"audio_only": input.AudioOnly,
	}

	reqBody, err := json.Marshal(downloadReq)
	if err != nil {
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to marshal request: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/video/download", a.douyinAPIEndpoint),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to create request: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", a.douyinAPIKey)

	// Execute request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		// Record metrics on error
		if m := metrics.GetGlobalDouyinMetrics(); m != nil {
			m.RecordActivityExecution("DownloadDouyinVideo", false, time.Since(startTime).Seconds())
			m.RecordActivityError("DownloadDouyinVideo", "NETWORK_ERROR")
		}
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to download video: %v", err),
			ErrorCode: "NETWORK_ERROR",
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Download failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
			ErrorCode: "VIDEO_UNAVAILABLE",
		}, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Parse response
	var downloadResp struct {
		Success       bool                     `json:"success"`
		VideoFileURL  string                   `json:"video_file_url"`
		AudioFileURL  string                   `json:"audio_file_url"`
		FileSize      int64                    `json:"file_size"`
		Duration      int                      `json:"duration"`
		Metadata      *common.DouyinVideoMetadata `json:"metadata"`
		Error         string                   `json:"error"`
		ErrorCode     string                   `json:"error_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&downloadResp); err != nil {
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to parse response: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	if !downloadResp.Success {
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     downloadResp.Error,
			ErrorCode: downloadResp.ErrorCode,
		}, fmt.Errorf("download failed: %s", downloadResp.Error)
	}

	// Step 2: Download file to local shared directory
	var localFilePath string
	var downloadURL string

	if input.AudioOnly && downloadResp.AudioFileURL != "" {
		downloadURL = downloadResp.AudioFileURL
		localFilePath = filepath.Join(input.TargetPath, "audio.mp3")
	} else {
		downloadURL = downloadResp.VideoFileURL
		localFilePath = filepath.Join(input.TargetPath, "video.mp4")
	}

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to create directory: %v", err),
			ErrorCode: "STORAGE_FULL",
		}, err
	}

	// Download file from URL
	activity.RecordHeartbeat(ctx, "Downloading file to local storage")
	fileResp, err := a.httpClient.Get(downloadURL)
	if err != nil {
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to download file from URL: %v", err),
			ErrorCode: "NETWORK_ERROR",
		}, err
	}
	defer fileResp.Body.Close()

	outFile, err := os.Create(localFilePath)
	if err != nil {
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to create file: %v", err),
			ErrorCode: "STORAGE_FULL",
		}, err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, fileResp.Body)
	if err != nil {
		return common.DownloadVideoActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to save file: %v", err),
			ErrorCode: "STORAGE_FULL",
		}, err
	}

	result := common.DownloadVideoActivityResult{
		Success:  true,
		FileSize: downloadResp.FileSize,
		Duration: downloadResp.Duration,
		Metadata: downloadResp.Metadata,
	}

	if input.AudioOnly {
		result.AudioFilePath = localFilePath
	} else {
		result.VideoFilePath = localFilePath
	}

	logger.Info("Video download completed",
		"videoFilePath", result.VideoFilePath,
		"audioFilePath", result.AudioFilePath,
		"fileSize", result.FileSize)

	// Record metrics
	if m := metrics.GetGlobalDouyinMetrics(); m != nil {
		duration := time.Since(startTime).Seconds()
		m.RecordActivityExecution("DownloadDouyinVideo", true, duration)
		m.VideoDownloadBytes.Add(float64(result.FileSize))
	}

	return result, nil
}

// ExtractAudioFromVideo extracts audio from video file using ffmpeg
func (a *DouyinActivities) ExtractAudioFromVideo(ctx context.Context, input common.ExtractAudioActivityInput) (common.ExtractAudioActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Extracting audio from video", "videoPath", input.VideoFilePath)

	activity.RecordHeartbeat(ctx, "Extracting audio with ffmpeg")

	// Validate input file exists
	if _, err := os.Stat(input.VideoFilePath); err != nil {
		return common.ExtractAudioActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Video file not found: %v", err),
		}, err
	}

	// Determine output format (default: wav)
	format := input.Format
	if format == "" {
		format = "wav"
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(input.OutputAudioPath), 0755); err != nil {
		return common.ExtractAudioActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create output directory: %v", err),
		}, err
	}

	// Use ffmpeg to extract audio
	// Convert to 16kHz mono WAV for Whisper
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", input.VideoFilePath,
		"-vn", // No video
		"-acodec", "pcm_s16le", // 16-bit PCM
		"-ar", "16000", // 16kHz sample rate
		"-ac", "1", // Mono
		"-y", // Overwrite output file
		input.OutputAudioPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return common.ExtractAudioActivityResult{
			Success: false,
			Error:   fmt.Sprintf("ffmpeg failed: %v, output: %s", err, string(output)),
		}, err
	}

	// Get file size
	fileInfo, err := os.Stat(input.OutputAudioPath)
	if err != nil {
		return common.ExtractAudioActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to stat audio file: %v", err),
		}, err
	}

	// Get audio duration
	duration, err := audio.GetAudioDuration(input.OutputAudioPath)
	if err != nil {
		logger.Warn("Failed to get audio duration", "error", err)
		duration = 0
	}

	result := common.ExtractAudioActivityResult{
		Success:       true,
		AudioFilePath: input.OutputAudioPath,
		FileSize:      fileInfo.Size(),
		Duration:      duration,
	}

	logger.Info("Audio extraction completed",
		"audioPath", result.AudioFilePath,
		"fileSize", result.FileSize,
		"duration", result.Duration)

	return result, nil
}

// TranscribeDouyinAudio transcribes audio using existing Whisper transcription
func (a *DouyinActivities) TranscribeDouyinAudio(ctx context.Context, input common.TranscribeAudioActivityInput) (common.TranscribeAudioActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Transcribing audio", "audioPath", input.AudioFilePath, "language", input.Language)

	activity.RecordHeartbeat(ctx, "Starting transcription")

	// This activity is designed to be called through workflow orchestration
	// The workflow will call the existing TranscribeActivities.TranscribeFile activity
	// For direct calls, we return an error to indicate incorrect usage

	return common.TranscribeAudioActivityResult{
		Success: false,
		Error:   "This activity should be called through workflow orchestration",
	}, fmt.Errorf("direct call not supported, use workflow orchestration")
}

// UpdateDouyinVideoRecord updates the video record in Next.js database via HTTP callback
func (a *DouyinActivities) UpdateDouyinVideoRecord(ctx context.Context, input common.UpdateVideoRecordActivityInput) (common.UpdateVideoRecordActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating video record", "jobId", input.JobID, "status", input.Status)

	activity.RecordHeartbeat(ctx, "Updating database record")

	// Get Next.js API endpoint from environment
	nextjsAPIEndpoint := os.Getenv("NEXTJS_API_ENDPOINT")
	if nextjsAPIEndpoint == "" {
		logger.Warn("NEXTJS_API_ENDPOINT not set, skipping database update")
		return common.UpdateVideoRecordActivityResult{
			Success: true, // Don't fail the workflow if callback is not configured
		}, nil
	}

	apiKey := os.Getenv("NEXTJS_API_KEY")
	if apiKey == "" {
		logger.Warn("NEXTJS_API_KEY not set, skipping database update")
		return common.UpdateVideoRecordActivityResult{
			Success: true,
		}, nil
	}

	// Prepare update payload
	updateReq := map[string]interface{}{
		"jobId":  input.JobID,
		"status": input.Status,
	}

	if input.Metadata != nil {
		updateReq["metadata"] = input.Metadata
	}

	if input.TranscriptText != "" {
		updateReq["transcriptText"] = input.TranscriptText
	}

	if len(input.TranscriptSegments) > 0 {
		updateReq["transcriptSegments"] = input.TranscriptSegments
	}

	if input.ErrorMessage != "" {
		updateReq["errorMessage"] = input.ErrorMessage
	}

	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return common.UpdateVideoRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to marshal request: %v", err),
		}, err
	}

	// Call Next.js API
	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/api/douyin/jobs/%s", nextjsAPIEndpoint, input.JobID),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return common.UpdateVideoRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create request: %v", err),
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return common.UpdateVideoRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to update record: %v", err),
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return common.UpdateVideoRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Update failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
		}, fmt.Errorf("update failed with status %d", resp.StatusCode)
	}

	logger.Info("Video record updated successfully", "jobId", input.JobID)

	return common.UpdateVideoRecordActivityResult{
		Success: true,
	}, nil
}

// ========================================
// Engagement & Comments Scraping Activities
// ========================================

// ScrapeDouyinEngagement scrapes engagement data (likes, comments, shares) for a video
func (a *DouyinActivities) ScrapeDouyinEngagement(ctx context.Context, input common.ScrapeEngagementActivityInput) (common.ScrapeEngagementActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Scraping engagement data", "videoId", input.VideoID)

	activity.RecordHeartbeat(ctx, "Scraping engagement from external service")

	// Prepare request
	scrapeReq := map[string]interface{}{
		"video_id": input.VideoID,
	}

	reqBody, err := json.Marshal(scrapeReq)
	if err != nil {
		return common.ScrapeEngagementActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to marshal request: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	// Call external Douyin API
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/engagement/scrape", a.douyinAPIEndpoint),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return common.ScrapeEngagementActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to create request: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", a.douyinAPIKey)

	// Execute request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return common.ScrapeEngagementActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to scrape engagement: %v", err),
			ErrorCode: "NETWORK_ERROR",
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return common.ScrapeEngagementActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Scraping failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
			ErrorCode: "SCRAPE_FAILED",
		}, fmt.Errorf("scrape failed with status %d", resp.StatusCode)
	}

	// Parse response
	var scrapeResp struct {
		Success    bool                   `json:"success"`
		Engagement *common.DouyinEngagement `json:"engagement"`
		Error      string                  `json:"error"`
		ErrorCode  string                  `json:"error_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&scrapeResp); err != nil {
		return common.ScrapeEngagementActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to parse response: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	if !scrapeResp.Success {
		return common.ScrapeEngagementActivityResult{
			Success:   false,
			Error:     scrapeResp.Error,
			ErrorCode: scrapeResp.ErrorCode,
		}, fmt.Errorf("scrape failed: %s", scrapeResp.Error)
	}

	logger.Info("Engagement data scraped successfully",
		"videoId", input.VideoID,
		"likesCount", scrapeResp.Engagement.LikesCount,
		"commentsCount", scrapeResp.Engagement.CommentsCount)

	return common.ScrapeEngagementActivityResult{
		Success:    true,
		Engagement: scrapeResp.Engagement,
	}, nil
}

// UpdateEngagementRecord updates engagement data in Next.js database via HTTP callback
func (a *DouyinActivities) UpdateEngagementRecord(ctx context.Context, input common.UpdateEngagementRecordActivityInput) (common.UpdateEngagementRecordActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating engagement record", "videoId", input.VideoID)

	activity.RecordHeartbeat(ctx, "Updating engagement database record")

	// Get Next.js API endpoint from environment
	nextjsAPIEndpoint := os.Getenv("NEXTJS_API_ENDPOINT")
	if nextjsAPIEndpoint == "" {
		logger.Warn("NEXTJS_API_ENDPOINT not set, skipping database update")
		return common.UpdateEngagementRecordActivityResult{
			Success: true, // Don't fail the workflow
		}, nil
	}

	apiKey := os.Getenv("NEXTJS_API_KEY")
	if apiKey == "" {
		logger.Warn("NEXTJS_API_KEY not set, skipping database update")
		return common.UpdateEngagementRecordActivityResult{
			Success: true,
		}, nil
	}

	// Prepare update payload
	updateReq := map[string]interface{}{
		"videoId":        input.VideoID,
		"likesCount":     input.Engagement.LikesCount,
		"commentsCount":  input.Engagement.CommentsCount,
		"sharesCount":    input.Engagement.SharesCount,
		"favoritesCount": input.Engagement.FavoritesCount,
		"viewsCount":     input.Engagement.ViewsCount,
		"scrapedAt":      input.Engagement.ScrapedAt,
	}

	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return common.UpdateEngagementRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to marshal request: %v", err),
		}, err
	}

	// Call Next.js API
	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/api/douyin/internal/engagement/%s", nextjsAPIEndpoint, input.VideoID),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return common.UpdateEngagementRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create request: %v", err),
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return common.UpdateEngagementRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to update record: %v", err),
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return common.UpdateEngagementRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Update failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
		}, fmt.Errorf("update failed with status %d", resp.StatusCode)
	}

	logger.Info("Engagement record updated successfully", "videoId", input.VideoID)

	return common.UpdateEngagementRecordActivityResult{
		Success: true,
	}, nil
}

// ScrapeDouyinComments scrapes comments for a video
func (a *DouyinActivities) ScrapeDouyinComments(ctx context.Context, input common.ScrapeCommentsActivityInput) (common.ScrapeCommentsActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Scraping comments", "videoId", input.VideoID, "limit", input.Limit)

	activity.RecordHeartbeat(ctx, "Scraping comments from external service")

	// Prepare request
	scrapeReq := map[string]interface{}{
		"video_id":        input.VideoID,
		"limit":           input.Limit,
		"include_replies": input.IncludeReplies,
	}

	reqBody, err := json.Marshal(scrapeReq)
	if err != nil {
		return common.ScrapeCommentsActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to marshal request: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	// Call external Douyin API
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/comments/scrape", a.douyinAPIEndpoint),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return common.ScrapeCommentsActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to create request: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", a.douyinAPIKey)

	// Execute request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return common.ScrapeCommentsActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to scrape comments: %v", err),
			ErrorCode: "NETWORK_ERROR",
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return common.ScrapeCommentsActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Scraping failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
			ErrorCode: "SCRAPE_FAILED",
		}, fmt.Errorf("scrape failed with status %d", resp.StatusCode)
	}

	// Parse response
	var scrapeResp struct {
		Success   bool                `json:"success"`
		Comments  []common.DouyinComment `json:"comments"`
		Error     string              `json:"error"`
		ErrorCode string              `json:"error_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&scrapeResp); err != nil {
		return common.ScrapeCommentsActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to parse response: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	if !scrapeResp.Success {
		return common.ScrapeCommentsActivityResult{
			Success:   false,
			Error:     scrapeResp.Error,
			ErrorCode: scrapeResp.ErrorCode,
		}, fmt.Errorf("scrape failed: %s", scrapeResp.Error)
	}

	logger.Info("Comments scraped successfully",
		"videoId", input.VideoID,
		"commentCount", len(scrapeResp.Comments))

	return common.ScrapeCommentsActivityResult{
		Success:  true,
		Comments: scrapeResp.Comments,
	}, nil
}

// UpdateCommentsRecord updates comments data in Next.js database via HTTP callback
func (a *DouyinActivities) UpdateCommentsRecord(ctx context.Context, input common.UpdateCommentsRecordActivityInput) (common.UpdateCommentsRecordActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating comments record", "videoId", input.VideoID, "commentCount", len(input.Comments))

	activity.RecordHeartbeat(ctx, "Updating comments database record")

	// Get Next.js API endpoint from environment
	nextjsAPIEndpoint := os.Getenv("NEXTJS_API_ENDPOINT")
	if nextjsAPIEndpoint == "" {
		logger.Warn("NEXTJS_API_ENDPOINT not set, skipping database update")
		return common.UpdateCommentsRecordActivityResult{
			Success: true, // Don't fail the workflow
		}, nil
	}

	apiKey := os.Getenv("NEXTJS_API_KEY")
	if apiKey == "" {
		logger.Warn("NEXTJS_API_KEY not set, skipping database update")
		return common.UpdateCommentsRecordActivityResult{
			Success: true,
		}, nil
	}

	// Prepare update payload
	updateReq := map[string]interface{}{
		"videoId":  input.VideoID,
		"comments": input.Comments,
	}

	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return common.UpdateCommentsRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to marshal request: %v", err),
		}, err
	}

	// Call Next.js API
	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/api/douyin/internal/comments/%s", nextjsAPIEndpoint, input.VideoID),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return common.UpdateCommentsRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create request: %v", err),
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return common.UpdateCommentsRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to update record: %v", err),
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return common.UpdateCommentsRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Update failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
		}, fmt.Errorf("update failed with status %d", resp.StatusCode)
	}

	logger.Info("Comments record updated successfully", "videoId", input.VideoID, "commentCount", len(input.Comments))

	return common.UpdateCommentsRecordActivityResult{
		Success: true,
	}, nil
}

// ========================================
// Report Generation Activities
// ========================================

// GenerateAIReport generates an AI analysis report for a video
func (a *DouyinActivities) GenerateAIReport(ctx context.Context, input common.GenerateAIReportActivityInput) (common.GenerateAIReportActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating AI report", "videoId", input.VideoID, "promptId", input.PromptID)

	activity.RecordHeartbeat(ctx, "Fetching video data")

	// Get Next.js API endpoint from environment
	nextjsAPIEndpoint := os.Getenv("NEXTJS_API_ENDPOINT")
	if nextjsAPIEndpoint == "" {
		return common.GenerateAIReportActivityResult{
			Success:   false,
			Error:     "NEXTJS_API_ENDPOINT not set",
			ErrorCode: "CONFIGURATION_ERROR",
		}, fmt.Errorf("NEXTJS_API_ENDPOINT not set")
	}

	apiKey := os.Getenv("NEXTJS_API_KEY")
	if apiKey == "" {
		return common.GenerateAIReportActivityResult{
			Success:   false,
			Error:     "NEXTJS_API_KEY not set",
			ErrorCode: "CONFIGURATION_ERROR",
		}, fmt.Errorf("NEXTJS_API_KEY not set")
	}

	// Step 1: Fetch video data from Next.js API (if not provided)
	if input.Metadata == nil || input.Transcript == "" {
		activity.RecordHeartbeat(ctx, "Fetching video data from database")

		getReq, err := http.NewRequestWithContext(ctx, "GET",
			fmt.Sprintf("%s/api/douyin/videos/%s/data", nextjsAPIEndpoint, input.VideoID),
			nil)
		if err != nil {
			return common.GenerateAIReportActivityResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to create request: %v", err),
				ErrorCode: "UNKNOWN",
			}, err
		}

		getReq.Header.Set("X-API-Key", apiKey)

		resp, err := a.httpClient.Do(getReq)
		if err != nil {
			return common.GenerateAIReportActivityResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to fetch video data: %v", err),
				ErrorCode: "NETWORK_ERROR",
			}, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return common.GenerateAIReportActivityResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to fetch video data with status %d: %s", resp.StatusCode, string(bodyBytes)),
				ErrorCode: "DATA_FETCH_FAILED",
			}, fmt.Errorf("fetch failed with status %d", resp.StatusCode)
		}

		// Parse video data
		var videoData struct {
			Success    bool                      `json:"success"`
			Metadata   *common.DouyinVideoMetadata `json:"metadata"`
			Transcript string                    `json:"transcript"`
			Engagement *common.DouyinEngagement    `json:"engagement"`
			Comments   []common.DouyinComment      `json:"comments"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&videoData); err != nil {
			return common.GenerateAIReportActivityResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to parse video data: %v", err),
				ErrorCode: "UNKNOWN",
			}, err
		}

		// Update input with fetched data
		if input.Metadata == nil {
			input.Metadata = videoData.Metadata
		}
		if input.Transcript == "" {
			input.Transcript = videoData.Transcript
		}
		if input.Engagement == nil {
			input.Engagement = videoData.Engagement
		}
		if len(input.Comments) == 0 {
			input.Comments = videoData.Comments
		}
	}

	// Step 2: Generate AI report by calling Next.js API
	activity.RecordHeartbeat(ctx, "Generating AI report")

	reportReq := map[string]interface{}{
		"videoId":    input.VideoID,
		"metadata":   input.Metadata,
		"transcript": input.Transcript,
		"engagement": input.Engagement,
		"comments":   input.Comments,
		"promptId":   input.PromptID,
	}

	if input.CustomPrompt != "" {
		reportReq["customPrompt"] = input.CustomPrompt
	}

	reqBody, err := json.Marshal(reportReq)
	if err != nil {
		return common.GenerateAIReportActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to marshal request: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/douyin/analysis/generate", nextjsAPIEndpoint),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return common.GenerateAIReportActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to create request: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return common.GenerateAIReportActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to generate report: %v", err),
			ErrorCode: "NETWORK_ERROR",
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return common.GenerateAIReportActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Report generation failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
			ErrorCode: "AI_GENERATION_FAILED",
		}, fmt.Errorf("generation failed with status %d", resp.StatusCode)
	}

	// Parse response
	var reportResp struct {
		Success    bool   `json:"success"`
		ReportText string `json:"reportText"`
		TokensUsed int    `json:"tokensUsed"`
		Error      string `json:"error"`
		ErrorCode  string `json:"errorCode"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&reportResp); err != nil {
		return common.GenerateAIReportActivityResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to parse response: %v", err),
			ErrorCode: "UNKNOWN",
		}, err
	}

	if !reportResp.Success {
		return common.GenerateAIReportActivityResult{
			Success:   false,
			Error:     reportResp.Error,
			ErrorCode: reportResp.ErrorCode,
		}, fmt.Errorf("report generation failed: %s", reportResp.Error)
	}

	logger.Info("AI report generated successfully",
		"videoId", input.VideoID,
		"tokensUsed", reportResp.TokensUsed,
		"reportLength", len(reportResp.ReportText))

	return common.GenerateAIReportActivityResult{
		Success:    true,
		ReportText: reportResp.ReportText,
		TokensUsed: reportResp.TokensUsed,
	}, nil
}

// UpdateReportRecord updates report/session record in Next.js database via HTTP callback
func (a *DouyinActivities) UpdateReportRecord(ctx context.Context, input common.UpdateReportRecordActivityInput) (common.UpdateReportRecordActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating report record", "sessionId", input.SessionID, "status", input.Status)

	activity.RecordHeartbeat(ctx, "Updating report database record")

	// Get Next.js API endpoint from environment
	nextjsAPIEndpoint := os.Getenv("NEXTJS_API_ENDPOINT")
	if nextjsAPIEndpoint == "" {
		logger.Warn("NEXTJS_API_ENDPOINT not set, skipping database update")
		return common.UpdateReportRecordActivityResult{
			Success: true, // Don't fail the workflow
		}, nil
	}

	apiKey := os.Getenv("NEXTJS_API_KEY")
	if apiKey == "" {
		logger.Warn("NEXTJS_API_KEY not set, skipping database update")
		return common.UpdateReportRecordActivityResult{
			Success: true,
		}, nil
	}

	// Prepare update payload
	updateReq := map[string]interface{}{
		"sessionId": input.SessionID,
		"status":    input.Status,
	}

	if input.ReportText != "" {
		updateReq["reportText"] = input.ReportText
	}

	if input.Error != "" {
		updateReq["error"] = input.Error
	}

	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return common.UpdateReportRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to marshal request: %v", err),
		}, err
	}

	// Call Next.js API
	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/api/douyin/internal/reports/%s", nextjsAPIEndpoint, input.SessionID),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return common.UpdateReportRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create request: %v", err),
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return common.UpdateReportRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to update record: %v", err),
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return common.UpdateReportRecordActivityResult{
			Success: false,
			Error:   fmt.Sprintf("Update failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
		}, fmt.Errorf("update failed with status %d", resp.StatusCode)
	}

	logger.Info("Report record updated successfully", "sessionId", input.SessionID, "status", input.Status)

	return common.UpdateReportRecordActivityResult{
		Success: true,
	}, nil
}
