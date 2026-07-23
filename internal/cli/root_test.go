package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunShowsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--help"}, &stdout, &stderr, "dev")

	if exitCode != 0 {
		t.Fatalf("Run(--help) exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage: pi") {
		t.Fatalf("Run(--help) output = %q, want usage", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run(--help) stderr = %q, want empty", stderr.String())
	}
}

func TestRunShowsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--version"}, &stdout, &stderr, "v0.0.0-dev")

	if exitCode != 0 {
		t.Fatalf("Run(--version) exit code = %d, want 0", exitCode)
	}
	if got := stdout.String(); got != "pi v0.0.0-dev\n" {
		t.Fatalf("Run(--version) output = %q, want %q", got, "pi v0.0.0-dev\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run(--version) stderr = %q, want empty", stderr.String())
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--unknown"}, &stdout, &stderr, "dev")

	if exitCode != 2 {
		t.Fatalf("Run(--unknown) exit code = %d, want 2", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Run(--unknown) stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown option") {
		t.Fatalf("Run(--unknown) stderr = %q, want unknown option error", stderr.String())
	}
}
