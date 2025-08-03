package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"tiktok-whisper/internal/api/errors"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/api/v1/handlers"
	"tiktok-whisper/internal/app/testutil"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *testutil.MockServices) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mockServices := testutil.NewMockServices(t)
	return router, mockServices
}

func TestTranscriptionHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		request        dto.CreateTranscriptionRequest
		setupMocks     func(*testutil.MockServices)
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful transcription creation",
			request: dto.CreateTranscriptionRequest{
				FilePath: "/tmp/test.mp3",
				Provider: "openai/whisper",
				Language: "en",
			},
			setupMocks: func(ms *testutil.MockServices) {
				ms.TranscriptionService.On("CreateTranscription", mock.Anything, mock.Anything).
					Return(&dto.TranscriptionResponse{
						ID:        1,
						FilePath:  "/tmp/test.mp3",
						Status:    "pending",
						Provider:  "openai/whisper",
						CreatedAt: time.Now(),
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(1), body["id"])
				assert.Equal(t, "pending", body["status"])
				assert.Equal(t, "/tmp/test.mp3", body["file_path"])
			},
		},
		{
			name: "validation error - missing file path",
			request: dto.CreateTranscriptionRequest{
				Provider: "openai/whisper",
			},
			setupMocks:     func(ms *testutil.MockServices) {},
			expectedStatus: http.StatusUnprocessableEntity,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "validation", body["kind"])
				assert.NotNil(t, body["details"])
			},
		},
		{
			name: "invalid provider",
			request: dto.CreateTranscriptionRequest{
				FilePath: "/tmp/test.mp3",
				Provider: "invalid_provider",
			},
			setupMocks:     func(ms *testutil.MockServices) {},
			expectedStatus: http.StatusUnprocessableEntity,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "validation", body["kind"])
				details := body["details"].(map[string]interface{})
				assert.Contains(t, details["provider"], "invalid")
			},
		},
		{
			name: "service error",
			request: dto.CreateTranscriptionRequest{
				FilePath: "/tmp/test.mp3",
			},
			setupMocks: func(ms *testutil.MockServices) {
				ms.TranscriptionService.On("CreateTranscription", mock.Anything, mock.Anything).
					Return(nil, errors.NewInternalError("service unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "internal", body["kind"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mockServices := setupTestRouter(t)
			tt.setupMocks(mockServices)

			handler := handlers.NewTranscriptionHandler(mockServices.TranscriptionService)
			router.POST("/api/v1/transcriptions", handler.Create)

			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/v1/transcriptions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var responseBody map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			tt.validateBody(t, responseBody)
		})
	}
}

