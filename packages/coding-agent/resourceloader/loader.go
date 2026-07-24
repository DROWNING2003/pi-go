// Package resourceloader loads project resources: config files, context, skills.
package resourceloader

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/DROWNING2003/pi-go/packages/agent/prompt"
)

// Resources holds loaded project resources.
type Resources struct {
	Config       map[string]interface{}
	ContextFiles []string
	Skills       []prompt.Skill
	Extensions   []string
	Warnings     []string
}

// Load loads project resources from a working directory.
func Load(cwd string) *Resources {
	r := &Resources{
		Config: make(map[string]interface{}),
	}

	// Load .pi/ directory if it exists
	piDir := filepath.Join(cwd, ".pi")
	if info, err := os.Stat(piDir); err == nil && info.IsDir() {
		r.loadConfig(filepath.Join(piDir, "config.json"))
		r.loadContextFiles(cwd)
		r.Skills = prompt.LoadSkills(filepath.Join(piDir, "skills"))
	}

	// Load AGENTS.md / CLAUDE.md from project root and home
	r.ContextFiles = append(r.ContextFiles, loadContextFiles(cwd)...)
	if home, err := os.UserHomeDir(); err == nil && home != cwd {
		r.ContextFiles = append(r.ContextFiles, loadContextFiles(home)...)
	}

	return r
}

func (r *Resources) loadConfig(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Simple key-value config
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		// Simple JSON-like key: value
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(strings.Trim(parts[0], "\"'"))
			val := strings.TrimSpace(strings.Trim(parts[1], "\"',"))
			r.Config[key] = val
		}
	}
}

func (r *Resources) loadContextFiles(cwd string) {
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		path := filepath.Join(cwd, name)
		if data, err := os.ReadFile(path); err == nil {
			r.ContextFiles = append(r.ContextFiles, string(data))
		}
	}
}

func loadContextFiles(dir string) []string {
	var files []string
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		path := filepath.Join(dir, name)
		if data, err := os.ReadFile(path); err == nil {
			files = append(files, string(data))
		}
	}
	return files
}
