package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	apiProvider "tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
)

type fakeOrchestrator struct{}

func (f *fakeOrchestrator) Transcribe(ctx context.Context, request *apiProvider.TranscriptionRequest) (*apiProvider.TranscriptionResponse, error) {
	return &apiProvider.TranscriptionResponse{Text: "ok", Duration: time.Second}, nil
}
func (f *fakeOrchestrator) TranscribeWithProvider(ctx context.Context, providerName string, request *apiProvider.TranscriptionRequest) (*apiProvider.TranscriptionResponse, error) {
	return f.Transcribe(ctx, request)
}
func (f *fakeOrchestrator) RecommendProvider(request *apiProvider.TranscriptionRequest) ([]string, error) {
	return []string{apiProvider.ProviderNameWhisperCpp}, nil
}
func (f *fakeOrchestrator) GetStats() apiProvider.OrchestratorStats { return apiProvider.OrchestratorStats{} }

type fakeDAOV2 struct {
	recordCalls []*model.TranscriptionFull
}

func (f *fakeDAOV2) Close() error                                            { return nil }
func (f *fakeDAOV2) CheckIfFileProcessed(fileName string) (int, error) { return 0, nil }
func (f *fakeDAOV2) DeleteByID(id int) error                            { return nil }
func (f *fakeDAOV2) RecordToDB(input repository.RecordInput)            {}
func (f *fakeDAOV2) GetAllByUser(userNickname string) ([]model.Transcription, error) {
	return nil, nil
}
func (f *fakeDAOV2) RecordToDBV2(t *model.TranscriptionFull) error {
	if t.ID == 0 {
		t.ID = 42
	}
	clone := *t
	f.recordCalls = append(f.recordCalls, &clone)
	return nil
}
func (f *fakeDAOV2) GetAllByUserV2(userNickname string) ([]model.TranscriptionFull, error) {
	return nil, nil
}
func (f *fakeDAOV2) GetByFileHash(fileHash string) (*model.TranscriptionFull, error) { return nil, nil }
func (f *fakeDAOV2) GetByProvider(providerType string, limit int) ([]model.TranscriptionFull, error) {
	return nil, nil
}
func (f *fakeDAOV2) UpdateFileMetadata(id int, fileHash string, fileSize int64) error { return nil }
func (f *fakeDAOV2) SoftDelete(id int) error                                           { return nil }
func (f *fakeDAOV2) GetActiveTranscriptions(limit int) ([]model.TranscriptionFull, error) {
	return nil, nil
}
func (f *fakeDAOV2) GetTranscriptionByID(id int) (*model.TranscriptionFull, error) { return nil, nil }

func TestCreateTranscriptionReturnsAssignedID(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "audio.mp3")
	if err := os.WriteFile(filePath, []byte("audio"), 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	dao := &fakeDAOV2{}
	service := &TranscriptionServiceImpl{
		orchestrator: &fakeOrchestrator{},
		repository:   dao,
	}

	resp, err := service.CreateTranscription(context.Background(), &dto.CreateTranscriptionRequest{
		FilePath: filePath,
		UserID:   "user-1",
	})
	if err != nil {
		t.Fatalf("CreateTranscription returned error: %v", err)
	}

	if resp.ID != 42 {
		t.Fatalf("expected response ID 42, got %d", resp.ID)
	}
	if len(dao.recordCalls) == 0 || dao.recordCalls[0].ID != 42 {
		t.Fatalf("expected DAO to assign ID before response, calls: %+v", dao.recordCalls)
	}
}
