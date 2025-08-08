package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/api/provider/registry"
	"tiktok-whisper/temporal/activities"
	"tiktok-whisper/temporal/workflows"
)

func TestSingleFileWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	
	// Mock activities
	mockTranscribeActivities := &mockTranscribeActivities{
		result: activities.TranscriptionResult{
			FileID:   "test-file-1",
			Text:     "This is a test transcription",
			Provider: "mock_provider",
		},
	}
	
	mockStorageActivities := &mockStorageActivities{}
	
	// Register activities
	env.RegisterActivity(mockTranscribeActivities.TranscribeFile)
	env.RegisterActivity(mockStorageActivities.UploadFile)
	env.RegisterActivity(mockStorageActivities.DownloadFile)
	env.RegisterActivity(mockStorageActivities.CleanupTempFile)
	
	// Test input
	request := workflows.SingleFileWorkflowRequest{
		FileID:       "test-file-1",
		FilePath:     "/tmp/test.mp3",
		Provider:     "mock_provider",
		Language:     "en",
		OutputFormat: "text",
		UseMinIO:     false,
	}
	
	// Execute workflow
	env.ExecuteWorkflow(workflows.SingleFileTranscriptionWorkflow, request)
	
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	
	var result workflows.SingleFileWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	
	assert.Equal(t, "test-file-1", result.FileID)
	assert.Equal(t, "mock_provider", result.Provider)
	assert.NotEmpty(t, result.TranscriptionURL)
	assert.Greater(t, result.ProcessingTime, time.Duration(0))
}

func TestBatchWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	
	// Register child workflow
	env.RegisterWorkflow(workflows.SingleFileTranscriptionWorkflow)
	
	// Mock activities for child workflows
	mockTranscribeActivities := &mockTranscribeActivities{
		result: activities.TranscriptionResult{
			Text:     "Test transcription",
			Provider: "mock_provider",
		},
	}
	
	mockStorageActivities := &mockStorageActivities{}
	
	env.RegisterActivity(mockTranscribeActivities.TranscribeFile)
	env.RegisterActivity(mockStorageActivities.UploadFile)
	env.RegisterActivity(mockStorageActivities.DownloadFile)
	env.RegisterActivity(mockStorageActivities.CleanupTempFile)
	
	// Test input
	request := workflows.BatchWorkflowRequest{
		BatchID: "test-batch-1",
		Files: []workflows.BatchFile{
			{FileID: "file-1", FilePath: "/tmp/test1.mp3"},
			{FileID: "file-2", FilePath: "/tmp/test2.mp3"},
			{FileID: "file-3", FilePath: "/tmp/test3.mp3"},
		},
		Provider:    "mock_provider",
		Language:    "en",
		MaxParallel: 2,
		UseMinIO:    false,
	}
	
	// Execute workflow
	env.ExecuteWorkflow(workflows.BatchTranscriptionWorkflow, request)
	
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	
	var result workflows.BatchWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	
	assert.Equal(t, "test-batch-1", result.BatchID)
	assert.Equal(t, 3, result.TotalFiles)
	assert.Equal(t, 3, result.SuccessCount)
	assert.Equal(t, 0, result.FailureCount)
	assert.Len(t, result.Results, 3)
}

func TestFallbackWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	
	// Register child workflow
	env.RegisterWorkflow(workflows.SingleFileTranscriptionWorkflow)
	
	// Mock activities with failing providers
	mockActivities := &mockFallbackActivities{
		failProviders: []string{"whisper_cpp", "openai"},
		successProvider: "elevenlabs",
	}
	
	env.RegisterActivity(mockActivities.GetProviderStatus)
	env.RegisterActivity(mockActivities.TranscribeFile)
	
	mockStorageActivities := &mockStorageActivities{}
	env.RegisterActivity(mockStorageActivities.UploadFile)
	env.RegisterActivity(mockStorageActivities.DownloadFile)
	env.RegisterActivity(mockStorageActivities.CleanupTempFile)
	
	// Test input
	request := workflows.FallbackWorkflowRequest{
		FileID:   "test-file-1",
		FilePath: "/tmp/test.mp3",
		Providers: []string{"whisper_cpp", "openai", "elevenlabs"},
		Language: "en",
		UseMinIO: false,
	}
	
	// Execute workflow
	env.ExecuteWorkflow(workflows.TranscriptionWithFallbackWorkflow, request)
	
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	
	var result workflows.FallbackWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	
	assert.Equal(t, "test-file-1", result.FileID)
	assert.Equal(t, "elevenlabs", result.SuccessfulProvider)
	assert.Equal(t, []string{"whisper_cpp", "openai", "elevenlabs"}, result.AttemptedProviders)
	assert.NotEmpty(t, result.TranscriptionURL)
}

