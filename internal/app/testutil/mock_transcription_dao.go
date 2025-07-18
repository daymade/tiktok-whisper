package testutil

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
)

// MockTranscriptionDAO is a comprehensive mock implementation of the repository.TranscriptionDAO interface
// It provides in-memory storage and configurable behavior for testing various database scenarios
type MockTranscriptionDAO struct {
	mock.Mock
	mu sync.RWMutex

	// In-memory storage
	transcriptions map[int]*model.Transcription
	nextID         int
	fileIndex      map[string]int // filename -> transcription ID

	// Configuration options
	DefaultError     error
	EnableCallTracking bool
	EnableRealistic   bool
	SimulateLatency   time.Duration

	// State tracking
	CallCount   int
	CallHistory []DAOCall
	ErrorMap    map[string]error // method -> error
	LatencyMap  map[string]time.Duration // method -> latency

	// Transaction simulation
	InTransaction bool
	TransactionCommitted bool
	TransactionRolledBack bool
}

// DAOCall represents a single DAO method call for tracking
type DAOCall struct {
	Method    string
	Arguments []interface{}
	Result    interface{}
	Error     error
	Timestamp time.Time
	Duration  time.Duration
}

// NewMockTranscriptionDAO creates a new MockTranscriptionDAO with sensible defaults
func NewMockTranscriptionDAO() *MockTranscriptionDAO {
	return &MockTranscriptionDAO{
		transcriptions:     make(map[int]*model.Transcription),
		nextID:            1,
		fileIndex:         make(map[string]int),
		EnableCallTracking: true,
		EnableRealistic:    true,
		SimulateLatency:    5 * time.Millisecond,
		CallHistory:       make([]DAOCall, 0),
		ErrorMap:          make(map[string]error),
		LatencyMap:        make(map[string]time.Duration),
	}
}

// Close implements the TranscriptionDAO interface
func (m *MockTranscriptionDAO) Close() error {
	startTime := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	m.trackCall("Close", nil, nil, nil, startTime)

	// Check for method-specific error
	if err, exists := m.ErrorMap["Close"]; exists {
		return err
	}

	// Check for default error
	if m.DefaultError != nil {
		return m.DefaultError
	}

	// Simulate latency
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	// Call testify mock if expected
	args := m.Called()
	if args.Get(0) != nil {
		return args.Error(0)
	}

	return nil
}

// GetAllByUser implements the TranscriptionDAO interface
func (m *MockTranscriptionDAO) GetAllByUser(userNickname string) ([]model.Transcription, error) {
	startTime := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for method-specific error
	if err, exists := m.ErrorMap["GetAllByUser"]; exists {
		m.trackCall("GetAllByUser", []interface{}{userNickname}, nil, err, startTime)
		return nil, err
	}

	// Check for default error
	if m.DefaultError != nil {
		m.trackCall("GetAllByUser", []interface{}{userNickname}, nil, m.DefaultError, startTime)
		return nil, m.DefaultError
	}

	// Simulate latency
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	// Filter transcriptions by user
	var result []model.Transcription
	for _, transcription := range m.transcriptions {
		if transcription.User == userNickname {
			result = append(result, *transcription)
		}
	}

	// Sort by ID for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	m.trackCall("GetAllByUser", []interface{}{userNickname}, result, nil, startTime)

	// Call testify mock if expected
	args := m.Called(userNickname)
	if args.Get(0) != nil {
		return args.Get(0).([]model.Transcription), args.Error(1)
	}

	return result, nil
}

