package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
)

// WhisperJobService defines the interface for whisper job operations
type WhisperJobService interface {
	CreateJob(ctx context.Context, userID string, req *dto.CreateWhisperJobRequest) (*dto.WhisperJobResponse, error)
	GetJob(ctx context.Context, jobID string) (*dto.WhisperJobResponse, error)
	ListJobs(ctx context.Context, userID string, page, limit int, status string) ([]dto.WhisperJobResponse, int, error)
	DeleteJob(ctx context.Context, jobID string) error
	GetUserStats(ctx context.Context, userID string) (*dto.UserStatsResponse, error)
	ProcessJob(ctx context.Context, jobID string) error
}

// WhisperJobServiceImpl implements WhisperJobService
type WhisperJobServiceImpl struct {
	repo                 repository.TranscriptionDAO
	transcriptionService TranscriptionService
	providerService      ProviderService
	jobs                 map[string]*model.WhisperJob // In-memory storage for now
	redisService         *RedisJobService           // Redis storage for production
	useRedis             bool                       // Whether to use Redis storage
}

// NewWhisperJobService creates a new whisper job service
func NewWhisperJobService(
	repo repository.TranscriptionDAO,
	transcriptionService TranscriptionService,
	providerService ProviderService,
) *WhisperJobServiceImpl {
	// Check if Redis is available
	redisService := NewRedisJobService(repo)
	
	// Test Redis connection
	ctx := context.Background()
	if _, err := redisService.redisClient.Ping(ctx).Result(); err != nil {
		// Redis not available, use in-memory storage
		return &WhisperJobServiceImpl{
			repo:                 repo,
			transcriptionService: transcriptionService,
			providerService:      providerService,
			jobs:                 make(map[string]*model.WhisperJob),
			useRedis:             false,
		}
	}
	
	// Redis is available
	return &WhisperJobServiceImpl{
		repo:                 repo,
		transcriptionService: transcriptionService,
		providerService:      providerService,
		jobs:                 make(map[string]*model.WhisperJob), // Fallback
		redisService:         redisService,
		useRedis:             true,
	}
}

