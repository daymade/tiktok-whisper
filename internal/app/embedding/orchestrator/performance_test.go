package orchestrator

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/app/embedding/provider"
	"tiktok-whisper/internal/app/storage/vector"
	"tiktok-whisper/internal/app/testutil"
)

// =============================================================================
// PERFORMANCE AND CONCURRENCY TESTS
// =============================================================================

// TestEmbeddingOrchestrator_ConcurrencyPerformance tests parallel processing performance
func TestEmbeddingOrchestrator_ConcurrencyPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockGemini := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Track concurrent execution metrics
	var (
		maxConcurrent     int64
		currentConcurrent int64
		totalCalls        int64
	)

	processingDelay := 100 * time.Millisecond

	// Setup mocks with concurrency tracking
	mockOpenAI.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(func(ctx context.Context, text string) ([]float32, error) {
		current := atomic.AddInt64(&currentConcurrent, 1)
		atomic.AddInt64(&totalCalls, 1)

		// Update max concurrent if necessary
		for {
			max := atomic.LoadInt64(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt64(&maxConcurrent, max, current) {
				break
			}
		}

		time.Sleep(processingDelay)
		atomic.AddInt64(&currentConcurrent, -1)
		return make([]float32, 1536), nil
	})

	mockGemini.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(func(ctx context.Context, text string) ([]float32, error) {
		current := atomic.AddInt64(&currentConcurrent, 1)
		atomic.AddInt64(&totalCalls, 1)

		// Update max concurrent if necessary
		for {
			max := atomic.LoadInt64(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt64(&maxConcurrent, max, current) {
				break
			}
		}

		time.Sleep(processingDelay)
		atomic.AddInt64(&currentConcurrent, -1)
		return make([]float32, 768), nil
	})

	mockStorage.On("StoreDualEmbeddings", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockLogger.SetEnabled(false) // Disable for performance testing

	// Test sequential vs concurrent processing
	const numRequests = 10

	// Sequential processing (for comparison)
	start := time.Now()
	for i := 0; i < numRequests; i++ {
		err := orchestrator.ProcessTranscription(context.Background(), i+1, fmt.Sprintf("sequential text %d", i))
		assert.NoError(t, err)
	}
	sequentialDuration := time.Since(start)

	// Wait for all goroutines to complete
	for atomic.LoadInt64(&currentConcurrent) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// Reset counters
	atomic.StoreInt64(&maxConcurrent, 0)
	atomic.StoreInt64(&totalCalls, 0)

	// Concurrent processing
	start = time.Now()
	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := orchestrator.ProcessTranscription(context.Background(), id+100, fmt.Sprintf("concurrent text %d", id))
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
	concurrentDuration := time.Since(start)

	// Wait for all goroutines to complete
	for atomic.LoadInt64(&currentConcurrent) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// Performance assertions
	maxConcurrentReached := atomic.LoadInt64(&maxConcurrent)
	totalCallsMade := atomic.LoadInt64(&totalCalls)

	t.Logf("Sequential duration: %v", sequentialDuration)
	t.Logf("Concurrent duration: %v", concurrentDuration)
	t.Logf("Max concurrent calls: %d", maxConcurrentReached)
	t.Logf("Total calls made: %d", totalCallsMade)

	// Concurrent processing should be significantly faster
	assert.Less(t, concurrentDuration, sequentialDuration*8/10,
		"Concurrent processing should be at least 20%% faster")

	// Should achieve reasonable concurrency
	assert.Greater(t, maxConcurrentReached, int64(4),
		"Should achieve reasonable concurrency (expected > 4)")

	// Should process all requests
	assert.Equal(t, int64(numRequests*2), totalCallsMade,
		"Should have made calls for all requests to both providers")

	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// TestBatchProcessor_ThroughputPerformance tests batch processing throughput
func TestBatchProcessor_ThroughputPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	// Test different batch sizes for throughput optimization
	batchSizes := []int{1, 5, 10, 20, 50}
	const totalItems = 100

	for _, batchSize := range batchSizes {
		t.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(t *testing.T) {
			// Arrange
			mockOrchestrator := new(MockEmbeddingOrchestrator)
			mockStorage := new(MockVectorStorage)
			mockLogger := testutil.NewMockLogger()

			processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

			// Create transcriptions
			transcriptions := make([]*vector.Transcription, totalItems)
			for i := 0; i < totalItems; i++ {
				transcriptions[i] = &vector.Transcription{
					ID:                i + 1,
					TranscriptionText: fmt.Sprintf("Throughput test %d", i+1),
					User:              "throughput_user",
				}
			}

			// Track processing metrics
			var processedCount int64
			var totalDuration time.Duration
			var mu sync.Mutex

			// Setup mocks with timing
			for i := 0; i < totalItems; i++ {
				mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(func(ctx context.Context, id int, text string) error {
					start := time.Now()

					// Simulate processing time
					time.Sleep(10 * time.Millisecond)

					duration := time.Since(start)

					mu.Lock()
					processedCount++
					totalDuration += duration
					mu.Unlock()

					return nil
				})
			}
			mockLogger.SetEnabled(false)

			// Act
			start := time.Now()
			result, err := processor.ProcessBatch(context.Background(), transcriptions, batchSize)
			wallClockTime := time.Since(start)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, totalItems, result.Processed)
			assert.Equal(t, 0, result.Failed)

			mu.Lock()
			avgProcessingTime := totalDuration / time.Duration(processedCount)
			mu.Unlock()

			t.Logf("Batch size: %d, Wall clock: %v, Avg processing: %v, Throughput: %.2f items/sec",
				batchSize, wallClockTime, avgProcessingTime,
				float64(totalItems)/wallClockTime.Seconds())

			// Performance expectations
			assert.Less(t, wallClockTime, 5*time.Second,
				"Should complete within reasonable time")
			assert.Equal(t, int64(totalItems), processedCount,
				"Should have processed all items")

			mockOrchestrator.AssertExpectations(t)
		})
	}
}

