package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Limiter is a token bucket rate limiter safe for concurrent use.
type Limiter struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	rate       float64 // tokens per nanosecond
	lastRefill time.Time
}

// New returns a Limiter that allows requestsPerMinute tokens per minute.
// burst controls the maximum tokens that can accumulate; if ≤ 0 it defaults to 1.
func New(requestsPerMinute, burst int) *Limiter {
	if burst <= 0 {
		burst = 1
	}
	rate := float64(requestsPerMinute) / float64(time.Minute)
	cap := float64(burst)
	return &Limiter{
		tokens:     cap,
		capacity:   cap,
		rate:       rate,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available or ctx is done.
func (l *Limiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(l.lastRefill)
		newTokens := float64(elapsed) * l.rate
		if newTokens > 0 {
			l.tokens = min(l.capacity, l.tokens+newTokens)
			l.lastRefill = now
		}
		if l.tokens >= 1.0 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		deficit := 1.0 - l.tokens
		waitDuration := time.Duration(deficit / l.rate)
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}
}