// CreateJob creates a new transcription job
func (s *WhisperJobServiceImpl) CreateJob(ctx context.Context, userID string, req *dto.CreateWhisperJobRequest) (*dto.WhisperJobResponse, error) {
	if s.useRedis {
		// Use Redis storage
		redisJob, err := s.redisService.CreateJob(ctx, userID, req.FileURL, req.FileName, req.FileSize, req.Provider, req.Language)
		if err != nil {
			return nil, fmt.Errorf("failed to create job in Redis: %w", err)
		}
		
		// Convert Redis job to WhisperJob for compatibility
		job := &model.WhisperJob{
			ID:            redisJob.ID,
			UserID:        redisJob.UserID,
			Status:        redisJob.Status,
			FileName:      redisJob.FileName,
			FileURL:       redisJob.FileKey,
			FileSize:      redisJob.FileSize,
			Provider:      redisJob.Provider,
			Language:      redisJob.Language,
			CreditCost:    max(5, (req.AudioDuration/60)*10),
			Metadata:      redisJob.Metadata,
			CreatedAt:     redisJob.CreatedAt,
			UpdatedAt:     redisJob.UpdatedAt,
		}
		
		// Start async processing
		go s.ProcessJob(context.Background(), redisJob.ID)
		
		return s.jobToResponse(job), nil
	}
	
	// Fallback to in-memory storage
	jobID := uuid.New().String()
	
	// Calculate credit cost (10 credits per minute, minimum 5)
	creditCost := 5
	if req.AudioDuration > 0 {
		creditCost = max(5, (req.AudioDuration/60)*10)
	}
	
	// Create job record
	job := &model.WhisperJob{
		ID:            jobID,
		UserID:        userID,
		Status:        string(dto.JobStatusPending),
		FileName:      req.FileName,
		FileURL:       req.FileURL,
		FileSize:      req.FileSize,
		AudioDuration: req.AudioDuration,
		Provider:      req.Provider,
		Language:      req.Language,
		OutputFormat:  req.OutputFormat,
		CreditCost:    creditCost,
		Metadata:      req.Metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	// Store in memory
	s.jobs[jobID] = job
	
	// Start async processing
	go s.ProcessJob(context.Background(), jobID)
	
	return s.jobToResponse(job), nil
}

// GetJob retrieves a job by ID
func (s *WhisperJobServiceImpl) GetJob(ctx context.Context, jobID string) (*dto.WhisperJobResponse, error) {
	if s.useRedis {
		redisJob, err := s.redisService.GetJob(ctx, jobID)
		if err != nil {
			return nil, err
		}
		
		// Convert Redis job to WhisperJob for compatibility
		job := &model.WhisperJob{
			ID:            redisJob.ID,
			UserID:        redisJob.UserID,
			Status:        redisJob.Status,
			FileName:      redisJob.FileName,
			FileURL:       redisJob.FileKey,
			FileSize:      redisJob.FileSize,
			Provider:      redisJob.Provider,
			Language:      redisJob.Language,
			CreditCost:    5, // Default if not available
			Metadata:      redisJob.Metadata,
			CreatedAt:     redisJob.CreatedAt,
			UpdatedAt:     redisJob.UpdatedAt,
		}
		
		return s.jobToResponse(job), nil
	}
	
	// Fallback to in-memory storage
	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	
	return s.jobToResponse(job), nil
}

// ListJobs lists jobs for a user
func (s *WhisperJobServiceImpl) ListJobs(ctx context.Context, userID string, page, limit int, status string) ([]dto.WhisperJobResponse, int, error) {
	if s.useRedis {
		redisJobs, total, err := s.redisService.ListJobs(ctx, userID, status, page, limit)
		if err != nil {
			return nil, 0, err
		}
		
		// Convert Redis jobs to WhisperJob responses
		responses := make([]dto.WhisperJobResponse, 0)
		for _, redisJob := range redisJobs {
			job := &model.WhisperJob{
				ID:            redisJob.ID,
				UserID:        redisJob.UserID,
				Status:        redisJob.Status,
				FileName:      redisJob.FileName,
				FileURL:       redisJob.FileKey,
				FileSize:      redisJob.FileSize,
				Provider:      redisJob.Provider,
				Language:      redisJob.Language,
				CreditCost:    5, // Default if not available
				Metadata:      redisJob.Metadata,
				CreatedAt:     redisJob.CreatedAt,
				UpdatedAt:     redisJob.UpdatedAt,
			}
			responses = append(responses, *s.jobToResponse(job))
		}
		
		return responses, total, nil
	}
	
	// Fallback to in-memory storage
	var userJobs []*model.WhisperJob
	for _, job := range s.jobs {
		if job.UserID == userID {
			if status == "" || job.Status == status {
				userJobs = append(userJobs, job)
			}
		}
	}
	
	// Calculate pagination
	total := len(userJobs)
	start := (page - 1) * limit
	end := min(start+limit, total)
	
	if start >= total {
		return []dto.WhisperJobResponse{}, total, nil
	}
	
	// Convert to response format
	responses := make([]dto.WhisperJobResponse, 0)
	for i := start; i < end; i++ {
		responses = append(responses, *s.jobToResponse(userJobs[i]))
	}
	
	return responses, total, nil
}

// DeleteJob deletes/cancels a job
func (s *WhisperJobServiceImpl) DeleteJob(ctx context.Context, jobID string) error {
	if s.useRedis {
		// Update job status to cancelled in Redis
		err := s.redisService.UpdateJob(ctx, jobID, "cancelled", 0, "")
		if err != nil {
			return fmt.Errorf("failed to cancel job: %w", err)
		}
		return nil
	}
	
	// Fallback to in-memory storage
	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	
	// Mark as cancelled
	job.Status = string(dto.JobStatusCancelled)
	job.UpdatedAt = time.Now()
	
	return nil
}

// GetUserStats gets statistics for a user
func (s *WhisperJobServiceImpl) GetUserStats(ctx context.Context, userID string) (*dto.UserStatsResponse, error) {
	stats := &dto.UserStatsResponse{
		UserID:        userID,
		ProviderUsage: make(map[string]int),
	}
	
	// Calculate stats from jobs
	for _, job := range s.jobs {
		if job.UserID == userID {
			stats.TotalJobs++
			stats.TotalCreditsUsed += job.CreditCost
			stats.TotalAudioMinutes += job.AudioDuration / 60
			
			if job.Status == string(dto.JobStatusCompleted) {
				stats.CompletedJobs++
				stats.TotalTranscriptions++
			} else if job.Status == string(dto.JobStatusFailed) {
				stats.FailedJobs++
			}
			
			if job.Provider != "" {
				stats.ProviderUsage[job.Provider]++
			}
			
			if stats.LastJobAt.Before(job.CreatedAt) {
				stats.LastJobAt = job.CreatedAt
			}
		}
	}
	
	return stats, nil
}

// ProcessJob processes a transcription job asynchronously
func (s *WhisperJobServiceImpl) ProcessJob(ctx context.Context, jobID string) error {
	var job *model.WhisperJob
	var err error
	
	if s.useRedis {
		// Get job from Redis
		redisJob, err := s.redisService.GetJob(ctx, jobID)
		if err != nil {
			return fmt.Errorf("job not found: %s", jobID)
		}
		
		// Update status to processing in Redis
		if err := s.redisService.UpdateJob(ctx, jobID, "processing", 0, ""); err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}
		
		// Convert Redis job to WhisperJob for processing
		job = &model.WhisperJob{
			ID:            redisJob.ID,
			UserID:        redisJob.UserID,
			Status:        redisJob.Status,
			FileName:      redisJob.FileName,
			FileURL:       redisJob.FileKey,
			FileSize:      redisJob.FileSize,
			Provider:      redisJob.Provider,
			Language:      redisJob.Language,
			Metadata:      redisJob.Metadata,
			CreatedAt:     redisJob.CreatedAt,
			UpdatedAt:     redisJob.UpdatedAt,
		}
	} else {
		// Fallback to in-memory storage
		var exists bool
		job, exists = s.jobs[jobID]
		if !exists {
			return fmt.Errorf("job not found: %s", jobID)
		}
		
		// Update status to processing
		job.Status = string(dto.JobStatusProcessing)
		now := time.Now()
		job.StartedAt = &now
		job.UpdatedAt = now
	}
	
	// Use the new processor for whisper_server provider
	if job.Provider == "whisper_server" {
		err = ProcessWhisperJobWithProvider(job)
	} else {
		// Fallback to original transcription service for other providers
		transcriptionReq := &dto.CreateTranscriptionRequest{
			FilePath: job.FileURL,
			UserID:   job.UserID,
			Language: job.Language,
			Provider: job.Provider,
			Options: map[string]interface{}{
				"output_format": job.OutputFormat,
			},
		}
		
		// Call transcription service
		transcription, err := s.transcriptionService.CreateTranscription(ctx, transcriptionReq)
		if err == nil && transcription != nil {
			job.WhisperJobID = &transcription.ID
			job.TranscriptionText = transcription.Transcription
		}
	}
	
	if err != nil {
		if s.useRedis {
			// Update status in Redis
			s.redisService.UpdateJob(ctx, jobID, "failed", 0, err.Error())
		} else {
			// Update status in memory
			job.Status = string(dto.JobStatusFailed)
			job.Error = err.Error()
			job.UpdatedAt = time.Now()
		}
		return err
	}
	
	// Update job with results
	if s.useRedis {
		// Update status in Redis
		s.redisService.UpdateJob(ctx, jobID, "completed", 100, "")
	} else {
		// Update status in memory
		job.Status = string(dto.JobStatusCompleted)
		// Note: transcription data is already set by ProcessWhisperJobWithProvider
		completedAt := time.Now()
		job.CompletedAt = &completedAt
		job.UpdatedAt = completedAt
	}
	
	return nil
}

