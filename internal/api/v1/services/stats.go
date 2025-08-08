package services

import (
	"context"
	"fmt"
	// "time"

	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/app/repository"
)

// StatsServiceImpl implements the StatsService interface
type StatsServiceImpl struct {
	repo repository.TranscriptionDAOV2
}

// NewStatsService creates a new stats service
func NewStatsService(repo repository.TranscriptionDAOV2) StatsService {
	return &StatsServiceImpl{
		repo: repo,
	}
}

// GetSystemStats returns system-wide statistics
func (s *StatsServiceImpl) GetSystemStats(ctx context.Context) (*dto.SystemStats, error) {
	stats := &dto.SystemStats{}

	// Get overall counts
	// TODO: Implement GetCounts in repository
	// For now, return placeholder data
	counts := &repository.RepositoryCounts{
		Total:            0,
		GeminiEmbeddings: 0,
		OpenAIEmbeddings: 0,
	}
	var err error
	if err != nil {
		return nil, fmt.Errorf("failed to get counts: %w", err)
	}

	stats.TotalTranscripts = counts.Total
	stats.GeminiEmbeddings = counts.GeminiEmbeddings
	stats.OpenAIEmbeddings = counts.OpenAIEmbeddings
	stats.PendingProcessing = counts.Total - counts.GeminiEmbeddings - counts.OpenAIEmbeddings

	// Get top users
	topUsers, err := s.GetUserStats(ctx, dto.StatsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Limit to top 10 users
	if len(topUsers) > 10 {
		topUsers = topUsers[:10]
	}
	stats.TopUsers = topUsers

	return stats, nil
}

// GetUserStats returns user statistics
func (s *StatsServiceImpl) GetUserStats(ctx context.Context, req dto.StatsRequest) ([]dto.UserStats, error) {
	// Parse date filters if provided
	// var startTime, endTime *time.Time
	// if req.StartDate != "" {
	// 	t, err := time.Parse("2006-01-02", req.StartDate)
	// 	if err == nil {
	// 		startTime = &t
	// 	}
	// }
	// if req.EndDate != "" {
	// 	t, err := time.Parse("2006-01-02", req.EndDate)
	// 	if err == nil {
	// 		endTime = &t
	// 	}
	// }

	// Get user statistics from repository
	// TODO: Implement GetUserStats in repository
	// For now, return empty result
	userStats := []repository.UserStatistics{}
	var err error
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Convert to DTOs
	stats := make([]dto.UserStats, 0, len(userStats))
	for _, us := range userStats {
		stat := dto.UserStats{
			User:             us.User,
			TotalTranscripts: us.TotalTranscripts,
			GeminiEmbeddings: us.GeminiEmbeddings,
			OpenAIEmbeddings: us.OpenAIEmbeddings,
		}
		stats = append(stats, stat)
	}

	return stats, nil
}