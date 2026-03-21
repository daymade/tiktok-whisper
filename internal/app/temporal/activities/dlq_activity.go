package activities

import (
	"context"

	"go.temporal.io/sdk/activity"
	"tiktok-whisper/internal/app/temporal/pkg/dlq"
)

// DLQActivities handles Dead Letter Queue operations
type DLQActivities struct {
	dlqClient *dlq.DLQClient
}

// NewDLQActivities creates a new DLQ activities instance
func NewDLQActivities() *DLQActivities {
	return &DLQActivities{
		dlqClient: dlq.NewDLQClient(),
	}
}

// ReportFailedWorkflowInput represents input for reporting a failed workflow
type ReportFailedWorkflowInput struct {
	WorkflowID   string                 `json:"workflowId"`
	WorkflowType string                 `json:"workflowType"`
	Input        interface{}            `json:"input"`
	Error        string                 `json:"error"`
	ErrorStack   string                 `json:"errorStack,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ReportFailedWorkflowResult represents result of reporting a failed workflow
type ReportFailedWorkflowResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ReportFailedWorkflow reports a failed workflow to the DLQ
func (a *DLQActivities) ReportFailedWorkflow(ctx context.Context, input ReportFailedWorkflowInput) (ReportFailedWorkflowResult, error) {
	logger := activity.GetLogger(ctx)

	// Skip if DLQ not configured (graceful degradation)
	if !a.dlqClient.IsConfigured() {
		logger.Warn("DLQ not configured, skipping failed workflow report",
			"workflowId", input.WorkflowID)
		return ReportFailedWorkflowResult{Success: true}, nil
	}

	logger.Info("Reporting failed workflow to DLQ",
		"workflowId", input.WorkflowID,
		"workflowType", input.WorkflowType,
		"error", input.Error)

	// Prepare DLQ entry
	entry := dlq.DLQEntry{
		WorkflowID:   input.WorkflowID,
		WorkflowType: input.WorkflowType,
		Input:        input.Input,
		Error:        input.Error,
		ErrorStack:   input.ErrorStack,
		Metadata:     input.Metadata,
	}

	// Send to DLQ
	err := a.dlqClient.AddFailedWorkflow(ctx, entry)
	if err != nil {
		logger.Error("Failed to report workflow to DLQ", "error", err)
		// Don't fail the activity - DLQ reporting is best-effort
		return ReportFailedWorkflowResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	logger.Info("Successfully reported workflow to DLQ", "workflowId", input.WorkflowID)
	return ReportFailedWorkflowResult{Success: true}, nil
}
