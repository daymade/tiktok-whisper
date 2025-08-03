package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/api/v1/dto"
)

// MockServices contains all mock services for testing
type MockServices struct {
	TranscriptionService *MockTranscriptionService
	ProviderService      *MockProviderService
	DownloadService      *MockDownloadService
	EmbeddingService     *MockEmbeddingService
	ExportService        *MockExportService
	ConfigService        *MockConfigService
}

// NewMockServices creates a new instance of mock services
func NewMockServices(t *testing.T) *MockServices {
	return &MockServices{
		TranscriptionService: NewMockTranscriptionService(t),
		ProviderService:      NewMockProviderService(t),
		DownloadService:      NewMockDownloadService(t),
		EmbeddingService:     NewMockEmbeddingService(t),
		ExportService:        NewMockExportService(t),
		ConfigService:        NewMockConfigService(t),
	}
}

// MockTranscriptionService is a mock implementation of TranscriptionService
type MockTranscriptionService struct {
	mock.Mock
}

func NewMockTranscriptionService(t *testing.T) *MockTranscriptionService {
	m := &MockTranscriptionService{}
	m.Test(t)
	return m
}

func (m *MockTranscriptionService) CreateTranscription(ctx context.Context, req *dto.CreateTranscriptionRequest) (*dto.TranscriptionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TranscriptionResponse), args.Error(1)
}

func (m *MockTranscriptionService) GetTranscription(ctx context.Context, id int) (*dto.TranscriptionResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TranscriptionResponse), args.Error(1)
}

func (m *MockTranscriptionService) ListTranscriptions(ctx context.Context, query dto.ListTranscriptionsQuery) (*dto.PaginatedTranscriptionsResponse, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PaginatedTranscriptionsResponse), args.Error(1)
}

func (m *MockTranscriptionService) DeleteTranscription(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockProviderService is a mock implementation of ProviderService
type MockProviderService struct {
	mock.Mock
}

func NewMockProviderService(t *testing.T) *MockProviderService {
	m := &MockProviderService{}
	m.Test(t)
	return m
}

func (m *MockProviderService) ListProviders(ctx context.Context) ([]dto.ProviderResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.ProviderResponse), args.Error(1)
}

func (m *MockProviderService) GetProvider(ctx context.Context, id string) (*dto.ProviderResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProviderResponse), args.Error(1)
}

func (m *MockProviderService) GetProviderStatus(ctx context.Context, id string) (*dto.ProviderStatusResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProviderStatusResponse), args.Error(1)
}

func (m *MockProviderService) GetProviderStats(ctx context.Context, id string) (*dto.ProviderStatsResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProviderStatsResponse), args.Error(1)
}

func (m *MockProviderService) TestProvider(ctx context.Context, id string, req *dto.TestProviderRequest) (*dto.TestProviderResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TestProviderResponse), args.Error(1)
}

// MockDownloadService is a mock implementation of DownloadService
type MockDownloadService struct {
	mock.Mock
}

func NewMockDownloadService(t *testing.T) *MockDownloadService {
	m := &MockDownloadService{}
	m.Test(t)
	return m
}

// MockEmbeddingService is a mock implementation of EmbeddingService
type MockEmbeddingService struct {
	mock.Mock
}

func NewMockEmbeddingService(t *testing.T) *MockEmbeddingService {
	m := &MockEmbeddingService{}
	m.Test(t)
	return m
}

// MockExportService is a mock implementation of ExportService
type MockExportService struct {
	mock.Mock
}

func NewMockExportService(t *testing.T) *MockExportService {
	m := &MockExportService{}
	m.Test(t)
	return m
}

// MockConfigService is a mock implementation of ConfigService
type MockConfigService struct {
	mock.Mock
}

func NewMockConfigService(t *testing.T) *MockConfigService {
	m := &MockConfigService{}
	m.Test(t)
	return m
}