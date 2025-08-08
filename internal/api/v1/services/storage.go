package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// StorageService handles file storage operations
type StorageService interface {
	UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (*FileUploadResult, error)
	GeneratePresignedURL(ctx context.Context, operation string, userID string, filename string) (*PresignedURLResult, error)
	GetFileURL(key string) string
	DeleteFile(ctx context.Context, key string) error
}

// FileUploadResult contains the result of a file upload
type FileUploadResult struct {
	URL           string    `json:"url"`
	Key           string    `json:"key"`
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	AudioDuration float64   `json:"audioDuration,omitempty"`
	UploadedAt    time.Time `json:"uploadedAt"`
}

// PresignedURLResult contains a presigned URL for uploads
type PresignedURLResult struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expiresAt"`
	Key       string            `json:"key"`
}

// MinioStorageService implements StorageService using MinIO
type MinioStorageService struct {
	client   *minio.Client
	bucket   string
	endpoint string
	useSSL   bool
}

// NewMinioStorageService creates a new MinIO storage service
func NewMinioStorageService() (StorageService, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "v2t-transcriptions"
	}

	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	// Initialize MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	service := &MinioStorageService{
		client:   client,
		bucket:   bucket,
		endpoint: endpoint,
		useSSL:   useSSL,
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return service, nil
}

// UploadFile uploads a file to MinIO storage
func (s *MinioStorageService) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (*FileUploadResult, error) {
	// Generate unique key for the file
	timestamp := time.Now().Unix()
	fileID := uuid.New().String()[:8]
	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("whisper/%s/%d-%s%s", userID, timestamp, fileID, ext)

	// Read file content
	buf := bytes.NewBuffer(nil)
	size, err := io.Copy(buf, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Upload to MinIO
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.client.PutObject(ctx, s.bucket, key, buf, size, minio.PutObjectOptions{
		ContentType: contentType,
		UserMetadata: map[string]string{
			"original-name": header.Filename,
			"user-id":       userID,
			"uploaded-at":   time.Now().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to MinIO: %w", err)
	}

	// Generate URL for the uploaded file
	fileURL := s.GetFileURL(key)

	return &FileUploadResult{
		URL:        fileURL,
		Key:        key,
		Name:       header.Filename,
		Size:       size,
		UploadedAt: time.Now(),
	}, nil
}

// GeneratePresignedURL generates a presigned URL for direct uploads
func (s *MinioStorageService) GeneratePresignedURL(ctx context.Context, operation string, userID string, filename string) (*PresignedURLResult, error) {
	// Generate key for the file
	timestamp := time.Now().Unix()
	fileID := uuid.New().String()[:8]
	ext := filepath.Ext(filename)
	key := fmt.Sprintf("whisper/%s/%d-%s%s", userID, timestamp, fileID, ext)

	// Set expiration time (1 hour)
	expiration := time.Hour

	var presignedURL *url.URL
	var err error

	switch operation {
	case "PUT", "upload":
		presignedURL, err = s.client.PresignedPutObject(ctx, s.bucket, key, expiration)
	case "GET", "download":
		presignedURL, err = s.client.PresignedGetObject(ctx, s.bucket, key, url.Values{}, expiration)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return &PresignedURLResult{
		URL:       presignedURL.String(),
		Method:    strings.ToUpper(operation),
		ExpiresAt: time.Now().Add(expiration),
		Key:       key,
	}, nil
}

// GetFileURL returns the URL for accessing a file
func (s *MinioStorageService) GetFileURL(key string) string {
	protocol := "http"
	if s.useSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", protocol, s.endpoint, s.bucket, key)
}

// DeleteFile deletes a file from storage
func (s *MinioStorageService) DeleteFile(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// MockStorageService implements StorageService with mock responses (for testing)
type MockStorageService struct{}

// NewMockStorageService creates a mock storage service
func NewMockStorageService() StorageService {
	return &MockStorageService{}
}

func (s *MockStorageService) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (*FileUploadResult, error) {
	timestamp := time.Now().Unix()
	fileID := uuid.New().String()[:8]
	key := fmt.Sprintf("whisper/%s/%d-%s", userID, timestamp, header.Filename)

	return &FileUploadResult{
		URL:        fmt.Sprintf("/storage/%s", key),
		Key:        key,
		Name:       header.Filename,
		Size:       header.Size,
		UploadedAt: time.Now(),
	}, nil
}

func (s *MockStorageService) GeneratePresignedURL(ctx context.Context, operation string, userID string, filename string) (*PresignedURLResult, error) {
	timestamp := time.Now().Unix()
	fileID := uuid.New().String()[:8]
	key := fmt.Sprintf("whisper/%s/%d-%s", userID, timestamp, filename)

	return &PresignedURLResult{
		URL:       fmt.Sprintf("https://mock-storage.example.com/presigned/%s", key),
		Method:    strings.ToUpper(operation),
		ExpiresAt: time.Now().Add(time.Hour),
		Key:       key,
	}, nil
}

func (s *MockStorageService) GetFileURL(key string) string {
	return fmt.Sprintf("/storage/%s", key)
}

func (s *MockStorageService) DeleteFile(ctx context.Context, key string) error {
	return nil
}