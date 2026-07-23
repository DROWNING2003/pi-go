package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Info holds metadata about a session file.
type Info struct {
	ID        string
	Path      string
	CWD       string
	Timestamp string
	Entries   int
}

// List scans a directory for session JSONL files and returns their metadata.
func List(sessionsDir string) ([]Info, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Info
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(sessionsDir, entry.Name())
		info, err := statSession(path)
		if err != nil {
			continue
		}
		sessions = append(sessions, *info)
	}

	// Sort by timestamp descending (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp > sessions[j].Timestamp
	})

	return sessions, nil
}

func statSession(path string) (*Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)

	// Read header
	var header Header
	if err := decoder.Decode(&header); err != nil {
		return nil, err
	}

	// Count remaining entries
	count := 0
	for decoder.More() {
		var dummy json.RawMessage
		if decoder.Decode(&dummy) == nil {
			count++
		}
	}

	return &Info{
		ID:        header.ID,
		Path:      path,
		CWD:       header.CWD,
		Timestamp: header.Timestamp,
		Entries:   count,
	}, nil
}

// Latest returns the most recent session from a directory.
func Latest(sessionsDir string) (*Session, error) {
	sessions, err := List(sessionsDir)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return Load(sessions[0].Path)
}

// FindByID finds a session by ID prefix.
func FindByID(sessionsDir, idPrefix string) (*Session, error) {
	sessions, err := List(sessionsDir)
	if err != nil {
		return nil, err
	}
	for _, s := range sessions {
		if strings.HasPrefix(s.ID, idPrefix) {
			return Load(s.Path)
		}
	}
	return nil, nil
}
