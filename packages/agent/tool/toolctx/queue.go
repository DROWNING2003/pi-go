// Package toolctx provides a file mutation queue for the edit tool,
// ensuring consistent file state across multiple edits.
package toolctx

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Mutation represents a pending file change.
type Mutation struct {
	Path    string
	OldText string
	NewText string
	Applied bool
}

// Queue tracks pending file mutations to ensure consistency.
type Queue struct {
	mu        sync.Mutex
	mutations []*Mutation
}

// NewQueue creates an empty mutation queue.
func NewQueue() *Queue {
	return &Queue{}
}

// Add adds a mutation to the queue.
func (q *Queue) Add(path, oldText, newText string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.mutations = append(q.mutations, &Mutation{Path: path, OldText: oldText, NewText: newText})
}

// Apply applies all queued mutations. Returns an error if any mutation fails,
// and all successful mutations remain applied.
func (q *Queue) Apply() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, m := range q.mutations {
		if m.Applied {
			continue
		}
		data, err := os.ReadFile(m.Path)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.Path, err)
		}
		content := string(data)
		count := strings.Count(content, m.OldText)
		if count == 0 {
			return fmt.Errorf("oldText not found in %s", m.Path)
		}
		if count > 1 {
			return fmt.Errorf("oldText matches %d times in %s, must be unique", count, m.Path)
		}
		newContent := strings.Replace(content, m.OldText, m.NewText, 1)
		if err := os.WriteFile(m.Path, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("write %s: %w", m.Path, err)
		}
		m.Applied = true
	}
	return nil
}

// Pending returns the number of pending mutations.
func (q *Queue) Pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	count := 0
	for _, m := range q.mutations {
		if !m.Applied {
			count++
		}
	}
	return count
}

// Reset clears the queue without applying.
func (q *Queue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.mutations = nil
}
