package testutil

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
)

// MockTranscriber is a mock implementation of the Transcriber interface
type MockTranscriber struct {
	TranscriptFunc func(inputFilePath string) (string, error)
	responses      map[string]string
	errors         map[string]error
	callCount      int
	lastFilePath   string
}

// NewMockTranscriber creates a new MockTranscriber with default behavior
func NewMockTranscriber() *MockTranscriber {
	return &MockTranscriber{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

// NewMockTranscriberWithDefaults creates a MockTranscriber with sensible defaults
func NewMockTranscriberWithDefaults() *MockTranscriber {
	mock := NewMockTranscriber()
	mock.WithDefaultResponse("This is a mock transcription of the audio file.")
	return mock
}

// WithResponse sets a specific response for a file path
func (m *MockTranscriber) WithResponse(filePath, response string) *MockTranscriber {
	m.responses[filePath] = response
	return m
}

// WithError sets an error for a specific file path
func (m *MockTranscriber) WithError(filePath string, err error) *MockTranscriber {
	m.errors[filePath] = err
	return m
}

// WithDefaultResponse sets a default response for any file path
func (m *MockTranscriber) WithDefaultResponse(response string) *MockTranscriber {
	m.TranscriptFunc = func(inputFilePath string) (string, error) {
		m.callCount++
		m.lastFilePath = inputFilePath

		// Check for specific error first
		if err, exists := m.errors[inputFilePath]; exists {
			return "", err
		}

		// Check for specific response
		if resp, exists := m.responses[inputFilePath]; exists {
			return resp, nil
		}

		// Return default response
		return response, nil
	}
	return m
}

// WithLatency adds artificial latency to transcription calls
func (m *MockTranscriber) WithLatency(duration time.Duration) *MockTranscriber {
	originalFunc := m.TranscriptFunc
	m.TranscriptFunc = func(inputFilePath string) (string, error) {
		time.Sleep(duration)
		if originalFunc != nil {
			return originalFunc(inputFilePath)
		}
		return "", errors.New("no transcript function set")
	}
	return m
}

// Transcript implements the Transcriber interface
func (m *MockTranscriber) Transcript(inputFilePath string) (string, error) {
	if m.TranscriptFunc != nil {
		return m.TranscriptFunc(inputFilePath)
	}

	m.callCount++
	m.lastFilePath = inputFilePath

	// Check for specific error first
	if err, exists := m.errors[inputFilePath]; exists {
		return "", err
	}

	// Check for specific response
	if resp, exists := m.responses[inputFilePath]; exists {
		return resp, nil
	}

	// Default behavior
	return fmt.Sprintf("Mock transcription for %s", filepath.Base(inputFilePath)), nil
}

// GetCallCount returns the number of times Transcript was called
func (m *MockTranscriber) GetCallCount() int {
	return m.callCount
}

// GetLastFilePath returns the last file path that was transcribed
func (m *MockTranscriber) GetLastFilePath() string {
	return m.lastFilePath
}

// Reset resets the mock state
func (m *MockTranscriber) Reset() {
	m.callCount = 0
	m.lastFilePath = ""
	m.responses = make(map[string]string)
	m.errors = make(map[string]error)
	m.TranscriptFunc = nil
}

// MockTranscriptionDAO is a mock implementation of the TranscriptionDAO interface
type MockTranscriptionDAO struct {
	transcriptions   []model.Transcription
	processedFiles   map[string]int
	closeFunc        func() error
	getAllByUserFunc func(userNickname string) ([]model.Transcription, error)
	recordToDBFunc   func(user, inputDir, fileName, mp3FileName string, audioDuration int, transcription string, lastConversionTime time.Time, hasError int, errorMessage string)
	checkFileFunc    func(fileName string) (int, error)
	closeCalled      bool
	recordCalls      []RecordCall
}

// RecordCall represents a call to RecordToDB
type RecordCall struct {
	User               string
	InputDir           string
	FileName           string
	Mp3FileName        string
	AudioDuration      int
	Transcription      string
	LastConversionTime time.Time
	HasError           int
	ErrorMessage       string
}

// NewMockTranscriptionDAO creates a new MockTranscriptionDAO
func NewMockTranscriptionDAO() *MockTranscriptionDAO {
	return &MockTranscriptionDAO{
		transcriptions: make([]model.Transcription, 0),
		processedFiles: make(map[string]int),
		recordCalls:    make([]RecordCall, 0),
	}
}

// WithTranscriptions sets up the mock with predefined transcriptions
func (m *MockTranscriptionDAO) WithTranscriptions(transcriptions []model.Transcription) *MockTranscriptionDAO {
	m.transcriptions = transcriptions
	return m
}

// WithProcessedFile marks a file as processed
func (m *MockTranscriptionDAO) WithProcessedFile(fileName string, id int) *MockTranscriptionDAO {
	m.processedFiles[fileName] = id
	return m
}

// WithCloseError sets an error to be returned when Close is called
func (m *MockTranscriptionDAO) WithCloseError(err error) *MockTranscriptionDAO {
	m.closeFunc = func() error {
		return err
	}
	return m
}

// Close implements the TranscriptionDAO interface
func (m *MockTranscriptionDAO) Close() error {
	m.closeCalled = true
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// GetAllByUser implements the TranscriptionDAO interface
func (m *MockTranscriptionDAO) GetAllByUser(userNickname string) ([]model.Transcription, error) {
	if m.getAllByUserFunc != nil {
		return m.getAllByUserFunc(userNickname)
	}

	var userTranscriptions []model.Transcription
	for _, t := range m.transcriptions {
		if t.User == userNickname {
			userTranscriptions = append(userTranscriptions, t)
		}
	}
	return userTranscriptions, nil
}

// CheckIfFileProcessed implements the TranscriptionDAO interface
func (m *MockTranscriptionDAO) CheckIfFileProcessed(fileName string) (int, error) {
	if m.checkFileFunc != nil {
		return m.checkFileFunc(fileName)
	}

	if id, exists := m.processedFiles[fileName]; exists {
		return id, nil
	}
	return 0, sql.ErrNoRows
}

// RecordToDB implements the TranscriptionDAO interface
func (m *MockTranscriptionDAO) RecordToDB(user, inputDir, fileName, mp3FileName string, audioDuration int, transcription string, lastConversionTime time.Time, hasError int, errorMessage string) {
	call := RecordCall{
		User:               user,
		InputDir:           inputDir,
		FileName:           fileName,
		Mp3FileName:        mp3FileName,
		AudioDuration:      audioDuration,
		Transcription:      transcription,
		LastConversionTime: lastConversionTime,
		HasError:           hasError,
		ErrorMessage:       errorMessage,
	}
	m.recordCalls = append(m.recordCalls, call)

	if m.recordToDBFunc != nil {
		m.recordToDBFunc(user, inputDir, fileName, mp3FileName, audioDuration, transcription, lastConversionTime, hasError, errorMessage)
		return
	}

	// Add to transcriptions
	transcriptionRecord := model.Transcription{
		ID:                 len(m.transcriptions) + 1,
		User:               user,
		LastConversionTime: lastConversionTime,
		Mp3FileName:        mp3FileName,
		AudioDuration:      float64(audioDuration),
		Transcription:      transcription,
		ErrorMessage:       errorMessage,
	}
	m.transcriptions = append(m.transcriptions, transcriptionRecord)

	// Mark file as processed
	m.processedFiles[fileName] = transcriptionRecord.ID
}

// GetRecordCalls returns all calls made to RecordToDB
func (m *MockTranscriptionDAO) GetRecordCalls() []RecordCall {
	return m.recordCalls
}

// GetTranscriptions returns all transcriptions in the mock
func (m *MockTranscriptionDAO) GetTranscriptions() []model.Transcription {
	return m.transcriptions
}

// WasCloseCalled returns true if Close was called
func (m *MockTranscriptionDAO) WasCloseCalled() bool {
	return m.closeCalled
}

// Reset resets the mock state
func (m *MockTranscriptionDAO) Reset() {
	m.transcriptions = make([]model.Transcription, 0)
	m.processedFiles = make(map[string]int)
	m.recordCalls = make([]RecordCall, 0)
	m.closeCalled = false
	m.closeFunc = nil
	m.getAllByUserFunc = nil
	m.recordToDBFunc = nil
	m.checkFileFunc = nil
}

// MockLogger is a mock implementation of a logger
type MockLogger struct {
	logs    []LogEntry
	logFunc func(level LogLevel, message string, args ...interface{})
	enabled bool
}

// LogLevel represents different log levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogEntry represents a log entry
type LogEntry struct {
	Level   LogLevel
	Message string
	Args    []interface{}
	Time    time.Time
}

// NewMockLogger creates a new MockLogger
func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs:    make([]LogEntry, 0),
		enabled: true,
	}
}

// WithCustomLogFunc sets a custom log function
func (m *MockLogger) WithCustomLogFunc(logFunc func(level LogLevel, message string, args ...interface{})) *MockLogger {
	m.logFunc = logFunc
	return m
}

// Debug logs a debug message
func (m *MockLogger) Debug(message string, args ...interface{}) {
	m.log(LogLevelDebug, message, args...)
}

// Info logs an info message
func (m *MockLogger) Info(message string, args ...interface{}) {
	m.log(LogLevelInfo, message, args...)
}

// Warn logs a warning message
func (m *MockLogger) Warn(message string, args ...interface{}) {
	m.log(LogLevelWarn, message, args...)
}

// Error logs an error message
func (m *MockLogger) Error(message string, args ...interface{}) {
	m.log(LogLevelError, message, args...)
}

// log is the internal logging function
func (m *MockLogger) log(level LogLevel, message string, args ...interface{}) {
	if !m.enabled {
		return
	}

	entry := LogEntry{
		Level:   level,
		Message: message,
		Args:    args,
		Time:    time.Now(),
	}

	m.logs = append(m.logs, entry)

	if m.logFunc != nil {
		m.logFunc(level, message, args...)
	}
}

// GetLogs returns all logged entries
func (m *MockLogger) GetLogs() []LogEntry {
	return m.logs
}

// GetLogsByLevel returns logs filtered by level
func (m *MockLogger) GetLogsByLevel(level LogLevel) []LogEntry {
	var filtered []LogEntry
	for _, log := range m.logs {
		if log.Level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// ContainsMessage checks if any log contains the specified message
func (m *MockLogger) ContainsMessage(message string) bool {
	for _, log := range m.logs {
		if strings.Contains(log.Message, message) {
			return true
		}
	}
	return false
}

// Reset clears all logs
func (m *MockLogger) Reset() {
	m.logs = make([]LogEntry, 0)
}

// SetEnabled enables or disables logging
func (m *MockLogger) SetEnabled(enabled bool) {
	m.enabled = enabled
}

// MockFileSystem is a mock implementation of file system operations
type MockFileSystem struct {
	files         map[string]MockFile
	directories   map[string]bool
	workingDir    string
	readFileFunc  func(filename string) ([]byte, error)
	writeFileFunc func(filename string, data []byte, perm os.FileMode) error
	statFunc      func(name string) (os.FileInfo, error)
	mkdirFunc     func(name string, perm os.FileMode) error
}

// MockFile represents a mock file
type MockFile struct {
	Name    string
	Content []byte
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
	Size    int64
}

// NewMockFileSystem creates a new MockFileSystem
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:       make(map[string]MockFile),
		directories: make(map[string]bool),
		workingDir:  "/mock/working/dir",
	}
}

// WithFile adds a file to the mock filesystem
func (m *MockFileSystem) WithFile(path string, content []byte, mode os.FileMode) *MockFileSystem {
	m.files[path] = MockFile{
		Name:    filepath.Base(path),
		Content: content,
		Mode:    mode,
		ModTime: time.Now(),
		IsDir:   false,
		Size:    int64(len(content)),
	}

	// Ensure parent directories exist
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		m.directories[dir] = true
	}

	return m
}