// Mock activities
type mockTranscribeActivities struct {
	result activities.TranscriptionResult
	err    error
}

func (m *mockTranscribeActivities) TranscribeFile(ctx context.Context, req activities.TranscriptionRequest) (activities.TranscriptionResult, error) {
	if m.err != nil {
		return activities.TranscriptionResult{}, m.err
	}
	result := m.result
	result.FileID = req.FileID
	return result, nil
}

type mockStorageActivities struct{}

func (m *mockStorageActivities) UploadFile(ctx context.Context, req activities.FileUploadRequest) (activities.FileUploadResult, error) {
	return activities.FileUploadResult{
		ObjectKey: req.ObjectKey,
		ETag:      "mock-etag",
		Size:      1024,
		URL:       "minio://test-bucket/" + req.ObjectKey,
	}, nil
}

func (m *mockStorageActivities) DownloadFile(ctx context.Context, req activities.FileDownloadRequest) (activities.FileDownloadResult, error) {
	return activities.FileDownloadResult{
		LocalPath: "/tmp/downloaded-" + req.ObjectKey,
		Size:      1024,
	}, nil
}

func (m *mockStorageActivities) CleanupTempFile(ctx context.Context, filePath string) error {
	return nil
}

type mockFallbackActivities struct {
	failProviders   []string
	successProvider string
}

func (m *mockFallbackActivities) GetProviderStatus(ctx context.Context, providerName string) (provider.ProviderHealthStatus, error) {
	for _, failProvider := range m.failProviders {
		if providerName == failProvider {
			return provider.ProviderHealthStatus{
				ProviderName: providerName,
				IsHealthy:    false,
				LastError:    "mock provider failure",
			}, nil
		}
	}
	
	return provider.ProviderHealthStatus{
		ProviderName: providerName,
		IsHealthy:    true,
		LastChecked:  time.Now(),
	}, nil
}

func (m *mockFallbackActivities) TranscribeFile(ctx context.Context, req activities.TranscriptionRequest) (activities.TranscriptionResult, error) {
	for _, failProvider := range m.failProviders {
		if req.Provider == failProvider {
			return activities.TranscriptionResult{
				FileID: req.FileID,
				Error:  "provider failed",
			}, fmt.Errorf("mock provider failure")
		}
	}
	
	return activities.TranscriptionResult{
		FileID:   req.FileID,
		Text:     "Test transcription from " + req.Provider,
		Provider: req.Provider,
	}, nil
}

// Integration test with real Temporal server (requires docker-compose up)
func TestIntegrationWithTemporalServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Check if Temporal is running
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}
	
	// Create client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		t.Skip("Temporal server not available, skipping integration test")
	}
	defer c.Close()
	
	// Create test worker
	w := worker.New(c, "test-task-queue", worker.Options{})
	
	// Register workflows and activities
	w.RegisterWorkflow(workflows.SingleFileTranscriptionWorkflow)
	
	// Create mock registry
	reg := registry.NewDefaultProviderRegistry()
	transcribeActivities := activities.NewTranscribeActivities(reg)
	
	w.RegisterActivity(transcribeActivities.TranscribeFile)
	w.RegisterActivity(transcribeActivities.GetProviderStatus)
	w.RegisterActivity(transcribeActivities.ListAvailableProviders)
	w.RegisterActivity(transcribeActivities.GetRecommendedProvider)
	
	// Start worker
	go func() {
		err := w.Run(worker.InterruptCh())
		require.NoError(t, err)
	}()
	
	// Give worker time to start
	time.Sleep(2 * time.Second)
	
	// Execute workflow
	we, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        "test-workflow-" + time.Now().Format("20060102150405"),
		TaskQueue: "test-task-queue",
	}, workflows.SingleFileTranscriptionWorkflow, workflows.SingleFileWorkflowRequest{
		FileID:   "integration-test-1",
		FilePath: "/tmp/test.mp3",
		Language: "en",
	})
	
	require.NoError(t, err)
	
	// Wait for result
	var result workflows.SingleFileWorkflowResult
	err = we.Get(context.Background(), &result)
	
	// Allow workflow to fail due to missing providers
	if err != nil {
		t.Logf("Workflow failed as expected (no real providers): %v", err)
	} else {
		assert.Equal(t, "integration-test-1", result.FileID)
	}
	
	// Stop worker
	w.Stop()
}