// File: internal/core/port/prices.go
package port

import (
	"context"
	"time"

	"cryptomarket/internal/core/domain"
)

type PriceRepository interface {
	GetLatestPrice(symbol string) (domain.GetPrice, error)
	GetLatestPriceByExchange(symbol string, exchange string) (domain.GetPrice, error)

	GetHighestPrice(symbol string) (domain.GetPrice, error)
	GetHighestPriceExchange(symbol string, exchange string) (domain.GetPrice, error)
	GetHighestPriceInDuration(symbol string, from time.Time, to time.Time) (domain.GetPrice, error)
	GetHighestPriceInDurationExchange(symbol string, exchange string, from time.Time, to time.Time) (domain.GetPrice, error)

	GetLowestPrice(symbol string) (domain.GetPrice, error)
	GetLowestPriceExchange(symbol string, exchange string) (domain.GetPrice, error)
	GetLowestPriceInDuration(symbol string, from time.Time, to time.Time) (domain.GetPrice, error)
	GetLowestPriceInDurationExchange(symbol string, exchange string, from time.Time, to time.Time) (domain.GetPrice, error)

	GetAveragePrice(symbol string) (domain.GetPrice, error)
	GetAveragePriceExchange(symbol string, exchange string) (domain.GetPrice, error)
	GetAveragePriceInDurationExchange(symbol string, exchange string, from time.Time, to time.Time) (domain.GetPrice, error)
}

// Enhanced PriceService interface with new methods
type PriceService interface {
	// Latest price methods (already implemented)
	GetLatestPrice(ctx context.Context, symbol string) (*domain.MarketData, error)
	GetLatestPriceByExchange(ctx context.Context, symbol, exchange string) (*domain.MarketData, error)

	// Highest price methods
	GetHighestPrice(ctx context.Context, symbol string, period time.Duration) (*domain.PriceStatistic, error)
	GetHighestPriceByExchange(ctx context.Context, symbol, exchange string, period time.Duration) (*domain.PriceStatistic, error)

	// Lowest price methods
	GetLowestPrice(ctx context.Context, symbol string, period time.Duration) (*domain.PriceStatistic, error)
	GetLowestPriceByExchange(ctx context.Context, symbol, exchange string, period time.Duration) (*domain.PriceStatistic, error)

	// Average price methods
	GetAveragePrice(ctx context.Context, symbol string, period time.Duration) (*domain.PriceStatistic, error)
	GetAveragePriceByExchange(ctx context.Context, symbol, exchange string, period time.Duration) (*domain.PriceStatistic, error)
}
