package port

import (
	"context"
	"cryptomarket/internal/core/domain"
)

type HealthRepository interface{}

type HealthService interface {
	GetSystemHealth(ctx context.Context) (*domain.HealthStatus, error)
	GetDetailedHealth(ctx context.Context) (*domain.HealthStatus, error)
}