// CheckIfFileProcessed implements the TranscriptionDAO interface
func (m *MockTranscriptionDAO) CheckIfFileProcessed(fileName string) (int, error) {
	startTime := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for method-specific error
	if err, exists := m.ErrorMap["CheckIfFileProcessed"]; exists {
		m.trackCall("CheckIfFileProcessed", []interface{}{fileName}, 0, err, startTime)
		return 0, err
	}

	// Check for default error
	if m.DefaultError != nil {
		m.trackCall("CheckIfFileProcessed", []interface{}{fileName}, 0, m.DefaultError, startTime)
		return 0, m.DefaultError
	}

	// Simulate latency
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	// Check if file exists in our index
	if transcriptionID, exists := m.fileIndex[fileName]; exists {
		m.trackCall("CheckIfFileProcessed", []interface{}{fileName}, transcriptionID, nil, startTime)
		
		// Call testify mock if expected
		args := m.Called(fileName)
		if args.Get(0) != nil {
			return args.Int(0), args.Error(1)
		}
		
		return transcriptionID, nil
	}

	m.trackCall("CheckIfFileProcessed", []interface{}{fileName}, 0, nil, startTime)

	// Call testify mock if expected
	args := m.Called(fileName)
	if args.Get(0) != nil {
		return args.Int(0), args.Error(1)
	}

	return 0, nil
}

// RecordToDB implements the TranscriptionDAO interface
func (m *MockTranscriptionDAO) RecordToDB(user, inputDir, fileName, mp3FileName string, audioDuration int, transcription string,
	lastConversionTime time.Time, hasError int, errorMessage string) {
	startTime := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	args := []interface{}{user, inputDir, fileName, mp3FileName, audioDuration, transcription, lastConversionTime, hasError, errorMessage}

	// Check for method-specific error
	if err, exists := m.ErrorMap["RecordToDB"]; exists {
		m.trackCall("RecordToDB", args, nil, err, startTime)
		return
	}

	// Check for default error (note: this method doesn't return an error in the interface)
	if m.DefaultError != nil {
		m.trackCall("RecordToDB", args, nil, m.DefaultError, startTime)
		return
	}

	// Simulate latency
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	// Create new transcription record
	transcriptionRecord := &model.Transcription{
		ID:                 m.nextID,
		User:               user,
		LastConversionTime: lastConversionTime,
		Mp3FileName:        mp3FileName,
		AudioDuration:      float64(audioDuration),
		Transcription:      transcription,
		ErrorMessage:       errorMessage,
	}

	// Store in memory
	m.transcriptions[m.nextID] = transcriptionRecord
	m.fileIndex[fileName] = m.nextID
	m.nextID++

	m.trackCall("RecordToDB", args, transcriptionRecord.ID, nil, startTime)

	// Call testify mock if expected
	m.Called(user, inputDir, fileName, mp3FileName, audioDuration, transcription, lastConversionTime, hasError, errorMessage)
}

// Configuration Methods

// WithDefaultError sets the default error to return for all methods
func (m *MockTranscriptionDAO) WithDefaultError(err error) *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DefaultError = err
	return m
}

// WithCallTracking enables/disables call history tracking
func (m *MockTranscriptionDAO) WithCallTracking(enabled bool) *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EnableCallTracking = enabled
	return m
}

// WithRealistic enables/disables realistic behavior simulation
func (m *MockTranscriptionDAO) WithRealistic(enabled bool) *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EnableRealistic = enabled
	return m
}

// WithLatency sets the simulated latency for all operations
func (m *MockTranscriptionDAO) WithLatency(latency time.Duration) *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SimulateLatency = latency
	return m
}

// Method-specific Configuration

// SetErrorForMethod sets a specific error for a given method
func (m *MockTranscriptionDAO) SetErrorForMethod(method string, err error) *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorMap[method] = err
	return m
}

// SetLatencyForMethod sets a specific latency for a given method
func (m *MockTranscriptionDAO) SetLatencyForMethod(method string, latency time.Duration) *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LatencyMap[method] = latency
	return m
}

// Common Error Scenarios

// SimulateDatabaseConnectionError simulates database connection errors
func (m *MockTranscriptionDAO) SimulateDatabaseConnectionError() *MockTranscriptionDAO {
	return m.WithDefaultError(fmt.Errorf("database connection failed"))
}

// SimulateConstraintViolation simulates database constraint violations
func (m *MockTranscriptionDAO) SimulateConstraintViolation() *MockTranscriptionDAO {
	return m.SetErrorForMethod("RecordToDB", fmt.Errorf("constraint violation: duplicate key"))
}

