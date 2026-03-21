package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/app/model"
)

// ProcessWhisperJobWithProvider processes a whisper job using the whisper_server provider
func ProcessWhisperJobWithProvider(job *model.WhisperJob) error {
	// Download the file from the URL
	tempFile, err := downloadFile(job.FileURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer os.Remove(tempFile)

	// Call whisper server API
	transcription, err := callWhisperServer(tempFile, job.Language)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	// Update job with result
	job.TranscriptionText = transcription
	job.Status = string(dto.JobStatusCompleted)
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	job.UpdatedAt = completedAt

	return nil
}

// downloadFile downloads a file from URL to a temporary location
func downloadFile(url string) (string, error) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "whisper-*.wav")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	defer tempFile.Close()

	// Download file
	resp, err := http.Get(url)
	if err != nil {
		os.Remove(tempPath)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to download: status %d", resp.StatusCode)
	}

	// Copy to temp file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempPath)
		return "", err
	}

	return tempPath, nil
}

// callWhisperServer calls the whisper.cpp server API
func callWhisperServer(filePath string, language string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file field
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}

	// Add other fields
	writer.WriteField("response_format", "json")
	if language != "" && language != "auto" {
		writer.WriteField("language", language)
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	// Make request to whisper server
	// Use host.docker.internal to access host machine from container
	req, err := http.NewRequest("POST", "http://host.docker.internal:8080/inference", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper server error: %s", string(body))
	}

	// Parse JSON response to extract text
	// For simplicity, we'll just look for the "text" field
	// In production, you'd properly unmarshal the JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil // Return raw response if JSON parsing fails
	}

	if text, ok := result["text"].(string); ok {
		return text, nil
	}

	return string(body), nil
}