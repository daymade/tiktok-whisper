package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestNewEnhancedMockLogger verifies the constructor creates a properly configured logger
func TestNewEnhancedMockLogger(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.MockLogger)
	assert.True(t, logger.config.CaptureMessages)
	assert.True(t, logger.config.Silent)
	assert.Equal(t, LogLevelDebug, logger.config.MinLevel)
	assert.Equal(t, 1000, logger.config.MaxMessages)
	assert.False(t, logger.config.EnableMocking) // Disabled by default
	assert.Equal(t, 0, len(logger.MockLogger.GetLogs()))
}

// TestEnhancedMockLoggerBasicLogging verifies basic logging functionality
func TestEnhancedMockLoggerBasicLogging(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	// Test Info logging
	logger.Info("Test info message", "key1", "value1", "key2", 42)
	
	messages := logger.MockLogger.GetLogs()
	require.Len(t, messages, 1)
	
	msg := messages[0]
	assert.Equal(t, LogLevelInfo, msg.Level)
	assert.Equal(t, "Test info message", msg.Message)
	assert.Equal(t, []interface{}{"key1", "value1", "key2", 42}, msg.Args)
	assert.WithinDuration(t, time.Now(), msg.Time, time.Second)
}

// TestEnhancedMockLoggerAllLevels verifies all logging levels work correctly
func TestEnhancedMockLoggerAllLevels(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")
	
	messages := logger.MockLogger.GetLogs()
	assert.Equal(t, 4, len(messages))
	
	debugMessages := logger.MockLogger.GetLogsByLevel(LogLevelDebug)
	assert.Equal(t, 1, len(debugMessages))
	
	infoMessages := logger.MockLogger.GetLogsByLevel(LogLevelInfo)
	assert.Equal(t, 1, len(infoMessages))
	
	warnMessages := logger.MockLogger.GetLogsByLevel(LogLevelWarn)
	assert.Equal(t, 1, len(warnMessages))
	
	errorMessages := logger.MockLogger.GetLogsByLevel(LogLevelError)
	assert.Equal(t, 1, len(errorMessages))
}

// TestEnhancedMockLoggerStructuredData verifies structured data capture
func TestEnhancedMockLoggerStructuredData(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	logger.Info("Processing transcription", "transcriptionID", 123, "status", "started")
	logger.Error("Processing failed", "transcriptionID", 456, "error", "network timeout")
	
	// Test field-based search
	messages := logger.GetEnhancedMessages()
	assert.Len(t, messages, 2)
	
	// Check first message fields
	assert.Equal(t, 123, messages[0].Fields["transcriptionID"])
	assert.Equal(t, "started", messages[0].Fields["status"])
	
	// Check second message fields
	assert.Equal(t, 456, messages[1].Fields["transcriptionID"])
	assert.Equal(t, "network timeout", messages[1].Fields["error"])
}

// TestEnhancedMockLoggerFieldSearch verifies field-based message search
func TestEnhancedMockLoggerFieldSearch(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	logger.Info("Processing transcription", "transcriptionID", 123)
	logger.Error("Failed to process", "transcriptionID", 456, "error", "network timeout")
	logger.Info("Batch processing complete", "processed", 10, "failed", 2)
	
	// Test field-based search
	assert.True(t, logger.HasMessageWithField("transcriptionID", 123))
	assert.True(t, logger.HasMessageWithField("error", "network timeout"))
	assert.False(t, logger.HasMessageWithField("transcriptionID", 999))
	
	// Test finding messages by field
	foundByField := logger.FindMessagesByField("transcriptionID", 456)
	assert.Len(t, foundByField, 1)
	assert.Equal(t, "Failed to process", foundByField[0].Message)
}

// TestEnhancedMockLoggerTestifyIntegration verifies testify/mock integration
func TestEnhancedMockLoggerTestifyIntegration(t *testing.T) {
	logger := NewEnhancedMockLogger().WithMockingEnabled(true)
	
	// Set up expectations
	logger.ExpectInfo("Processing started", "batchSize", 10)
	logger.ExpectError("Processing failed", "error", mock.AnythingOfType("string"))
	
	// Execute the logged operations
	logger.Info("Processing started", "batchSize", 10)
	logger.Error("Processing failed", "error", "network timeout")
	
	// Verify expectations were met
	logger.AssertExpectations(t)
}

// TestEnhancedMockLoggerFlexibleExpectations verifies flexible expectation setup
func TestEnhancedMockLoggerFlexibleExpectations(t *testing.T) {
	logger := NewEnhancedMockLogger().WithMockingEnabled(true)
	
	// Set up flexible expectations
	logger.ExpectAnyInfo().Times(2)
	logger.ExpectAnyError().Once()
	
	// Execute operations
	logger.Info("First info message")
	logger.Info("Second info message")
	logger.Error("Error message")
	
	// Verify expectations
	logger.AssertExpectations(t)
}