// WithDirectory adds a directory to the mock filesystem
func (m *MockFileSystem) WithDirectory(path string) *MockFileSystem {
	m.directories[path] = true
	m.files[path] = MockFile{
		Name:    filepath.Base(path),
		Content: nil,
		Mode:    os.ModeDir | 0755,
		ModTime: time.Now(),
		IsDir:   true,
		Size:    0,
	}
	return m
}

// WithWorkingDir sets the working directory
func (m *MockFileSystem) WithWorkingDir(dir string) *MockFileSystem {
	m.workingDir = dir
	return m
}

// ReadFile reads a file from the mock filesystem
func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if m.readFileFunc != nil {
		return m.readFileFunc(filename)
	}

	file, exists := m.files[filename]
	if !exists {
		return nil, &fs.PathError{Op: "read", Path: filename, Err: fs.ErrNotExist}
	}

	if file.IsDir {
		return nil, &fs.PathError{Op: "read", Path: filename, Err: fs.ErrInvalid}
	}

	return file.Content, nil
}

// WriteFile writes a file to the mock filesystem
func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if m.writeFileFunc != nil {
		return m.writeFileFunc(filename, data, perm)
	}

	m.files[filename] = MockFile{
		Name:    filepath.Base(filename),
		Content: data,
		Mode:    perm,
		ModTime: time.Now(),
		IsDir:   false,
		Size:    int64(len(data)),
	}

	// Ensure parent directories exist
	dir := filepath.Dir(filename)
	if dir != "." && dir != "/" {
		m.directories[dir] = true
	}

	return nil
}

