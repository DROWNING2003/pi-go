package sandbox

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := LoadConfig("/nonexistent")
	if !cfg.Enabled {
		t.Error("sandbox should be enabled by default")
	}
	if cfg.Network == nil || len(cfg.Network.AllowedDomains) == 0 {
		t.Error("should have default allowed domains")
	}
	if cfg.Filesystem == nil {
		t.Error("should have filesystem config")
	}
}

func TestLoadConfig_ProjectOverride(t *testing.T) {
	dir := t.TempDir()
	piDir := filepath.Join(dir, ".pi")
	os.MkdirAll(piDir, 0755)
	os.WriteFile(filepath.Join(piDir, "sandbox.json"), []byte(`{"enabled":false}`), 0644)

	cfg := LoadConfig(dir)
	if cfg.Enabled {
		t.Error("sandbox should be disabled by project config")
	}
}

func TestMergeConfig(t *testing.T) {
	dst := DefaultConfig()
	src := Config{Enabled: false}
	mergeConfig(&dst, &src)
	if dst.Enabled {
		t.Error("should merge enabled=false")
	}
}

func TestWrapCommand_MacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS only")
	}

	m := New(DefaultConfig())
	if !m.IsEnabled() {
		t.Skip("sandbox-exec not available")
	}

	cmd, err := m.WrapCommand("echo hello", "/tmp")
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	if cmd == "echo hello" {
		t.Error("command should be wrapped")
	}
}

func TestWrapCommand_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only")
	}

	m := New(DefaultConfig())
	if !m.IsEnabled() {
		t.Skip("bubblewrap not available")
	}

	cmd, err := m.WrapCommand("echo hello", "/tmp")
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	if cmd == "echo hello" {
		t.Error("command should be wrapped")
	}
}

func TestDisabledSandbox(t *testing.T) {
	m := New(Config{Enabled: false})
	if m.IsEnabled() {
		t.Error("should be disabled")
	}
	cmd, _ := m.WrapCommand("rm -rf /", "/tmp")
	if cmd != "rm -rf /" {
		t.Error("disabled sandbox should not wrap")
	}
}

func TestResolvePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	result := resolvePath("~/.ssh")
	expected := filepath.Join(home, ".ssh")
	if result != expected {
		t.Errorf("resolvePath: %q != %q", result, expected)
	}
	if resolvePath("/etc") != "/etc" {
		t.Error("absolute path should stay same")
	}
}