// SimulateQueryTimeout simulates query timeout errors
func (m *MockTranscriptionDAO) SimulateQueryTimeout() *MockTranscriptionDAO {
	return m.SetErrorForMethod("GetAllByUser", fmt.Errorf("query timeout"))
}

// SimulateTableNotFound simulates table not found errors
func (m *MockTranscriptionDAO) SimulateTableNotFound() *MockTranscriptionDAO {
	return m.WithDefaultError(fmt.Errorf("table 'transcriptions' doesn't exist"))
}

// Data Management Methods

// AddTranscription adds a transcription record to the in-memory storage
func (m *MockTranscriptionDAO) AddTranscription(transcription *model.Transcription) *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if transcription.ID == 0 {
		transcription.ID = m.nextID
		m.nextID++
	}
	
	m.transcriptions[transcription.ID] = transcription
	if transcription.Mp3FileName != "" {
		m.fileIndex[transcription.Mp3FileName] = transcription.ID
	}
	
	return m
}

// AddTranscriptions adds multiple transcription records
func (m *MockTranscriptionDAO) AddTranscriptions(transcriptions []*model.Transcription) *MockTranscriptionDAO {
	for _, transcription := range transcriptions {
		m.AddTranscription(transcription)
	}
	return m
}

// RemoveTranscription removes a transcription record by ID
func (m *MockTranscriptionDAO) RemoveTranscription(id int) *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if transcription, exists := m.transcriptions[id]; exists {
		// Remove from file index
		for fileName, transcriptionID := range m.fileIndex {
			if transcriptionID == id {
				delete(m.fileIndex, fileName)
				break
			}
		}
		delete(m.transcriptions, id)
	}
	
	return m
}

// GetTranscriptionByID retrieves a transcription by ID
func (m *MockTranscriptionDAO) GetTranscriptionByID(id int) *model.Transcription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if transcription, exists := m.transcriptions[id]; exists {
		// Return a copy to prevent external modification
		copy := *transcription
		return &copy
	}
	
	return nil
}

// GetAllTranscriptions returns all transcriptions
func (m *MockTranscriptionDAO) GetAllTranscriptions() []*model.Transcription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var transcriptions []*model.Transcription
	for _, transcription := range m.transcriptions {
		copy := *transcription
		transcriptions = append(transcriptions, &copy)
	}
	
	// Sort by ID for consistent ordering
	sort.Slice(transcriptions, func(i, j int) bool {
		return transcriptions[i].ID < transcriptions[j].ID
	})
	
	return transcriptions
}

// GetTranscriptionCount returns the total number of transcriptions
func (m *MockTranscriptionDAO) GetTranscriptionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.transcriptions)
}

// GetUserCount returns the number of unique users
func (m *MockTranscriptionDAO) GetUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	users := make(map[string]bool)
	for _, transcription := range m.transcriptions {
		users[transcription.User] = true
	}
	
	return len(users)
}

// GetUsers returns all unique users
func (m *MockTranscriptionDAO) GetUsers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	users := make(map[string]bool)
	for _, transcription := range m.transcriptions {
		users[transcription.User] = true
	}
	
	var userList []string
	for user := range users {
		userList = append(userList, user)
	}
	
	sort.Strings(userList)
	return userList
}

// State Inspection Methods

// GetCallCount returns the total number of method calls
func (m *MockTranscriptionDAO) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CallCount
}

// GetCallHistory returns the complete call history
func (m *MockTranscriptionDAO) GetCallHistory() []DAOCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history := make([]DAOCall, len(m.CallHistory))
	copy(history, m.CallHistory)
	return history
}

// GetCallsForMethod returns all calls for a specific method
func (m *MockTranscriptionDAO) GetCallsForMethod(method string) []DAOCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var calls []DAOCall
	for _, call := range m.CallHistory {
		if call.Method == method {
			calls = append(calls, call)
		}
	}
	
	return calls
}

// WasMethodCalled checks if a specific method was called
func (m *MockTranscriptionDAO) WasMethodCalled(method string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, call := range m.CallHistory {
		if call.Method == method {
			return true
		}
	}
	
	return false
}

