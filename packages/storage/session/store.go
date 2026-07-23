// Package session implements JSONL-based session storage with atomic writes,
// resume, fork, and tree navigation.
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Header is the first line of a session JSONL file.
type Header struct {
	Type          string `json:"type"`
	Version       int    `json:"version"`
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	CWD           string `json:"cwd"`
	ParentSession string `json:"parentSession,omitempty"`
}

// Entry is a generic entry in the session JSONL file.
type Entry struct {
	Type string `json:"type"`
	// Message entry fields
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
	// Tree entry fields
	Action   string `json:"action,omitempty"`
	TargetID string `json:"targetId,omitempty"`
	Label    string `json:"label,omitempty"`
}

// Session represents an in-memory session with its entries.
type Session struct {
	mu      sync.Mutex
	Header  Header
	Entries []json.RawMessage
	path    string
	dirty   bool
}

// New creates a new session with a unique ID.
func New(id, cwd string) *Session {
	return &Session{
		Header: Header{
			Type:      "session",
			Version:   3,
			ID:        id,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CWD:       cwd,
		},
	}
}

// Load reads a session from a JSONL file.
func Load(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	lineNum := 0
	var header Header
	var entries []json.RawMessage

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		if lineNum == 1 {
			if err := json.Unmarshal(line, &header); err != nil {
				return nil, fmt.Errorf("line %d: invalid session header: %w", lineNum, err)
			}
			if header.Type != "session" {
				return nil, fmt.Errorf("line %d: expected session header, got %q", lineNum, header.Type)
			}
			continue
		}

		// Trim trailing whitespace for corrupt recovery
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			entries = append(entries, line)
			continue
		}

		if !json.Valid([]byte(trimmed)) {
			// Corrupt tail: stop reading, keep valid prefix
			break
		}

		entries = append(entries, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}

	return &Session{
		Header:  header,
		Entries: entries,
		path:    path,
	}, nil
}

// Append adds entries to the session.
func (s *Session) Append(entries ...json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Entries = append(s.Entries, entries...)
	s.dirty = true
}

// Save writes the session atomically to its file path.
func (s *Session) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Session) saveLocked() error {
	if s.path == "" {
		return fmt.Errorf("session has no path")
	}

	dir := filepath.Dir(s.path)
	tmpFile, err := os.CreateTemp(dir, ".session-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write header
	headerData, _ := json.Marshal(s.Header)
	if _, err := tmpFile.Write(append(headerData, '\n')); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write header: %w", err)
	}

	// Write entries
	for _, entry := range s.Entries {
		if _, err := tmpFile.Write(append(entry, '\n')); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("write entry: %w", err)
		}
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp: %w", err)
	}

	s.dirty = false
	return nil
}

// Fork creates a new session from this one with a new ID.
func (s *Session) Fork(newID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := make([]json.RawMessage, len(s.Entries))
	copy(entries, s.Entries)
	return &Session{
		Header: Header{
			Type:          "session",
			Version:       3,
			ID:            newID,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			CWD:           s.Header.CWD,
			ParentSession: s.Header.ID,
		},
		Entries: entries,
	}
}

// Path returns the session file path.
func (s *Session) Path() string {
	return s.path
}

// SetPath sets the session file path.
func (s *Session) SetPath(path string) {
	s.path = path
}

// IsDirty returns true if there are unsaved changes.
func (s *Session) IsDirty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirty
}

// Ensure json is used
var _ = json.RawMessage{}
