package common

import "time"

// DouyinVideoMetadata represents video metadata from Douyin
type DouyinVideoMetadata struct {
	VideoID        string    `json:"video_id"`
	Title          string    `json:"title"`
	AuthorName     string    `json:"author_name"`
	AuthorID       string    `json:"author_id"`
	CoverURL       string    `json:"cover_url"`
	PublishTime    time.Time `json:"publish_time"`
	Duration       int       `json:"duration"` // seconds
	LikesCount     int       `json:"likes_count"`
	CommentsCount  int       `json:"comments_count"`
	SharesCount    int       `json:"shares_count"`
	FavoritesCount int       `json:"favorites_count"`
}

// ImportVideoWorkflowInput represents input for import video workflow
type ImportVideoWorkflowInput struct {
	UserID             string `json:"userId"`
	VideoURL           string `json:"videoUrl"`
	JobID              string `json:"jobId"`
	SkipTranscription  bool   `json:"skipTranscription,omitempty"`
	Language           string `json:"language,omitempty"`
}

// ImportVideoWorkflowResult represents result of import video workflow
type ImportVideoWorkflowResult struct {
	Success             bool                     `json:"success"`
	VideoID             string                   `json:"videoId"`
	Metadata            *DouyinVideoMetadata     `json:"metadata,omitempty"`
	TranscriptText      string                   `json:"transcriptText,omitempty"`
	TranscriptSegments  []TranscriptSegment      `json:"transcriptSegments,omitempty"`
	Error               string                   `json:"error,omitempty"`
	ErrorCode           string                   `json:"errorCode,omitempty"`
	DurationMs          int64                    `json:"durationMs,omitempty"`
}

// TranscriptSegment represents a segment of transcript with timing
type TranscriptSegment struct {
	Start float64 `json:"start"` // seconds
	End   float64 `json:"end"`   // seconds
	Text  string  `json:"text"`
}

// BatchImportVideosWorkflowInput represents input for batch import workflow
type BatchImportVideosWorkflowInput struct {
	UserID    string                      `json:"userId"`
	VideoURLs []string                    `json:"videoUrls"`
	BatchID   string                      `json:"batchId"`
	Options   BatchImportOptions          `json:"options,omitempty"`
}

// BatchImportOptions represents options for batch import
type BatchImportOptions struct {
	Concurrency       int    `json:"concurrency,omitempty"` // Default 5
	SkipTranscription bool   `json:"skipTranscription,omitempty"`
	Language          string `json:"language,omitempty"`
}

// BatchImportVideosWorkflowResult represents result of batch import workflow
type BatchImportVideosWorkflowResult struct {
	BatchID             string              `json:"batchId"`
	TotalCount          int                 `json:"totalCount"`
	SuccessCount        int                 `json:"successCount"`
	FailedCount         int                 `json:"failedCount"`
	SuccessfulVideoIDs  []string            `json:"successfulVideoIds"`
	FailedVideos        []FailedVideoInfo   `json:"failedVideos"`
	DurationMs          int64               `json:"durationMs"`
}

// FailedVideoInfo represents information about a failed video import
type FailedVideoInfo struct {
	VideoURL string `json:"videoUrl"`
	Error    string `json:"error"`
}

// DownloadVideoActivityInput represents input for download video activity
type DownloadVideoActivityInput struct {
	VideoURL   string `json:"videoUrl"`
	TargetPath string `json:"targetPath"`
	AudioOnly  bool   `json:"audioOnly,omitempty"`
}

// DownloadVideoActivityResult represents result of download video activity
type DownloadVideoActivityResult struct {
	Success       bool                  `json:"success"`
	VideoFilePath string                `json:"videoFilePath,omitempty"`
	AudioFilePath string                `json:"audioFilePath,omitempty"`
	FileSize      int64                 `json:"fileSize,omitempty"`
	Duration      int                   `json:"duration,omitempty"` // seconds
	Metadata      *DouyinVideoMetadata  `json:"metadata,omitempty"`
	Error         string                `json:"error,omitempty"`
	ErrorCode     string                `json:"errorCode,omitempty"` // VIDEO_UNAVAILABLE, NETWORK_ERROR, STORAGE_FULL, UNKNOWN
}

// ExtractAudioActivityInput represents input for extract audio activity
type ExtractAudioActivityInput struct {
	VideoFilePath   string `json:"videoFilePath"`
	OutputAudioPath string `json:"outputAudioPath"`
	Format          string `json:"format,omitempty"` // Default: wav
}

// ExtractAudioActivityResult represents result of extract audio activity
type ExtractAudioActivityResult struct {
	Success       bool   `json:"success"`
	AudioFilePath string `json:"audioFilePath,omitempty"`
	FileSize      int64  `json:"fileSize,omitempty"`
	Duration      int    `json:"duration,omitempty"` // seconds
	Error         string `json:"error,omitempty"`
}

