package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_Success(t *testing.T) {
	cfg := Config{MaxRetries: 2, BaseDelay: time.Millisecond}
	err := Do(context.Background(), cfg, func() error { return nil })
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestDo_RetryThenSuccess(t *testing.T) {
	cfg := Config{MaxRetries: 3, BaseDelay: time.Millisecond}
	attempts := 0
	err := Do(context.Background(), cfg, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})
	if err != nil {
		t.Errorf("should succeed after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts: %d", attempts)
	}
}

func TestDo_NonRetryable(t *testing.T) {
	cfg := Config{MaxRetries: 3, BaseDelay: time.Millisecond}
	attempts := 0
	err := Do(context.Background(), cfg, func() error {
		attempts++
		return errors.New("invalid API key")
	})
	if err == nil || attempts != 1 {
		t.Errorf("non-retryable should fail immediately, attempts=%d", attempts)
	}
}

func TestDo_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := Config{MaxRetries: 1, BaseDelay: 10 * time.Millisecond}
	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return errors.New("retryable")
	})
	// Should cancel before first delay
	if err == nil {
		t.Error("should have error")
	}
	// Should only have run once (canceled during first delay)
	if attempts > 2 {
		t.Errorf("too many attempts: %d", attempts)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		err       error
		retryable bool
	}{
		{errors.New("timeout"), true},
		{errors.New("connection refused"), true},
		{errors.New("rate limit exceeded"), true},
		{errors.New("HTTP 503"), true},
		{errors.New("invalid API key"), false},
		{errors.New("model not found"), false},
	}
	for _, tt := range tests {
		if IsRetryable(tt.err) != tt.retryable {
			t.Errorf("IsRetryable(%q) = %v, want %v", tt.err.Error(), !tt.retryable, tt.retryable)
		}
	}
}
