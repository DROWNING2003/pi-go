// Package sessioncwd manages session working directory.
package sessioncwd

import "os"

// CWD returns the current working directory for a session.
// Uses the process CWD by default, can be overridden.
func CWD() string {
	cwd, _ := os.Getwd()
	return cwd
}

// WithFallback returns cwd or fallback if cwd is empty.
func WithFallback(cwd, fallback string) string {
	if cwd == "" {
		return fallback
	}
	return cwd
}
