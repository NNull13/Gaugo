package gaugo

import (
	"fmt"

	"github.com/nnull13/gaugo/internal/failure"
)

// RateLimitConfig limits outgoing judge requests to avoid provider rate limits.
// Set RequestsPerMinute to a positive value to enable throttling.
//
// The limiter is shared across all concurrent evaluations that use the same
// judge instance, so the limit applies per API key, not per goroutine.
type RateLimitConfig struct {
	// RequestsPerMinute is the maximum number of requests per minute.
	// Zero disables rate limiting.
	RequestsPerMinute int
	// Burst is the maximum number of tokens that can accumulate.
	// Zero defaults to 1 (strict: one request at a time before throttling).
	Burst int
}

// Validate reports whether cfg is internally consistent.
func (cfg RateLimitConfig) Validate() error {
	if cfg.RequestsPerMinute < 0 {
		return failure.Validation(failure.CodeRateLimitInvalid, "RateLimitConfig.Validate", "requests_per_minute",
			fmt.Sprintf("rate limit requests per minute must be non-negative, got %d", cfg.RequestsPerMinute), nil)
	}
	if cfg.Burst < 0 {
		return failure.Validation(failure.CodeRateLimitInvalid, "RateLimitConfig.Validate", "burst",
			fmt.Sprintf("rate limit burst must be non-negative, got %d", cfg.Burst), nil)
	}
	return nil
}