// TestEmbeddingOrchestrator_MemoryUsage tests memory usage under load
func TestEmbeddingOrchestrator_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Force garbage collection and get baseline
	runtime.GC()
	var baselineMemStats runtime.MemStats
	runtime.ReadMemStats(&baselineMemStats)

	// Arrange
	mockOpenAI := new(MockEmbeddingProvider)
	mockGemini := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	// Create large embeddings to test memory handling
	largeOpenAIEmbedding := make([]float32, 10000)
	largeGeminiEmbedding := make([]float32, 10000)

	for i := range largeOpenAIEmbedding {
		largeOpenAIEmbedding[i] = float32(i) * 0.001
	}
	for i := range largeGeminiEmbedding {
		largeGeminiEmbedding[i] = float32(i) * 0.002
	}

	// Setup mocks
	mockOpenAI.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(largeOpenAIEmbedding, nil)
	mockGemini.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(largeGeminiEmbedding, nil)
	mockStorage.On("StoreDualEmbeddings", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockLogger.SetEnabled(false)

	// Process multiple transcriptions to test memory accumulation
	const numTranscriptions = 50
	for i := 0; i < numTranscriptions; i++ {
		err := orchestrator.ProcessTranscription(context.Background(), i+1, fmt.Sprintf("memory test %d", i))
		assert.NoError(t, err)

		// Periodically check memory usage
		if i%10 == 0 {
			runtime.GC()
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			memoryIncrease := memStats.Alloc - baselineMemStats.Alloc
			t.Logf("After %d transcriptions: Memory increase: %d bytes", i+1, memoryIncrease)
		}
	}

	// Final memory check
	runtime.GC()
	var finalMemStats runtime.MemStats
	runtime.ReadMemStats(&finalMemStats)

	totalMemoryIncrease := finalMemStats.Alloc - baselineMemStats.Alloc
	memoryPerTranscription := totalMemoryIncrease / numTranscriptions

	t.Logf("Total memory increase: %d bytes", totalMemoryIncrease)
	t.Logf("Memory per transcription: %d bytes", memoryPerTranscription)

	// Memory usage should be reasonable (allowing for test overhead)
	assert.Less(t, memoryPerTranscription, uint64(1024*1024), // 1MB per transcription
		"Memory usage per transcription should be reasonable")

	mockOpenAI.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// TestBatchProcessor_ContextCancellationPerformance tests performance of context cancellation
func TestBatchProcessor_ContextCancellationPerformance(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create many transcriptions
	const totalTranscriptions = 100
	transcriptions := make([]*vector.Transcription, totalTranscriptions)
	for i := 0; i < totalTranscriptions; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Cancellation perf test %d", i+1),
			User:              "cancel_perf_user",
		}
	}

	// Track cancellation responsiveness
	var (
		processedBeforeCancel int64
		processedAfterCancel  int64
		cancellationTime      time.Time
	)

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Setup mocks
	for i := 0; i < totalTranscriptions; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(func(ctx context.Context, id int, text string) error {
			// Cancel after processing 10 items
			if atomic.LoadInt64(&processedBeforeCancel) == 10 {
				cancellationTime = time.Now()
				cancel()
			}

			// Track processing before/after cancellation
			if cancellationTime.IsZero() {
				atomic.AddInt64(&processedBeforeCancel, 1)
			} else {
				atomic.AddInt64(&processedAfterCancel, 1)
			}

			// Simulate work that respects cancellation
			select {
			case <-time.After(50 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).Maybe() // Use Maybe() since cancellation will interrupt processing
	}
	mockLogger.SetEnabled(false)

	// Act
	start := time.Now()
	result, err := processor.ProcessBatch(ctx, transcriptions, 5)
	totalDuration := time.Since(start)

	// Assert
	assert.NoError(t, err) // Should handle cancellation gracefully

	processedBefore := atomic.LoadInt64(&processedBeforeCancel)
	processedAfter := atomic.LoadInt64(&processedAfterCancel)

	t.Logf("Processed before cancellation: %d", processedBefore)
	t.Logf("Processed after cancellation: %d", processedAfter)
	t.Logf("Total duration: %v", totalDuration)
	t.Logf("Result: Processed=%d, Failed=%d", result.Processed, result.Failed)

	// Cancellation should be responsive
	assert.Equal(t, int64(10), processedBefore, "Should have processed exactly 10 before cancellation")
	assert.Less(t, processedAfter, int64(20), "Should not have processed too many after cancellation")
	assert.Less(t, totalDuration, 2*time.Second, "Should respond to cancellation quickly")
}

// TestEmbeddingOrchestrator_ScalabilityLimits tests behavior at scale limits
func TestEmbeddingOrchestrator_ScalabilityLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability test in short mode")
	}

	// Test different scales to find performance characteristics
	scales := []int{10, 50, 100, 200}

	for _, scale := range scales {
		t.Run(fmt.Sprintf("Scale_%d", scale), func(t *testing.T) {
			// Arrange
			mockOpenAI := new(MockEmbeddingProvider)
			mockGemini := new(MockEmbeddingProvider)
			mockStorage := new(MockVectorStorage)
			mockLogger := testutil.NewMockLogger()

			providers := map[string]provider.EmbeddingProvider{
				"openai": mockOpenAI,
				"gemini": mockGemini,
			}

			orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

			// Track resource usage
			var (
				totalProcessingTime time.Duration
				maxGoroutines       int
				mu                  sync.Mutex
			)

			// Setup mocks
			for i := 0; i < scale; i++ {
				mockOpenAI.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(func(ctx context.Context, text string) ([]float32, error) {
					start := time.Now()

					// Track goroutine count
					goroutines := runtime.NumGoroutine()
					mu.Lock()
					if goroutines > maxGoroutines {
						maxGoroutines = goroutines
					}
					mu.Unlock()

					time.Sleep(20 * time.Millisecond) // Simulate processing

					mu.Lock()
					totalProcessingTime += time.Since(start)
					mu.Unlock()

					return make([]float32, 1536), nil
				})

				mockGemini.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(func(ctx context.Context, text string) ([]float32, error) {
					start := time.Now()

					time.Sleep(20 * time.Millisecond) // Simulate processing

					mu.Lock()
					totalProcessingTime += time.Since(start)
					mu.Unlock()

					return make([]float32, 768), nil
				})

				mockStorage.On("StoreDualEmbeddings", mock.Anything, i+1, mock.Anything, mock.Anything).Return(nil)
			}
			mockLogger.SetEnabled(false)

			// Act - Process all concurrently
			start := time.Now()
			var wg sync.WaitGroup
			for i := 0; i < scale; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					err := orchestrator.ProcessTranscription(context.Background(), id+1, fmt.Sprintf("scale test %d", id))
					assert.NoError(t, err)
				}(i)
			}
			wg.Wait()
			wallClockTime := time.Since(start)

			// Metrics
			mu.Lock()
			avgProcessingTime := totalProcessingTime / time.Duration(scale*2) // Both providers
			mu.Unlock()

			throughput := float64(scale) / wallClockTime.Seconds()

			t.Logf("Scale: %d, Wall clock: %v, Avg processing: %v, Throughput: %.2f items/sec, Max goroutines: %d",
				scale, wallClockTime, avgProcessingTime, throughput, maxGoroutines)

			// Performance assertions
			assert.Less(t, wallClockTime, 10*time.Second,
				"Should complete within reasonable time even at scale")
			assert.Greater(t, throughput, 5.0,
				"Should maintain reasonable throughput")
			assert.Less(t, maxGoroutines, scale*3,
				"Should not create excessive goroutines")

			mockOpenAI.AssertExpectations(t)
			mockGemini.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
		})
	}
}

