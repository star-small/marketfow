// File: internal/core/port/exchange.go
package port

import (
	"context"

	"cryptomarket/internal/core/domain"
)

type ExchangeAdapter interface {
	// Start streaming market data
	Start(ctx context.Context) (<-chan domain.MarketData, error)

	// Stop streaming
	Stop() error

	// Get exchange name/identifier
	Name() string

	// Health check
	IsHealthy() bool
}

type ExchangeService interface {
	// Switch to live mode (connect to real exchanges)
	SwitchToLiveMode(ctx context.Context) error

	// Switch to test mode (use synthetic data)
	SwitchToTestMode(ctx context.Context) error

	// Get current mode
	GetCurrentMode() string

	// Start data processing
	StartDataProcessing(ctx context.Context) error

	// Stop data processing
	StopDataProcessing() error

	// Get aggregated data channel
	GetDataStream() <-chan domain.MarketData

	// Get service statistics (already implemented in your service)
	GetStats() map[string]interface{}

	// Check if service is running (already implemented in your service)
	IsRunning() bool
}
