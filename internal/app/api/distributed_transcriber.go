package api

import (
	"context"
	"fmt"
	"time"
	
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"tiktok-whisper/internal/app/common"
)

// DistributedTranscriber provides a simple interface for distributed transcription
type DistributedTranscriber struct {
	temporalClient client.Client
	taskQueue      string
}

// TranscriptionJob represents a transcription job
type TranscriptionJob struct {
	ID          string    `json:"id"`
	FilePath    string    `json:"file_path"`
	Status      string    `json:"status"`
	Result      string    `json:"result,omitempty"`
	Error       string    `json:"error,omitempty"`
	SubmittedAt time.Time `json:"submitted_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	WorkflowID  string    `json:"workflow_id"`
}

// NewDistributedTranscriber creates a new distributed transcriber
func NewDistributedTranscriber(temporalHost string) (*DistributedTranscriber, error) {
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}
	
	return &DistributedTranscriber{
		temporalClient: c,
		taskQueue:      "v2t-transcription-queue",
	}, nil
}

// SubmitJob submits a single file for transcription
func (dt *DistributedTranscriber) SubmitJob(ctx context.Context, filePath string) (*TranscriptionJob, error) {
	jobID := uuid.New().String()
	workflowID := fmt.Sprintf("transcribe-%s-%d", jobID, time.Now().Unix())
	
	// Create workflow request
	request := common.SingleFileWorkflowRequest{
		FileID:       jobID,
		FilePath:     filePath,
		Language:     "auto",
		OutputFormat: "text",
		UseMinIO:     false,
	}
	
	// Start workflow
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: dt.taskQueue,
	}
	
	we, err := dt.temporalClient.ExecuteWorkflow(ctx, options, "SingleFileTranscriptionWorkflow", request)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}
	
	job := &TranscriptionJob{
		ID:          jobID,
		FilePath:    filePath,
		Status:      "submitted",
		SubmittedAt: time.Now(),
		WorkflowID:  we.GetID(),
	}
	
	return job, nil
}

// GetJobStatus retrieves the status of a job
func (dt *DistributedTranscriber) GetJobStatus(ctx context.Context, workflowID string) (*TranscriptionJob, error) {
	// Check if workflow is still running
	desc, err := dt.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to describe workflow: %w", err)
	}
	
	job := &TranscriptionJob{
		WorkflowID: workflowID,
	}
	
	// Determine status based on workflow state
	if desc.WorkflowExecutionInfo.Status.String() == "Running" {
		job.Status = "processing"
	} else {
		job.Status = "completed"
		
		// Get workflow result
		we := dt.temporalClient.GetWorkflow(ctx, workflowID, "")
		var result common.SingleFileWorkflowResult
		if err := we.Get(ctx, &result); err == nil {
			job.Result = result.TranscriptionURL
			if result.Error != "" {
				job.Status = "failed"
				job.Error = result.Error
			}
		}
	}
	
	return job, nil
}

// WaitForResult waits for a workflow to complete and returns the result
func (dt *DistributedTranscriber) WaitForResult(ctx context.Context, workflowID string) (*TranscriptionJob, error) {
	// Get workflow handle
	we := dt.temporalClient.GetWorkflow(ctx, workflowID, "")
	
	// Wait for result
	var result common.SingleFileWorkflowResult
	err := we.Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("workflow failed: %w", err)
	}
	
	job := &TranscriptionJob{
		WorkflowID:  workflowID,
		Status:      "completed",
		Result:      result.TranscriptionURL,
		CompletedAt: time.Now(),
	}
	
	if result.Error != "" {
		job.Status = "failed"
		job.Error = result.Error
	}
	
	return job, nil
}

// SubmitBatch submits multiple files for transcription
func (dt *DistributedTranscriber) SubmitBatch(ctx context.Context, filePaths []string, maxParallel int) (*TranscriptionJob, error) {
	batchID := uuid.New().String()
	workflowID := fmt.Sprintf("batch-%s-%d", batchID, time.Now().Unix())
	
	// Build batch files
	files := make([]common.BatchFile, 0, len(filePaths))
	for _, path := range filePaths {
		files = append(files, common.BatchFile{
			FileID:   uuid.New().String(),
			FilePath: path,
		})
	}
	
	// Create batch request
	request := common.BatchWorkflowRequest{
		BatchID:     batchID,
		Files:       files,
		Language:    "auto",
		MaxParallel: maxParallel,
		UseMinIO:    false,
	}
	
	// Start batch workflow
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: dt.taskQueue,
	}
	
	we, err := dt.temporalClient.ExecuteWorkflow(ctx, options, "BatchTranscriptionWorkflow", request)
	if err != nil {
		return nil, fmt.Errorf("failed to start batch workflow: %w", err)
	}
	
	return &TranscriptionJob{
		ID:          batchID,
		Status:      "submitted",
		SubmittedAt: time.Now(),
		WorkflowID:  we.GetID(),
	}, nil
}

// Close closes the distributed transcriber
func (dt *DistributedTranscriber) Close() {
	if dt.temporalClient != nil {
		dt.temporalClient.Close()
	}
}