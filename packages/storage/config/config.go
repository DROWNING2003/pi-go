// Package config manages user configuration, context files, and trust
// settings for the coding agent.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the user's configuration settings.
type Config struct {
	Model        string            `json:"model,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	SystemPrompt string            `json:"systemPrompt,omitempty"`
	ContextFiles []string          `json:"contextFiles,omitempty"`
	TrustedDirs  []string          `json:"trustedDirs,omitempty"`
	MaxTurns     int               `json:"maxTurns,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Theme        string            `json:"theme,omitempty"`
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxTurns: 10,
	}
}

// Load reads configuration from the following locations (in priority order):
// 1. Project config: <cwd>/.pi.json
// 2. Global config: <configDir>/config.json
func Load(cwd, configDir string) (*Config, error) {
	cfg := DefaultConfig()

	// Global config
	globalPath := filepath.Join(configDir, "config.json")
	if data, err := os.ReadFile(globalPath); err == nil {
		var global Config
		if err := json.Unmarshal(data, &global); err != nil {
			return nil, fmt.Errorf("global config: %w", err)
		}
		merge(cfg, &global)
	}

	// Project config (overrides global)
	if cwd != "" {
		projectPath := filepath.Join(cwd, ".pi.json")
		if data, err := os.ReadFile(projectPath); err == nil {
			var project Config
			if err := json.Unmarshal(data, &project); err != nil {
				return nil, fmt.Errorf("project config: %w", err)
			}
			merge(cfg, &project)
		}
	}

	return cfg, nil
}

// LoadContextFiles reads AGENTS.md / CLAUDE.md style context files.
// Loading order: project root -> home directory.
func LoadContextFiles(cwd, homeDir string) ([]string, error) {
	var files []string

	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		if cwd != "" {
			path := filepath.Join(cwd, name)
			if data, err := os.ReadFile(path); err == nil {
				files = append(files, string(data))
			}
		}
		if homeDir != "" && homeDir != cwd {
			path := filepath.Join(homeDir, name)
			if data, err := os.ReadFile(path); err == nil {
				files = append(files, string(data))
			}
		}
	}

	return files, nil
}

// TrustManager tracks which directories the user has trusted.
type TrustManager struct {
	configDir string
	trusted   map[string]bool
}

// NewTrustManager creates a trust manager.
func NewTrustManager(configDir string) *TrustManager {
	tm := &TrustManager{
		configDir: configDir,
		trusted:   make(map[string]bool),
	}
	tm.load()
	return tm
}

func (tm *TrustManager) load() {
	path := filepath.Join(tm.configDir, "trusted.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var trusted []string
	if json.Unmarshal(data, &trusted) == nil {
		for _, d := range trusted {
			tm.trusted[d] = true
		}
	}
}

func (tm *TrustManager) save() error {
	dir := tm.configDir
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	var trusted []string
	for d := range tm.trusted {
		trusted = append(trusted, d)
	}
	data, _ := json.Marshal(trusted)
	return os.WriteFile(filepath.Join(dir, "trusted.json"), data, 0600)
}

// IsTrusted returns whether a directory is trusted.
func (tm *TrustManager) IsTrusted(dir string) bool {
	// Resolve symlinks
	resolved, err := filepath.EvalSymlinks(dir)
	if err == nil {
		dir = resolved
	}
	return tm.trusted[dir]
}

// Trust marks a directory as trusted.
func (tm *TrustManager) Trust(dir string) error {
	resolved, err := filepath.EvalSymlinks(dir)
	if err == nil {
		dir = resolved
	}
	tm.trusted[dir] = true
	return tm.save()
}

func merge(dst, src *Config) {
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.Provider != "" {
		dst.Provider = src.Provider
	}
	if src.SystemPrompt != "" {
		dst.SystemPrompt = src.SystemPrompt
	}
	if len(src.ContextFiles) > 0 {
		dst.ContextFiles = src.ContextFiles
	}
	if len(src.TrustedDirs) > 0 {
		dst.TrustedDirs = src.TrustedDirs
	}
	if src.MaxTurns != 0 {
		dst.MaxTurns = src.MaxTurns
	}
	if src.Theme != "" {
		dst.Theme = src.Theme
	}
}
