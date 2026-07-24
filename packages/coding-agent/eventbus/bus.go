// Package eventbus provides a simple pub/sub event bus for agent events.
package eventbus

import "sync"

// Bus is a simple event bus for agent lifecycle events.
type Bus struct {
	mu   sync.RWMutex
	subs map[string][]chan interface{}
}

// New creates a new event bus.
func New() *Bus {
	return &Bus{subs: make(map[string][]chan interface{})}
}

// Subscribe registers a channel for an event type.
func (b *Bus) Subscribe(eventType string) <-chan interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan interface{}, 64)
	b.subs[eventType] = append(b.subs[eventType], ch)
	return ch
}

// Emit sends an event to all subscribers.
func (b *Bus) Emit(eventType string, data interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[eventType] {
		select {
		case ch <- data:
		default:
		}
	}
}

// CloseAll closes all subscriber channels.
func (b *Bus) CloseAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, chs := range b.subs {
		for _, ch := range chs {
			close(ch)
		}
	}
	b.subs = make(map[string][]chan interface{})
}
