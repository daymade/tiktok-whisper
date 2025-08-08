package common

import "time"

// TranscriptionRequest represents a request to transcribe a file
type TranscriptionRequest struct {
	FileID       string                 `json:"file_id"`
	FilePath     string                 `json:"file_path"`
	Provider     string                 `json:"provider"`
	Language     string                 `json:"language"`
	OutputFormat string                 `json:"output_format"`
	Options      map[string]interface{} `json:"options"`
}

// TranscriptionResult represents the result of transcription
type TranscriptionResult struct {
	FileID         string        `json:"file_id"`
	Transcription  string        `json:"transcription"`
	Provider       string        `json:"provider"`
	ProcessingTime time.Duration `json:"processing_time"`
	Error          string        `json:"error,omitempty"`
}

// FileUploadRequest represents a request to upload a file
type FileUploadRequest struct {
	LocalPath  string                 `json:"local_path"`
	RemotePath string                 `json:"remote_path"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// FileUploadResult represents the result of file upload
type FileUploadResult struct {
	RemoteURL string `json:"remote_url"`
	Size      int64  `json:"size"`
	Error     string `json:"error,omitempty"`
}

// FileDownloadRequest represents a request to download a file
type FileDownloadRequest struct {
	RemoteURL  string `json:"remote_url"`
	LocalPath  string `json:"local_path"`
}

// FileDownloadResult represents the result of file download
type FileDownloadResult struct {
	LocalPath string `json:"local_path"`
	Size      int64  `json:"size"`
	Error     string `json:"error,omitempty"`
}

// ProviderStatusRequest represents a request to check provider status
type ProviderStatusRequest struct {
	Provider string `json:"provider"`
}

// ProviderStatusResult represents provider status
type ProviderStatusResult struct {
	Provider    string `json:"provider"`
	Available   bool   `json:"available"`
	Healthy     bool   `json:"healthy"`
	Message     string `json:"message"`
	LastChecked time.Time `json:"last_checked"`
}

// DownloadVideoRequest represents a request to download video
type DownloadVideoRequest struct {
	URL      string                 `json:"url"`
	Type     string                 `json:"type"` // youtube, xiaoyuzhou, etc.
	OutputDir string                `json:"output_dir"`
	Options  map[string]interface{} `json:"options"`
}

// DownloadVideoResult represents the result of video download
type DownloadVideoResult struct {
	Files      []string `json:"files"`
	TotalSize  int64    `json:"total_size"`
	Error      string   `json:"error,omitempty"`
}