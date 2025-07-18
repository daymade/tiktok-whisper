package testutil

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
)

// EnhancedMockLogger is a comprehensive testify/mock based implementation of the Logger interface
// It extends the existing MockLogger with full testify/mock integration and advanced testing capabilities.
type EnhancedMockLogger struct {
	*MockLogger // Embed the existing MockLogger
	mock.Mock   // Add testify/mock capabilities
	
	// Enhanced configuration and state
	config          EnhancedLoggerConfig
	callTracking    bool
	mutex           sync.RWMutex
	structuredData  map[string]interface{}
}

// EnhancedLoggerConfig extends the basic logger configuration with additional testing features
type EnhancedLoggerConfig struct {
	// Inherit basic configuration
	CaptureMessages bool
	Silent         bool
	MinLevel       LogLevel
	MaxMessages    int
	TimestampFormat string
	
	// Enhanced features
	TrackCalls      bool
	CaptureFields   bool
	EnableMocking   bool
	VerboseOutput   bool
}

// EnhancedLogMessage extends LogEntry with additional structured data
type EnhancedLogMessage struct {
	LogEntry
	Fields map[string]interface{}
	CallID string
}

// NewEnhancedMockLogger creates a new EnhancedMockLogger with comprehensive testing capabilities
func NewEnhancedMockLogger() *EnhancedMockLogger {
	return &EnhancedMockLogger{
		MockLogger: NewMockLogger(),
		config: EnhancedLoggerConfig{
			CaptureMessages: true,
			Silent:         true,
			MinLevel:       LogLevelDebug,
			MaxMessages:    1000,
			TimestampFormat: "2006-01-02 15:04:05.000",
			TrackCalls:     true,
			CaptureFields:  true,
			EnableMocking:  false, // Disabled by default to avoid unexpected call panics
			VerboseOutput:  false,
		},
		callTracking:   true,
		structuredData: make(map[string]interface{}),
	}
}

// NewEnhancedMockLoggerWithConfig creates a new EnhancedMockLogger with custom configuration
func NewEnhancedMockLoggerWithConfig(config EnhancedLoggerConfig) *EnhancedMockLogger {
	return &EnhancedMockLogger{
		MockLogger:     NewMockLogger(),
		config:         config,
		callTracking:   config.TrackCalls,
		structuredData: make(map[string]interface{}),
	}
}

// WithVerboseOutput configures the logger to print messages to stdout
func (m *EnhancedMockLogger) WithVerboseOutput() *EnhancedMockLogger {
	m.config.VerboseOutput = true
	m.config.Silent = false
	return m
}

// WithSilentOutput configures the logger to suppress output (default)
func (m *EnhancedMockLogger) WithSilentOutput() *EnhancedMockLogger {
	m.config.VerboseOutput = false
	m.config.Silent = true
	return m
}

// WithMinLevel sets the minimum log level to capture/process
func (m *EnhancedMockLogger) WithMinLevel(level LogLevel) *EnhancedMockLogger {
	m.config.MinLevel = level
	return m
}

// WithMaxMessages sets the maximum number of messages to keep in memory
func (m *EnhancedMockLogger) WithMaxMessages(max int) *EnhancedMockLogger {
	m.config.MaxMessages = max
	return m
}

// WithMockingEnabled enables/disables testify/mock integration
func (m *EnhancedMockLogger) WithMockingEnabled(enabled bool) *EnhancedMockLogger {
	m.config.EnableMocking = enabled
	return m
}

// Debug logs a debug message with both legacy and enhanced functionality
func (m *EnhancedMockLogger) Debug(msg string, keysAndValues ...interface{}) {
	// Call the original MockLogger method
	m.MockLogger.Debug(msg, keysAndValues...)
	
	// Add testify/mock integration if enabled
	if m.config.EnableMocking {
		args := append([]interface{}{msg}, keysAndValues...)
		m.Called(args...)
	}
	
	// Enhanced logging
	m.logEnhanced(LogLevelDebug, msg, keysAndValues...)
}

// Info logs an info message with both legacy and enhanced functionality
func (m *EnhancedMockLogger) Info(msg string, keysAndValues ...interface{}) {
	// Call the original MockLogger method
	m.MockLogger.Info(msg, keysAndValues...)
	
	// Add testify/mock integration if enabled
	if m.config.EnableMocking {
		args := append([]interface{}{msg}, keysAndValues...)
		m.Called(args...)
	}
	
	// Enhanced logging
	m.logEnhanced(LogLevelInfo, msg, keysAndValues...)
}

// Warn logs a warning message with both legacy and enhanced functionality
func (m *EnhancedMockLogger) Warn(msg string, keysAndValues ...interface{}) {
	// Call the original MockLogger method
	m.MockLogger.Warn(msg, keysAndValues...)
	
	// Add testify/mock integration if enabled
	if m.config.EnableMocking {
		args := append([]interface{}{msg}, keysAndValues...)
		m.Called(args...)
	}
	
	// Enhanced logging
	m.logEnhanced(LogLevelWarn, msg, keysAndValues...)
}

