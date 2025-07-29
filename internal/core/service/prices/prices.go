// File: internal/core/service/prices/prices.go
package prices

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"cryptomarket/internal/core/domain"
	"cryptomarket/internal/core/port"
	"cryptomarket/internal/utils"
)

// Supported symbols and exchanges (simple maps)
var supportedSymbols = map[string]bool{
	"BTCUSDT":  true,
	"DOGEUSDT": true,
	"TONUSDT":  true,
	"SOLUSDT":  true,
	"ETHUSDT":  true,
}

var supportedExchanges = map[string]bool{
	"exchange1":      true,
	"exchange2":      true,
	"exchange3":      true,
	"test-exchange1": true,
	"test-exchange2": true,
	"test-exchange3": true,
}

type PriceService struct {
	cache port.Cache
	db    *sql.DB
}

// NewPriceService creates a new price service with proper interface dependency
func NewPriceService(cache port.Cache, db *sql.DB) port.PriceService {
	return &PriceService{
		cache: cache,
		db:    db,
	}
}

// GetLatestPrice gets the latest price for a symbol across all exchanges
func (s *PriceService) GetLatestPrice(ctx context.Context, symbol string) (*domain.MarketData, error) {
	// Validate symbol
	validSymbol, err := s.validateSymbol(symbol)
	if err != nil {
		return nil, err
	}

	// If cache is available, try to get from cache first
	if s.cache != nil {
		data, err := s.cache.GetLatestPrice(ctx, validSymbol)
		if err == nil && data != nil {
			return data, nil
		}
		// If cache fails or no data, continue to fallback
	}

	// TODO: Fallback to PostgreSQL if cache is unavailable
	// For now, return error if cache is not available
	if s.cache == nil {
		return nil, fmt.Errorf("no cache available and PostgreSQL fallback not implemented")
	}

	return nil, fmt.Errorf("no price data found for symbol %s", symbol)
}

// GetLatestPriceByExchange gets the latest price for a symbol from specific exchange
func (s *PriceService) GetLatestPriceByExchange(ctx context.Context, symbol, exchange string) (*domain.MarketData, error) {
	// Validate symbol and exchange
	validSymbol, err := s.validateSymbol(symbol)
	if err != nil {
		return nil, err
	}

	validExchange, err := s.validateExchange(exchange)
	if err != nil {
		return nil, err
	}

	// If cache is available, try to get from cache first
	if s.cache != nil {
		data, err := s.cache.GetLatestPriceByExchange(ctx, validSymbol, validExchange)
		if err == nil && data != nil {
			return data, nil
		}
		// If cache fails or no data, continue to fallback
	}

	// TODO: Fallback to PostgreSQL if cache is unavailable
	// For now, return error if cache is not available
	if s.cache == nil {
		return nil, fmt.Errorf("no cache available and PostgreSQL fallback not implemented")
	}

	return nil, fmt.Errorf("no price data found for symbol %s on exchange %s", symbol, exchange)
}

// GetHighestPrice gets the highest price for a symbol across all exchanges within a period
func (s *PriceService) GetHighestPrice(ctx context.Context, symbol string, period time.Duration) (*domain.PriceStatistic, error) {
	validSymbol, err := s.validateSymbol(symbol)
	if err != nil {
		return nil, err
	}

	from, to := utils.CalculateTimeRange(period)

	// Add debug logging
	slog.Info("GetHighestPrice request",
		"symbol", validSymbol,
		"period", period.String(),
		"from", from.Format(time.RFC3339),
		"to", to.Format(time.RFC3339))

	if s.cache != nil {
		// Get data from cache for the time range
		priceData, err := s.cache.GetPricesInRange(ctx, validSymbol, from, to)
		if err != nil {
			slog.Error("Failed to get prices from cache", "error", err, "symbol", validSymbol)
			return nil, fmt.Errorf("failed to get price data from cache: %w", err)
		}

		slog.Info("Retrieved price data from cache",
			"symbol", validSymbol,
			"count", len(priceData),
			"period", utils.FormatPeriod(period))

		if len(priceData) > 0 {
			highest := s.findHighestPrice(priceData)
			slog.Info("Found highest price", "symbol", validSymbol, "highest", highest, "data_points", len(priceData))

			return &domain.PriceStatistic{
				Symbol:    validSymbol,
				Price:     highest,
				Period:    utils.FormatPeriod(period),
				StartTime: from,
				EndTime:   to,
				Timestamp: time.Now().Unix(),
			}, nil
		} else {
			slog.Warn("No price data found in cache",
				"symbol", validSymbol,
				"period", utils.FormatPeriod(period),
				"from", from.Format(time.RFC3339),
				"to", to.Format(time.RFC3339))
		}
	} else {
		slog.Warn("Cache is not available", "symbol", validSymbol)
	}

	// TODO: Fallback to PostgreSQL
	return nil, fmt.Errorf("no data found for symbol %s in period %s (cache returned %d records)",
		symbol, utils.FormatPeriod(period), 0)
}

