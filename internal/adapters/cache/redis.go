package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"cryptomarket/internal/core/domain"
	"cryptomarket/internal/core/port"

	"github.com/redis/go-redis/v9"
)

type RedisAdapter struct {
	client *redis.Client
}

func NewRedisAdapter(client *redis.Client) port.Cache {
	return &RedisAdapter{
		client: client,
	}
}

// SetPrice stores price data with timestamp
func (r *RedisAdapter) SetPrice(ctx context.Context, key string, data domain.MarketData) error {
	// Store latest price for quick access
	latestKey := fmt.Sprintf("latest:%s:%s", data.Symbol, data.Exchange)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal market data: %w", err)
	}

	// Set latest price with 2 minute expiration (to ensure cleanup)
	if err := r.client.Set(ctx, latestKey, dataBytes, 2*time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to set latest price: %w", err)
	}

	// Store in time-series sorted set for range queries
	timeSeriesKey := fmt.Sprintf("timeseries:%s:%s", data.Symbol, data.Exchange)
	score := float64(data.Timestamp)
	member := fmt.Sprintf("%f", data.Price)

	// Add to sorted set with score as timestamp
	if err := r.client.ZAdd(ctx, timeSeriesKey, redis.Z{
		Score:  score,
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to time series: %w", err)
	}

	// Set expiration for time series (2 minutes to ensure cleanup)
	r.client.Expire(ctx, timeSeriesKey, 2*time.Minute)

	return nil
}

// GetLatestPrice retrieves the latest price for a symbol across all exchanges
func (r *RedisAdapter) GetLatestPrice(ctx context.Context, symbol string) (*domain.MarketData, error) {
	pattern := fmt.Sprintf("latest:%s:*", symbol)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no price data found for symbol %s", symbol)
	}

	// Get the most recent price
	var latestData *domain.MarketData
	var latestTimestamp int64

	for _, key := range keys {
		dataStr, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var data domain.MarketData
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			continue
		}

		if data.Timestamp > latestTimestamp {
			latestTimestamp = data.Timestamp
			latestData = &data
		}
	}

	if latestData == nil {
		return nil, fmt.Errorf("no valid price data found for symbol %s", symbol)
	}

	return latestData, nil
}

// GetLatestPriceByExchange retrieves the latest price for a symbol from specific exchange
func (r *RedisAdapter) GetLatestPriceByExchange(ctx context.Context, symbol, exchange string) (*domain.MarketData, error) {
	key := fmt.Sprintf("latest:%s:%s", symbol, exchange)

	dataStr, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("no price data found for %s on %s", symbol, exchange)
		}
		return nil, fmt.Errorf("failed to get price data: %w", err)
	}

	var data domain.MarketData
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal price data: %w", err)
	}

	return &data, nil
}

// GetPricesInRange retrieves all prices for a symbol within time range across all exchanges
func (r *RedisAdapter) GetPricesInRange(ctx context.Context, symbol string, from, to time.Time) ([]domain.MarketData, error) {
	// Get all exchanges for this symbol
	pattern := fmt.Sprintf("timeseries:%s:*", symbol)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		slog.Error("Failed to get timeseries keys", "error", err, "pattern", pattern)
		return nil, fmt.Errorf("failed to get timeseries keys: %w", err)
	}

	slog.Info("GetPricesInRange debug",
		"symbol", symbol,
		"pattern", pattern,
		"keys_found", len(keys),
		"keys", keys,
		"from", from.Format(time.RFC3339),
		"to", to.Format(time.RFC3339))

	if len(keys) == 0 {
		slog.Warn("No timeseries keys found for symbol", "symbol", symbol, "pattern", pattern)
		return []domain.MarketData{}, nil
	}

	var allData []domain.MarketData
	fromScore := float64(from.Unix())
	toScore := float64(to.Unix())

	slog.Info("Time range for Redis query",
		"fromScore", fromScore,
		"toScore", toScore,
		"from_time", from.Format(time.RFC3339),
		"to_time", to.Format(time.RFC3339))

	for _, key := range keys {
		// Extract exchange name from key
		// timeseries:SYMBOL:EXCHANGE -> EXCHANGE
		parts := parseTimeSeriesKey(key)
		if len(parts) < 3 {
			slog.Warn("Invalid timeseries key format", "key", key, "parts", parts)
			continue
		}
		exchange := parts[2]

		slog.Debug("Processing timeseries key", "key", key, "exchange", exchange)

		// Get prices in range using ZRANGEBYSCORE
		results, err := r.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
			Min: strconv.FormatFloat(fromScore, 'f', -1, 64),
			Max: strconv.FormatFloat(toScore, 'f', -1, 64),
		}).Result()
		if err != nil {
			slog.Error("Failed to get prices by score", "error", err, "key", key)
			continue
		}

		slog.Info("Retrieved prices from Redis key",
			"key", key,
			"exchange", exchange,
			"results_count", len(results),
			"sample_results", results[:min(3, len(results))])

		// Convert to MarketData
		for i, result := range results {
			price, err := strconv.ParseFloat(result, 64)
			if err != nil {
				slog.Warn("Failed to parse price", "result", result, "error", err)
				continue
			}

			// Get timestamp from score - we need to use ZRANGEBYSCORE with WITHSCORES
			// Let's get the score for this member
			score, err := r.client.ZScore(ctx, key, result).Result()
			if err != nil {
				slog.Warn("Failed to get score for member", "key", key, "member", result, "error", err)
				// Fallback: estimate timestamp based on position and time range
				totalResults := len(results)
				if totalResults > 1 {
					ratio := float64(i) / float64(totalResults-1)
					estimatedTimestamp := fromScore + ratio*(toScore-fromScore)
					score = estimatedTimestamp
				} else {
					score = fromScore
				}
			}

			data := domain.MarketData{
				Symbol:    symbol,
				Price:     price,
				Timestamp: int64(score),
				Exchange:  exchange,
			}
			allData = append(allData, data)
		}
	}

	slog.Info("GetPricesInRange result",
		"symbol", symbol,
		"total_data_points", len(allData),
		"time_range", fmt.Sprintf("%s to %s", from.Format("15:04:05"), to.Format("15:04:05")))

	return allData, nil
}

