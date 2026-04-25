package etl

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"tiktok-whisper/internal/app/temporal/pkg/command"
)

var (
	youtubeURL    string
	language      string
	temporalHost  string
	waitForResult bool
)

// NewETLCommand creates the ETL command
func NewETLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "etl",
		Short: "Run ETL pipeline for YouTube videos (download → convert → transcribe)",
		Long: `Run the complete ETL pipeline:
1. Download video/audio from YouTube using yt-dlp
2. Convert to optimal audio format using ffmpeg
3. Transcribe using faster-whisper
4. Store results in MinIO`,
		Example: `  # Process a YouTube video
  v2t etl --url "https://www.youtube.com/watch?v=VIDEO_ID" --language zh

  # Process and wait for result
  v2t etl --url "https://www.youtube.com/watch?v=VIDEO_ID" --wait

  # Use custom Temporal host
  v2t etl --url "URL" --temporal-host "192.168.1.100:7233"`,
		RunE: runETL,
	}

	// Add flags
	cmd.Flags().StringVarP(&youtubeURL, "url", "u", "", "YouTube URL to process (required)")
	cmd.Flags().StringVarP(&language, "language", "l", "auto", "Language code (e.g., 'zh', 'en', 'auto')")
	cmd.Flags().StringVar(&temporalHost, "temporal-host", "127.0.0.1:7233", "Temporal server address")
	cmd.Flags().BoolVarP(&waitForResult, "wait", "w", false, "Wait for transcription to complete")

	// Mark required flags
	cmd.MarkFlagRequired("url")

	return cmd
}

func runETL(cmd *cobra.Command, args []string) error {
	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create distributed transcriber
	dt, err := command.NewDistributedTranscriber()
	if err != nil {
		return fmt.Errorf("failed to create distributed transcriber: %w", err)
	}
	defer dt.Close()

	ctx := context.Background()

	// Submit ETL job
	logger.Info("Submitting ETL job",
		zap.String("url", youtubeURL),
		zap.String("language", language))

	// TODO: Implement SubmitETLJob
	// job, err := dt.SubmitETLJob(ctx, youtubeURL, language)
	job, err := dt.SubmitJob(ctx, youtubeURL)
	if err != nil {
		return fmt.Errorf("failed to submit ETL job: %w", err)
	}

	fmt.Printf("ETL job submitted successfully!\n")
	fmt.Printf("Job ID: %s\n", job.ID)
	fmt.Printf("Workflow ID: %s\n", job.WorkflowID)
	fmt.Printf("Status: %s\n", job.Status)

	if !waitForResult {
		fmt.Printf("\nTo check status:\n")
		fmt.Printf("  v2t job status --workflow-id %s\n", job.WorkflowID)
		return nil
	}

	// Wait for result
	fmt.Printf("\nWaiting for transcription to complete...\n")
	
	result, err := command.WaitForJobWithProgress(ctx, dt, job.WorkflowID, func(status string) {
		fmt.Printf("Status: %s\n", status)
	})
	
	if err != nil {
		return err
	}
	
	fmt.Printf("\nTranscription completed successfully!\n")
	fmt.Printf("Result URL: %s\n", result.Result)
	return nil
}