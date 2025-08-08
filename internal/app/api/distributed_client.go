package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"tiktok-whisper/internal/app/common"
)

// DistributedClient provides a simple interface for distributed transcription
type DistributedClient struct {
	temporalClient client.Client
	taskQueue      string
}

// NewDistributedClient creates a new distributed client
func NewDistributedClient(temporalHost string) (*DistributedClient, error) {
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	return &DistributedClient{
		temporalClient: c,
		taskQueue:      "v2t-transcription-queue",
	}, nil
}

// StartTranscription starts a single file transcription workflow
func (d *DistributedClient) StartTranscription(ctx context.Context, filePath string) (string, error) {
	workflowID := fmt.Sprintf("transcribe-%s-%d", uuid.New().String(), time.Now().Unix())
	
	request := common.SingleFileWorkflowRequest{
		FileID:       uuid.New().String(),
		FilePath:     filePath,
		Language:     "auto",
		OutputFormat: "text",
		UseMinIO:     true,
	}

	we, err := d.temporalClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: d.taskQueue,
	}, "SingleFileTranscriptionWorkflow", request)
	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	return we.GetID(), nil
}

// GetWorkflowResult gets the result of a workflow
func (d *DistributedClient) GetWorkflowResult(ctx context.Context, workflowID string) (*common.SingleFileWorkflowResult, error) {
	workflow := d.temporalClient.GetWorkflow(ctx, workflowID, "")
	
	var result common.SingleFileWorkflowResult
	err := workflow.Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetWorkflowStatus gets the current status of a workflow
func (d *DistributedClient) GetWorkflowStatus(ctx context.Context, workflowID string) (string, error) {
	resp, err := d.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return "", err
	}

	status := resp.WorkflowExecutionInfo.Status.String()
	return status, nil
}

// Close closes the temporal client connection
func (d *DistributedClient) Close() {
	d.temporalClient.Close()
}