// GetHighestPriceByExchange gets the highest price for a symbol from specific exchange within a period
func (s *PriceService) GetHighestPriceByExchange(ctx context.Context, symbol, exchange string, period time.Duration) (*domain.PriceStatistic, error) {
	validSymbol, err := s.validateSymbol(symbol)
	if err != nil {
		return nil, err
	}

	validExchange, err := s.validateExchange(exchange)
	if err != nil {
		return nil, err
	}

	from, to := utils.CalculateTimeRange(period)

	// Add debug logging
	slog.Info("GetHighestPriceByExchange request",
		"symbol", validSymbol,
		"exchange", validExchange,
		"period", period.String(),
		"from", from.Format(time.RFC3339),
		"to", to.Format(time.RFC3339))

	if s.cache != nil {
		// Get data from cache for the time range
		priceData, err := s.cache.GetPricesInRangeByExchange(ctx, validSymbol, validExchange, from, to)
		if err != nil {
			slog.Error("Failed to get prices from cache by exchange",
				"error", err,
				"symbol", validSymbol,
				"exchange", validExchange)
			return nil, fmt.Errorf("failed to get price data from cache: %w", err)
		}

		slog.Info("Retrieved price data from cache by exchange",
			"symbol", validSymbol,
			"exchange", validExchange,
			"count", len(priceData),
			"period", utils.FormatPeriod(period))

		if len(priceData) > 0 {
			highest := s.findHighestPrice(priceData)
			slog.Info("Found highest price by exchange",
				"symbol", validSymbol,
				"exchange", validExchange,
				"highest", highest,
				"data_points", len(priceData))

			return &domain.PriceStatistic{
				Symbol:    validSymbol,
				Exchange:  validExchange,
				Price:     highest,
				Period:    utils.FormatPeriod(period),
				StartTime: from,
				EndTime:   to,
				Timestamp: time.Now().Unix(),
			}, nil
		} else {
			slog.Warn("No price data found in cache by exchange",
				"symbol", validSymbol,
				"exchange", validExchange,
				"period", utils.FormatPeriod(period),
				"from", from.Format(time.RFC3339),
				"to", to.Format(time.RFC3339))
		}
	} else {
		slog.Warn("Cache is not available", "symbol", validSymbol, "exchange", validExchange)
	}

	return nil, fmt.Errorf("no data found for symbol %s on exchange %s in period %s",
		symbol, exchange, utils.FormatPeriod(period))
}

// GetLowestPrice gets the lowest price for a symbol across all exchanges within a period
func (s *PriceService) GetLowestPrice(ctx context.Context, symbol string, period time.Duration) (*domain.PriceStatistic, error) {
	validSymbol, err := s.validateSymbol(symbol)
	if err != nil {
		return nil, err
	}

	from, to := utils.CalculateTimeRange(period)

	slog.Info("GetLowestPrice request",
		"symbol", validSymbol,
		"period", period.String())

	if s.cache != nil {
		priceData, err := s.cache.GetPricesInRange(ctx, validSymbol, from, to)
		if err != nil {
			slog.Error("Failed to get prices from cache", "error", err, "symbol", validSymbol)
			return nil, fmt.Errorf("failed to get price data from cache: %w", err)
		}

		slog.Info("Retrieved price data for lowest", "symbol", validSymbol, "count", len(priceData))

		if len(priceData) > 0 {
			lowest := s.findLowestPrice(priceData)
			return &domain.PriceStatistic{
				Symbol:    validSymbol,
				Price:     lowest,
				Period:    utils.FormatPeriod(period),
				StartTime: from,
				EndTime:   to,
				Timestamp: time.Now().Unix(),
			}, nil
		}
	}

	return nil, fmt.Errorf("no data found for symbol %s in period %s", symbol, utils.FormatPeriod(period))
}

// GetLowestPriceByExchange gets the lowest price for a symbol from specific exchange within a period
func (s *PriceService) GetLowestPriceByExchange(ctx context.Context, symbol, exchange string, period time.Duration) (*domain.PriceStatistic, error) {
	validSymbol, err := s.validateSymbol(symbol)
	if err != nil {
		return nil, err
	}

	validExchange, err := s.validateExchange(exchange)
	if err != nil {
		return nil, err
	}

	from, to := utils.CalculateTimeRange(period)

	if s.cache != nil {
		priceData, err := s.cache.GetPricesInRangeByExchange(ctx, validSymbol, validExchange, from, to)
		if err != nil {
			return nil, fmt.Errorf("failed to get price data from cache: %w", err)
		}

		if len(priceData) > 0 {
			lowest := s.findLowestPrice(priceData)
			return &domain.PriceStatistic{
				Symbol:    validSymbol,
				Exchange:  validExchange,
				Price:     lowest,
				Period:    utils.FormatPeriod(period),
				StartTime: from,
				EndTime:   to,
				Timestamp: time.Now().Unix(),
			}, nil
		}
	}

	return nil, fmt.Errorf("no data found for symbol %s on exchange %s in period %s", symbol, exchange, utils.FormatPeriod(period))
}