// TranscribeAudioActivityInput represents input for transcribe audio activity
type TranscribeAudioActivityInput struct {
	AudioFilePath string `json:"audioFilePath"`
	Language      string `json:"language,omitempty"`
	Model         string `json:"model,omitempty"`
}

// TranscribeAudioActivityResult represents result of transcribe audio activity
type TranscribeAudioActivityResult struct {
	Success    bool                 `json:"success"`
	Text       string               `json:"text,omitempty"`
	Segments   []TranscriptSegment  `json:"segments,omitempty"`
	TokensUsed int                  `json:"tokensUsed,omitempty"`
	Error      string               `json:"error,omitempty"`
}

// UpdateVideoRecordActivityInput represents input for update video record activity
type UpdateVideoRecordActivityInput struct {
	JobID              string               `json:"jobId"`
	Status             string               `json:"status"` // pending, downloading, transcribing, completed, failed
	Metadata           *DouyinVideoMetadata `json:"metadata,omitempty"`
	TranscriptText     string               `json:"transcriptText,omitempty"`
	TranscriptSegments []TranscriptSegment  `json:"transcriptSegments,omitempty"`
	ErrorMessage       string               `json:"errorMessage,omitempty"`
}

// UpdateVideoRecordActivityResult represents result of update video record activity
type UpdateVideoRecordActivityResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ========================================
// Engagement Scraping Workflow Types
// ========================================

// ScrapeEngagementWorkflowInput represents input for scrape engagement workflow
type ScrapeEngagementWorkflowInput struct {
	UserID  string `json:"userId"`
	VideoID string `json:"videoId"`
	JobID   string `json:"jobId"`
}

// ScrapeEngagementWorkflowResult represents result of scrape engagement workflow
type ScrapeEngagementWorkflowResult struct {
	Success    bool               `json:"success"`
	VideoID    string             `json:"videoId"`
	Engagement *DouyinEngagement  `json:"engagement,omitempty"`
	Error      string             `json:"error,omitempty"`
	ErrorCode  string             `json:"errorCode,omitempty"`
	DurationMs int64              `json:"durationMs,omitempty"`
}

// DouyinEngagement represents engagement data for a video
type DouyinEngagement struct {
	VideoID        string    `json:"videoId"`
	LikesCount     int       `json:"likesCount"`
	CommentsCount  int       `json:"commentsCount"`
	SharesCount    int       `json:"sharesCount"`
	FavoritesCount int       `json:"favoritesCount"`
	ViewsCount     int       `json:"viewsCount,omitempty"`
	ScrapedAt      time.Time `json:"scrapedAt"`
}

// ========================================
// Comment Scraping Workflow Types
// ========================================

// ScrapeCommentsWorkflowInput represents input for scrape comments workflow
type ScrapeCommentsWorkflowInput struct {
	UserID         string `json:"userId"`
	VideoID        string `json:"videoId"`
	JobID          string `json:"jobId"`
	Limit          int    `json:"limit,omitempty"`          // Max comments to scrape
	IncludeReplies bool   `json:"includeReplies,omitempty"` // Include comment replies
}

// ScrapeCommentsWorkflowResult represents result of scrape comments workflow
type ScrapeCommentsWorkflowResult struct {
	Success      bool              `json:"success"`
	VideoID      string            `json:"videoId"`
	CommentCount int               `json:"commentCount"`
	Comments     []DouyinComment   `json:"comments,omitempty"`
	Error        string            `json:"error,omitempty"`
	ErrorCode    string            `json:"errorCode,omitempty"`
	DurationMs   int64             `json:"durationMs,omitempty"`
}

// DouyinComment represents a comment on a video
type DouyinComment struct {
	CommentID      string          `json:"commentId"`
	Content        string          `json:"content"`
	AuthorName     string          `json:"authorName"`
	AuthorID       string          `json:"authorId"`
	LikesCount     int             `json:"likesCount"`
	CreatedAt      time.Time       `json:"createdAt"`
	ParentCommentID string         `json:"parentCommentId,omitempty"` // For replies
	Replies        []DouyinComment `json:"replies,omitempty"`
}

// ========================================
// Report Generation Workflow Types
// ========================================

// GenerateReportWorkflowInput represents input for generate report workflow
type GenerateReportWorkflowInput struct {
	UserID    string `json:"userId"`
	VideoID   string `json:"videoId"`
	JobID     string `json:"jobId"`
	PromptID  string `json:"promptId,omitempty"` // Custom prompt ID
	SessionID string `json:"sessionId,omitempty"` // Existing analysis session
}

// GenerateReportWorkflowResult represents result of generate report workflow
type GenerateReportWorkflowResult struct {
	Success    bool   `json:"success"`
	VideoID    string `json:"videoId"`
	SessionID  string `json:"sessionId,omitempty"`
	ReportText string `json:"reportText,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorCode  string `json:"errorCode,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
}

