// Package modelconfig provides model configuration loading.
package modelconfig

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// Config holds resolved model configuration.
type Config struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	System   string `json:"system,omitempty"`
}

// Load reads model config from project and global locations.
func Load(cwd, configDir string) (*Config, error) {
	cfg := &Config{}

	// Global: ~/.pi/config.json
	globalPath := filepath.Join(configDir, "config.json")
	if data, err := os.ReadFile(globalPath); err == nil {
		json.Unmarshal(data, cfg)
	}

	// Project: .pi/config.json
	if cwd != "" {
		projectPath := filepath.Join(cwd, ".pi", "config.json")
		if data, err := os.ReadFile(projectPath); err == nil {
			json.Unmarshal(data, cfg)
		}
	}

	return cfg, nil
}

// ResolveModel resolves a model reference to a concrete model.
func ResolveModel(ref string, reg *provider.Registry) *provider.ProviderModel {
	if ref == "" {
		return reg.ResolveModel("deepseek/deepseek-chat")
	}
	return reg.ResolveModel(ref)
}
