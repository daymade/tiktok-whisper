package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"go.temporal.io/sdk/client"
	
	"tiktok-whisper/temporal/workflows"
)

var (
	temporalHost string
	namespace    string
	taskQueue    string
	useMinIO     bool
	provider     string
	language     string
	maxParallel  int
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	rootCmd := &cobra.Command{
		Use:   "v2t-distributed",
		Short: "Distributed transcription client for v2t",
		Long:  "Submit transcription jobs to the distributed v2t Temporal cluster",
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&temporalHost, "temporal-host", getEnv("TEMPORAL_HOST", "localhost:7233"), "Temporal server address")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", getEnv("TEMPORAL_NAMESPACE", "default"), "Temporal namespace")
	rootCmd.PersistentFlags().StringVar(&taskQueue, "task-queue", getEnv("TASK_QUEUE", "v2t-transcription-queue"), "Task queue name")
	rootCmd.PersistentFlags().BoolVar(&useMinIO, "use-minio", true, "Use MinIO for file storage")

	// Add subcommands
	rootCmd.AddCommand(transcribeCmd())
	rootCmd.AddCommand(batchCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(listCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func transcribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcribe <file-path>",
		Short: "Transcribe a single file",
		Args:  cobra.ExactArgs(1),
		RunE:  runTranscribe,
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Transcription provider (default: auto-select)")
	cmd.Flags().StringVar(&language, "language", "en", "Language code")

	return cmd
}

func batchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch <directory>",
		Short: "Transcribe all files in a directory",
		Args:  cobra.ExactArgs(1),
		RunE:  runBatch,
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Transcription provider (default: auto-select)")
	cmd.Flags().StringVar(&language, "language", "en", "Language code")
	cmd.Flags().IntVar(&maxParallel, "parallel", 5, "Maximum parallel transcriptions")
	cmd.Flags().StringP("extension", "e", "mp3,wav,m4a,mp4", "File extensions to process (comma-separated)")

	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <workflow-id>",
		Short: "Check workflow status",
		Args:  cobra.ExactArgs(1),
		RunE:  runStatus,
	}
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent workflows",
		RunE:  runList,
	}

	cmd.Flags().IntP("limit", "l", 10, "Number of workflows to list")

	return cmd
}

func runTranscribe(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Validate file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Create Temporal client
	c, err := createTemporalClient()
	if err != nil {
		return err
	}
	defer c.Close()

	// Generate workflow ID
	workflowID := fmt.Sprintf("transcribe-%s-%d", 
		strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		time.Now().Unix())

	// Prepare request
	request := workflows.SingleFileWorkflowRequest{
		FileID:       uuid.New().String(),
		FilePath:     filePath,
		Provider:     provider,
		Language:     language,
		OutputFormat: "text",
		UseMinIO:     useMinIO,
	}

	// Execute workflow
	we, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
	}, workflows.SingleFileTranscriptionWorkflow, request)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	fmt.Printf("Started workflow: %s\n", we.GetID())
	fmt.Printf("Run ID: %s\n", we.GetRunID())

	// Wait for result
	fmt.Println("Waiting for transcription to complete...")
	
	var result workflows.SingleFileWorkflowResult
	err = we.Get(context.Background(), &result)
	if err != nil {
		return fmt.Errorf("workflow failed: %w", err)
	}

	if result.Error != "" {
		return fmt.Errorf("transcription failed: %s", result.Error)
	}

	fmt.Printf("\nTranscription completed successfully!\n")
	fmt.Printf("Provider: %s\n", result.Provider)
	fmt.Printf("Processing time: %s\n", result.ProcessingTime)
	fmt.Printf("Result location: %s\n", result.TranscriptionURL)

	return nil
}

