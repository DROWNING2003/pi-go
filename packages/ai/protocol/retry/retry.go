// Package retry provides exponential backoff retry logic for provider API calls.
package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Config configures retry behavior.
type Config struct {
	MaxRetries int           // maximum retry attempts (default: 3)
	BaseDelay  time.Duration // initial delay (default: 1s)
	MaxDelay   time.Duration // maximum delay (default: 30s)
	Multiplier float64       // backoff multiplier (default: 2.0)
	Jitter     float64       // jitter factor 0-1 (default: 0.1)
}

// DefaultConfig returns sensible retry defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.1,
	}
}

// IsRetryable returns true for errors that should be retried (network, rate limit, server errors).
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Common retryable patterns
	retryable := []string{
		"timeout", "connection refused", "connection reset",
		"rate limit", "too many requests", "429",
		"500", "502", "503", "504",
		"temporary failure", "try again",
	}
	for _, p := range retryable {
		if contains(err.Error(), p) {
			return true
		}
	}
	_ = msg
	return false
}

// Do executes a function with exponential backoff retry.
// Returns the first non-retryable error or the last retryable error.
func Do(ctx context.Context, cfg Config, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := backoff(cfg, attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if !IsRetryable(err) {
			return err
		}
	}

	return lastErr
}

// DoWithValue executes a function returning a value with retry.
func DoWithValue[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := backoff(cfg, attempt)
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
			}
		}

		val, err := fn()
		if err == nil {
			return val, nil
		}

		lastErr = err

		if !IsRetryable(err) {
			return zero, err
		}
	}

	return zero, lastErr
}

func backoff(cfg Config, attempt int) time.Duration {
	delay := float64(cfg.BaseDelay) * math.Pow(cfg.Multiplier, float64(attempt-1))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	// Add jitter
	jitter := delay * cfg.Jitter * (rand.Float64()*2 - 1)
	delay += jitter
	if delay < 0 {
		delay = 0
	}
	return time.Duration(delay)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
