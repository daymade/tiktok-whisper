package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/app/repository"
)

// ExportServiceImpl implements the ExportService interface
type ExportServiceImpl struct {
	repo repository.TranscriptionDAOV2
}

// NewExportService creates a new export service
func NewExportService(repo repository.TranscriptionDAOV2) ExportService {
	return &ExportServiceImpl{
		repo: repo,
	}
}

// ExportTranscriptions exports transcriptions in the requested format
func (s *ExportServiceImpl) ExportTranscriptions(ctx context.Context, req dto.ExportRequest, writer io.Writer) error {
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

	// Fetch transcriptions
	limit := req.Limit
	if limit == 0 {
		limit = 10000 // Default max export limit
	}

	// TODO: Implement FindByFilters in repository
	// For now, return empty result
	transcriptions := []repository.Transcription{}
	var err error
	if err != nil {
		return fmt.Errorf("failed to fetch transcriptions: %w", err)
	}

	// Export based on format
	switch req.Format {
	case "csv":
		return s.exportCSV(transcriptions, writer)
	case "json":
		return s.exportJSON(transcriptions, writer)
	case "xlsx":
		// TODO: Implement Excel export
		return fmt.Errorf("Excel export not yet implemented")
	default:
		return fmt.Errorf("unsupported export format: %s", req.Format)
	}
}

// exportCSV exports transcriptions as CSV
func (s *ExportServiceImpl) exportCSV(transcriptions []repository.Transcription, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	header := []string{
		"ID",
		"User",
		"File Name",
		"MP3 File Name",
		"Audio Duration",
		"Transcription",
		"Conversion Time",
		"Has Error",
		"Error Message",
		"Has OpenAI Embedding",
		"Has Gemini Embedding",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, t := range transcriptions {
		row := []string{
			strconv.Itoa(t.ID),
			t.UserNickname,
			t.FileName,
			t.Mp3FileName,
			fmt.Sprintf("%.2f", t.AudioDuration),
			t.Transcription,
			t.LastConversionTime.Format(time.RFC3339),
			strconv.Itoa(t.HasError),
			t.ErrorMessage,
			strconv.FormatBool(t.EmbeddingOpenAI != nil),
			strconv.FormatBool(t.EmbeddingGemini != nil),
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// exportJSON exports transcriptions as JSON
func (s *ExportServiceImpl) exportJSON(transcriptions []repository.Transcription, writer io.Writer) error {
	// Convert to exportable format (without embedding vectors for size)
	exportData := make([]map[string]interface{}, 0, len(transcriptions))
	for _, t := range transcriptions {
		data := map[string]interface{}{
			"id":                 t.ID,
			"user":               t.UserNickname,
			"file_name":          t.FileName,
			"mp3_file_name":      t.Mp3FileName,
			"audio_duration":     t.AudioDuration,
			"transcription":      t.Transcription,
			"conversion_time":    t.LastConversionTime,
			"has_error":          t.HasError,
			"error_message":      t.ErrorMessage,
			"has_openai_embedding": t.EmbeddingOpenAI != nil,
			"has_gemini_embedding": t.EmbeddingGemini != nil,
		}
		exportData = append(exportData, data)
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exportData)
}