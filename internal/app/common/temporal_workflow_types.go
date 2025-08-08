package common

import "time"

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

// BatchFile represents a single file in a batch
type BatchFile struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	Provider string `json:"provider,omitempty"` // Optional provider override for this file
}

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

// BatchWorkflowResult represents the output of batch transcription workflow
type BatchWorkflowResult struct {
	BatchID        string                      `json:"batch_id"`
	TotalFiles     int                         `json:"total_files"`
	SuccessCount   int                         `json:"success_count"`
	FailureCount   int                         `json:"failure_count"`
	Results        []SingleFileWorkflowResult  `json:"results"`
	ProcessingTime time.Duration               `json:"processing_time"`
}

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

// FallbackWorkflowResult represents the output of fallback transcription workflow
type FallbackWorkflowResult struct {
	FileID           string        `json:"file_id"`
	TranscriptionURL string        `json:"transcription_url"`
	Provider         string        `json:"provider"`        // The provider that succeeded
	AttemptedProviders []string    `json:"attempted_providers"`
	ProcessingTime   time.Duration `json:"processing_time"`
	Error            string        `json:"error,omitempty"`
}

// ETLWorkflowRequest represents the input for ETL workflow
type ETLWorkflowRequest struct {
	Source      string                 `json:"source"`       // Source URL or path
	Type        string                 `json:"type"`         // Source type (youtube, xiaoyuzhou, etc.)
	Language    string                 `json:"language"`
	Provider    string                 `json:"provider"`
	MaxParallel int                    `json:"max_parallel"`
	Options     map[string]interface{} `json:"options"`
}

// ETLWorkflowResult represents the output of ETL workflow
type ETLWorkflowResult struct {
	TotalFiles     int                         `json:"total_files"`
	SuccessCount   int                         `json:"success_count"`
	FailureCount   int                         `json:"failure_count"`
	Results        []SingleFileWorkflowResult  `json:"results"`
	ProcessingTime time.Duration               `json:"processing_time"`
}