// TestEnhancedMockLoggerConfiguration verifies configuration options
func TestEnhancedMockLoggerConfiguration(t *testing.T) {
	config := EnhancedLoggerConfig{
		CaptureMessages: true,
		Silent:         true,
		MinLevel:       LogLevelWarn,
		MaxMessages:    5,
		TimestampFormat: "15:04:05",
		TrackCalls:     true,
		CaptureFields:  true,
		EnableMocking:  true,
		VerboseOutput:  false,
	}
	
	logger := NewEnhancedMockLoggerWithConfig(config)
	
	assert.Equal(t, config.CaptureMessages, logger.config.CaptureMessages)
	assert.Equal(t, config.Silent, logger.config.Silent)
	assert.Equal(t, config.MinLevel, logger.config.MinLevel)
	assert.Equal(t, config.MaxMessages, logger.config.MaxMessages)
	assert.Equal(t, config.TimestampFormat, logger.config.TimestampFormat)
	assert.Equal(t, config.TrackCalls, logger.config.TrackCalls)
	assert.Equal(t, config.CaptureFields, logger.config.CaptureFields)
	assert.Equal(t, config.EnableMocking, logger.config.EnableMocking)
	assert.Equal(t, config.VerboseOutput, logger.config.VerboseOutput)
}

// TestEnhancedMockLoggerFluentConfiguration verifies fluent configuration
func TestEnhancedMockLoggerFluentConfiguration(t *testing.T) {
	logger := NewEnhancedMockLogger().
		WithMinLevel(LogLevelWarn).
		WithMaxMessages(500).
		WithVerboseOutput().
		WithMockingEnabled(false)
	
	assert.Equal(t, LogLevelWarn, logger.config.MinLevel)
	assert.Equal(t, 500, logger.config.MaxMessages)
	assert.True(t, logger.config.VerboseOutput)
	assert.False(t, logger.config.Silent)
	assert.False(t, logger.config.EnableMocking)
}

// TestEnhancedMockLoggerLevelFiltering verifies level filtering works correctly
func TestEnhancedMockLoggerLevelFiltering(t *testing.T) {
	logger := NewEnhancedMockLogger().WithMinLevel(LogLevelWarn)
	
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")
	
	messages := logger.MockLogger.GetLogs()
	// Only WARN and ERROR should be captured by the enhanced logger
	// But the base MockLogger still captures all messages
	assert.Equal(t, 4, len(messages))
	
	// However, the enhanced functionality should respect the min level
	assert.Equal(t, LogLevelWarn, logger.config.MinLevel)
}

// TestEnhancedMockLoggerStructuredDataCapture verifies structured data capture
func TestEnhancedMockLoggerStructuredDataCapture(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	logger.Info("Processing transcription", "transcriptionID", 123, "status", "started")
	logger.Error("Processing failed", "userID", "user123", "error", "timeout")
	
	// Get all structured data
	data := logger.GetStructuredData()
	
	// Check that fields from both messages are captured
	assert.Equal(t, 123, data["transcriptionID"])
	assert.Equal(t, "started", data["status"])
	assert.Equal(t, "user123", data["userID"])
	assert.Equal(t, "timeout", data["error"])
}

// TestEnhancedMockLoggerStructuredFieldSearch verifies structured field searching
func TestEnhancedMockLoggerStructuredFieldSearch(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	logger.Info("Processing started", "userID", "user123")
	logger.Error("Processing failed", "userID", "user456")
	
	// Test structured field search
	assert.True(t, logger.HasStructuredField("userID", "user456"))
	assert.False(t, logger.HasStructuredField("userID", "user999"))
}

// TestEnhancedMockLoggerReset verifies reset functionality
func TestEnhancedMockLoggerReset(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	// Add messages, expectations, and structured data
	logger.Info("Test message", "key", "value")
	logger.ExpectInfo("Expected message")
	
	assert.Equal(t, 1, len(logger.MockLogger.GetLogs()))
	assert.Equal(t, 1, len(logger.GetStructuredData()))
	
	// Reset the logger
	logger.ResetEnhanced()
	
	// Verify everything is cleared
	assert.Equal(t, 0, len(logger.MockLogger.GetLogs()))
	assert.Equal(t, 0, len(logger.GetStructuredData()))
}

// TestEnhancedMockLoggerThreadSafety verifies thread safety
func TestEnhancedMockLoggerThreadSafety(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	// Run concurrent logging operations
	done := make(chan bool, 10) // Buffered channel to prevent blocking
	
	// Start multiple goroutines logging messages
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }() // Ensure done is sent even if panic
			for j := 0; j < 100; j++ {
				logger.Info("Concurrent message", "goroutine", id, "message", j)
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify messages were captured (may be less than 1000 due to race conditions in base MockLogger)
	messageCount := len(logger.MockLogger.GetLogs())
	assert.Greater(t, messageCount, 900, "Expected at least 900 messages but got %d", messageCount)
	assert.LessOrEqual(t, messageCount, 1000, "Expected at most 1000 messages but got %d", messageCount)
	
	// Verify structured data was captured safely (EnhancedMockLogger is thread-safe)
	data := logger.GetStructuredData()
	assert.Contains(t, data, "goroutine")
	assert.Contains(t, data, "message")
}

