// File: internal/adapters/handler/http/v1/modeHandler.go
package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cryptomarket/internal/core/domain"
	"cryptomarket/internal/core/port"
)

type ExchangeHandler struct {
	exchangeService port.ExchangeService
	ctx             context.Context
}

func NewExchangeHandler(
	exchangeService port.ExchangeService,
	ctx context.Context,
) *ExchangeHandler {
	return &ExchangeHandler{
		exchangeService: exchangeService,
		ctx:             ctx,
	}
}

type ModeResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SwitchToTestExchange handles POST /mode/test
func (h *ExchangeHandler) SwitchToTestExchange(w http.ResponseWriter, r *http.Request) {
	if h.exchangeService == nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "exchange service not available")
		return
	}

	// Use context with timeout
	ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()

	err := h.exchangeService.SwitchToTestMode(ctx)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "failed to switch to test mode: "+err.Error())
		return
	}

	response := ModeResponse{
		Status:  "success",
		Message: "switched to test mode successfully",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SwitchToLiveExchange handles POST /mode/live
func (h *ExchangeHandler) SwitchToLiveExchange(w http.ResponseWriter, r *http.Request) {
	if h.exchangeService == nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "exchange service not available")
		return
	}

	// Use context with timeout
	ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()

	err := h.exchangeService.SwitchToLiveMode(ctx)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "failed to switch to live mode: "+err.Error())
		return
	}

	response := ModeResponse{
		Status:  "success",
		Message: "switched to live mode successfully",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetCurrentExchange handles GET /mode/current
func (h *ExchangeHandler) GetCurrentExchange(w http.ResponseWriter, r *http.Request) {
	if h.exchangeService == nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "exchange service not available")
		return
	}

	mode := h.exchangeService.GetCurrentMode()
	stats := h.exchangeService.GetStats()

	// Convert adapter_names to string slice for JSON serialization
	var adapterNames []string
	if names, ok := stats["adapter_names"].([]string); ok {
		adapterNames = names
	}

	// Safely extract statistics with type assertions
	var (
		isRunning        bool
		activeAdapters   int
		healthyAdapters  int
		aggregatedBuffer int
		resultBuffer     int
	)

	if running, ok := stats["is_running"].(bool); ok {
		isRunning = running
	}
	if active, ok := stats["active_adapters"].(int); ok {
		activeAdapters = active
	}
	if healthy, ok := stats["healthy_adapters"].(int); ok {
		healthyAdapters = healthy
	}
	if aggBuffer, ok := stats["aggregated_buffer"].(int); ok {
		aggregatedBuffer = aggBuffer
	}
	if resBuffer, ok := stats["result_buffer"].(int); ok {
		resultBuffer = resBuffer
	}

	response := domain.SystemMode{
		CurrentMode:      mode,
		IsRunning:        isRunning,
		ActiveAdapters:   activeAdapters,
		HealthyAdapters:  healthyAdapters,
		AdapterNames:     adapterNames,
		AggregatedBuffer: aggregatedBuffer,
		ResultBuffer:     resultBuffer,
		Timestamp:        time.Now().Unix(),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper methods
func (h *ExchangeHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If we can't encode the response, send a simple error message
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal_error","message":"failed to encode response"}`))
	}
}

func (h *ExchangeHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	errorType := "internal_error"
	switch statusCode {
	case http.StatusServiceUnavailable:
		errorType = "service_unavailable"
	case http.StatusBadRequest:
		errorType = "bad_request"
	}

	response := map[string]interface{}{
		"error":     errorType,
		"message":   message,
		"timestamp": time.Now().Unix(),
	}

	h.writeJSONResponse(w, statusCode, response)
}
