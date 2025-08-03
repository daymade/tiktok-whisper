package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"tiktok-whisper/internal/api/errors"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
)

// TranscriptionServiceImpl implements TranscriptionService
type TranscriptionServiceImpl struct {
	orchestrator provider.TranscriptionOrchestrator
	repository   repository.TranscriptionDAOV2
}

// NewTranscriptionService creates a new transcription service
func NewTranscriptionService(
	orchestrator provider.TranscriptionOrchestrator,
	repository repository.TranscriptionDAOV2,
) TranscriptionService {
	return &TranscriptionServiceImpl{
		orchestrator: orchestrator,
		repository:   repository,
	}
}

// CreateTranscription creates a new transcription job
func (s *TranscriptionServiceImpl) CreateTranscription(ctx context.Context, req *dto.CreateTranscriptionRequest) (*dto.TranscriptionResponse, error) {
	// Validate file exists
	fileInfo, err := os.Stat(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewBadRequestError("File not found: " + req.FilePath)
		}
		return nil, errors.NewInternalError("Failed to access file")
	}

	// Create transcription record in database
	transcription := &model.TranscriptionFull{
		User:         req.UserID,
		FileName:     req.FilePath,
		InputDir:     filepath.Dir(req.FilePath),
		FileSize:     fileInfo.Size(),
		ProviderType: req.Provider,
		Language:     req.Language,
		ModelName:    req.Model,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save to database
	err = s.repository.RecordToDBV2(transcription)
	if err != nil {
		return nil, errors.NewInternalError("Failed to create transcription record")
	}

	// Start async transcription process
	go s.processTranscription(context.Background(), transcription, req)

	// Return response
	return &dto.TranscriptionResponse{
		ID:        transcription.ID,
		UserID:    transcription.User,
		FilePath:  transcription.FileName,
		Status:    "pending",
		Provider:  transcription.ProviderType,
		Language:  transcription.Language,
		Model:     transcription.ModelName,
		FileSize:  transcription.FileSize,
		CreatedAt: transcription.CreatedAt,
		UpdatedAt: transcription.UpdatedAt,
	}, nil
}

// processTranscription handles the async transcription process
func (s *TranscriptionServiceImpl) processTranscription(ctx context.Context, transcription *model.TranscriptionFull, req *dto.CreateTranscriptionRequest) {
	// Create provider request
	providerReq := &provider.TranscriptionRequest{
		InputFilePath:   req.FilePath,
		Language:        req.Language,
		Model:           req.Model,
		ProviderOptions: req.Options,
	}

	// Execute transcription
	response, err := s.orchestrator.Transcribe(ctx, providerReq)
	
	// Update transcription record
	now := time.Now()
	transcription.UpdatedAt = now
	transcription.LastConversionTime = now

	if err != nil {
		transcription.HasError = 1
		transcription.ErrorMessage = err.Error()
	} else {
		transcription.Transcription = response.Text
		transcription.AudioDuration = int(response.Duration.Seconds())
		if response.Language != "" {
			transcription.Language = response.Language
		}
		// Keep the original provider and model from request
		// as the response doesn't have these fields
	}

	// Save update to database
	_ = s.repository.RecordToDBV2(transcription)
}

// GetTranscription retrieves a transcription by ID
func (s *TranscriptionServiceImpl) GetTranscription(ctx context.Context, id int) (*dto.TranscriptionResponse, error) {
	transcription, err := s.repository.GetTranscriptionByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("transcription")
		}
		return nil, errors.NewInternalError("Failed to retrieve transcription")
	}

	resp := dto.ToTranscriptionResponse(transcription)
	return &resp, nil
}

// ListTranscriptions lists transcriptions with pagination and filtering
func (s *TranscriptionServiceImpl) ListTranscriptions(ctx context.Context, query dto.ListTranscriptionsQuery) (*dto.PaginatedTranscriptionsResponse, error) {
	// Use available repository methods
	// For now, we'll use GetActiveTranscriptions for all transcriptions
	// and filter in memory based on query parameters
	
	limit := 1000 // Get a large batch to filter
	transcriptions, err := s.repository.GetActiveTranscriptions(limit)
	if err != nil {
		return nil, errors.NewInternalError("Failed to list transcriptions")
	}

	// Filter based on query parameters
	filtered := make([]model.TranscriptionFull, 0)
	for _, t := range transcriptions {
		// Filter by user
		if query.UserID != "" && t.User != query.UserID {
			continue
		}
		
		// Filter by provider
		if query.Provider != "" && t.ProviderType != query.Provider {
			continue
		}
		
		// Filter by status
		status := dto.DetermineStatus(&t)
		if query.Status != "" && status != query.Status {
			continue
		}
		
		filtered = append(filtered, t)
	}

	// Apply pagination
	total := len(filtered)
	start := (query.Page - 1) * query.Limit
	end := start + query.Limit
	if end > total {
		end = total
	}
	
	// Handle out of bounds
	if start >= total {
		start = total
		end = total
	}
	
	paginated := filtered[start:end]

	// Convert to response DTOs
	responses := make([]dto.TranscriptionResponse, len(paginated))
	for i, t := range paginated {
		resp := dto.ToTranscriptionResponse(&t)
		responses[i] = resp
	}

	// Calculate pagination
	totalPages := (total + query.Limit - 1) / query.Limit
	hasNext := query.Page < totalPages
	hasPrev := query.Page > 1

	return &dto.PaginatedTranscriptionsResponse{
		Transcriptions: responses,
		Pagination: dto.PaginationResponse{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}, nil
}

// DeleteTranscription deletes a transcription by ID
func (s *TranscriptionServiceImpl) DeleteTranscription(ctx context.Context, id int) error {
	// Check if exists
	_, err := s.repository.GetTranscriptionByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewNotFoundError("transcription")
		}
		return errors.NewInternalError("Failed to check transcription")
	}

	// Soft delete
	if err := s.repository.SoftDelete(id); err != nil {
		return errors.NewInternalError("Failed to delete transcription")
	}

	return nil
}