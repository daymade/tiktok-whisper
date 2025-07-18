package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"tiktok-whisper/internal/app/embedding/provider"
	"tiktok-whisper/internal/app/storage/vector"
)

// Logger interface for dependency injection
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// EmbeddingOrchestrator orchestrates embedding generation across multiple providers
// Following Single Responsibility Principle - only coordinates embedding generation
type EmbeddingOrchestrator struct {
	providers map[string]provider.EmbeddingProvider
	storage   vector.VectorStorage
	logger    Logger
}

// EmbeddingStatus represents the status of embeddings for a transcription
type EmbeddingStatus struct {
	TranscriptionID  int
	OpenAICompleted  bool
	GeminiCompleted  bool
	OpenAIError      string
	GeminiError      string
}

// NewEmbeddingOrchestrator creates a new embedding orchestrator
func NewEmbeddingOrchestrator(
	providers map[string]provider.EmbeddingProvider,
	storage vector.VectorStorage,
	logger Logger,
) *EmbeddingOrchestrator {
	return &EmbeddingOrchestrator{
		providers: providers,
		storage:   storage,
		logger:    logger,
	}
}

// ProcessTranscription processes a single transcription with all available providers
func (o *EmbeddingOrchestrator) ProcessTranscription(ctx context.Context, transcriptionID int, text string) error {
	// Handle dual embedding case
	if len(o.providers) == 2 {
		return o.processDualEmbeddings(ctx, transcriptionID, text)
	}
	
	// Handle single provider case
	return o.processSingleProvider(ctx, transcriptionID, text)
}

// processDualEmbeddings handles both OpenAI and Gemini providers
func (o *EmbeddingOrchestrator) processDualEmbeddings(ctx context.Context, transcriptionID int, text string) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(o.providers))
	
	var openaiEmbedding, geminiEmbedding []float32
	var openaiErr, geminiErr error
	
	// Process OpenAI
	if openaiProvider, exists := o.providers["openai"]; exists {
		wg.Add(1)
		go func() {
			defer wg.Done()
			openaiEmbedding, openaiErr = openaiProvider.GenerateEmbedding(ctx, text)
			if openaiErr != nil {
				o.logger.Error("Failed to generate OpenAI embedding", 
					"transcriptionID", transcriptionID, "error", openaiErr)
				errors <- openaiErr
			}
		}()
	}
	
	// Process Gemini
	if geminiProvider, exists := o.providers["gemini"]; exists {
		wg.Add(1)
		go func() {
			defer wg.Done()
			geminiEmbedding, geminiErr = geminiProvider.GenerateEmbedding(ctx, text)
			if geminiErr != nil {
				o.logger.Error("Failed to generate Gemini embedding", 
					"transcriptionID", transcriptionID, "error", geminiErr)
				errors <- geminiErr
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}
	
	if len(errorList) > 0 {
		return fmt.Errorf("embedding generation failed: %v", errorList)
	}
	
	// Store dual embeddings
	if openaiEmbedding != nil && geminiEmbedding != nil {
		err := o.storage.StoreDualEmbeddings(ctx, transcriptionID, openaiEmbedding, geminiEmbedding)
		if err != nil {
			return fmt.Errorf("failed to store dual embeddings: %w", err)
		}
		
		o.logger.Info("Successfully processed dual embeddings", 
			"transcriptionID", transcriptionID)
	}
	
	return nil
}

// processSingleProvider handles single provider case
func (o *EmbeddingOrchestrator) processSingleProvider(ctx context.Context, transcriptionID int, text string) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(o.providers))
	
	for providerName, prov := range o.providers {
		wg.Add(1)
		go func(name string, p provider.EmbeddingProvider) {
			defer wg.Done()
			
			embedding, err := p.GenerateEmbedding(ctx, text)
			if err != nil {
				o.logger.Error("Failed to generate embedding", 
					"provider", name, "transcriptionID", transcriptionID, "error", err)
				errors <- err
				return
			}
			
			err = o.storage.StoreEmbedding(ctx, transcriptionID, name, embedding)
			if err != nil {
				o.logger.Error("Failed to store embedding", 
					"provider", name, "transcriptionID", transcriptionID, "error", err)
				errors <- err
				return
			}
			
			o.logger.Info("Successfully processed embedding", 
				"provider", name, "transcriptionID", transcriptionID)
		}(providerName, prov)
	}
	
	wg.Wait()
	close(errors)
	
	// Check if any errors occurred
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}
	
	if len(errorList) > 0 {
		return fmt.Errorf("embedding generation failed: %v", errorList)
	}
	
	return nil
}

// GetEmbeddingStatus returns the status of embeddings for a transcription
func (o *EmbeddingOrchestrator) GetEmbeddingStatus(ctx context.Context, transcriptionID int) (*EmbeddingStatus, error) {
	status := &EmbeddingStatus{
		TranscriptionID: transcriptionID,
	}
	
	// Check if we have dual embeddings
	dualEmbedding, err := o.storage.GetDualEmbeddings(ctx, transcriptionID)
	if err == nil && dualEmbedding != nil {
		status.OpenAICompleted = dualEmbedding.OpenAI != nil
		status.GeminiCompleted = dualEmbedding.Gemini != nil
	}
	
	return status, nil
}