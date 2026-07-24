// Package codingutil provides shared utilities for the coding agent.
package codingutil

import (
	"fmt"
	"sync"
	"time"
)

// Timer measures elapsed time for operations.
type Timer struct {
	start time.Time
	name  string
}

// StartTimer creates a new timer.
func StartTimer(name string) *Timer {
	return &Timer{start: time.Now(), name: name}
}

// Elapsed returns the elapsed time and resets.
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// Stop logs and returns the elapsed time.
func (t *Timer) Stop() time.Duration {
	elapsed := time.Since(t.start)
	return elapsed
}

// FormatDuration formats a duration for display.
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// CacheStats tracks cache hit/miss statistics.
type CacheStats struct {
	mu     sync.Mutex
	hits   int64
	misses int64
}

// Hit records a cache hit.
func (c *CacheStats) Hit() {
	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
}

// Miss records a cache miss.
func (c *CacheStats) Miss() {
	c.mu.Lock()
	c.misses++
	c.mu.Unlock()
}

// Stats returns hit/miss counts and hit rate.
func (c *CacheStats) Stats() (hits, misses int64, rate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hits = c.hits
	misses = c.misses
	total := hits + misses
	if total > 0 {
		rate = float64(hits) / float64(total)
	}
	return
}

// UsageTotals tracks cumulative token usage.
type UsageTotals struct {
	mu           sync.Mutex
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	TotalCost    float64
}

// Add adds usage from a response.
func (u *UsageTotals) Add(input, output int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.InputTokens += input
	u.OutputTokens += output
	u.TotalTokens += input + output
}

// Summary returns a formatted usage summary.
func (u *UsageTotals) Summary() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return fmt.Sprintf("tokens: %d in / %d out / %d total | cost: $%.4f",
		u.InputTokens, u.OutputTokens, u.TotalTokens, u.TotalCost)
}