// BatchGenerateReportsWorkflowInput represents input for batch report generation
type BatchGenerateReportsWorkflowInput struct {
	UserID    string                      `json:"userId"`
	VideoIDs  []string                    `json:"videoIds"`
	BatchID   string                      `json:"batchId"`
	PromptID  string                      `json:"promptId,omitempty"`
	Options   BatchGenerateReportsOptions `json:"options,omitempty"`
}

// BatchGenerateReportsOptions represents options for batch report generation
type BatchGenerateReportsOptions struct {
	Concurrency int `json:"concurrency,omitempty"` // Default 3 (AI calls are expensive)
}

// BatchGenerateReportsWorkflowResult represents result of batch report generation
type BatchGenerateReportsWorkflowResult struct {
	BatchID          string               `json:"batchId"`
	TotalCount       int                  `json:"totalCount"`
	SuccessCount     int                  `json:"successCount"`
	FailedCount      int                  `json:"failedCount"`
	SuccessfulReports []GeneratedReportInfo `json:"successfulReports"`
	FailedReports    []FailedReportInfo    `json:"failedReports"`
	DurationMs       int64                 `json:"durationMs"`
}

// GeneratedReportInfo represents information about a successfully generated report
type GeneratedReportInfo struct {
	VideoID   string `json:"videoId"`
	SessionID string `json:"sessionId"`
}

// FailedReportInfo represents information about a failed report generation
type FailedReportInfo struct {
	VideoID string `json:"videoId"`
	Error   string `json:"error"`
}

// ========================================
// Douyin Service Activity Types
// ========================================

// ScrapeEngagementActivityInput represents input for scrape engagement activity
type ScrapeEngagementActivityInput struct {
	VideoID string `json:"videoId"`
}

// ScrapeEngagementActivityResult represents result of scrape engagement activity
type ScrapeEngagementActivityResult struct {
	Success    bool              `json:"success"`
	Engagement *DouyinEngagement `json:"engagement,omitempty"`
	Error      string            `json:"error,omitempty"`
	ErrorCode  string            `json:"errorCode,omitempty"`
}

// ScrapeCommentsActivityInput represents input for scrape comments activity
type ScrapeCommentsActivityInput struct {
	VideoID        string `json:"videoId"`
	Limit          int    `json:"limit,omitempty"`
	IncludeReplies bool   `json:"includeReplies,omitempty"`
}

// ScrapeCommentsActivityResult represents result of scrape comments activity
type ScrapeCommentsActivityResult struct {
	Success  bool            `json:"success"`
	Comments []DouyinComment `json:"comments,omitempty"`
	Error    string          `json:"error,omitempty"`
	ErrorCode string         `json:"errorCode,omitempty"`
}

// GenerateAIReportActivityInput represents input for generate AI report activity
type GenerateAIReportActivityInput struct {
	VideoID      string                `json:"videoId"`
	Metadata     *DouyinVideoMetadata  `json:"metadata,omitempty"`
	Transcript   string                `json:"transcript,omitempty"`
	Engagement   *DouyinEngagement     `json:"engagement,omitempty"`
	Comments     []DouyinComment       `json:"comments,omitempty"`
	PromptID     string                `json:"promptId,omitempty"`
	CustomPrompt string                `json:"customPrompt,omitempty"`
}

// GenerateAIReportActivityResult represents result of generate AI report activity
type GenerateAIReportActivityResult struct {
	Success    bool   `json:"success"`
	ReportText string `json:"reportText,omitempty"`
	TokensUsed int    `json:"tokensUsed,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorCode  string `json:"errorCode,omitempty"`
}

// UpdateEngagementRecordActivityInput represents input for update engagement record activity
type UpdateEngagementRecordActivityInput struct {
	VideoID    string            `json:"videoId"`
	Engagement *DouyinEngagement `json:"engagement"`
}

// UpdateEngagementRecordActivityResult represents result of update engagement record activity
type UpdateEngagementRecordActivityResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// UpdateCommentsRecordActivityInput represents input for update comments record activity
type UpdateCommentsRecordActivityInput struct {
	VideoID  string          `json:"videoId"`
	Comments []DouyinComment `json:"comments"`
}

// UpdateCommentsRecordActivityResult represents result of update comments record activity
type UpdateCommentsRecordActivityResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// UpdateReportRecordActivityInput represents input for update report record activity
type UpdateReportRecordActivityInput struct {
	SessionID  string `json:"sessionId"`
	ReportText string `json:"reportText"`
	Status     string `json:"status"` // completed, failed
	Error      string `json:"error,omitempty"`
}

// UpdateReportRecordActivityResult represents result of update report record activity
type UpdateReportRecordActivityResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
