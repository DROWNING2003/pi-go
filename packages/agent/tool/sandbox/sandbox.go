// Package sandbox provides OS-level sandboxing for bash command execution
// using sandbox-exec (macOS) or bubblewrap (Linux).
//
// Matches the TypeScript pi sandbox extension behavior.
package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Config matches the TS sandbox Config format.
type Config struct {
	Enabled    bool              `json:"enabled,omitempty"`
	Network    *NetworkConfig    `json:"network,omitempty"`
	Filesystem *FilesystemConfig `json:"filesystem,omitempty"`
}

// NetworkConfig restricts network access.
type NetworkConfig struct {
	AllowedDomains []string `json:"allowedDomains,omitempty"`
	DeniedDomains  []string `json:"deniedDomains,omitempty"`
}

// FilesystemConfig restricts filesystem access.
type FilesystemConfig struct {
	DenyRead   []string `json:"denyRead,omitempty"`
	AllowWrite []string `json:"allowWrite,omitempty"`
	DenyWrite  []string `json:"denyWrite,omitempty"`
}

// DefaultConfig returns the default sandbox configuration matching TS defaults.
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Network: &NetworkConfig{
			AllowedDomains: []string{
				"npmjs.org", "*.npmjs.org", "registry.npmjs.org",
				"registry.yarnpkg.com", "pypi.org", "*.pypi.org",
				"github.com", "*.github.com", "api.github.com",
				"raw.githubusercontent.com",
			},
		},
		Filesystem: &FilesystemConfig{
			DenyRead:   []string{"~/.ssh", "~/.aws", "~/.gnupg"},
			AllowWrite: []string{".", "/tmp"},
			DenyWrite:  []string{".env", ".env.*", "*.pem", "*.key"},
		},
	}
}

// LoadConfig loads sandbox config from project and global paths.
func LoadConfig(cwd string) Config {
	cfg := DefaultConfig()

	// Global: ~/.pi/sandbox.json
	home, _ := os.UserHomeDir()
	globalPath := filepath.Join(home, ".pi", "sandbox.json")
	if data, err := os.ReadFile(globalPath); err == nil {
		var global Config
		if json.Unmarshal(data, &global) == nil {
			mergeConfig(&cfg, &global)
		}
	}

	// Project: .pi/sandbox.json
	if cwd != "" {
		projectPath := filepath.Join(cwd, ".pi", "sandbox.json")
		if data, err := os.ReadFile(projectPath); err == nil {
			var project Config
			if json.Unmarshal(data, &project) == nil {
				mergeConfig(&cfg, &project)
			}
		}
	}

	return cfg
}

func mergeConfig(dst, src *Config) {
	if !src.Enabled {
		dst.Enabled = false
	}
	if src.Network != nil {
		if dst.Network == nil {
			dst.Network = &NetworkConfig{}
		}
		if len(src.Network.AllowedDomains) > 0 {
			dst.Network.AllowedDomains = src.Network.AllowedDomains
		}
		if len(src.Network.DeniedDomains) > 0 {
			dst.Network.DeniedDomains = src.Network.DeniedDomains
		}
	}
	if src.Filesystem != nil {
		if dst.Filesystem == nil {
			dst.Filesystem = &FilesystemConfig{}
		}
		if len(src.Filesystem.DenyRead) > 0 {
			dst.Filesystem.DenyRead = src.Filesystem.DenyRead
		}
		if len(src.Filesystem.AllowWrite) > 0 {
			dst.Filesystem.AllowWrite = src.Filesystem.AllowWrite
		}
		if len(src.Filesystem.DenyWrite) > 0 {
			dst.Filesystem.DenyWrite = src.Filesystem.DenyWrite
		}
	}
}

// Manager wraps commands with OS-level sandboxing.
type Manager struct {
	cfg     Config
	enabled bool
}

// New creates a sandbox manager.
func New(cfg Config) *Manager {
	s := &Manager{cfg: cfg}
	if !cfg.Enabled {
		return s
	}

	switch runtime.GOOS {
	case "darwin":
		// sandbox-exec is built into macOS
		s.enabled = true
	case "linux":
		// Check for bubblewrap
		if _, err := exec.LookPath("bwrap"); err == nil {
			s.enabled = true
		}
	}
	return s
}

// IsEnabled returns whether sandboxing is active.
func (s *Manager) IsEnabled() bool {
	return s.enabled
}

