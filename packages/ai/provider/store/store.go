// Package store provides persistent model catalog storage.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// ModelStoreEntry is a cached model catalog for a provider.
type ModelStoreEntry struct {
	Models       []provider.ModelConfig `json:"models"`
	LastModified int64                  `json:"lastModified,omitempty"`
	CheckedAt    int64                  `json:"checkedAt,omitempty"`
}

// Store persists model catalogs to disk.
type Store struct {
	mu  sync.RWMutex
	dir string
}

// NewStore creates a model store in the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Read returns the cached model catalog for a provider.
func (s *Store) Read(providerID string) (*ModelStoreEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.dir, "models", providerID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entry ModelStoreEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// Write saves a model catalog for a provider.
func (s *Store) Write(providerID string, entry *ModelStoreEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.dir, "models")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, providerID+".json"), data, 0644)
}

// Delete removes a cached model catalog.
func (s *Store) Delete(providerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(filepath.Join(s.dir, "models", providerID+".json"))
}