func TestTranscriptionHandler_Get(t *testing.T) {
	tests := []struct {
		name           string
		transcriptionID string
		setupMocks     func(*testutil.MockServices)
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name:            "successful get",
			transcriptionID: "123",
			setupMocks: func(ms *testutil.MockServices) {
				ms.TranscriptionService.On("GetTranscription", mock.Anything, 123).
					Return(&dto.TranscriptionResponse{
						ID:            123,
						FilePath:      "/tmp/test.mp3",
						Status:        "completed",
						Transcription: "Hello world",
						Duration:      10.5,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(123), body["id"])
				assert.Equal(t, "completed", body["status"])
				assert.Equal(t, "Hello world", body["transcription"])
			},
		},
		{
			name:            "not found",
			transcriptionID: "999",
			setupMocks: func(ms *testutil.MockServices) {
				ms.TranscriptionService.On("GetTranscription", mock.Anything, 999).
					Return(nil, errors.NewNotFoundError("transcription"))
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "not_found", body["kind"])
			},
		},
		{
			name:            "invalid ID",
			transcriptionID: "abc",
			setupMocks:     func(ms *testutil.MockServices) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "bad_request", body["kind"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mockServices := setupTestRouter(t)
			tt.setupMocks(mockServices)

			handler := handlers.NewTranscriptionHandler(mockServices.TranscriptionService)
			router.GET("/api/v1/transcriptions/:id", handler.Get)

			req := httptest.NewRequest("GET", "/api/v1/transcriptions/"+tt.transcriptionID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var responseBody map[string]interface{}
			err := json.Unmarshal(rec.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			tt.validateBody(t, responseBody)
		})
	}
}

func TestTranscriptionHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMocks     func(*testutil.MockServices)
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name:        "successful list with pagination",
			queryParams: "?page=1&limit=10",
			setupMocks: func(ms *testutil.MockServices) {
				ms.TranscriptionService.On("ListTranscriptions", mock.Anything, mock.Anything).
					Return(&dto.PaginatedTranscriptionsResponse{
						Transcriptions: []dto.TranscriptionResponse{
							{ID: 1, Status: "completed"},
							{ID: 2, Status: "processing"},
						},
						Pagination: dto.PaginationResponse{
							Page:       1,
							Limit:      10,
							Total:      2,
							TotalPages: 1,
							HasNext:    false,
							HasPrev:    false,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				transcriptions := body["transcriptions"].([]interface{})
				assert.Len(t, transcriptions, 2)
				
				pagination := body["pagination"].(map[string]interface{})
				assert.Equal(t, float64(1), pagination["page"])
				assert.Equal(t, float64(10), pagination["limit"])
				assert.Equal(t, float64(2), pagination["total"])
			},
		},
		{
			name:        "filter by status",
			queryParams: "?status=completed",
			setupMocks: func(ms *testutil.MockServices) {
				ms.TranscriptionService.On("ListTranscriptions", mock.Anything, mock.MatchedBy(func(query dto.ListTranscriptionsQuery) bool {
					return query.Status == "completed"
				})).Return(&dto.PaginatedTranscriptionsResponse{
					Transcriptions: []dto.TranscriptionResponse{
						{ID: 1, Status: "completed"},
					},
					Pagination: dto.PaginationResponse{
						Page: 1, Limit: 20, Total: 1, TotalPages: 1,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				transcriptions := body["transcriptions"].([]interface{})
				assert.Len(t, transcriptions, 1)
			},
		},
		{
			name:        "invalid query parameters",
			queryParams: "?page=0&limit=200",
			setupMocks:  func(ms *testutil.MockServices) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "bad_request", body["kind"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mockServices := setupTestRouter(t)
			tt.setupMocks(mockServices)

			handler := handlers.NewTranscriptionHandler(mockServices.TranscriptionService)
			router.GET("/api/v1/transcriptions", handler.List)

			req := httptest.NewRequest("GET", "/api/v1/transcriptions"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var responseBody map[string]interface{}
			err := json.Unmarshal(rec.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			tt.validateBody(t, responseBody)
		})
	}
}

func TestTranscriptionHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		transcriptionID string
		setupMocks     func(*testutil.MockServices)
		expectedStatus int
	}{
		{
			name:            "successful delete",
			transcriptionID: "123",
			setupMocks: func(ms *testutil.MockServices) {
				ms.TranscriptionService.On("DeleteTranscription", mock.Anything, 123).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:            "not found",
			transcriptionID: "999",
			setupMocks: func(ms *testutil.MockServices) {
				ms.TranscriptionService.On("DeleteTranscription", mock.Anything, 999).
					Return(errors.NewNotFoundError("transcription"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:            "invalid ID",
			transcriptionID: "abc",
			setupMocks:     func(ms *testutil.MockServices) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mockServices := setupTestRouter(t)
			tt.setupMocks(mockServices)

			handler := handlers.NewTranscriptionHandler(mockServices.TranscriptionService)
			router.DELETE("/api/v1/transcriptions/:id", handler.Delete)

			req := httptest.NewRequest("DELETE", "/api/v1/transcriptions/"+tt.transcriptionID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}