// TestBatchProcessor_LoadBalancing tests load distribution in batch processing
func TestBatchProcessor_LoadBalancing(t *testing.T) {
	// Arrange
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	// Create transcriptions with varying processing times
	const numTranscriptions = 20
	transcriptions := make([]*vector.Transcription, numTranscriptions)
	for i := 0; i < numTranscriptions; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Load balance test %d", i+1),
			User:              "load_balance_user",
		}
	}

	// Track processing distribution
	var (
		processingTimes = make(map[int]time.Duration)
		startTimes      = make(map[int]time.Time)
		mu              sync.Mutex
	)

	// Setup mocks with varying processing times
	for i := 0; i < numTranscriptions; i++ {
		processingDelay := time.Duration(10+i*5) * time.Millisecond // Increasing delay
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(func(ctx context.Context, id int, text string) error {
			start := time.Now()
			mu.Lock()
			startTimes[id] = start
			mu.Unlock()

			time.Sleep(processingDelay)

			mu.Lock()
			processingTimes[id] = time.Since(start)
			mu.Unlock()

			return nil
		})
	}
	mockLogger.SetEnabled(false)

	// Act
	start := time.Now()
	result, err := processor.ProcessBatch(context.Background(), transcriptions, 5)
	totalDuration := time.Since(start)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, numTranscriptions, result.Processed)

	// Analyze load distribution
	mu.Lock()
	var totalProcessingTime time.Duration
	var earliestStart, latestEnd time.Time

	for id, startTime := range startTimes {
		if earliestStart.IsZero() || startTime.Before(earliestStart) {
			earliestStart = startTime
		}

		endTime := startTime.Add(processingTimes[id])
		if latestEnd.IsZero() || endTime.After(latestEnd) {
			latestEnd = endTime
		}

		totalProcessingTime += processingTimes[id]
	}
	mu.Unlock()

	actualSpan := latestEnd.Sub(earliestStart)
	theoreticalSequentialTime := totalProcessingTime
	parallelizationEfficiency := float64(theoreticalSequentialTime) / float64(actualSpan*time.Duration(5)) // batch size 5

	t.Logf("Total duration: %v", totalDuration)
	t.Logf("Actual processing span: %v", actualSpan)
	t.Logf("Total processing time: %v", totalProcessingTime)
	t.Logf("Parallelization efficiency: %.2f", parallelizationEfficiency)

	// Load balancing assertions
	assert.Greater(t, parallelizationEfficiency, 0.6,
		"Should achieve reasonable parallelization efficiency")
	assert.Less(t, actualSpan, theoreticalSequentialTime/2,
		"Parallel processing should be significantly faster than sequential")

	mockOrchestrator.AssertExpectations(t)
}

