// Package util provides agent harness utilities matching TS harness/utils/
package util

import (
	"fmt"
	"strings"
)

// Truncate truncates a string to maxLen characters, adding indicator if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}

// TruncateBytes truncates to bytes limit, roughly.
func TruncateBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "\n... (truncated)"
}

// FormatShellOutput formats a shell command's output for display.
func FormatShellOutput(command, stdout, stderr string, exitCode int, cancelled, timedOut bool) string {
	var parts []string

	if cancelled {
		return fmt.Sprintf("Command cancelled: %s", command)
	}
	if timedOut {
		return fmt.Sprintf("Command timed out: %s", command)
	}

	parts = append(parts, fmt.Sprintf("Command: %s", command))

	if stdout != "" {
		parts = append(parts, stdout)
	} else if stderr == "" {
		parts = append(parts, "(no output)")
	}

	if stderr != "" {
		parts = append(parts, "[stderr]\n"+stderr)
	}

	if exitCode != 0 {
		parts = append(parts, fmt.Sprintf("[exit code: %d]", exitCode))
	}

	return Truncate(strings.Join(parts, "\n"), 4000)
}

// NormalizeToolPath normalizes a path from a tool argument.
func NormalizeToolPath(path string) string {
	// Normalize unicode spaces
	replacer := strings.NewReplacer(
		"\u00A0", " ", "\u2000", " ", "\u2001", " ",
		"\u2002", " ", "\u2003", " ", "\u2004", " ",
		"\u2005", " ", "\u2006", " ", "\u2007", " ",
		"\u2008", " ", "\u2009", " ", "\u200A", " ",
		"\u202F", " ", "\u205F", " ", "\u3000", " ",
	)
	normalized := replacer.Replace(path)
	if strings.HasPrefix(normalized, "@") {
		normalized = normalized[1:]
	}
	return normalized
}

// ResolveReadToolPath attempts multiple path variants for the read tool.
func ResolveReadToolPath(resolved string, exists func(string) bool) string {
	variants := []string{
		resolved,
		strings.Replace(resolved, " AM.", "\u202FAM.", -1),
		strings.Replace(resolved, " PM.", "\u202FPM.", -1),
	}
	for _, v := range variants {
		if exists(v) {
			return v
		}
	}
	return resolved
}
