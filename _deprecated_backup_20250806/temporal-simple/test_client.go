package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"
)

func main() {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: "127.0.0.1:7233",
	})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	// Test file path
	testFile := "/Volumes/SSD2T/workspace/go/tiktok-whisper/test/data/jfk.wav"
	
	// Submit workflow
	workflowID := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	we, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "v2t-transcription-queue",
	}, "SingleFileTranscriptionWorkflow", map[string]interface{}{
		"file_id":   "e2e-test-1",
		"file_path": testFile,
		"language":  "en",
	})
	
	if err != nil {
		log.Fatalf("Unable to execute workflow: %v", err)
	}
	
	log.Printf("Started workflow ID: %s, RunID: %s", we.GetID(), we.GetRunID())
	
	// Wait for result
	var result map[string]interface{}
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalf("Unable to get workflow result: %v", err)
	}
	
	fmt.Printf("\n=== E2E Test Successful! ===\n")
	fmt.Printf("File ID: %v\n", result["file_id"])
	fmt.Printf("Transcription URL: %v\n", result["transcription_url"])
	fmt.Printf("Provider: %v\n", result["provider"])
	fmt.Printf("Processing Time: %v\n", result["processing_time"])
}