// Package streamfn provides a global default stream function registry
// matching TS stream-fn.ts.
package streamfn

import (
	"context"
	"sync"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// StreamFunc is the signature for streaming model responses.
type StreamFunc func(ctx context.Context, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent

var (
	mu              sync.Mutex
	defaultStreamFn StreamFunc
)

// SetDefault configures the fallback stream function.
func SetDefault(fn StreamFunc) {
	mu.Lock()
	defer mu.Unlock()
	defaultStreamFn = fn
}

// GetDefault returns the default stream function.
func GetDefault() StreamFunc {
	mu.Lock()
	defer mu.Unlock()
	if defaultStreamFn == nil {
		panic("no default stream function configured")
	}
	return defaultStreamFn
}
