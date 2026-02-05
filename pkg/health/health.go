// Package health provides health check endpoints for Kubernetes probes.
package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	dapr "github.com/dapr/go-sdk/client"
)

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult represents the result of a health check.
type CheckResult struct {
	Status    Status `json:"status"`
	Component string `json:"component,omitempty"`
	Message   string `json:"message,omitempty"`
	Latency   string `json:"latency,omitempty"`
}

// HealthResponse is the response from health endpoints.
type HealthResponse struct {
	Status  Status        `json:"status"`
	Checks  []CheckResult `json:"checks,omitempty"`
	Version string        `json:"version,omitempty"`
}

// Handler provides health check HTTP handlers.
type Handler struct {
	daprClient  dapr.Client
	logger      *slog.Logger
	version     string
	ready       atomic.Bool
	startupDone atomic.Bool
}

// NewHandler creates a new health handler.
func NewHandler(daprClient dapr.Client, version string) *Handler {
	h := &Handler{
		daprClient: daprClient,
		version:    version,
	}
	h.ready.Store(true)
	h.startupDone.Store(true)
	return h
}

// NewChecker creates a new health checker with logging support.
func NewChecker(daprClient dapr.Client, logger *slog.Logger) *Handler {
	h := &Handler{
		daprClient: daprClient,
		logger:     logger,
	}
	h.ready.Store(true)
	h.startupDone.Store(true)
	return h
}

// SetReady sets the readiness state.
func (h *Handler) SetReady(ready bool) {
	h.ready.Store(ready)
}

// SetStartupDone sets the startup completion state.
func (h *Handler) SetStartupDone(done bool) {
	h.startupDone.Store(done)
}

// LivenessHandler handles /livez requests.
// Liveness probes should be simple - just check if the server is running.
func (h *Handler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:  StatusHealthy,
		Version: h.version,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// ReadinessHandler handles /readyz requests.
// Readiness probes check if the service can accept traffic.
func (h *Handler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	checks := make([]CheckResult, 0)
	overallStatus := StatusHealthy
	httpStatus := http.StatusOK

	// Check if marked as ready
	if !h.ready.Load() {
		overallStatus = StatusUnhealthy
		httpStatus = http.StatusServiceUnavailable
		checks = append(checks, CheckResult{
			Status:    StatusUnhealthy,
			Component: "server",
			Message:   "server not ready",
		})
	}

	// Check Dapr sidecar connectivity
	if h.daprClient != nil {
		daprCheck := h.checkDapr(r.Context())
		checks = append(checks, daprCheck)
		if daprCheck.Status != StatusHealthy {
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}
	}

	resp := HealthResponse{
		Status:  overallStatus,
		Checks:  checks,
		Version: h.version,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

// StartupHandler handles /startupz requests.
// Startup probes check if the application has finished initialization.
func (h *Handler) StartupHandler(w http.ResponseWriter, r *http.Request) {
	status := StatusHealthy
	httpStatus := http.StatusOK

	if !h.startupDone.Load() {
		status = StatusUnhealthy
		httpStatus = http.StatusServiceUnavailable
	}

	resp := HealthResponse{
		Status:  status,
		Version: h.version,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

// checkDapr checks the Dapr sidecar connectivity.
func (h *Handler) checkDapr(ctx context.Context) CheckResult {
	if h.daprClient == nil {
		return CheckResult{
			Status:    StatusUnhealthy,
			Component: "dapr",
			Message:   "dapr client not initialized",
		}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := h.daprClient.GetMetadata(ctx)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:    StatusUnhealthy,
			Component: "dapr",
			Message:   err.Error(),
			Latency:   latency.String(),
		}
	}

	return CheckResult{
		Status:    StatusHealthy,
		Component: "dapr",
		Message:   "sidecar connected",
		Latency:   latency.String(),
	}
}

// RegisterHandlers registers health check handlers with the given mux.
func (h *Handler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/livez", h.LivenessHandler)
	mux.HandleFunc("/readyz", h.ReadinessHandler)
	mux.HandleFunc("/startupz", h.StartupHandler)
}
