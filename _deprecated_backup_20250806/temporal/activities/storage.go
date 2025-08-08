package activities

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.temporal.io/sdk/activity"
)

// StorageActivities provides MinIO storage operations
type StorageActivities struct {
	client     *minio.Client
	bucket     string
	tempDir    string
}

// NewStorageActivities creates a new instance of storage activities
func NewStorageActivities(endpoint, accessKey, secretKey, bucket string) (*StorageActivities, error) {
	// Initialize MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // Set to true for HTTPS
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Create temp directory for downloads
	tempDir := "/tmp/v2t-temporal"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &StorageActivities{
		client:  client,
		bucket:  bucket,
		tempDir: tempDir,
	}, nil
}

// FileUploadRequest represents a file upload request
type FileUploadRequest struct {
	LocalPath  string            `json:"local_path"`
	ObjectKey  string            `json:"object_key"`
	Metadata   map[string]string `json:"metadata"`
}

// FileUploadResult represents the result of a file upload
type FileUploadResult struct {
	ObjectKey string `json:"object_key"`
	ETag      string `json:"etag"`
	Size      int64  `json:"size"`
	URL       string `json:"url"`
}

// UploadFile uploads a file to MinIO
func (s *StorageActivities) UploadFile(ctx context.Context, req FileUploadRequest) (FileUploadResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Uploading file to MinIO", "localPath", req.LocalPath, "objectKey", req.ObjectKey)

	// Get file info
	fileInfo, err := os.Stat(req.LocalPath)
	if err != nil {
		return FileUploadResult{}, fmt.Errorf("failed to stat file: %w", err)
	}

	// Upload file with progress
	uploadInfo, err := s.client.FPutObject(ctx, s.bucket, req.ObjectKey, req.LocalPath, minio.PutObjectOptions{
		UserMetadata: req.Metadata,
		Progress:     s.createProgressReader(ctx, fileInfo.Size()),
	})
	if err != nil {
		return FileUploadResult{}, fmt.Errorf("failed to upload file: %w", err)
	}

	result := FileUploadResult{
		ObjectKey: req.ObjectKey,
		ETag:      uploadInfo.ETag,
		Size:      uploadInfo.Size,
		URL:       fmt.Sprintf("minio://%s/%s", s.bucket, req.ObjectKey),
	}

	logger.Info("File uploaded successfully", "objectKey", req.ObjectKey, "size", result.Size)
	return result, nil
}

// FileDownloadRequest represents a file download request
type FileDownloadRequest struct {
	ObjectKey string `json:"object_key"`
	LocalPath string `json:"local_path"` // Optional, will generate if empty
}

// FileDownloadResult represents the result of a file download
type FileDownloadResult struct {
	LocalPath string `json:"local_path"`
	Size      int64  `json:"size"`
}

// DownloadFile downloads a file from MinIO
func (s *StorageActivities) DownloadFile(ctx context.Context, req FileDownloadRequest) (FileDownloadResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Downloading file from MinIO", "objectKey", req.ObjectKey)

	// Generate local path if not provided
	localPath := req.LocalPath
	if localPath == "" {
		localPath = filepath.Join(s.tempDir, fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(req.ObjectKey)))
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return FileDownloadResult{}, fmt.Errorf("failed to create directory: %w", err)
	}

	// Get object info for progress tracking
	objInfo, err := s.client.StatObject(ctx, s.bucket, req.ObjectKey, minio.StatObjectOptions{})
	if err != nil {
		return FileDownloadResult{}, fmt.Errorf("failed to stat object: %w", err)
	}

	// Download file with progress
	err = s.client.FGetObject(ctx, s.bucket, req.ObjectKey, localPath, minio.GetObjectOptions{
		Progress: s.createProgressReader(ctx, objInfo.Size),
	})
	if err != nil {
		return FileDownloadResult{}, fmt.Errorf("failed to download file: %w", err)
	}

	result := FileDownloadResult{
		LocalPath: localPath,
		Size:      objInfo.Size,
	}

	logger.Info("File downloaded successfully", "localPath", localPath, "size", result.Size)
	return result, nil
}

// CleanupTempFile removes a temporary file
func (s *StorageActivities) CleanupTempFile(ctx context.Context, filePath string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cleaning up temp file", "path", filePath)

	// Only allow cleanup within temp directory for safety
	if !strings.HasPrefix(filePath, s.tempDir) {
		return fmt.Errorf("cannot cleanup file outside temp directory: %s", filePath)
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

// ListFilesRequest represents a request to list files
type ListFilesRequest struct {
	Prefix    string `json:"prefix"`
	Recursive bool   `json:"recursive"`
	MaxKeys   int    `json:"max_keys"`
}

// ListFiles lists files in MinIO bucket
func (s *StorageActivities) ListFiles(ctx context.Context, req ListFilesRequest) ([]string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Listing files in MinIO", "prefix", req.Prefix, "recursive", req.Recursive)

	opts := minio.ListObjectsOptions{
		Prefix:    req.Prefix,
		Recursive: req.Recursive,
		MaxKeys:   req.MaxKeys,
	}

	var files []string
	for object := range s.client.ListObjects(ctx, s.bucket, opts) {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		files = append(files, object.Key)
	}

	logger.Info("Listed files", "count", len(files))
	return files, nil
}

// EnsureBucketExists creates the bucket if it doesn't exist
func (s *StorageActivities) EnsureBucketExists(ctx context.Context) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Ensuring bucket exists", "bucket", s.bucket)

	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info("Bucket created", "bucket", s.bucket)
	}

	return nil
}

// createProgressReader creates a progress reader for file operations
func (s *StorageActivities) createProgressReader(ctx context.Context, totalSize int64) io.Reader {
	pr := &progressReader{
		ctx:       ctx,
		totalSize: totalSize,
		lastHeart: time.Now(),
	}
	return pr
}

// progressReader implements io.Reader with activity heartbeat
type progressReader struct {
	ctx       context.Context
	totalSize int64
	readBytes int64
	lastHeart time.Time
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	pr.readBytes += int64(len(p))
	
	// Send heartbeat every 5 seconds
	if time.Since(pr.lastHeart) > 5*time.Second {
		percent := float64(pr.readBytes) / float64(pr.totalSize) * 100
		activity.RecordHeartbeat(pr.ctx, fmt.Sprintf("Progress: %.1f%%", percent))
		pr.lastHeart = time.Now()
	}
	
	return len(p), nil
}