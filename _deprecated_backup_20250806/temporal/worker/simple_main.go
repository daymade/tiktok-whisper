package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"tiktok-whisper/temporal/activities"
	"tiktok-whisper/temporal/workflows"
)

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
		MaxConcurrentActivityExecutionSize: 10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register workflows
	w.RegisterWorkflow(workflows.SimpleSingleFileWorkflow)
	w.RegisterWorkflow(workflows.SingleFileTranscriptionWorkflow)
	w.RegisterWorkflow(workflows.BatchTranscriptionWorkflow)
	w.RegisterWorkflow(workflows.TranscriptionWithFallbackWorkflow)

	// Register activities
	simpleActivities := activities.NewSimpleTranscribeActivities()
	w.RegisterActivity(simpleActivities.TranscribeFileSimple)
	w.RegisterActivity(simpleActivities.GetProviderStatus)

	// Register storage activities
	storageActivities := activities.NewStorageActivities()
	w.RegisterActivity(storageActivities.UploadFile)
	w.RegisterActivity(storageActivities.DownloadFile)
	w.RegisterActivity(storageActivities.CleanupTempFile)

	log.Printf("Starting worker on task queue: %s", taskQueue)
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