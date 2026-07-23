// Package queue implements abort, continue, steering, and follow-up message
// queues for the agent loop.
package queue

import (
	"sync"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// QueueMode controls how many pending messages are injected at each drain point.
type QueueMode string

const (
	// QueueModeAll drains all pending messages at once.
	QueueModeAll QueueMode = "all"
	// QueueModeOneAtATime drains one message per drain point.
	QueueModeOneAtATime QueueMode = "one-at-a-time"
)

// Manager handles steering and follow-up message injection.
type Manager struct {
	mu          sync.Mutex
	steering    []*model.UserMessage
	followUp    []*model.UserMessage
	aborted     bool
	abortReason string
	mode        QueueMode
}

// NewManager creates a queue manager.
func NewManager(mode QueueMode) *Manager {
	if mode == "" {
		mode = QueueModeAll
	}
	return &Manager{mode: mode}
}

// Abort signals the agent loop to stop.
func (m *Manager) Abort(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.aborted = true
	if reason != "" {
		m.abortReason = reason
	}
}

// IsAborted returns whether abort has been requested.
func (m *Manager) IsAborted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.aborted
}

// AbortReason returns the abort reason.
func (m *Manager) AbortReason() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.abortReason
}

// Reset clears the abort flag.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.aborted = false
	m.abortReason = ""
}

// PushSteering adds a steering message to inject mid-run.
func (m *Manager) PushSteering(msg *model.UserMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.steering = append(m.steering, msg)
}

// PushFollowUp adds a follow-up message for after the agent stops.
func (m *Manager) PushFollowUp(msg *model.UserMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.followUp = append(m.followUp, msg)
}

// DrainSteering returns all pending steering messages.
func (m *Manager) DrainSteering() []*model.UserMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.steering) == 0 {
		return nil
	}
	out := m.steering
	m.steering = nil
	return out
}

// DrainFollowUp returns pending follow-up messages according to the queue mode.
func (m *Manager) DrainFollowUp() []*model.UserMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.followUp) == 0 {
		return nil
	}
	if m.mode == QueueModeOneAtATime {
		msg := m.followUp[0]
		m.followUp = m.followUp[1:]
		return []*model.UserMessage{msg}
	}
	out := m.followUp
	m.followUp = nil
	return out
}

// HasPending returns true if there are pending steering or follow-up messages.
func (m *Manager) HasPending() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.steering) > 0 || len(m.followUp) > 0
}
