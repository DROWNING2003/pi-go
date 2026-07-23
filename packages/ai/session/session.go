// Package session provides session resource cleanup management.
package session

import (
	"fmt"
	"sync"
)

// CleanupFunc is called to release resources when a session ends.
type CleanupFunc func(sessionID string)

var (
	mu       sync.Mutex
	cleanups []CleanupFunc
)

// RegisterCleanup registers a cleanup function that runs on session shutdown.
func RegisterCleanup(fn CleanupFunc) func() {
	mu.Lock()
	defer mu.Unlock()
	cleanups = append(cleanups, fn)
	idx := len(cleanups) - 1
	return func() {
		mu.Lock()
		defer mu.Unlock()
		cleanups[idx] = nil
	}
}

// Cleanup runs all registered cleanup functions.
func Cleanup(sessionID string) error {
	mu.Lock()
	fns := make([]CleanupFunc, len(cleanups))
	copy(fns, cleanups)
	mu.Unlock()

	var errs []error
	for _, fn := range fns {
		if fn == nil {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					errs = append(errs, fmt.Errorf("cleanup panic: %v", r))
				}
			}()
			fn(sessionID)
		}()
	}
	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
}