// GetPricesInRangeByExchange retrieves prices for a symbol from specific exchange within time range
func (r *RedisAdapter) GetPricesInRangeByExchange(ctx context.Context, symbol, exchange string, from, to time.Time) ([]domain.MarketData, error) {
	key := fmt.Sprintf("timeseries:%s:%s", symbol, exchange)
	fromScore := float64(from.Unix())
	toScore := float64(to.Unix())

	slog.Info("GetPricesInRangeByExchange debug",
		"symbol", symbol,
		"exchange", exchange,
		"key", key,
		"from", from.Format(time.RFC3339),
		"to", to.Format(time.RFC3339),
		"fromScore", fromScore,
		"toScore", toScore)

	// First check if the key exists
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		slog.Error("Failed to check key existence", "error", err, "key", key)
		return nil, fmt.Errorf("failed to check key existence: %w", err)
	}

	if exists == 0 {
		slog.Warn("Timeseries key does not exist", "key", key)
		return []domain.MarketData{}, nil
	}

	// Get the total count of items in the sorted set for debugging
	totalCount, err := r.client.ZCard(ctx, key).Result()
	if err == nil {
		slog.Info("Sorted set info", "key", key, "total_items", totalCount)
	}

	// Get a sample of the most recent items for debugging
	recentItems, err := r.client.ZRevRangeWithScores(ctx, key, 0, 4).Result()
	if err == nil && len(recentItems) > 0 {
		slog.Info("Recent items in sorted set", "key", key, "recent_items", recentItems)
	}

	// Get prices in range
	results, err := r.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: strconv.FormatFloat(fromScore, 'f', -1, 64),
		Max: strconv.FormatFloat(toScore, 'f', -1, 64),
	}).Result()
	if err != nil {
		slog.Error("Failed to get prices in range", "error", err, "key", key)
		return nil, fmt.Errorf("failed to get prices in range: %w", err)
	}

	slog.Info("ZRangeByScore result",
		"key", key,
		"results_count", len(results),
		"sample_results", results[:min(3, len(results))])

	var data []domain.MarketData
	for _, result := range results {
		price, err := strconv.ParseFloat(result, 64)
		if err != nil {
			slog.Warn("Failed to parse price", "result", result, "error", err)
			continue
		}

		// Get timestamp from score
		score, err := r.client.ZScore(ctx, key, result).Result()
		if err != nil {
			slog.Warn("Failed to get score", "key", key, "member", result, "error", err)
			continue
		}

		marketData := domain.MarketData{
			Symbol:    symbol,
			Price:     price,
			Timestamp: int64(score),
			Exchange:  exchange,
		}
		data = append(data, marketData)
	}

	slog.Info("GetPricesInRangeByExchange result",
		"symbol", symbol,
		"exchange", exchange,
		"data_points", len(data))

	return data, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CleanupOldData removes data older than specified duration
func (r *RedisAdapter) CleanupOldData(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	cutoffScore := float64(cutoffTime.Unix())

	// Clean up time series data
	timeSeriesPattern := "timeseries:*"
	keys, err := r.client.Keys(ctx, timeSeriesPattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get timeseries keys for cleanup: %w", err)
	}

	for _, key := range keys {
		// Remove old entries from sorted set
		_, err := r.client.ZRemRangeByScore(ctx, key, "0", strconv.FormatFloat(cutoffScore, 'f', -1, 64)).Result()
		if err != nil {
			continue // Continue with other keys
		}
	}

	// Clean up latest price data (handled by TTL, but we can also check manually)
	latestPattern := "latest:*"
	latestKeys, err := r.client.Keys(ctx, latestPattern).Result()
	if err == nil {
		for _, key := range latestKeys {
			// Check if the data is too old
			dataStr, err := r.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var data domain.MarketData
			if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
				continue
			}

			if time.Unix(data.Timestamp, 0).Before(cutoffTime) {
				r.client.Del(ctx, key)
			}
		}
	}

	return nil
}

// Ping checks Redis connection health
func (r *RedisAdapter) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// parseTimeSeriesKey parses a timeseries key to extract components
func parseTimeSeriesKey(key string) []string {
	result := make([]string, 0)
	current := ""
	for _, char := range key {
		if char == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
