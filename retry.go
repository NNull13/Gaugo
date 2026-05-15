package gaugo

import (
	"fmt"
	"time"

	"github.com/nnull13/gaugo/internal/failure"
)

// RetryConfig controls retries for transient provider HTTP failures.
type RetryConfig struct {
	// MaxAttempts is the total number of attempts, including the first request.
	// Zero uses the package default.
	MaxAttempts int
	// BaseDelay is the fallback delay before the second attempt.
	// Zero uses the package default.
	BaseDelay time.Duration
	// MaxDelay caps exponential backoff and Retry-After delays.
	// Zero uses the package default.
	MaxDelay time.Duration
}

// DefaultRetryConfig returns the retry defaults used by bundled providers.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    2 * time.Second,
	}
}

// Validate reports whether cfg is internally consistent.
func (cfg RetryConfig) Validate() error {
	if cfg.MaxAttempts < 0 {
		return failure.Validation(failure.CodeRetryInvalid, "RetryConfig.Validate", "max_attempts", fmt.Sprintf("retry max attempts must be non-negative, got %d", cfg.MaxAttempts), nil)
	}
	if cfg.BaseDelay < 0 {
		return failure.Validation(failure.CodeRetryInvalid, "RetryConfig.Validate", "base_delay", fmt.Sprintf("retry base delay must be non-negative, got %s", cfg.BaseDelay), nil)
	}
	if cfg.MaxDelay < 0 {
		return failure.Validation(failure.CodeRetryInvalid, "RetryConfig.Validate", "max_delay", fmt.Sprintf("retry max delay must be non-negative, got %s", cfg.MaxDelay), nil)
	}
	if cfg.BaseDelay > 0 && cfg.MaxDelay > 0 && cfg.BaseDelay > cfg.MaxDelay {
		return failure.Validation(failure.CodeRetryInvalid, "RetryConfig.Validate", "base_delay", fmt.Sprintf("retry base delay %s exceeds max delay %s", cfg.BaseDelay, cfg.MaxDelay), nil)
	}
	return nil
}
