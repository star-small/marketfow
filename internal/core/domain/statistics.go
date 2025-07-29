// File: internal/core/domain/statistics.go
package domain

import "time"

// PriceStatistic represents aggregated price data over a time period
type PriceStatistic struct {
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange,omitempty"` // omitempty for cross-exchange queries
	Price     float64   `json:"price"`
	Period    string    `json:"period"` // Human readable period like "5m", "1h"
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Timestamp int64     `json:"timestamp"` // When this statistic was calculated
}

// HealthStatus represents system health information
type HealthStatus struct {
	Status     string            `json:"status"`     // "healthy", "degraded", "unhealthy"
	Components map[string]string `json:"components"` // Component name -> status
	Timestamp  int64             `json:"timestamp"`
	Message    string            `json:"message,omitempty"`
}

// SystemMode represents the current exchange mode
type SystemMode struct {
	CurrentMode      string   `json:"current_mode"` // "live" or "test"
	IsRunning        bool     `json:"is_running"`
	ActiveAdapters   int      `json:"active_adapters"`
	HealthyAdapters  int      `json:"healthy_adapters"`
	AdapterNames     []string `json:"adapter_names"`
	AggregatedBuffer int      `json:"aggregated_buffer"`
	ResultBuffer     int      `json:"result_buffer"`
	Timestamp        int64    `json:"timestamp"`
}
