package store

import (
	"testing"

	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

func TestReadWriteDelete(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	entry := &ModelStoreEntry{
		Models: []provider.ModelConfig{
			{ID: "test-model", Name: "Test"},
		},
		CheckedAt: 12345,
	}

	// Write
	if err := s.Write("test", entry); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read
	loaded, err := s.Read("test")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(loaded.Models) != 1 || loaded.Models[0].ID != "test-model" {
		t.Errorf("models: %+v", loaded.Models)
	}
	if loaded.CheckedAt != 12345 {
		t.Errorf("checkedAt: %d", loaded.CheckedAt)
	}

	// Delete
	if err := s.Delete("test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Read("test"); err == nil {
		t.Error("should fail after delete")
	}
}
