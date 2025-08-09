package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"tiktok-whisper/internal/app/repository"
)

// RedisJobService implements WhisperJobService with Redis backend
type RedisJobService struct {
	redisClient *redis.Client
	logger      *zap.Logger
	repository  repository.TranscriptionDAO
}

// NewRedisJobService creates a new Redis-based job service
func NewRedisJobService(repository repository.TranscriptionDAO) *RedisJobService {
	// Initialize Redis client
	redisAddr := "localhost:6379"
	if addr := ""; addr != "" {
		redisAddr = addr
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	logger, _ := zap.NewProduction()

	return &RedisJobService{
		redisClient: redisClient,
		logger:      logger,
		repository:  repository,
	}
}

// Job represents a transcription job in Redis
type Job struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	FileKey     string                 `json:"file_key"`
	FileName    string                 `json:"file_name"`
	FileSize    int64                  `json:"file_size"`
	Provider    string                 `json:"provider"`
	Language    string                 `json:"language"`
	Status      string                 `json:"status"` // pending, processing, completed, failed
	Progress    int                    `json:"progress"`
	Error       string                 `json:"error"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// CreateJob creates a new job and stores it in Redis
func (s *RedisJobService) CreateJob(ctx context.Context, userID, fileKey, fileName string, fileSize int64, provider, language string) (*Job, error) {
	job := &Job{
		ID:        uuid.New().String(),
		UserID:    userID,
		FileKey:   fileKey,
		FileName:  fileName,
		FileSize:  fileSize,
		Provider:  provider,
		Language:  language,
		Status:    "pending",
		Progress:  0,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Serialize job to JSON
	jobData, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job: %w", err)
	}

	// Store job in Redis
	pipe := s.redisClient.Pipeline()
	pipe.HSet(ctx, "jobs", job.ID, jobData)
	pipe.ZAdd(ctx, "jobs:pending", redis.Z{Score: float64(job.CreatedAt.Unix()), Member: job.ID})
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to store job: %w", err)
	}

	s.logger.Info("Job created", zap.String("job_id", job.ID))
	return job, nil
}

// GetJob retrieves a job by ID
func (s *RedisJobService) GetJob(ctx context.Context, jobID string) (*Job, error) {
	jobData, err := s.redisClient.HGet(ctx, "jobs", jobID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job Job
	if err := json.Unmarshal(jobData, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// UpdateJob updates a job's status and metadata
func (s *RedisJobService) UpdateJob(ctx context.Context, jobID string, status string, progress int, errorMsg string) error {
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	oldStatus := job.Status
	job.Status = status
	job.Progress = progress
	job.UpdatedAt = time.Now()

	if errorMsg != "" {
		job.Error = errorMsg
	}

	if status == "completed" || status == "failed" {
		now := time.Now()
		job.CompletedAt = &now
	}

	// Serialize and update job
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Update job in Redis and move between sets
	pipe := s.redisClient.Pipeline()
	pipe.HSet(ctx, "jobs", job.ID, jobData)

	// Move job between status sets
	if oldStatus != status {
		if oldStatus == "pending" {
			pipe.ZRem(ctx, "jobs:pending", job.ID)
		} else if oldStatus == "processing" {
			pipe.ZRem(ctx, "jobs:processing", job.ID)
		}

		if status == "processing" {
			pipe.ZAdd(ctx, "jobs:processing", redis.Z{Score: float64(job.UpdatedAt.Unix()), Member: job.ID})
		} else if status == "completed" {
			pipe.ZAdd(ctx, "jobs:completed", redis.Z{Score: float64(job.UpdatedAt.Unix()), Member: job.ID})
		} else if status == "failed" {
			pipe.ZAdd(ctx, "jobs:failed", redis.Z{Score: float64(job.UpdatedAt.Unix()), Member: job.ID})
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	s.logger.Info("Job updated", 
		zap.String("job_id", jobID),
		zap.String("status", status),
		zap.Int("progress", progress))

	return nil
}

// ListJobs lists jobs for a user with pagination
func (s *RedisJobService) ListJobs(ctx context.Context, userID string, status string, page, limit int) ([]*Job, int, error) {
	var jobIDs []string
	var err error

	// Get job IDs based on status
	switch status {
	case "pending":
		jobIDs, err = s.redisClient.ZRange(ctx, "jobs:pending", 0, -1).Result()
	case "processing":
		jobIDs, err = s.redisClient.ZRange(ctx, "jobs:processing", 0, -1).Result()
	case "completed":
		jobIDs, err = s.redisClient.ZRange(ctx, "jobs:completed", 0, -1).Result()
	case "failed":
		jobIDs, err = s.redisClient.ZRange(ctx, "jobs:failed", 0, -1).Result()
	default:
		// Get all job IDs
		jobIDs, err = s.redisClient.HKeys(ctx, "jobs").Result()
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get job IDs: %w", err)
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit - 1
	if end >= len(jobIDs) {
		end = len(jobIDs) - 1
	}
	if start >= len(jobIDs) {
		return []*Job{}, 0, nil
	}
	jobIDs = jobIDs[start : end+1]

	// Get job data
	pipe := s.redisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(jobIDs))
	for i, jobID := range jobIDs {
		cmds[i] = pipe.HGet(ctx, "jobs", jobID)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get job data: %w", err)
	}

	// Parse jobs
	var jobs []*Job
	for i, cmd := range cmds {
		jobData, err := cmd.Bytes()
		if err != nil {
			s.logger.Error("Failed to get job data", zap.String("job_id", jobIDs[i]), zap.Error(err))
			continue
		}

		var job Job
		if err := json.Unmarshal(jobData, &job); err != nil {
			s.logger.Error("Failed to unmarshal job", zap.String("job_id", jobIDs[i]), zap.Error(err))
			continue
		}

		// Filter by user ID if specified
		if userID != "" && job.UserID != userID {
			continue
		}

		jobs = append(jobs, &job)
	}

	// Get total count
	total := len(jobIDs)
	if userID != "" {
		// Recalculate total for specific user
		total = 0
		for _, jobID := range jobIDs {
			var job Job
			jobData, _ := s.redisClient.HGet(ctx, "jobs", jobID).Bytes()
			if jobData != nil {
				json.Unmarshal(jobData, &job)
				if job.UserID == userID {
					total++
				}
			}
		}
	}

	return jobs, total, nil
}

// DeleteJob deletes a job
func (s *RedisJobService) DeleteJob(ctx context.Context, jobID string) error {
	_, err := s.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Remove from Redis
	pipe := s.redisClient.Pipeline()
	pipe.HDel(ctx, "jobs", jobID)
	
	// Remove from status sets
	pipe.ZRem(ctx, "jobs:pending", jobID)
	pipe.ZRem(ctx, "jobs:processing", jobID)
	pipe.ZRem(ctx, "jobs:completed", jobID)
	pipe.ZRem(ctx, "jobs:failed", jobID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	s.logger.Info("Job deleted", zap.String("job_id", jobID))
	return nil
}

// GetJobStats returns job statistics
func (s *RedisJobService) GetJobStats(ctx context.Context) (map[string]int64, error) {
	pipe := s.redisClient.Pipeline()
	pendingCmd := pipe.ZCard(ctx, "jobs:pending")
	processingCmd := pipe.ZCard(ctx, "jobs:processing")
	completedCmd := pipe.ZCard(ctx, "jobs:completed")
	failedCmd := pipe.ZCard(ctx, "jobs:failed")
	totalCmd := pipe.HLen(ctx, "jobs")

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}

	stats := map[string]int64{
		"pending":   pendingCmd.Val(),
		"processing": processingCmd.Val(),
		"completed": completedCmd.Val(),
		"failed":    failedCmd.Val(),
		"total":     totalCmd.Val(),
	}

	return stats, nil
}

// CleanupOldJobs removes jobs older than specified duration
func (s *RedisJobService) CleanupOldJobs(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// Get all job IDs
	jobIDs, err := s.redisClient.HKeys(ctx, "jobs").Result()
	if err != nil {
		return fmt.Errorf("failed to get job IDs: %w", err)
	}

	deleted := 0
	for _, jobID := range jobIDs {
		jobData, err := s.redisClient.HGet(ctx, "jobs", jobID).Bytes()
		if err != nil {
			continue
		}

		var job Job
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue
		}

		if job.CreatedAt.Before(cutoff) {
			if err := s.DeleteJob(ctx, jobID); err == nil {
				deleted++
			}
		}
	}

	s.logger.Info("Cleaned up old jobs", zap.Int("deleted", deleted))
	return nil
}

// Close closes the Redis connection
func (s *RedisJobService) Close() error {
	return s.redisClient.Close()
}