// Error logs an error message with both legacy and enhanced functionality
func (m *EnhancedMockLogger) Error(msg string, keysAndValues ...interface{}) {
	// Call the original MockLogger method
	m.MockLogger.Error(msg, keysAndValues...)
	
	// Add testify/mock integration if enabled
	if m.config.EnableMocking {
		args := append([]interface{}{msg}, keysAndValues...)
		m.Called(args...)
	}
	
	// Enhanced logging
	m.logEnhanced(LogLevelError, msg, keysAndValues...)
}

// logEnhanced provides enhanced logging capabilities
func (m *EnhancedMockLogger) logEnhanced(level LogLevel, msg string, keysAndValues ...interface{}) {
	if !m.shouldLog(level) {
		return
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Extract structured fields if enabled
	if m.config.CaptureFields {
		fields := m.extractFields(keysAndValues)
		for k, v := range fields {
			m.structuredData[k] = v
		}
	}
	
	// Print to stdout if verbose output is enabled
	if m.config.VerboseOutput {
		fmt.Printf("[%s] %s %s %s\n", 
			time.Now().Format(m.config.TimestampFormat),
			level,
			msg,
			m.formatKeyValues(keysAndValues),
		)
	}
}

// shouldLog determines if a message should be logged based on level
func (m *EnhancedMockLogger) shouldLog(level LogLevel) bool {
	levelOrder := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
	}
	
	return levelOrder[level] >= levelOrder[m.config.MinLevel]
}

// extractFields extracts structured data from key-value pairs
func (m *EnhancedMockLogger) extractFields(keysAndValues []interface{}) map[string]interface{} {
	fields := make(map[string]interface{})
	
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			value := keysAndValues[i+1]
			fields[key] = value
		}
	}
	
	return fields
}

// formatKeyValues formats key-value pairs for display
func (m *EnhancedMockLogger) formatKeyValues(keysAndValues []interface{}) string {
	if len(keysAndValues) == 0 {
		return ""
	}
	
	var parts []string
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			parts = append(parts, fmt.Sprintf("%v=%v", keysAndValues[i], keysAndValues[i+1]))
		}
	}
	
	return strings.Join(parts, " ")
}

// GetEnhancedMessages returns all messages with enhanced metadata
func (m *EnhancedMockLogger) GetEnhancedMessages() []EnhancedLogMessage {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	entries := m.MockLogger.GetLogs()
	enhanced := make([]EnhancedLogMessage, len(entries))
	
	for i, entry := range entries {
		enhanced[i] = EnhancedLogMessage{
			LogEntry: entry,
			Fields:   m.extractFields(entry.Args),
			CallID:   fmt.Sprintf("call_%d", i),
		}
	}
	
	return enhanced
}

// HasMessageWithField checks if any message has the specified field with the given value
func (m *EnhancedMockLogger) HasMessageWithField(fieldName string, fieldValue interface{}) bool {
	messages := m.GetEnhancedMessages()
	for _, msg := range messages {
		if value, exists := msg.Fields[fieldName]; exists {
			if reflect.DeepEqual(value, fieldValue) {
				return true
			}
		}
	}
	return false
}

// FindMessagesByField searches for messages with the specified field value
func (m *EnhancedMockLogger) FindMessagesByField(fieldName string, fieldValue interface{}) []EnhancedLogMessage {
	messages := m.GetEnhancedMessages()
	var found []EnhancedLogMessage
	
	for _, msg := range messages {
		if value, exists := msg.Fields[fieldName]; exists {
			if reflect.DeepEqual(value, fieldValue) {
				found = append(found, msg)
			}
		}
	}
	
	return found
}

// GetStructuredData returns all captured structured data
func (m *EnhancedMockLogger) GetStructuredData() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Return a copy to prevent data races
	result := make(map[string]interface{})
	for k, v := range m.structuredData {
		result[k] = v
	}
	
	return result
}

// HasStructuredField checks if a structured field exists with the given value
func (m *EnhancedMockLogger) HasStructuredField(fieldName string, fieldValue interface{}) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if value, exists := m.structuredData[fieldName]; exists {
		return reflect.DeepEqual(value, fieldValue)
	}
	
	return false
}

// ClearStructuredData clears all captured structured data
func (m *EnhancedMockLogger) ClearStructuredData() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.structuredData = make(map[string]interface{})
}

// ExpectInfo sets up an expectation for an Info call
func (m *EnhancedMockLogger) ExpectInfo(msg string, keysAndValues ...interface{}) *mock.Call {
	if !m.config.EnableMocking {
		return nil
	}
	args := append([]interface{}{msg}, keysAndValues...)
	return m.On("Info", args...)
}

// ExpectError sets up an expectation for an Error call
func (m *EnhancedMockLogger) ExpectError(msg string, keysAndValues ...interface{}) *mock.Call {
	if !m.config.EnableMocking {
		return nil
	}
	args := append([]interface{}{msg}, keysAndValues...)
	return m.On("Error", args...)
}