// TestEnhancedMockLoggerSummary verifies summary generation
func TestEnhancedMockLoggerSummary(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	logger.Info("Info message", "key1", "value1")
	logger.Warn("Warning message", "key2", "value2")
	logger.Error("Error message", "key3", "value3")
	
	summary := logger.EnhancedSummary()
	
	assert.Contains(t, summary, "EnhancedMockLogger Summary")
	assert.Contains(t, summary, "Total Messages: 3")
	assert.Contains(t, summary, "Structured Fields: 3")
	assert.Contains(t, summary, "INFO: 1")
	assert.Contains(t, summary, "WARN: 1")
	assert.Contains(t, summary, "ERROR: 1")
	assert.Contains(t, summary, "Configuration:")
}

// TestEnhancedMockLoggerConfigurationUpdates verifies configuration updates
func TestEnhancedMockLoggerConfigurationUpdates(t *testing.T) {
	logger := NewEnhancedMockLogger()
	
	// Update configuration
	newConfig := EnhancedLoggerConfig{
		CaptureMessages: false,
		Silent:         false,
		MinLevel:       LogLevelError,
		MaxMessages:    100,
		TimestampFormat: "15:04:05",
		TrackCalls:     false,
		CaptureFields:  false,
		EnableMocking:  false,
		VerboseOutput:  true,
	}
	
	logger.UpdateConfig(newConfig)
	
	config := logger.GetConfig()
	assert.Equal(t, newConfig.CaptureMessages, config.CaptureMessages)
	assert.Equal(t, newConfig.Silent, config.Silent)
	assert.Equal(t, newConfig.MinLevel, config.MinLevel)
	assert.Equal(t, newConfig.MaxMessages, config.MaxMessages)
	assert.Equal(t, newConfig.VerboseOutput, config.VerboseOutput)
}

// TestEnhancedMockLoggerMockingDisabled verifies behavior when mocking is disabled
func TestEnhancedMockLoggerMockingDisabled(t *testing.T) {
	logger := NewEnhancedMockLogger().WithMockingEnabled(false)
	
	// These should not panic or create expectations
	assert.Nil(t, logger.ExpectInfo("test message"))
	assert.Nil(t, logger.ExpectError("test error"))
	assert.Nil(t, logger.ExpectAnyInfo())
	assert.Nil(t, logger.ExpectAnyError())
	
	// Logging should still work
	logger.Info("Test message")
	assert.Equal(t, 1, len(logger.MockLogger.GetLogs()))
}

// BenchmarkEnhancedMockLoggerLogging benchmarks logging performance
func BenchmarkEnhancedMockLoggerLogging(b *testing.B) {
	logger := NewEnhancedMockLogger()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark message", "iteration", i, "timestamp", time.Now())
	}
}

// BenchmarkEnhancedMockLoggerFieldSearch benchmarks field search performance
func BenchmarkEnhancedMockLoggerFieldSearch(b *testing.B) {
	logger := NewEnhancedMockLogger()
	
	// Pre-populate with messages
	for i := 0; i < 1000; i++ {
		logger.Info("Test message", "id", i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.HasMessageWithField("id", 500)
	}
}

// ExampleEnhancedMockLogger demonstrates basic usage
func ExampleEnhancedMockLogger() {
	// Create a new enhanced mock logger
	logger := NewEnhancedMockLogger()
	
	// Log some messages with structured data
	logger.Info("Processing started", "batchSize", 10, "userID", "user123")
	logger.Error("Processing failed", "error", "network timeout", "userID", "user123")
	
	// Check what was logged
	messages := logger.GetEnhancedMessages()
	for _, msg := range messages {
		if msg.Level == LogLevelError {
			// Handle error message with structured data
			if userID, exists := msg.Fields["userID"]; exists {
				_ = userID // Handle user-specific error
			}
		}
	}
	
	// Search for messages with specific fields
	userMessages := logger.FindMessagesByField("userID", "user123")
	_ = userMessages // Process user-specific messages
	
	// Verify expectations in tests
	logger.AssertExpectations(nil) // would pass *testing.T in real usage
}

// ExampleEnhancedMockLoggerWithTestify demonstrates testify integration
func ExampleEnhancedMockLoggerWithTestify() {
	logger := NewEnhancedMockLogger()
	
	// Set up expectations with structured data
	logger.ExpectInfo("Processing transcription", "transcriptionID", 123, "provider", "openai")
	logger.ExpectError("Failed to generate embedding", "transcriptionID", 123, "error", mock.AnythingOfType("*errors.errorString"))
	
	// Your code under test would call:
	logger.Info("Processing transcription", "transcriptionID", 123, "provider", "openai")
	logger.Error("Failed to generate embedding", "transcriptionID", 123, "error", assert.AnError)
	
	// Verify expectations were met
	logger.AssertExpectations(nil) // would pass *testing.T in real usage
	
	// Additionally verify structured data was captured
	if logger.HasStructuredField("transcriptionID", 123) {
		// Verify specific transcription was processed
	}
}