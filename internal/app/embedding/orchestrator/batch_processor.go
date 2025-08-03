package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tiktok-whisper/internal/app/storage/vector"
)

// BatchProcessor handles batch processing of transcriptions
// Following Single Responsibility Principle - only handles batch processing
type BatchProcessor struct {
	orchestrator EmbeddingOrchestratorInterface
	storage      vector.VectorStorage
	logger       Logger

	// Configuration
	batchSize   int
	concurrency int

	// State management
	isProcessing bool
	isPaused     bool
	currentBatch int
	totalBatches int

	// Control channels
	stopChan   chan struct{}
	pauseChan  chan struct{}
	resumeChan chan struct{}

	// Mutex for thread safety
	mu sync.RWMutex
}

// EmbeddingOrchestratorInterface for dependency injection
type EmbeddingOrchestratorInterface interface {
	ProcessTranscription(ctx context.Context, transcriptionID int, text string) error
	GetEmbeddingStatus(ctx context.Context, transcriptionID int) (*EmbeddingStatus, error)
}

// BatchResult represents the result of batch processing
type BatchResult struct {
	Processed int
	Failed    int
	Errors    []error
}

// ProcessingStatus represents the current status of batch processing
type ProcessingStatus struct {
	IsProcessing bool
	IsPaused     bool
	CurrentBatch int
	TotalBatches int
	Progress     float64
	StartTime    time.Time
	ETA          time.Time
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(
	orchestrator EmbeddingOrchestratorInterface,
	storage vector.VectorStorage,
	logger Logger,
) *BatchProcessor {
	return &BatchProcessor{
		orchestrator: orchestrator,
		storage:      storage,
		logger:       logger,
		batchSize:    10,
		concurrency:  5,
		stopChan:     make(chan struct{}, 1),
		pauseChan:    make(chan struct{}, 1),
		resumeChan:   make(chan struct{}, 1),
	}
}

// ProcessBatch processes a batch of transcriptions
func (p *BatchProcessor) ProcessBatch(ctx context.Context, transcriptions []*vector.Transcription, batchSize int) (*BatchResult, error) {
	p.mu.Lock()
	p.isProcessing = true
	p.totalBatches = (len(transcriptions) + batchSize - 1) / batchSize
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.isProcessing = false
		p.mu.Unlock()
	}()

	result := &BatchResult{}

	// Process in batches
	for i := 0; i < len(transcriptions); i += batchSize {
		// Check for stop signal
		select {
		case <-p.stopChan:
			return result, nil
		default:
		}

		// Check for pause signal
		select {
		case <-p.pauseChan:
			<-p.resumeChan // Wait for resume signal
		default:
		}

		end := min(i+batchSize, len(transcriptions))
		batch := transcriptions[i:end]

		p.mu.Lock()
		p.currentBatch = i/batchSize + 1
		p.mu.Unlock()

		// Process batch concurrently
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, t := range batch {
			wg.Add(1)
			go func(transcription *vector.Transcription) {
				defer wg.Done()

				err := p.orchestrator.ProcessTranscription(ctx, transcription.ID, transcription.TranscriptionText)

				mu.Lock()
				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, err)
				} else {
					result.Processed++
				}
				mu.Unlock()
			}(t)
		}

		wg.Wait()

		// Progress logging
		progress := float64(i+len(batch)) / float64(len(transcriptions)) * 100
		p.logger.Info("Batch processing progress",
			"progress", progress,
			"processed", result.Processed,
			"failed", result.Failed)
	}

	return result, nil
}

// ProcessAllTranscriptions processes all transcriptions without embeddings for specified providers
func (p *BatchProcessor) ProcessAllTranscriptions(ctx context.Context, providers []string, batchSize int) error {
	var allTranscriptions []*vector.Transcription
	processedIDs := make(map[int]bool)

	// Get transcriptions without embeddings for each provider
	for _, provider := range providers {
		// Use a large limit to get all available records
		transcriptions, err := p.storage.GetTranscriptionsWithoutEmbeddings(ctx, provider, 10000)
		if err != nil {
			return err
		}

		// Add unique transcriptions to the processing list
		for _, t := range transcriptions {
			if !processedIDs[t.ID] {
				allTranscriptions = append(allTranscriptions, t)
				processedIDs[t.ID] = true
			}
		}
	}

	if len(allTranscriptions) == 0 {
		p.logger.Info("No transcriptions to process")
		return nil
	}

	p.logger.Info("Starting batch processing",
		"totalTranscriptions", len(allTranscriptions),
		"providers", providers,
		"batchSize", batchSize)

	_, err := p.ProcessBatch(ctx, allTranscriptions, batchSize)
	return err
}

// ProcessUserTranscriptions processes transcriptions for a specific user without embeddings for specified providers
func (p *BatchProcessor) ProcessUserTranscriptions(ctx context.Context, userNickname string, providers []string, batchSize int) error {
	var allTranscriptions []*vector.Transcription
	processedIDs := make(map[int]bool)

	// Get user-specific transcriptions without embeddings for each provider
	for _, provider := range providers {
		// Use a large limit to get all available records for this user
		transcriptions, err := p.storage.GetTranscriptionsWithoutEmbeddingsByUser(ctx, provider, userNickname, 10000)
		if err != nil {
			return err
		}

		// Add unique transcriptions to the processing list
		for _, t := range transcriptions {
			if !processedIDs[t.ID] {
				allTranscriptions = append(allTranscriptions, t)
				processedIDs[t.ID] = true
			}
		}
	}

	if len(allTranscriptions) == 0 {
		p.logger.Info("No transcriptions to process for user",
			"user", userNickname)
		return nil
	}

	p.logger.Info("Starting user-specific batch processing",
		"user", userNickname,
		"totalTranscriptions", len(allTranscriptions),
		"providers", providers,
		"batchSize", batchSize)

	_, err := p.ProcessBatch(ctx, allTranscriptions, batchSize)
	if err != nil {
		return fmt.Errorf("failed to process user transcriptions: %w", err)
	}

	p.logger.Info("Completed user-specific batch processing",
		"user", userNickname,
		"totalTranscriptions", len(allTranscriptions))

	return nil
}

// GetProcessingStatus returns the current processing status
func (p *BatchProcessor) GetProcessingStatus(ctx context.Context) (*ProcessingStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	progress := float64(0)
	if p.totalBatches > 0 {
		progress = float64(p.currentBatch) / float64(p.totalBatches) * 100
	}

	return &ProcessingStatus{
		IsProcessing: p.isProcessing,
		IsPaused:     p.isPaused,
		CurrentBatch: p.currentBatch,
		TotalBatches: p.totalBatches,
		Progress:     progress,
	}, nil
}

// PauseProcessing pauses the current processing
func (p *BatchProcessor) PauseProcessing() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isProcessing && !p.isPaused {
		p.isPaused = true
		select {
		case p.pauseChan <- struct{}{}:
		default:
		}
	}
	return nil
}

// ResumeProcessing resumes paused processing
func (p *BatchProcessor) ResumeProcessing() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isProcessing && p.isPaused {
		p.isPaused = false
		select {
		case p.resumeChan <- struct{}{}:
		default:
		}
	}
	return nil
}

// StopProcessing stops the current processing
func (p *BatchProcessor) StopProcessing() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isProcessing {
		select {
		case p.stopChan <- struct{}{}:
		default:
		}
	}
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