// ExpectDebug sets up an expectation for a Debug call
func (m *EnhancedMockLogger) ExpectDebug(msg string, keysAndValues ...interface{}) *mock.Call {
	if !m.config.EnableMocking {
		return nil
	}
	args := append([]interface{}{msg}, keysAndValues...)
	return m.On("Debug", args...)
}

// ExpectWarn sets up an expectation for a Warn call
func (m *EnhancedMockLogger) ExpectWarn(msg string, keysAndValues ...interface{}) *mock.Call {
	if !m.config.EnableMocking {
		return nil
	}
	args := append([]interface{}{msg}, keysAndValues...)
	return m.On("Warn", args...)
}

// ExpectAnyInfo sets up an expectation for any Info call
func (m *EnhancedMockLogger) ExpectAnyInfo() *mock.Call {
	if !m.config.EnableMocking {
		return nil
	}
	return m.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ExpectAnyError sets up an expectation for any Error call
func (m *EnhancedMockLogger) ExpectAnyError() *mock.Call {
	if !m.config.EnableMocking {
		return nil
	}
	return m.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ExpectAnyDebug sets up an expectation for any Debug call
func (m *EnhancedMockLogger) ExpectAnyDebug() *mock.Call {
	if !m.config.EnableMocking {
		return nil
	}
	return m.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ExpectAnyWarn sets up an expectation for any Warn call
func (m *EnhancedMockLogger) ExpectAnyWarn() *mock.Call {
	if !m.config.EnableMocking {
		return nil
	}
	return m.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ResetEnhanced clears all captured messages, structured data, and mock expectations
func (m *EnhancedMockLogger) ResetEnhanced() {
	m.MockLogger.Reset()
	m.ClearStructuredData()
	m.Mock = mock.Mock{}
}

// GetConfig returns the current configuration
func (m *EnhancedMockLogger) GetConfig() EnhancedLoggerConfig {
	return m.config
}

// UpdateConfig updates the logger configuration
func (m *EnhancedMockLogger) UpdateConfig(config EnhancedLoggerConfig) {
	m.config = config
}

// HasError checks if any error-level messages were logged
func (m *EnhancedMockLogger) HasError() bool {
	return len(m.MockLogger.GetLogsByLevel(LogLevelError)) > 0
}

// HasWarning checks if any warning-level messages were logged
func (m *EnhancedMockLogger) HasWarning() bool {
	return len(m.MockLogger.GetLogsByLevel(LogLevelWarn)) > 0
}

// GetErrorMessages returns all error-level messages
func (m *EnhancedMockLogger) GetErrorMessages() []LogEntry {
	return m.MockLogger.GetLogsByLevel(LogLevelError)
}

// GetWarningMessages returns all warning-level messages
func (m *EnhancedMockLogger) GetWarningMessages() []LogEntry {
	return m.MockLogger.GetLogsByLevel(LogLevelWarn)
}

// GetInfoMessages returns all info-level messages
func (m *EnhancedMockLogger) GetInfoMessages() []LogEntry {
	return m.MockLogger.GetLogsByLevel(LogLevelInfo)
}

// GetDebugMessages returns all debug-level messages
func (m *EnhancedMockLogger) GetDebugMessages() []LogEntry {
	return m.MockLogger.GetLogsByLevel(LogLevelDebug)
}

// FindMessages searches for messages containing the specified text
func (m *EnhancedMockLogger) FindMessages(text string) []LogEntry {
	var found []LogEntry
	for _, entry := range m.MockLogger.GetLogs() {
		if strings.Contains(entry.Message, text) {
			found = append(found, entry)
		}
	}
	return found
}

// Summary returns a detailed summary of the logger state
func (m *EnhancedMockLogger) EnhancedSummary() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var summary strings.Builder
	summary.WriteString("EnhancedMockLogger Summary\n")
	summary.WriteString("========================\n")
	
	// Basic stats
	logs := m.MockLogger.GetLogs()
	summary.WriteString(fmt.Sprintf("Total Messages: %d\n", len(logs)))
	summary.WriteString(fmt.Sprintf("Structured Fields: %d\n", len(m.structuredData)))
	summary.WriteString(fmt.Sprintf("Mock Calls: %d\n", len(m.Mock.Calls)))
	
	// Level counts
	levelCounts := make(map[LogLevel]int)
	for _, log := range logs {
		levelCounts[log.Level]++
	}
	
	summary.WriteString("\nMessage Counts by Level:\n")
	for level, count := range levelCounts {
		summary.WriteString(fmt.Sprintf("  %s: %d\n", level, count))
	}
	
	// Configuration
	summary.WriteString("\nConfiguration:\n")
	summary.WriteString(fmt.Sprintf("  CaptureMessages: %t\n", m.config.CaptureMessages))
	summary.WriteString(fmt.Sprintf("  Silent: %t\n", m.config.Silent))
	summary.WriteString(fmt.Sprintf("  MinLevel: %s\n", m.config.MinLevel))
	summary.WriteString(fmt.Sprintf("  MaxMessages: %d\n", m.config.MaxMessages))
	summary.WriteString(fmt.Sprintf("  EnableMocking: %t\n", m.config.EnableMocking))
	
	return summary.String()
}