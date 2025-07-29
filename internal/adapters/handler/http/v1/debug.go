// File: internal/adapters/handler/http/v1/debug.go
package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cryptomarket/internal/core/port"

	"github.com/redis/go-redis/v9"
)

type DebugHandler struct {
	cache       port.Cache
	redisClient *redis.Client
}

func NewDebugHandler(cache port.Cache, redisClient *redis.Client) *DebugHandler {
	return &DebugHandler{
		cache:       cache,
		redisClient: redisClient,
	}
}

type DebugResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GET /debug/redis/keys - Show all Redis keys
func (h *DebugHandler) GetRedisKeys(w http.ResponseWriter, r *http.Request) {
	if h.redisClient == nil {
		h.writeResponse(w, http.StatusServiceUnavailable, "Redis client not available", nil)
		return
	}

	ctx := context.Background()

	// Get all keys
	allKeys, err := h.redisClient.Keys(ctx, "*").Result()
	if err != nil {
		h.writeResponse(w, http.StatusInternalServerError, "Failed to get Redis keys", nil)
		return
	}

	// Organize keys by type
	keyTypes := map[string][]string{
		"latest":     []string{},
		"timeseries": []string{},
		"other":      []string{},
	}

	for _, key := range allKeys {
		if strings.HasPrefix(key, "latest:") {
			keyTypes["latest"] = append(keyTypes["latest"], key)
		} else if strings.HasPrefix(key, "timeseries:") {
			keyTypes["timeseries"] = append(keyTypes["timeseries"], key)
		} else {
			keyTypes["other"] = append(keyTypes["other"], key)
		}
	}

	data := map[string]interface{}{
		"total_keys":   len(allKeys),
		"keys_by_type": keyTypes,
		"all_keys":     allKeys,
	}

	h.writeResponse(w, http.StatusOK, fmt.Sprintf("Found %d Redis keys", len(allKeys)), data)
}

// GET /debug/redis/symbol/{symbol} - Show data for specific symbol
func (h *DebugHandler) GetSymbolData(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	if symbol == "" {
		h.writeResponse(w, http.StatusBadRequest, "Missing symbol parameter", nil)
		return
	}

	symbol = strings.ToUpper(symbol)

	if h.redisClient == nil {
		h.writeResponse(w, http.StatusServiceUnavailable, "Redis client not available", nil)
		return
	}

	ctx := context.Background()

	// Get latest price keys for this symbol
	latestPattern := fmt.Sprintf("latest:%s:*", symbol)
	latestKeys, err := h.redisClient.Keys(ctx, latestPattern).Result()
	if err != nil {
		h.writeResponse(w, http.StatusInternalServerError, "Failed to get latest keys", nil)
		return
	}

	// Get timeseries keys for this symbol
	timeseriesPattern := fmt.Sprintf("timeseries:%s:*", symbol)
	timeseriesKeys, err := h.redisClient.Keys(ctx, timeseriesPattern).Result()
	if err != nil {
		h.writeResponse(w, http.StatusInternalServerError, "Failed to get timeseries keys", nil)
		return
	}

	// Get sample data from latest keys
	latestData := make(map[string]interface{})
	for _, key := range latestKeys {
		value, err := h.redisClient.Get(ctx, key).Result()
		if err == nil {
			latestData[key] = value
		}
	}

	// Get sample data from timeseries keys (last 10 entries)
	timeseriesData := make(map[string]interface{})
	for _, key := range timeseriesKeys {
		// Get last 10 entries from sorted set
		entries, err := h.redisClient.ZRevRangeWithScores(ctx, key, 0, 9).Result()
		if err == nil {
			timeseriesData[key] = entries
		}
	}

	data := map[string]interface{}{
		"symbol":          symbol,
		"latest_keys":     latestKeys,
		"timeseries_keys": timeseriesKeys,
		"latest_data":     latestData,
		"timeseries_data": timeseriesData,
	}

	h.writeResponse(w, http.StatusOK, fmt.Sprintf("Data for symbol %s", symbol), data)
}

// GET /debug/cache/test/{symbol} - Test cache methods directly
func (h *DebugHandler) TestCacheMethods(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	if symbol == "" {
		h.writeResponse(w, http.StatusBadRequest, "Missing symbol parameter", nil)
		return
	}

	symbol = strings.ToUpper(symbol)

	if h.cache == nil {
		h.writeResponse(w, http.StatusServiceUnavailable, "Cache not available", nil)
		return
	}

	ctx := context.Background()
	results := make(map[string]interface{})

	// Test GetLatestPrice
	latest, err := h.cache.GetLatestPrice(ctx, symbol)
	if err != nil {
		results["latest_price_error"] = err.Error()
	} else {
		results["latest_price"] = latest
	}

	// Test GetPricesInRange for last 5 minutes
	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)

	rangeData, err := h.cache.GetPricesInRange(ctx, symbol, fiveMinutesAgo, now)
	if err != nil {
		results["range_data_error"] = err.Error()
	} else {
		results["range_data_count"] = len(rangeData)
		if len(rangeData) > 0 {
			results["range_data_sample"] = rangeData[0:min(3, len(rangeData))] // First 3 entries
		}
	}

	// Test specific exchange
	exchangeData, err := h.cache.GetPricesInRangeByExchange(ctx, symbol, "exchange1", fiveMinutesAgo, now)
	if err != nil {
		results["exchange_data_error"] = err.Error()
	} else {
		results["exchange_data_count"] = len(exchangeData)
		if len(exchangeData) > 0 {
			results["exchange_data_sample"] = exchangeData[0:min(3, len(exchangeData))]
		}
	}

	data := map[string]interface{}{
		"symbol": symbol,
		"time_range": map[string]int64{
			"from": fiveMinutesAgo.Unix(),
			"to":   now.Unix(),
		},
		"results": results,
	}

	h.writeResponse(w, http.StatusOK, fmt.Sprintf("Cache test results for %s", symbol), data)
}

func (h *DebugHandler) writeResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := DebugResponse{
		Message: message,
		Data:    data,
	}

	if statusCode >= 400 {
		response.Error = message
		response.Message = ""
	}

	json.NewEncoder(w).Encode(response)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