// Stat returns file information
func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc(name)
	}

	file, exists := m.files[name]
	if !exists {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}

	return &MockFileInfo{
		name:    file.Name,
		size:    file.Size,
		mode:    file.Mode,
		modTime: file.ModTime,
		isDir:   file.IsDir,
	}, nil
}

// Mkdir creates a directory
func (m *MockFileSystem) Mkdir(name string, perm os.FileMode) error {
	if m.mkdirFunc != nil {
		return m.mkdirFunc(name, perm)
	}

	m.WithDirectory(name)
	return nil
}

// Exists checks if a file or directory exists
func (m *MockFileSystem) Exists(path string) bool {
	_, exists := m.files[path]
	return exists
}

// ListFiles returns all files in the mock filesystem
func (m *MockFileSystem) ListFiles() []string {
	var files []string
	for path := range m.files {
		files = append(files, path)
	}
	return files
}

// Reset clears all files and directories
func (m *MockFileSystem) Reset() {
	m.files = make(map[string]MockFile)
	m.directories = make(map[string]bool)
}

// MockFileInfo implements os.FileInfo
type MockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *MockFileInfo) ModTime() time.Time { return m.modTime }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() interface{}   { return nil }

// Ensure our mocks implement the required interfaces
var _ api.Transcriber = (*MockTranscriber)(nil)
var _ repository.TranscriptionDAO = (*MockTranscriptionDAO)(nil)
