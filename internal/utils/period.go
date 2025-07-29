// File: internal/utils/period.go
package utils

import (
	"fmt"
	"time"
)

// DefaultPeriod is the default time period when none is specified
const DefaultPeriod = 1 * time.Minute

// ValidPeriods defines the allowed time periods
var ValidPeriods = map[string]time.Duration{
	"1s":  1 * time.Second,
	"3s":  3 * time.Second,
	"5s":  5 * time.Second,
	"10s": 10 * time.Second,
	"30s": 30 * time.Second,
	"1m":  1 * time.Minute,
	"3m":  3 * time.Minute,
	"5m":  5 * time.Minute,
	"10m": 10 * time.Minute,
	"15m": 15 * time.Minute,
	"30m": 30 * time.Minute,
	"1h":  1 * time.Hour,
	"3h":  3 * time.Hour,
	"6h":  6 * time.Hour,
	"12h": 12 * time.Hour,
	"24h": 24 * time.Hour,
}

// ParsePeriod parses a period string and returns the duration
func ParsePeriod(periodStr string) (time.Duration, error) {
	if periodStr == "" {
		return DefaultPeriod, nil
	}

	// First try the predefined valid periods
	if duration, exists := ValidPeriods[periodStr]; exists {
		return duration, nil
	}

	// Try parsing as a Go duration string (e.g., "2m30s")
	duration, err := time.ParseDuration(periodStr)
	if err != nil {
		return 0, fmt.Errorf("invalid period format '%s': must be one of %v or a valid Go duration",
			periodStr, getValidPeriodKeys())
	}

	// Validate that the duration is reasonable (between 1 second and 24 hours)
	if duration < 1*time.Second {
		return 0, fmt.Errorf("period too short: minimum is 1 second")
	}
	if duration > 24*time.Hour {
		return 0, fmt.Errorf("period too long: maximum is 24 hours")
	}

	return duration, nil
}

// FormatPeriod converts a duration to a human-readable string
func FormatPeriod(duration time.Duration) string {
	// Check if it matches one of our predefined periods
	for key, validDuration := range ValidPeriods {
		if duration == validDuration {
			return key
		}
	}

	// Otherwise, use Go's string representation
	return duration.String()
}

// getValidPeriodKeys returns the keys from ValidPeriods map
func getValidPeriodKeys() []string {
	keys := make([]string, 0, len(ValidPeriods))
	for key := range ValidPeriods {
		keys = append(keys, key)
	}
	return keys
}

// CalculateTimeRange calculates the start and end time for a given period
func CalculateTimeRange(period time.Duration) (time.Time, time.Time) {
	now := time.Now()
	return now.Add(-period), now
}