// WasMethodCalledWith checks if a method was called with specific arguments
func (m *MockTranscriptionDAO) WasMethodCalledWith(method string, args ...interface{}) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, call := range m.CallHistory {
		if call.Method == method && len(call.Arguments) == len(args) {
			match := true
			for i, arg := range args {
				if call.Arguments[i] != arg {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	
	return false
}

// GetLastCall returns the last method call
func (m *MockTranscriptionDAO) GetLastCall() *DAOCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.CallHistory) == 0 {
		return nil
	}
	
	return &m.CallHistory[len(m.CallHistory)-1]
}

// GetAverageLatency returns the average method call latency
func (m *MockTranscriptionDAO) GetAverageLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.CallHistory) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, call := range m.CallHistory {
		total += call.Duration
	}
	
	return total / time.Duration(len(m.CallHistory))
}

// Reset Methods

// Reset clears all state and returns to default configuration
func (m *MockTranscriptionDAO) Reset() *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.transcriptions = make(map[int]*model.Transcription)
	m.nextID = 1
	m.fileIndex = make(map[string]int)
	m.CallCount = 0
	m.CallHistory = make([]DAOCall, 0)
	m.ErrorMap = make(map[string]error)
	m.LatencyMap = make(map[string]time.Duration)
	m.DefaultError = nil
	m.EnableCallTracking = true
	m.EnableRealistic = true
	m.SimulateLatency = 5 * time.Millisecond
	m.InTransaction = false
	m.TransactionCommitted = false
	m.TransactionRolledBack = false
	
	return m
}

// ClearHistory clears call history but keeps data and configuration
func (m *MockTranscriptionDAO) ClearHistory() *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCount = 0
	m.CallHistory = make([]DAOCall, 0)
	return m
}

// ClearData clears all transcription data but keeps configuration
func (m *MockTranscriptionDAO) ClearData() *MockTranscriptionDAO {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transcriptions = make(map[int]*model.Transcription)
	m.nextID = 1
	m.fileIndex = make(map[string]int)
	return m
}

// Test Helper Methods

// ExpectClose sets up an expectation for Close method
func (m *MockTranscriptionDAO) ExpectClose(err error) *MockTranscriptionDAO {
	m.On("Close").Return(err)
	return m
}

// ExpectGetAllByUser sets up an expectation for GetAllByUser method
func (m *MockTranscriptionDAO) ExpectGetAllByUser(userNickname string, result []model.Transcription, err error) *MockTranscriptionDAO {
	m.On("GetAllByUser", userNickname).Return(result, err)
	return m
}

// ExpectCheckIfFileProcessed sets up an expectation for CheckIfFileProcessed method
func (m *MockTranscriptionDAO) ExpectCheckIfFileProcessed(fileName string, result int, err error) *MockTranscriptionDAO {
	m.On("CheckIfFileProcessed", fileName).Return(result, err)
	return m
}

// ExpectRecordToDB sets up an expectation for RecordToDB method
func (m *MockTranscriptionDAO) ExpectRecordToDB(user, inputDir, fileName, mp3FileName string, audioDuration int, transcription string,
	lastConversionTime time.Time, hasError int, errorMessage string) *MockTranscriptionDAO {
	m.On("RecordToDB", user, inputDir, fileName, mp3FileName, audioDuration, transcription, lastConversionTime, hasError, errorMessage).Return()
	return m
}

// Private helper methods

// trackCall records a method call in the history
func (m *MockTranscriptionDAO) trackCall(method string, args []interface{}, result interface{}, err error, startTime time.Time) {
	if !m.EnableCallTracking {
		return
	}
	
	m.CallCount++
	m.CallHistory = append(m.CallHistory, DAOCall{
		Method:    method,
		Arguments: args,
		Result:    result,
		Error:     err,
		Timestamp: startTime,
		Duration:  time.Since(startTime),
	})
}

// Interface compliance check
var _ repository.TranscriptionDAO = (*MockTranscriptionDAO)(nil)