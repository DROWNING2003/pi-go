// Package settingsmgr provides settings management matching TS settings-manager.ts.
package settingsmgr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Settings holds user-configurable settings.
type Settings struct {
	Model        string            `json:"model,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	SystemPrompt string            `json:"systemPrompt,omitempty"`
	MaxTurns     int               `json:"maxTurns,omitempty"`
	Temperature  float64           `json:"temperature,omitempty"`
	Theme        string            `json:"theme,omitempty"`
	AutoCompact  bool              `json:"autoCompact,omitempty"`
	Extensions   []string          `json:"extensions,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	// Internal
	updatedAt time.Time `json:"-"`
}

// Manager manages in-memory settings with file persistence.
type Manager struct {
	mu       sync.RWMutex
	dir      string
	settings *Settings
}

// New creates a settings manager.
func New(configDir string) *Manager {
	m := &Manager{dir: configDir}
	m.load()
	return m
}

// Get returns a copy of the current settings.
func (m *Manager) Get() *Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.settings == nil {
		return &Settings{MaxTurns: 10, Temperature: 0.7}
	}
	s := *m.settings
	return &s
}

// Set updates a setting value and persists.
func (m *Manager) Set(updates *Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.settings == nil {
		m.settings = &Settings{}
	}

	if updates.Model != "" {
		m.settings.Model = updates.Model
	}
	if updates.Provider != "" {
		m.settings.Provider = updates.Provider
	}
	if updates.SystemPrompt != "" {
		m.settings.SystemPrompt = updates.SystemPrompt
	}
	if updates.MaxTurns > 0 {
		m.settings.MaxTurns = updates.MaxTurns
	}
	if updates.Temperature > 0 {
		m.settings.Temperature = updates.Temperature
	}
	if updates.Theme != "" {
		m.settings.Theme = updates.Theme
	}
	if len(updates.Extensions) > 0 {
		m.settings.Extensions = updates.Extensions
	}
	if updates.Env != nil {
		if m.settings.Env == nil {
			m.settings.Env = make(map[string]string)
		}
		for k, v := range updates.Env {
			m.settings.Env[k] = v
		}
	}

	m.settings.updatedAt = time.Now()
	return m.saveLocked()
}

// Reload re-reads settings from disk.
func (m *Manager) Reload() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.load()
}

// load reads settings from disk.
func (m *Manager) load() {
	path := filepath.Join(m.dir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		m.settings = &Settings{MaxTurns: 10, Temperature: 0.7}
		return
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		m.settings = &Settings{MaxTurns: 10, Temperature: 0.7}
		return
	}
	if s.MaxTurns <= 0 {
		s.MaxTurns = 10
	}
	m.settings = &s
}

// saveLocked persists settings to disk (caller must hold lock).
func (m *Manager) saveLocked() error {
	os.MkdirAll(m.dir, 0700)
	data, err := json.MarshalIndent(m.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.dir, "settings.json"), data, 0600)
}

// Save persists current settings.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

// Ensure json and time are used
var _ = json.Marshal
var _ = time.Now
