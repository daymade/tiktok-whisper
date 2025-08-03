package provider

import (
	"context"
	"fmt"
	"time"
)

// TranscriberAdapter adapts the Provider interface to the legacy Transcriber interface
type TranscriberAdapter struct {
	orchestrator TranscriptionOrchestrator
}

// NewTranscriberAdapter creates a new adapter that bridges Provider to Transcriber
func NewTranscriberAdapter(orchestrator TranscriptionOrchestrator) *TranscriberAdapter {
	return &TranscriberAdapter{
		orchestrator: orchestrator,
	}
}

// Transcript implements the Transcriber interface using the provider framework
func (a *TranscriberAdapter) Transcript(inputFilePath string) (string, error) {
	// Create a context with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create transcription request
	request := &TranscriptionRequest{
		InputFilePath: inputFilePath,
	}

	// Execute transcription through the orchestrator
	response, err := a.orchestrator.Transcribe(ctx, request)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}

	// Return the transcription text
	return response.Text, nil
}