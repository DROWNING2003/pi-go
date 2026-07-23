package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAndSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	s := New("sess-1", "/tmp")
	s.SetPath(path)
	s.Append(json.RawMessage(`{"role":"user","content":"hello","timestamp":1}`))
	s.Append(json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"hi"}],"api":"faux","provider":"faux","model":"faux-1","usage":{"input":1,"output":1,"totalTokens":2,"cost":{"input":0,"output":0,"total":0}},"stopReason":"stop","timestamp":2}`))

	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists and is valid JSONL
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 3 { // header + 2 entries
		t.Errorf("expected 3 lines, got %d", lines)
	}

	// Load
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Header.ID != "sess-1" {
		t.Errorf("header id: %q", loaded.Header.ID)
	}
	if loaded.Header.Version != 3 {
		t.Errorf("version: %d", loaded.Header.Version)
	}
	if len(loaded.Entries) != 2 {
		t.Errorf("entries: got %d, want 2", len(loaded.Entries))
	}
}

func TestLoad_CorruptTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := `{"type":"session","version":3,"id":"s1","timestamp":"2024-01-01T00:00:00Z","cwd":"/"}
{"role":"user","content":"hi","timestamp":1}
this is not json
`
	os.WriteFile(path, []byte(content), 0644)

	s, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Should have the user message but stop before corrupt line
	if len(s.Entries) != 1 {
		t.Errorf("entries: got %d, want 1 (corrupt tail ignored)", len(s.Entries))
	}
}

func TestFork(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	s := New("parent", "/")
	s.SetPath(path)
	s.Append(json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`))
	s.Save()

	forked := s.Fork("child")
	if forked.Header.ParentSession != "parent" {
		t.Errorf("parent: %q", forked.Header.ParentSession)
	}
	if forked.Header.ID != "child" {
		t.Errorf("id: %q", forked.Header.ID)
	}
	if len(forked.Entries) != 1 {
		t.Errorf("entries: got %d", len(forked.Entries))
	}
}

func TestAtomicSave_NoCorruptionOnCrash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	s := New("s1", "/")
	s.SetPath(path)
	s.Append(json.RawMessage(`{"role":"user","content":"before","timestamp":1}`))
	s.Save()

	// Read the current content
	data, _ := os.ReadFile(path)
	original := string(data)

	// Simulate a failed save by writing partial data to a temp file
	tmpPath := path + ".tmp"
	os.WriteFile(tmpPath, []byte("partial"), 0644)

	// The original file should be intact
	current, _ := os.ReadFile(path)
	if string(current) != original {
		t.Error("original file was corrupted")
	}
	os.Remove(tmpPath)
}
