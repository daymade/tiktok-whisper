package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DouyinMetrics holds all Prometheus metrics for Douyin workflows
type DouyinMetrics struct {
	// Workflow metrics
	WorkflowsTotal        *prometheus.CounterVec
	WorkflowDuration      *prometheus.HistogramVec
	WorkflowErrorsTotal   *prometheus.CounterVec

	// Activity metrics
	ActivityDuration      *prometheus.HistogramVec
	ActivityErrorsTotal   *prometheus.CounterVec

	// Business metrics
	VideosImportedTotal   prometheus.Counter
	VideoDownloadBytes    prometheus.Counter
	TranscriptionDuration *prometheus.HistogramVec
	EngagementScraped     prometheus.Counter
	CommentsScraped       prometheus.Counter
	ReportsGenerated      prometheus.Counter

	// DLQ metrics
	DLQEntriesTotal       *prometheus.CounterVec
}

// NewDouyinMetrics creates and registers all Douyin metrics
func NewDouyinMetrics(registry prometheus.Registerer) *DouyinMetrics {
	factory := promauto.With(registry)

	return &DouyinMetrics{
		// Workflow metrics
		WorkflowsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "douyin_workflows_total",
				Help: "Total number of Douyin workflows executed",
			},
			[]string{"workflow_type", "status"}, // status: success, failed
		),

		WorkflowDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "douyin_workflow_duration_seconds",
				Help:    "Duration of Douyin workflow executions",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s, 2s, 4s, ..., 512s
			},
			[]string{"workflow_type"},
		),

		WorkflowErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "douyin_workflow_errors_total",
				Help: "Total number of Douyin workflow errors by type",
			},
			[]string{"workflow_type", "error_code"},
		),

		// Activity metrics
		ActivityDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "douyin_activity_duration_seconds",
				Help:    "Duration of Douyin activity executions",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 100ms, 200ms, ..., 51.2s
			},
			[]string{"activity_name", "status"},
		),

		ActivityErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "douyin_activity_errors_total",
				Help: "Total number of Douyin activity errors",
			},
			[]string{"activity_name", "error_code"},
		),

		// Business metrics
		VideosImportedTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "douyin_videos_imported_total",
				Help: "Total number of Douyin videos successfully imported",
			},
		),

		VideoDownloadBytes: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "douyin_video_download_bytes_total",
				Help: "Total bytes downloaded from Douyin videos",
			},
		),

		TranscriptionDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "douyin_transcription_duration_seconds",
				Help:    "Duration of audio transcription by audio length",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},
			[]string{"language"},
		),

		EngagementScraped: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "douyin_engagement_scraped_total",
				Help: "Total number of engagement data scraped",
			},
		),

		CommentsScraped: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "douyin_comments_scraped_total",
				Help: "Total number of comments scraped",
			},
		),

		ReportsGenerated: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "douyin_reports_generated_total",
				Help: "Total number of AI reports generated",
			},
		),

		// DLQ metrics
		DLQEntriesTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "douyin_dlq_entries_total",
				Help: "Total number of workflows added to DLQ",
			},
			[]string{"workflow_type", "error_code"},
		),
	}
}

// RecordWorkflowCompletion records a workflow completion
func (m *DouyinMetrics) RecordWorkflowCompletion(workflowType string, success bool, durationSeconds float64) {
	status := "success"
	if !success {
		status = "failed"
	}

	m.WorkflowsTotal.WithLabelValues(workflowType, status).Inc()
	m.WorkflowDuration.WithLabelValues(workflowType).Observe(durationSeconds)
}

// RecordWorkflowError records a workflow error
func (m *DouyinMetrics) RecordWorkflowError(workflowType, errorCode string) {
	m.WorkflowErrorsTotal.WithLabelValues(workflowType, errorCode).Inc()
}

// RecordActivityExecution records an activity execution
func (m *DouyinMetrics) RecordActivityExecution(activityName string, success bool, durationSeconds float64) {
	status := "success"
	if !success {
		status = "failed"
	}

	m.ActivityDuration.WithLabelValues(activityName, status).Observe(durationSeconds)
}

// RecordActivityError records an activity error
func (m *DouyinMetrics) RecordActivityError(activityName, errorCode string) {
	m.ActivityErrorsTotal.WithLabelValues(activityName, errorCode).Inc()
}

// RecordDLQEntry records a DLQ entry
func (m *DouyinMetrics) RecordDLQEntry(workflowType, errorCode string) {
	m.DLQEntriesTotal.WithLabelValues(workflowType, errorCode).Inc()
}