// GetAveragePrice gets the average price for a symbol across all exchanges within a period
func (s *PriceService) GetAveragePrice(ctx context.Context, symbol string, period time.Duration) (*domain.PriceStatistic, error) {
	validSymbol, err := s.validateSymbol(symbol)
	if err != nil {
		return nil, err
	}

	from, to := utils.CalculateTimeRange(period)

	if s.cache != nil {
		priceData, err := s.cache.GetPricesInRange(ctx, validSymbol, from, to)
		if err != nil {
			return nil, fmt.Errorf("failed to get price data from cache: %w", err)
		}

		if len(priceData) > 0 {
			average := s.calculateAveragePrice(priceData)
			return &domain.PriceStatistic{
				Symbol:    validSymbol,
				Price:     average,
				Period:    utils.FormatPeriod(period),
				StartTime: from,
				EndTime:   to,
				Timestamp: time.Now().Unix(),
			}, nil
		}
	}

	return nil, fmt.Errorf("no data found for symbol %s in period %s", symbol, utils.FormatPeriod(period))
}

// GetAveragePriceByExchange gets the average price for a symbol from specific exchange within a period
func (s *PriceService) GetAveragePriceByExchange(ctx context.Context, symbol, exchange string, period time.Duration) (*domain.PriceStatistic, error) {
	validSymbol, err := s.validateSymbol(symbol)
	if err != nil {
		return nil, err
	}

	validExchange, err := s.validateExchange(exchange)
	if err != nil {
		return nil, err
	}

	from, to := utils.CalculateTimeRange(period)

	if s.cache != nil {
		priceData, err := s.cache.GetPricesInRangeByExchange(ctx, validSymbol, validExchange, from, to)
		if err != nil {
			return nil, fmt.Errorf("failed to get price data from cache: %w", err)
		}

		if len(priceData) > 0 {
			average := s.calculateAveragePrice(priceData)
			return &domain.PriceStatistic{
				Symbol:    validSymbol,
				Exchange:  validExchange,
				Price:     average,
				Period:    utils.FormatPeriod(period),
				StartTime: from,
				EndTime:   to,
				Timestamp: time.Now().Unix(),
			}, nil
		}
	}

	return nil, fmt.Errorf("no data found for symbol %s on exchange %s in period %s", symbol, exchange, utils.FormatPeriod(period))
}

// Helper methods for price calculations
func (s *PriceService) findHighestPrice(prices []domain.MarketData) float64 {
	if len(prices) == 0 {
		return 0
	}

	highest := prices[0].Price
	for _, price := range prices[1:] {
		if price.Price > highest {
			highest = price.Price
		}
	}
	return highest
}

func (s *PriceService) findLowestPrice(prices []domain.MarketData) float64 {
	if len(prices) == 0 {
		return 0
	}

	lowest := prices[0].Price
	for _, price := range prices[1:] {
		if price.Price < lowest {
			lowest = price.Price
		}
	}
	return lowest
}

func (s *PriceService) calculateAveragePrice(prices []domain.MarketData) float64 {
	if len(prices) == 0 {
		return 0
	}

	sum := 0.0
	for _, price := range prices {
		sum += price.Price
	}
	return math.Round((sum/float64(len(prices)))*100) / 100 // Round to 2 decimal places
}

// Simple validation functions
func (s *PriceService) validateSymbol(symbol string) (string, error) {
	if symbol == "" {
		return "", fmt.Errorf("symbol cannot be empty")
	}

	// Normalize to uppercase
	normalized := strings.ToUpper(strings.TrimSpace(symbol))

	// Check if supported
	if !supportedSymbols[normalized] {
		return "", fmt.Errorf("unsupported symbol: %s", symbol)
	}

	return normalized, nil
}

func (s *PriceService) validateExchange(exchange string) (string, error) {
	if exchange == "" {
		return "", fmt.Errorf("exchange cannot be empty")
	}

	// Normalize to lowercase
	normalized := strings.ToLower(strings.TrimSpace(exchange))

	// Check if supported
	if !supportedExchanges[normalized] {
		return "", fmt.Errorf("unsupported exchange: %s", exchange)
	}

	return normalized, nil
}