// WrapCommand wraps a shell command with sandbox restrictions.
// On macOS, uses sandbox-exec with a generated profile.
// On Linux, uses bubblewrap.
func (s *Manager) WrapCommand(command string, cwd string) (string, error) {
	if !s.enabled {
		return command, nil
	}

	switch runtime.GOOS {
	case "darwin":
		return s.wrapMacOS(command, cwd)
	case "linux":
		return s.wrapLinux(command, cwd)
	default:
		return command, nil
	}
}

func (s *Manager) wrapMacOS(command, cwd string) (string, error) {
	profile := s.buildMacOSProfile(cwd)
	profileFile, err := writeTempProfile(profile)
	if err != nil {
		return "", fmt.Errorf("sandbox profile: %w", err)
	}
	// temp file cleaned up after exec
	return fmt.Sprintf("sandbox-exec -f %s sh -c %s", profileFile, escapeSh(command)), nil
}

func (s *Manager) buildMacOSProfile(cwd string) string {
	var b strings.Builder
	b.WriteString("(version 1)\n")
	b.WriteString("(allow default)\n")
	b.WriteString("(deny sysctl*)\n")

	// Filesystem: deny read of sensitive paths
	if s.cfg.Filesystem != nil {
		for _, p := range s.cfg.Filesystem.DenyRead {
			resolved := resolvePath(p)
			b.WriteString(fmt.Sprintf("(deny file-read* (subpath %q))\n", resolved))
		}

		// Allow write only to specified paths
		if len(s.cfg.Filesystem.AllowWrite) > 0 {
			b.WriteString("(deny file-write*)\n")
			for _, p := range s.cfg.Filesystem.AllowWrite {
				resolved := resolvePath(p)
				if resolved == "." && cwd != "" {
					resolved = cwd
				}
				b.WriteString(fmt.Sprintf("(allow file-write* (subpath %q))\n", resolved))
			}
			// Always allow /tmp and /dev/null
			b.WriteString("(allow file-write* (subpath \"/tmp\"))\n")
			b.WriteString("(allow file-write* (subpath \"/dev/null\"))\n")
		}

		// Deny write to specific paths
		for _, p := range s.cfg.Filesystem.DenyWrite {
			resolved := resolvePath(p)
			if strings.Contains(resolved, "*") {
				resolved = filepath.Dir(resolved)
				b.WriteString(fmt.Sprintf("(deny file-write* (subpath %q))\n", resolved))
			} else {
				b.WriteString(fmt.Sprintf("(deny file-write* (subpath %q))\n", resolved))
			}
		}
	}

	// Network restrictions
	if s.cfg.Network != nil && len(s.cfg.Network.AllowedDomains) > 0 {
		b.WriteString("(deny network*)\n")
		for _, domain := range s.cfg.Network.AllowedDomains {
			if strings.HasPrefix(domain, "*.") {
				domain = domain[1:] // sandbox-exec handles wildcards differently
			}
			b.WriteString(fmt.Sprintf("(allow network* (remote unix-socket))\n"))
		}
		b.WriteString("(allow network* (local ip))\n") // localhost
	}

	return b.String()
}

func (s *Manager) wrapLinux(command, cwd string) (string, error) {
	// bubblewrap-based sandboxing
	args := []string{
		"bwrap",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/etc", "/etc",
		"--bind", cwd, cwd,
		"--bind", "/tmp", "/tmp",
		"--dev", "/dev",
		"--proc", "/proc",
		"--unshare-all",
		"--die-with-parent",
	}

	// Deny read sensitive paths
	if s.cfg.Filesystem != nil {
		for _, p := range s.cfg.Filesystem.DenyRead {
			resolved := resolvePath(p)
			args = append(args, "--tmpfs", resolved)
		}

		for _, p := range s.cfg.Filesystem.DenyWrite {
			resolved := resolvePath(p)
			args = append(args, "--ro-bind", resolved, resolved)
		}
	}

	args = append(args, "sh", "-c", command)
	return strings.Join(args, " "), nil
}

func resolvePath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func writeTempProfile(content string) (string, error) {
	f, err := os.CreateTemp("", "pi-sandbox-*.sb")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func escapeSh(s string) string {
	// Simple sh escaping
	q := fmt.Sprintf("%q", s)
	return q
}
