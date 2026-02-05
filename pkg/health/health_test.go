package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockDaprClient is a test-local mock that implements dapr.Client
// by embedding a minimal mock for GetMetadata (the only method health checks use).
type mockDaprClient struct {
	mock.Mock
	dapr.Client // Embed interface to satisfy all methods; panics if unused methods are called
}

func (m *mockDaprClient) GetMetadata(ctx context.Context) (*dapr.GetMetadataResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dapr.GetMetadataResponse), args.Error(1)
}

func TestNewHandler(t *testing.T) {
	mockClient := new(mockDaprClient)
	handler := NewHandler(mockClient, "v1.0.0")

	assert.NotNil(t, handler)
	assert.Equal(t, "v1.0.0", handler.version)
	assert.True(t, handler.ready.Load())
	assert.True(t, handler.startupDone.Load())
}

func TestNewChecker(t *testing.T) {
	mockClient := new(mockDaprClient)
	handler := NewChecker(mockClient, nil)

	assert.NotNil(t, handler)
	assert.True(t, handler.ready.Load())
	assert.True(t, handler.startupDone.Load())
}

func TestSetReady(t *testing.T) {
	handler := &Handler{}
	handler.ready.Store(true)

	handler.SetReady(false)
	assert.False(t, handler.ready.Load())

	handler.SetReady(true)
	assert.True(t, handler.ready.Load())
}

func TestSetStartupDone(t *testing.T) {
	handler := &Handler{}
	handler.startupDone.Store(false)

	handler.SetStartupDone(true)
	assert.True(t, handler.startupDone.Load())

	handler.SetStartupDone(false)
	assert.False(t, handler.startupDone.Load())
}

func TestLivenessHandler(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		expectedStatus int
		expectedBody   HealthResponse
	}{
		{
			name:           "healthy liveness",
			version:        "v1.0.0",
			expectedStatus: http.StatusOK,
			expectedBody: HealthResponse{
				Status:  StatusHealthy,
				Version: "v1.0.0",
			},
		},
		{
			name:           "healthy liveness without version",
			version:        "",
			expectedStatus: http.StatusOK,
			expectedBody: HealthResponse{
				Status:  StatusHealthy,
				Version: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil, tt.version)

			req := httptest.NewRequest(http.MethodGet, "/livez", nil)
			rec := httptest.NewRecorder()

			handler.LivenessHandler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var resp HealthResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody.Status, resp.Status)
			assert.Equal(t, tt.expectedBody.Version, resp.Version)
		})
	}
}

func TestReadinessHandler(t *testing.T) {
	tests := []struct {
		name           string
		ready          bool
		setupMock      func(*mockDaprClient)
		expectedStatus int
		expectedHealth Status
	}{
		{
			name:  "healthy with dapr connected",
			ready: true,
			setupMock: func(m *mockDaprClient) {
				m.On("GetMetadata", mock.Anything).Return(&dapr.GetMetadataResponse{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedHealth: StatusHealthy,
		},
		{
			name:  "degraded when dapr unavailable",
			ready: true,
			setupMock: func(m *mockDaprClient) {
				m.On("GetMetadata", mock.Anything).Return(nil, errors.New("connection refused"))
			},
			expectedStatus: http.StatusOK,
			expectedHealth: StatusDegraded,
		},
		{
			name:           "unhealthy when not ready",
			ready:          false,
			setupMock:      nil, // No mock - we're testing the ready flag, not dapr connectivity
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: StatusUnhealthy,
		},
		{
			name:           "healthy without dapr client",
			ready:          true,
			setupMock:      nil,
			expectedStatus: http.StatusOK,
			expectedHealth: StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.setupMock != nil {
				mockClient := new(mockDaprClient)
				tt.setupMock(mockClient)
				handler = NewHandler(mockClient, "v1.0.0")
			} else {
				handler = NewHandler(nil, "v1.0.0")
			}
			handler.SetReady(tt.ready)

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()

			handler.ReadinessHandler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var resp HealthResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedHealth, resp.Status)
		})
	}
}

func TestStartupHandler(t *testing.T) {
	tests := []struct {
		name           string
		startupDone    bool
		expectedStatus int
		expectedHealth Status
	}{
		{
			name:           "startup complete",
			startupDone:    true,
			expectedStatus: http.StatusOK,
			expectedHealth: StatusHealthy,
		},
		{
			name:           "startup not complete",
			startupDone:    false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil, "v1.0.0")
			handler.SetStartupDone(tt.startupDone)

			req := httptest.NewRequest(http.MethodGet, "/startupz", nil)
			rec := httptest.NewRecorder()

			handler.StartupHandler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var resp HealthResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedHealth, resp.Status)
		})
	}
}

