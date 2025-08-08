package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Request/Response types
type TranscriptionRequest struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	Language string `json:"language"`
}

type TranscriptionResult struct {
	FileID   string `json:"file_id"`
	Text     string `json:"text"`
	Provider string `json:"provider"`
	Error    string `json:"error,omitempty"`
}

type SingleFileWorkflowRequest struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	Language string `json:"language"`
}

type SingleFileWorkflowResult struct {
	FileID           string        `json:"file_id"`
	TranscriptionURL string        `json:"transcription_url"`
	Provider         string        `json:"provider"`
	ProcessingTime   time.Duration `json:"processing_time"`
	Error            string        `json:"error,omitempty"`
}

// Activity implementation
func TranscribeFileSimple(ctx context.Context, req TranscriptionRequest) (TranscriptionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting transcription", "fileId", req.FileID, "file", req.FilePath)

	// Get whisper binary path
	whisperBinary := getEnv("WHISPER_BINARY", "/Volumes/SSD2T/workspace/cpp/whisper.cpp-updated/build/bin/whisper-cli")
	modelPath := getEnv("WHISPER_MODEL", "/Volumes/SSD2T/workspace/cpp/whisper.cpp-updated/models/ggml-large-v3.bin")

	// Prepare command
	args := []string{
		"-m", modelPath,
		"-f", req.FilePath,
		"--no-timestamps",
		"--no-prints",  // Suppress progress output
		"-of", "txt",   // Output format text only
	}

	if req.Language != "" && req.Language != "auto" {
		args = append(args, "--language", req.Language)
	}

	// Log the exact command being executed
	logger.Info("Executing whisper command", "binary", whisperBinary, "args", args)
	
	// Execute whisper
	cmd := exec.CommandContext(ctx, whisperBinary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Whisper execution failed", 
			"error", err, 
			"output", string(output),
			"binary", whisperBinary,
			"args", args)
		return TranscriptionResult{
			FileID: req.FileID,
			Error:  fmt.Sprintf("whisper execution failed: %v - output: %s", err, string(output)),
		}, err
	}

	return TranscriptionResult{
		FileID:   req.FileID,
		Text:     string(output),
		Provider: "whisper_cpp",
	}, nil
}

// Workflow implementation
func SingleFileTranscriptionWorkflow(ctx workflow.Context, req SingleFileWorkflowRequest) (SingleFileWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting single file transcription workflow", "fileId", req.FileID)

	startTime := workflow.Now(ctx)

	// Configure activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    100 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Perform transcription
	var transcriptionResult TranscriptionResult
	err := workflow.ExecuteActivity(ctx, TranscribeFileSimple, TranscriptionRequest{
		FileID:   req.FileID,
		FilePath: req.FilePath,
		Language: req.Language,
	}).Get(ctx, &transcriptionResult)

	if err != nil {
		return SingleFileWorkflowResult{
			FileID: req.FileID,
			Error:  fmt.Sprintf("Failed to transcribe: %v", err),
		}, err
	}

	// Save transcription
	outputPath := filepath.Join(filepath.Dir(req.FilePath),
		fmt.Sprintf("%s_transcription.txt", req.FileID))

	// Use side effect for file writing
	err = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		return os.WriteFile(outputPath, []byte(transcriptionResult.Text), 0644)
	}).Get(&err)

	if err != nil {
		return SingleFileWorkflowResult{
			FileID: req.FileID,
			Error:  fmt.Sprintf("Failed to save transcription: %v", err),
		}, err
	}

	processingTime := workflow.Now(ctx).Sub(startTime)

	return SingleFileWorkflowResult{
		FileID:           req.FileID,
		TranscriptionURL: outputPath,
		Provider:         transcriptionResult.Provider,
		ProcessingTime:   processingTime,
	}, nil
}

func main() {
	// Create Temporal client
	temporalHost := getEnv("TEMPORAL_HOST", "127.0.0.1:7233")
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	// Create worker
	taskQueue := getEnv("TASK_QUEUE", "v2t-transcription-queue")
	w := worker.New(c, taskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     5,
		MaxConcurrentWorkflowTaskExecutionSize: 5,
	})

	// Register workflow and activity
	w.RegisterWorkflow(SingleFileTranscriptionWorkflow)
	w.RegisterActivity(TranscribeFileSimple)

	log.Printf("Starting simple worker on task queue: %s", taskQueue)
	log.Printf("Temporal host: %s", temporalHost)
	log.Printf("Whisper binary: %s", getEnv("WHISPER_BINARY", "default"))

	// Run worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}