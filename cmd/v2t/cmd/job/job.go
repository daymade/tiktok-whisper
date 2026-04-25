package job

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"tiktok-whisper/internal/app/temporal/pkg/command"
)

var (
	workflowID   string
	temporalHost string
)

// NewJobCommand creates the job management command
func NewJobCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Manage distributed transcription jobs",
		Long:  "Commands for checking status and managing distributed transcription jobs",
	}

	// Add subcommands
	cmd.AddCommand(newStatusCommand())

	return cmd
}

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Check job status",
		Long:    "Check the status of a distributed transcription job",
		Example: "  v2t job status --workflow-id etl-abc123-1234567890",
		RunE:    runStatus,
	}

	cmd.Flags().StringVarP(&workflowID, "workflow-id", "w", "", "Workflow ID to check (required)")
	cmd.Flags().StringVar(&temporalHost, "temporal-host", "127.0.0.1:7233", "Temporal server address")
	cmd.MarkFlagRequired("workflow-id")

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Get job status
	job, err := dt.GetJobStatus(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get job status: %w", err)
	}

	// Display status
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Workflow ID:\t%s\n", job.WorkflowID)
	fmt.Fprintf(w, "Status:\t%s\n", job.Status)
	
	if job.Result != "" {
		fmt.Fprintf(w, "Result:\t%s\n", job.Result)
	}
	
	if job.Error != "" {
		fmt.Fprintf(w, "Error:\t%s\n", job.Error)
	}
	
	if !job.CompletedAt.IsZero() {
		fmt.Fprintf(w, "Completed At:\t%s\n", job.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	
	w.Flush()
	
	return nil
}