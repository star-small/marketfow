// File: internal/adapters/handler/http/v1/endpoints.go
package v1

import (
	"net/http"

	"cryptomarket/internal/core/port"
	"github.com/redis/go-redis/v9"
)

// SetMarketRoutes sets up all market data API routes
func SetMarketRoutes(router *http.ServeMux, marketHandler *PriceHandler, healthHandler *HealthHandler, exchangeHandler *ExchangeHandler) {
	// Market Data API Routes
	setPriceRoutes(marketHandler, router)

	// Data Mode API Routes
	setModeRoutes(exchangeHandler, router)

	// System Health Routes
	setHealthRoutes(healthHandler, router)
}

// SetDebugRoutes sets up debug routes (call this separately for debugging)
func SetDebugRoutes(router *http.ServeMux, cache port.Cache, redisClient *redis.Client) {
	debugHandler := NewDebugHandler(cache, redisClient)

	router.HandleFunc("GET /debug/redis/keys", debugHandler.GetRedisKeys)
	router.HandleFunc("GET /debug/redis/symbol/{symbol}", debugHandler.GetSymbolData)
	router.HandleFunc("GET /debug/cache/test/{symbol}", debugHandler.TestCacheMethods)
}

// setPriceRoutes sets up all price-related endpoints
func setPriceRoutes(handler *PriceHandler, router *http.ServeMux) {
	// Latest Price Endpoints
	router.HandleFunc("GET /prices/latest/{symbol}", handler.GetLatestPrice)
	router.HandleFunc("GET /prices/latest/{exchange}/{symbol}", handler.GetLatestPriceByExchange)

	// Highest Price Endpoints
	router.HandleFunc("GET /prices/highest/{symbol}", handler.GetHighestPrice)                      // Default period
	router.HandleFunc("GET /prices/highest/{exchange}/{symbol}", handler.GetHighestPriceByExchange) // Default period
	// Note: Same endpoints handle ?period={duration} query parameter for custom periods

	// Lowest Price Endpoints
	router.HandleFunc("GET /prices/lowest/{symbol}", handler.GetLowestPrice)                      // Default period
	router.HandleFunc("GET /prices/lowest/{exchange}/{symbol}", handler.GetLowestPriceByExchange) // Default period
	// Note: Same endpoints handle ?period={duration} query parameter for custom periods

	// Average Price Endpoints
	router.HandleFunc("GET /prices/average/{symbol}", handler.GetAveragePrice)                      // Default period
	router.HandleFunc("GET /prices/average/{exchange}/{symbol}", handler.GetAveragePriceByExchange) // Default period
	// Note: Same endpoints handle ?period={duration} query parameter for custom periods
}

// setModeRoutes sets up data mode switching endpoints
func setModeRoutes(handler *ExchangeHandler, router *http.ServeMux) {
	router.HandleFunc("POST /mode/test", handler.SwitchToTestExchange)
	router.HandleFunc("POST /mode/live", handler.SwitchToLiveExchange)
	router.HandleFunc("GET /mode/current", handler.GetCurrentExchange) // Extra: get current mode
}

// setHealthRoutes sets up system health endpoints
func setHealthRoutes(handler *HealthHandler, router *http.ServeMux) {
	router.HandleFunc("GET /health", handler.GetSystemHealth)
	router.HandleFunc("GET /health/detailed", handler.GetDetailedHealth) // Extra: detailed health check
}
