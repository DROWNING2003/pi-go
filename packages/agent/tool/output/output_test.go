package output

import (
	"testing"
)

func TestShellResult_Format(t *testing.T) {
	r := ShellResult{Command: "ls", Stdout: "a\nb", ExitCode: 0}
	s := r.Format()
	if s == "" {
		t.Error("empty format")
	}
}

func TestShellResult_Error(t *testing.T) {
	r := ShellResult{Command: "bad", Stderr: "err", ExitCode: 1}
	s := r.Format()
	if s == "" {
		t.Error("empty format")
	}
}

func TestShellResult_Timeout(t *testing.T) {
	r := ShellResult{Command: "sleep 100", TimedOut: true}
	s := r.Format()
	if s == "" {
		t.Error("empty format")
	}
}