func TestCheckDapr(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mockDaprClient)
		nilClient      bool
		expectedStatus Status
		expectedMsg    string
	}{
		{
			name: "dapr healthy",
			setupMock: func(m *mockDaprClient) {
				m.On("GetMetadata", mock.Anything).Return(&dapr.GetMetadataResponse{}, nil)
			},
			expectedStatus: StatusHealthy,
			expectedMsg:    "sidecar connected",
		},
		{
			name: "dapr unhealthy",
			setupMock: func(m *mockDaprClient) {
				m.On("GetMetadata", mock.Anything).Return(nil, errors.New("sidecar not ready"))
			},
			expectedStatus: StatusUnhealthy,
			expectedMsg:    "sidecar not ready",
		},
		{
			name:           "nil client",
			nilClient:      true,
			expectedStatus: StatusUnhealthy,
			expectedMsg:    "dapr client not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *Handler
			if tt.nilClient {
				handler = &Handler{daprClient: nil}
			} else {
				mockClient := new(mockDaprClient)
				tt.setupMock(mockClient)
				handler = &Handler{daprClient: mockClient}
			}

			result := handler.checkDapr(context.Background())

			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Equal(t, "dapr", result.Component)
			assert.Contains(t, result.Message, tt.expectedMsg)
		})
	}
}

func TestRegisterHandlers(t *testing.T) {
	handler := NewHandler(nil, "v1.0.0")
	mux := http.NewServeMux()

	handler.RegisterHandlers(mux)

	// Test that handlers are registered by making requests
	tests := []struct {
		path   string
		status int
	}{
		{"/livez", http.StatusOK},
		{"/readyz", http.StatusOK},
		{"/startupz", http.StatusOK},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, tt.status, rec.Code, "path: %s", tt.path)
	}
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, Status("healthy"), StatusHealthy)
	assert.Equal(t, Status("unhealthy"), StatusUnhealthy)
	assert.Equal(t, Status("degraded"), StatusDegraded)
}

func TestCheckResultStruct(t *testing.T) {
	result := CheckResult{
		Status:    StatusHealthy,
		Component: "test",
		Message:   "all good",
		Latency:   "10ms",
	}

	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, "test", result.Component)
	assert.Equal(t, "all good", result.Message)
	assert.Equal(t, "10ms", result.Latency)
}

func TestHealthResponseStruct(t *testing.T) {
	resp := HealthResponse{
		Status:  StatusHealthy,
		Version: "v1.0.0",
		Checks: []CheckResult{
			{Status: StatusHealthy, Component: "test"},
		},
	}

	assert.Equal(t, StatusHealthy, resp.Status)
	assert.Equal(t, "v1.0.0", resp.Version)
	assert.Len(t, resp.Checks, 1)
}

func TestReadinessHandlerChecksContent(t *testing.T) {
	mockClient := new(mockDaprClient)
	mockClient.On("GetMetadata", mock.Anything).Return(&dapr.GetMetadataResponse{}, nil)

	handler := NewHandler(mockClient, "v1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ReadinessHandler(rec, req)

	var resp HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Should have dapr check in the checks list
	assert.Len(t, resp.Checks, 1)
	assert.Equal(t, "dapr", resp.Checks[0].Component)
	assert.Equal(t, StatusHealthy, resp.Checks[0].Status)
	assert.NotEmpty(t, resp.Checks[0].Latency)
}