// jobToResponse converts internal job model to API response
func (s *WhisperJobServiceImpl) jobToResponse(job *model.WhisperJob) *dto.WhisperJobResponse {
	resp := &dto.WhisperJobResponse{
		ID:            job.ID,
		UserID:        job.UserID,
		WhisperJobID:  job.WhisperJobID,
		Status:        job.Status,
		FileName:      job.FileName,
		FileURL:       job.FileURL,
		FileSize:      job.FileSize,
		AudioDuration: job.AudioDuration,
		CreditCost:    job.CreditCost,
		ProviderID:    job.Provider,
		Language:      job.Language,
		OutputFormat:  job.OutputFormat,
		Error:         job.Error,
		Metadata:      job.Metadata,
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
		StartedAt:     job.StartedAt,
		CompletedAt:   job.CompletedAt,
	}
	
	// Add provider name if available
	if job.Provider != "" {
		if provider, err := s.providerService.GetProvider(context.Background(), job.Provider); err == nil {
			resp.ProviderName = provider.Name
		}
	}
	
	// Add transcription data if available
	if job.TranscriptionText != "" {
		resp.TranscriptionText = job.TranscriptionText
		resp.TranscriptionURL = fmt.Sprintf("/api/whisper/jobs/%s/transcription", job.ID)
	}
	
	return resp
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}