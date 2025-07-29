package v1

import (
	"encoding/json"
	"net/http"

	"cryptomarket/internal/core/port"
)

type HealthHandler struct {
	healthService port.HealthService
}

func NewHealthHandler(
	healthService port.HealthService,
) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

type HealthResponse struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
	Timestamp  int64             `json:"timestamp"`
	Message    string            `json:"message,omitempty"`
}

// GetSystemHealth handles GET /health
func (h *HealthHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	if h.healthService == nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "health service not available")
		return
	}

	// Get system health status
	healthStatus, err := h.healthService.GetSystemHealth(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "failed to get system health: "+err.Error())
		return
	}

	// Prepare response
	response := HealthResponse{
		Status:     healthStatus.Status,
		Components: healthStatus.Components,
		Timestamp:  healthStatus.Timestamp,
		Message:    healthStatus.Message,
	}

	// Set appropriate HTTP status code based on health status
	statusCode := http.StatusOK
	switch healthStatus.Status {
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	case "degraded":
		statusCode = http.StatusOK // Still operational, but with warnings
	case "healthy":
		statusCode = http.StatusOK
	default:
		statusCode = http.StatusInternalServerError
	}

	h.writeJSONResponse(w, statusCode, response)
}

// GetDetailedHealth handles GET /health/detailed
func (h *HealthHandler) GetDetailedHealth(w http.ResponseWriter, r *http.Request) {
	if h.healthService == nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "health service not available")
		return
	}

	// Get detailed health status
	healthStatus, err := h.healthService.GetDetailedHealth(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "failed to get detailed health: "+err.Error())
		return
	}

	// Prepare response
	response := HealthResponse{
		Status:     healthStatus.Status,
		Components: healthStatus.Components,
		Timestamp:  healthStatus.Timestamp,
		Message:    healthStatus.Message,
	}

	// Set appropriate HTTP status code based on health status
	statusCode := http.StatusOK
	switch healthStatus.Status {
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	case "degraded":
		statusCode = http.StatusOK // Still operational, but with warnings
	case "healthy":
		statusCode = http.StatusOK
	default:
		statusCode = http.StatusInternalServerError
	}

	h.writeJSONResponse(w, statusCode, response)
}

// Helper methods
func (h *HealthHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If we can't encode the response, send a simple error message
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal_error","message":"failed to encode response"}`))
	}
}

func (h *HealthHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	errorType := "internal_error"
	switch statusCode {
	case http.StatusServiceUnavailable:
		errorType = "service_unavailable"
	case http.StatusBadRequest:
		errorType = "bad_request"
	}

	response := map[string]string{
		"error":   errorType,
		"message": message,
	}

	h.writeJSONResponse(w, statusCode, response)
}
