package health

import (
	"context"
	"database/sql"
	"time"

	"cryptomarket/internal/core/domain"
	"cryptomarket/internal/core/port"
)

type HealthService struct {
	db              *sql.DB
	cache           port.Cache
	exchangeService port.ExchangeService
}

func NewHealthService(db *sql.DB, cache port.Cache, exchangeService port.ExchangeService) port.HealthService {
	return &HealthService{
		db:              db,
		cache:           cache,
		exchangeService: exchangeService,
	}
}

func (s *HealthService) GetSystemHealth(ctx context.Context) (*domain.HealthStatus, error) {
	status := &domain.HealthStatus{
		Components: make(map[string]string),
		Timestamp:  time.Now().Unix(),
	}

	allHealthy := true

	// Check PostgreSQL
	if s.db != nil {
		if err := s.db.PingContext(ctx); err != nil {
			status.Components["database"] = "unhealthy"
			allHealthy = false
		} else {
			status.Components["database"] = "healthy"
		}
	} else {
		status.Components["database"] = "unavailable"
		allHealthy = false
	}

	// Check Redis
	if s.cache != nil {
		if err := s.cache.Ping(ctx); err != nil {
			status.Components["cache"] = "unhealthy"
			allHealthy = false
		} else {
			status.Components["cache"] = "healthy"
		}
	} else {
		status.Components["cache"] = "unavailable"
		allHealthy = false
	}

	// Check Exchange Service
	if s.exchangeService != nil {
		stats := s.exchangeService.GetStats()
		if isRunning, ok := stats["is_running"].(bool); ok && isRunning {
			if healthyAdapters, ok := stats["healthy_adapters"].(int); ok && healthyAdapters > 0 {
				status.Components["exchange_service"] = "healthy"
			} else {
				status.Components["exchange_service"] = "degraded"
				allHealthy = false
			}
		} else {
			status.Components["exchange_service"] = "unhealthy"
			allHealthy = false
		}
	} else {
		status.Components["exchange_service"] = "unavailable"
		allHealthy = false
	}

	// Determine overall status
	if allHealthy {
		status.Status = "healthy"
		status.Message = "All systems operational"
	} else {
		// Check if any critical components are down
		if status.Components["exchange_service"] == "unhealthy" {
			status.Status = "unhealthy"
			status.Message = "Exchange service is down"
		} else {
			status.Status = "degraded"
			status.Message = "Some components are not fully operational"
		}
	}

	return status, nil
}

func (s *HealthService) GetDetailedHealth(ctx context.Context) (*domain.HealthStatus, error) {
	status, err := s.GetSystemHealth(ctx)
	if err != nil {
		return nil, err
	}

	// Add detailed information about exchange service
	if s.exchangeService != nil {
		stats := s.exchangeService.GetStats()

		// Add more detailed exchange information
		if mode, ok := stats["current_mode"].(string); ok {
			status.Components["exchange_mode"] = mode
		}

		if activeAdapters, ok := stats["active_adapters"].(int); ok {
			if activeAdapters > 0 {
				status.Components["active_adapters"] = "healthy"
			} else {
				status.Components["active_adapters"] = "no_adapters"
			}
		}

		if aggregatedBuffer, ok := stats["aggregated_buffer"].(int); ok {
			if aggregatedBuffer < 800 { // Buffer is not too full
				status.Components["aggregated_buffer"] = "healthy"
			} else {
				status.Components["aggregated_buffer"] = "high_load"
			}
		}

		if resultBuffer, ok := stats["result_buffer"].(int); ok {
			if resultBuffer < 800 { // Buffer is not too full
				status.Components["result_buffer"] = "healthy"
			} else {
				status.Components["result_buffer"] = "high_load"
			}
		}
	}

	return status, nil
}