func runBatch(cmd *cobra.Command, args []string) error {
	directory := args[0]

	// Validate directory exists
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("directory not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", directory)
	}

	// Get file extensions
	extensions, _ := cmd.Flags().GetString("extension")
	extList := strings.Split(extensions, ",")
	extMap := make(map[string]bool)
	for _, ext := range extList {
		extMap["."+strings.TrimPrefix(ext, ".")] = true
	}

	// Find all matching files
	var files []workflows.BatchFile
	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && extMap[filepath.Ext(path)] {
			files = append(files, workflows.BatchFile{
				FileID:   uuid.New().String(),
				FilePath: path,
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no matching files found in directory")
	}

	fmt.Printf("Found %d files to transcribe\n", len(files))

	// Create Temporal client
	c, err := createTemporalClient()
	if err != nil {
		return err
	}
	defer c.Close()

	// Generate workflow ID
	workflowID := fmt.Sprintf("batch-%s-%d", 
		filepath.Base(directory),
		time.Now().Unix())

	// Prepare request
	request := workflows.BatchWorkflowRequest{
		BatchID:     uuid.New().String(),
		Files:       files,
		Provider:    provider,
		Language:    language,
		MaxParallel: maxParallel,
		UseMinIO:    useMinIO,
	}

	// Execute workflow
	we, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
	}, workflows.BatchTranscriptionWorkflow, request)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	fmt.Printf("\nStarted batch workflow: %s\n", we.GetID())
	fmt.Printf("Run ID: %s\n", we.GetRunID())
	fmt.Println("\nProcessing files...")

	// Wait for result
	var result workflows.BatchWorkflowResult
	err = we.Get(context.Background(), &result)
	if err != nil {
		return fmt.Errorf("workflow failed: %w", err)
	}

	fmt.Printf("\nBatch transcription completed!\n")
	fmt.Printf("Total files: %d\n", result.TotalFiles)
	fmt.Printf("Successful: %d\n", result.SuccessCount)
	fmt.Printf("Failed: %d\n", result.FailureCount)
	fmt.Printf("Total processing time: %s\n", result.ProcessingTime)

	// Show failed files if any
	if result.FailureCount > 0 {
		fmt.Println("\nFailed files:")
		for _, r := range result.Results {
			if r.Error != "" {
				fmt.Printf("  - %s: %s\n", r.FileID, r.Error)
			}
		}
	}

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	workflowID := args[0]

	c, err := createTemporalClient()
	if err != nil {
		return err
	}
	defer c.Close()

	// Describe workflow execution
	resp, err := c.DescribeWorkflowExecution(context.Background(), workflowID, "")
	if err != nil {
		return fmt.Errorf("failed to describe workflow: %w", err)
	}

	fmt.Printf("Workflow ID: %s\n", workflowID)
	fmt.Printf("Status: %s\n", resp.WorkflowExecutionInfo.Status)
	fmt.Printf("Type: %s\n", resp.WorkflowExecutionInfo.Type.Name)
	fmt.Printf("Start Time: %s\n", resp.WorkflowExecutionInfo.StartTime)
	
	if resp.WorkflowExecutionInfo.CloseTime != nil {
		fmt.Printf("Close Time: %s\n", *resp.WorkflowExecutionInfo.CloseTime)
		duration := resp.WorkflowExecutionInfo.CloseTime.Sub(resp.WorkflowExecutionInfo.StartTime)
		fmt.Printf("Duration: %s\n", duration)
	}

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")

	c, err := createTemporalClient()
	if err != nil {
		return err
	}
	defer c.Close()

	// List workflows
	var workflows []client.WorkflowExecutionInfo
	iter, err := c.ListWorkflow(context.Background(), &client.ListWorkflowExecutionsRequest{
		PageSize: int32(limit),
	})
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	for iter.HasNext() {
		exec, err := iter.Next()
		if err != nil {
			break
		}
		workflows = append(workflows, exec)
		if len(workflows) >= limit {
			break
		}
	}

	if len(workflows) == 0 {
		fmt.Println("No workflows found")
		return nil
	}

	fmt.Printf("Recent workflows (showing %d):\n\n", len(workflows))
	fmt.Printf("%-40s %-20s %-15s %s\n", "WORKFLOW ID", "TYPE", "STATUS", "START TIME")
	fmt.Println(strings.Repeat("-", 100))

	for _, wf := range workflows {
		fmt.Printf("%-40s %-20s %-15s %s\n",
			wf.Execution.WorkflowId,
			wf.Type.Name,
			wf.Status,
			wf.StartTime.Format("2006-01-02 15:04:05"),
		)
	}

	return nil
}

func createTemporalClient() (client.Client, error) {
	return client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}