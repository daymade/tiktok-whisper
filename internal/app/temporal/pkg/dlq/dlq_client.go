package dlq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.temporal.io/sdk/activity"
)

// DLQClient handles communication with Next.js DLQ API
type DLQClient struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

// DLQEntry represents a failed workflow entry
type DLQEntry struct {
	WorkflowID  string                 `json:"workflowId"`
	WorkflowType string                `json:"workflowType"`
	Input       interface{}            `json:"input"`
	Error       string                 `json:"error"`
	ErrorStack  string                 `json:"errorStack,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DLQResponse represents the API response
type DLQResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Entry   *struct {
		ID           string     `json:"id"`
		WorkflowID   string     `json:"workflowId"`
		Status       string     `json:"status"`
		AttemptCount int        `json:"attemptCount"`
		NextRetryAt  *time.Time `json:"nextRetryAt"`
	} `json:"entry,omitempty"`
}

// NewDLQClient creates a new DLQ client
func NewDLQClient() *DLQClient {
	endpoint := os.Getenv("NEXTJS_API_ENDPOINT")
	if endpoint == "" {
		// Fallback to default
		endpoint = "http://pod0-dev-app:3000"
	}

	apiKey := os.Getenv("NEXTJS_INTERNAL_API_KEY")
	if apiKey == "" {
		// Fallback to WHISPER_API_KEY for backward compatibility
		apiKey = os.Getenv("WHISPER_API_KEY")
	}

	return &DLQClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AddFailedWorkflow adds a failed workflow to the DLQ
func (c *DLQClient) AddFailedWorkflow(ctx context.Context, entry DLQEntry) error {
	// Skip if no API endpoint configured (graceful degradation)
	if c.endpoint == "" || c.apiKey == "" {
		if activity.GetInfo(ctx).Attempt == 0 {
			// Only log on first attempt to avoid spam
			logger := activity.GetLogger(ctx)
			logger.Warn("DLQ not configured, skipping failed workflow logging",
				"workflowId", entry.WorkflowID,
				"error", entry.Error)
		}
		return nil
	}

	// Prepare request
	reqBody, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ entry: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/internal/dlq", c.endpoint),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create DLQ request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DLQ request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var dlqResp DLQResponse
	if err := json.NewDecoder(resp.Body).Decode(&dlqResp); err != nil {
		return fmt.Errorf("failed to decode DLQ response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DLQ API returned error: %s (status: %d)", dlqResp.Error, resp.StatusCode)
	}

	if !dlqResp.Success {
		return fmt.Errorf("DLQ API reported failure: %s", dlqResp.Error)
	}

	// Log success
	logger := activity.GetLogger(ctx)
	logger.Info("Successfully added failed workflow to DLQ",
		"workflowId", entry.WorkflowID,
		"dlqEntryId", dlqResp.Entry.ID,
		"status", dlqResp.Entry.Status,
		"attemptCount", dlqResp.Entry.AttemptCount)

	return nil
}

// IsConfigured checks if DLQ client is properly configured
func (c *DLQClient) IsConfigured() bool {
	return c.endpoint != "" && c.apiKey != ""
}
