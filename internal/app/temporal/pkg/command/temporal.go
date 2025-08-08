package command

import (
	"context"
	"fmt"
	"time"
	
	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/temporal/pkg/common"
)

// TemporalClientFactory creates a distributed transcriber with common configuration
func NewDistributedTranscriber() (*api.DistributedTranscriber, error) {
	temporalHost := common.GetEnv("TEMPORAL_HOST", common.DefaultTemporalHost)
	return api.NewDistributedTranscriber(temporalHost)
}

// WaitForJobWithProgress waits for a job to complete with progress updates
func WaitForJobWithProgress(ctx context.Context, dt *api.DistributedTranscriber, workflowID string, progressFunc func(status string)) (*api.TranscriptionJob, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(30 * time.Minute)
	
	for {
		select {
		case <-ticker.C:
			status, err := dt.GetJobStatus(ctx, workflowID)
			if err != nil {
				return nil, fmt.Errorf("failed to get job status: %w", err)
			}
			
			if progressFunc != nil {
				progressFunc(status.Status)
			}
			
			if status.Status == "completed" {
				return status, nil
			} else if status.Status == "failed" {
				return status, fmt.Errorf("job failed: %s", status.Error)
			}
			
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for job completion")
			
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}