package toolctx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMutationQueue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	q := NewQueue()
	q.Add(path, "hello", "hi")
	q.Add(path, "world", "earth")

	if q.Pending() != 2 {
		t.Errorf("pending: %d", q.Pending())
	}

	if err := q.Apply(); err != nil {
		t.Fatalf("apply: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hi earth" {
		t.Errorf("content: %q", data)
	}
	if q.Pending() != 0 {
		t.Errorf("pending after apply: %d", q.Pending())
	}
}

func TestMutationQueue_Conflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	q := NewQueue()
	q.Add(path, "nonexistent", "x")
	if err := q.Apply(); err == nil {
		t.Error("should fail on missing oldText")
	}
}