// Benchmark functions for performance regression testing
func BenchmarkEmbeddingOrchestrator_SingleTranscription(b *testing.B) {
	// Setup
	mockOpenAI := new(MockEmbeddingProvider)
	mockGemini := new(MockEmbeddingProvider)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	providers := map[string]provider.EmbeddingProvider{
		"openai": mockOpenAI,
		"gemini": mockGemini,
	}

	orchestrator := NewEmbeddingOrchestrator(providers, mockStorage, mockLogger)

	openaiEmbedding := make([]float32, 1536)
	geminiEmbedding := make([]float32, 768)

	mockOpenAI.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(openaiEmbedding, nil)
	mockGemini.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(geminiEmbedding, nil)
	mockStorage.On("StoreDualEmbeddings", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockLogger.SetEnabled(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := orchestrator.ProcessTranscription(context.Background(), i, "benchmark text")
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkBatchProcessor_SmallBatch(b *testing.B) {
	// Setup
	mockOrchestrator := new(MockEmbeddingOrchestrator)
	mockStorage := new(MockVectorStorage)
	mockLogger := testutil.NewMockLogger()

	processor := NewBatchProcessor(mockOrchestrator, mockStorage, mockLogger)

	transcriptions := make([]*vector.Transcription, 10)
	for i := 0; i < 10; i++ {
		transcriptions[i] = &vector.Transcription{
			ID:                i + 1,
			TranscriptionText: fmt.Sprintf("Benchmark text %d", i+1),
			User:              "benchmark_user",
		}
	}

	for i := 0; i < 10; i++ {
		mockOrchestrator.On("ProcessTranscription", mock.Anything, i+1, mock.Anything).Return(nil)
	}
	mockLogger.SetEnabled(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessBatch(context.Background(), transcriptions, 5)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
