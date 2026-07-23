// Package output formats shell command output for display.
package output

import (
	"fmt"
	"strings"
)

// ShellResult formats a shell command result for display in the agent UI.
type ShellResult struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
	Aborted  bool
}

// Format formats a shell result for display to the model.
func (s *ShellResult) Format() string {
	var parts []string

	if s.TimedOut {
		parts = append(parts, fmt.Sprintf("Command timed out: %s", s.Command))
	} else {
		parts = append(parts, fmt.Sprintf("Command: %s", s.Command))
	}

	if s.Stdout != "" {
		parts = append(parts, s.Stdout)
	}
	if s.Stderr != "" {
		parts = append(parts, "[stderr]\n"+s.Stderr)
	}

	if s.Aborted {
		parts = append(parts, "[aborted]")
	} else if s.TimedOut {
		parts = append(parts, "[timed out]")
	} else if s.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("[exit code: %d]", s.ExitCode))
	} else {
		parts = append(parts, "[ok]")
	}

	return truncate(strings.Join(parts, "\n"), 4000)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
