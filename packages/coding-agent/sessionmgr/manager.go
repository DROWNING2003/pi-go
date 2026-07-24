// Package sessionmgr provides session management matching TS session-manager.ts.
package sessionmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/util"
	"github.com/DROWNING2003/pi-go/packages/storage/session"
)

// Manager handles session lifecycle: create, open, list, write entries.
type Manager struct {
	mu      sync.Mutex
	dir     string
	current *session.Session
	entries []session.Entry
}

// New creates a session manager rooted at the given directory.
func New(sessionsDir string) *Manager {
	return &Manager{dir: sessionsDir}
}

// Create creates a new session file.
func (m *Manager) Create(cwd string) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := util.UUIDv7()
	s := session.New(id, cwd)
	path := filepath.Join(m.dir, id+".jsonl")
	s.SetPath(path)

	if err := os.MkdirAll(m.dir, 0700); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}

	m.current = s
	m.entries = nil
	return s, nil
}

// Open opens an existing session by path or ID prefix.
func (m *Manager) Open(pathOrID string) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try direct path first
	if _, err := os.Stat(pathOrID); err == nil {
		s, err := session.Load(pathOrID)
		if err != nil {
			return nil, fmt.Errorf("load session: %w", err)
		}
		m.current = s
		return s, nil
	}

	// Try ID prefix
	s, err := session.FindByID(m.dir, pathOrID)
	if err != nil {
		return nil, fmt.Errorf("find session: %w", err)
	}
	if s == nil {
		return nil, fmt.Errorf("session not found: %s", pathOrID)
	}
	m.current = s
	return s, nil
}

// Current returns the current open session.
func (m *Manager) Current() *session.Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// List returns all sessions in the directory.
func (m *Manager) List() ([]session.Info, error) {
	return session.List(m.dir)
}

// AppendEntry appends an entry to the current session.
func (m *Manager) AppendEntry(entry session.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		return fmt.Errorf("no session open")
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	m.current.Append(data)
	m.entries = append(m.entries, entry)
	return nil
}

// Save persists the current session to disk.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		return fmt.Errorf("no session open")
	}
	return m.current.Save()
}

// Fork creates a new session forked from the current one at a given entry.
func (m *Manager) Fork(entryID string) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		return nil, fmt.Errorf("no session open")
	}

	newID := util.UUIDv7()
	forked := m.current.Fork(newID)
	path := filepath.Join(m.dir, newID+".jsonl")
	forked.SetPath(path)

	if err := forked.Save(); err != nil {
		return nil, fmt.Errorf("save fork: %w", err)
	}

	return forked, nil
}

// Stats computes statistics for the current session.
func (m *Manager) Stats() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil || len(m.entries) == 0 {
		return map[string]interface{}{"messageCount": 0}
	}

	var entries []json.RawMessage
	for _, e := range m.entries {
		data, _ := json.Marshal(e)
		entries = append(entries, data)
	}

	// Parse entries for stats
	msgCount := 0
	totalInput := 0
	totalOutput := 0
	lastModel := ""

	for _, raw := range entries {
		var e struct {
			Type  string `json:"type"`
			Role  string `json:"role"`
			Model string `json:"model"`
			Usage struct {
				Input  int `json:"input"`
				Output int `json:"output"`
			} `json:"usage"`
		}
		if json.Unmarshal(raw, &e) == nil {
			if e.Type == "message" {
				msgCount++
				if e.Usage.Input > 0 {
					totalInput += e.Usage.Input
					totalOutput += e.Usage.Output
				}
				if e.Model != "" {
					lastModel = e.Model
				}
			}
		}
	}

	return map[string]interface{}{
		"messageCount": msgCount,
		"totalInput":   totalInput,
		"totalOutput":  totalOutput,
		"lastModel":    lastModel,
	}
}

// GetEntries returns entries optionally filtered since a given entry ID.
func (m *Manager) GetEntries(since string) []session.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()

	if since == "" {
		result := make([]session.Entry, len(m.entries))
		copy(result, m.entries)
		return result
	}

	found := false
	var result []session.Entry
	for _, e := range m.entries {
		if e.ID == since {
			found = true
			continue
		}
		if found {
			result = append(result, e)
		}
	}
	return result
}

// Tree returns the session tree from all sessions.
func (m *Manager) Tree() []*session.TreeNode {
	sessions, _ := m.List()
	return session.BuildTree(sessions)
}

// SetLabel sets a label on a tree node.
func (m *Manager) SetLabel(targetID, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		return fmt.Errorf("no session open")
	}

	entry := session.CreateLabelEntry(targetID, label)
	data, _ := json.Marshal(entry)
	m.current.Append(data)
	return nil
}

// Ensure sort and time imports
var _ = sort.Strings
var _ = time.Now
