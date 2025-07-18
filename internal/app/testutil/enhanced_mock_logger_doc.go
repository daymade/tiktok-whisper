/*
Package testutil provides comprehensive testing utilities for the tiktok-whisper project.

EnhancedMockLogger Implementation
===============================

The EnhancedMockLogger extends the existing MockLogger with full testify/mock integration 
and advanced testing capabilities, providing a complete solution for testing logging 
behavior in the tiktok-whisper project.

Key Features:
- Full backward compatibility with existing MockLogger
- Complete testify/mock integration for method call verification
- Thread-safe structured data capture and retrieval
- Flexible configuration options for different testing scenarios
- Advanced message filtering and searching capabilities
- Performance optimized for high-volume logging scenarios
- Comprehensive helper methods for common testing patterns

Architecture:
- Embeds the existing MockLogger for backward compatibility
- Adds testify/mock.Mock for call verification
- Provides enhanced structured data capture
- Maintains thread safety through read-write mutexes
- Supports configurable behavior for different test scenarios

Basic Usage:
	logger := NewEnhancedMockLogger()
	
	// Log messages with structured data
	logger.Info("Processing started", "batchSize", 10, "userID", "user123")
	logger.Error("Processing failed", "error", "network timeout")
	
	// Verify logging behavior
	assert.True(t, logger.HasError())
	assert.True(t, logger.HasMessageWithField("userID", "user123"))
	assert.Equal(t, 2, len(logger.GetEnhancedMessages()))

Testify/Mock Integration:
	logger := NewEnhancedMockLogger()
	
	// Set up expectations
	logger.ExpectInfo("Processing transcription", "transcriptionID", 123)
	logger.ExpectAnyError().Once()
	
	// Execute code under test
	myEmbeddingOrchestrator.ProcessTranscription(ctx, 123, "test text")
	
	// Verify expectations
	logger.AssertExpectations(t)

Advanced Configuration:
	config := EnhancedLoggerConfig{
		CaptureMessages: true,    // Enable message capture
		Silent:         true,     // Suppress console output
		MinLevel:       LogLevelWarn, // Only capture WARN and ERROR
		MaxMessages:    500,      // Limit memory usage
		TimestampFormat: "15:04:05", // Custom timestamp format
		TrackCalls:     true,     // Enable call tracking
		CaptureFields:  true,     // Enable structured data capture
		EnableMocking:  true,     // Enable testify/mock integration
		VerboseOutput:  false,    // Disable verbose output
	}
	
	logger := NewEnhancedMockLoggerWithConfig(config)
	
	// Or use fluent configuration
	logger := NewEnhancedMockLogger().
		WithMinLevel(LogLevelInfo).
		WithMaxMessages(1000).
		WithVerboseOutput().
		WithMockingEnabled(true)

Structured Data Capture:
	logger := NewEnhancedMockLogger()
	
	// Log with structured data
	logger.Info("Processing transcription", "transcriptionID", 123, "provider", "openai")
	logger.Error("Processing failed", "transcriptionID", 123, "error", "network timeout")
	
	// Search by structured fields
	assert.True(t, logger.HasMessageWithField("transcriptionID", 123))
	assert.True(t, logger.HasStructuredField("provider", "openai"))
	
	// Find messages with specific fields
	transcriptionMessages := logger.FindMessagesByField("transcriptionID", 123)
	assert.Len(t, transcriptionMessages, 2)
	
	// Get all structured data
	data := logger.GetStructuredData()
	assert.Equal(t, 123, data["transcriptionID"])
	assert.Equal(t, "openai", data["provider"])

Enhanced Message Retrieval:
	logger := NewEnhancedMockLogger()
	
	// Log various messages
	logger.Info("Starting batch processing", "batchSize", 100)
	logger.Info("Processing progress", "completed", 50, "remaining", 50)
	logger.Error("Processing failed", "error", "network timeout")
	
	// Get enhanced messages with structured data
	messages := logger.GetEnhancedMessages()
	for _, msg := range messages {
		if msg.Level == LogLevelError {
			// Handle error with structured context
			if errorMsg, exists := msg.Fields["error"]; exists {
				// Process specific error type
				_ = errorMsg
			}
		}
	}
	
	// Use convenience methods
	errorMessages := logger.GetErrorMessages() // From base MockLogger
	assert.Len(t, errorMessages, 1)

Thread Safety:
	All EnhancedMockLogger methods are thread-safe and can be used safely in
	concurrent testing scenarios:
	
	logger := NewEnhancedMockLogger()
	
	// Safe for concurrent use
	go func() {
		logger.Info("Concurrent message", "goroutine", 1)
	}()
	
	go func() {
		logger.Error("Concurrent error", "goroutine", 2)
	}()
	
	// All operations are thread-safe
	data := logger.GetStructuredData()
	messages := logger.GetEnhancedMessages()

Testing EmbeddingOrchestrator:
	The EnhancedMockLogger is specifically designed for testing the EmbeddingOrchestrator
	and related components:
	
	func TestEmbeddingOrchestrator(t *testing.T) {
		logger := NewEnhancedMockLogger()
		mockProvider := &MockEmbeddingProvider{}
		mockStorage := &MockVectorStorage{}
		
		orchestrator := NewEmbeddingOrchestrator(
			map[string]provider.EmbeddingProvider{"openai": mockProvider},
			mockStorage,
			logger,
		)
		
		// Set up expectations
		logger.ExpectInfo("Successfully processed embedding", "provider", "openai", "transcriptionID", 123)
		mockProvider.On("GenerateEmbedding", mock.Anything, "test text").Return([]float32{0.1, 0.2}, nil)
		mockStorage.On("StoreEmbedding", mock.Anything, 123, "openai", mock.Anything).Return(nil)
		
		// Execute test
		err := orchestrator.ProcessTranscription(context.Background(), 123, "test text")
		
		// Verify results
		assert.NoError(t, err)
		logger.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
		
		// Verify structured logging
		assert.True(t, logger.HasMessageWithField("transcriptionID", 123))
		assert.True(t, logger.HasMessageWithField("provider", "openai"))
		assert.False(t, logger.HasError())
	}

Testing BatchProcessor:
	The EnhancedMockLogger provides excellent support for testing batch processing scenarios:
	
	func TestBatchProcessor(t *testing.T) {
		logger := NewEnhancedMockLogger()
		orchestrator := &MockEmbeddingOrchestrator{}
		storage := &MockVectorStorage{}
		
		processor := NewBatchProcessor(orchestrator, storage, logger)
		
		// Set up expectations for batch processing
		logger.ExpectInfo("Starting batch processing", "totalTranscriptions", 100, "batchSize", 10)
		logger.ExpectInfo("Batch processing progress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Times(10)
		
		// Execute batch processing
		err := processor.ProcessAllTranscriptions(context.Background(), 10)
		
		// Verify results
		assert.NoError(t, err)
		logger.AssertExpectations(t)
		
		// Verify progress logging
		progressMessages := logger.FindMessages("Batch processing progress")
		assert.Greater(t, len(progressMessages), 0)
		
		// Verify no errors occurred
		assert.False(t, logger.HasError())
	}

Performance Considerations:
	The EnhancedMockLogger is optimized for performance while maintaining comprehensive
	testing capabilities:
	
	- Message capture is optimized for high-volume scenarios
	- Structured data extraction is performed efficiently
	- Memory usage is controlled through MaxMessages configuration
	- Thread-safe operations use read-write mutexes for optimal performance
	- Search operations are optimized for typical test patterns

Best Practices:
	1. Use NewEnhancedMockLogger() for most testing scenarios
	2. Use configuration options to optimize for specific test needs
	3. Use ExpectAny* methods for flexible expectations
	4. Use structured field search for detailed verification
	5. Use ResetEnhanced() between test cases for clean state
	6. Use thread-safe operations in concurrent tests
	7. Use HasError() and HasWarning() for quick status checks
	8. Use structured data capture for detailed test verification

Common Test Patterns:
	// Pattern 1: Basic logging verification
	logger := NewEnhancedMockLogger()
	myFunction(logger)
	assert.False(t, logger.HasError())
	assert.True(t, logger.HasMessage("Operation completed"))
	
	// Pattern 2: Structured data verification
	logger := NewEnhancedMockLogger()
	myFunction(logger)
	assert.True(t, logger.HasMessageWithField("transcriptionID", 123))
	assert.True(t, logger.HasStructuredField("status", "completed"))
	
	// Pattern 3: Mock expectation verification
	logger := NewEnhancedMockLogger()
	logger.ExpectInfo("Processing started", "batchSize", 10)
	logger.ExpectAnyError().Times(0) // Expect no errors
	myFunction(logger)
	logger.AssertExpectations(t)
	
	// Pattern 4: Progress monitoring
	logger := NewEnhancedMockLogger()
	myBatchFunction(logger)
	progressMessages := logger.FindMessages("progress")
	assert.Greater(t, len(progressMessages), 0)
	
	// Pattern 5: Error analysis
	logger := NewEnhancedMockLogger()
	myFunction(logger)
	if logger.HasError() {
		errorMessages := logger.GetErrorMessages()
		for _, msg := range errorMessages {
			// Analyze error details
			assert.Contains(t, msg.Fields, "error")
		}
	}

Memory Management:
	The EnhancedMockLogger automatically manages memory through:
	- Configurable MaxMessages limit
	- Automatic cleanup of old messages
	- Efficient structured data storage
	- Thread-safe memory operations

Integration with Existing Code:
	The EnhancedMockLogger is fully compatible with existing MockLogger usage:
	
	// Existing code using MockLogger
	logger := NewMockLogger()
	logger.Info("Test message")
	assert.True(t, logger.ContainsMessage("Test message"))
	
	// Enhanced version with additional capabilities
	enhancedLogger := NewEnhancedMockLogger()
	enhancedLogger.Info("Test message", "key", "value")
	assert.True(t, enhancedLogger.ContainsMessage("Test message"))
	assert.True(t, enhancedLogger.HasMessageWithField("key", "value"))
	enhancedLogger.AssertExpectations(t)

The EnhancedMockLogger provides a complete solution for testing logging behavior 
in the tiktok-whisper project, combining the simplicity of the existing MockLogger 
with the power of testify/mock integration and advanced structured data capabilities.
*/
package testutil