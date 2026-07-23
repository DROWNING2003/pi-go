// Package guard provides destructive command detection for the bash tool.
// Before executing a shell command, it checks against dangerous patterns and
// blocks potentially destructive operations.
package guard

import (
	"regexp"
	"strings"
	"sync"
)

// Rule defines a dangerous command pattern to block.
type Rule struct {
	Pattern     string // regex pattern
	Description string // human-readable reason
	Tip         string // safer alternative
}

// Guard checks shell commands against destructive patterns.
type Guard struct {
	mu    sync.RWMutex
	rules []compiledRule
}

type compiledRule struct {
	re   *regexp.Regexp
	desc string
	tip  string
}

// DefaultRules returns the built-in destructive command rules.
func DefaultRules() []Rule {
	return []Rule{
		// Git destructive
		{Pattern: `git\s+reset\s+--hard`, Description: "git reset --hard destroys uncommitted changes", Tip: "Use 'git stash' first to save your changes"},
		{Pattern: `git\s+push\s+.*--force`, Description: "git push --force overwrites remote history", Tip: "Use 'git push --force-with-lease' instead"},
		{Pattern: `git\s+clean\s+-.*[fdx]`, Description: "git clean removes untracked files", Tip: "Run 'git clean --dry-run' first to see what will be deleted"},
		{Pattern: `git\s+branch\s+-D`, Description: "git branch -D force-deletes a branch", Tip: "Use 'git branch -d' for safe deletion"},

		// Filesystem destructive
		{Pattern: `rm\s+-rf?\s*(/|~|\.\.?/|\.\s)`, Description: "rm -rf removes files recursively and forcefully", Tip: "Remove with explicit paths and review before running"},
		{Pattern: `rm\s+-rf?\s+/`, Description: "rm -rf / destroys the entire filesystem", Tip: "Never run this command"},
		{Pattern: `rm\s+-rf?\s+~`, Description: "rm -rf ~ destroys your home directory", Tip: "Never run this command"},
		{Pattern: `>\s*/dev/sd[a-z]`, Description: "Writing directly to disk devices can corrupt data", Tip: "Use proper disk utilities with safety checks"},
		{Pattern: `dd\s+if=.*of=/dev/`, Description: "dd to disk devices can overwrite partitions", Tip: "Double-check the output device before running"},
		{Pattern: `mkfs\.`, Description: "mkfs formats a filesystem, destroying all data", Tip: "Ensure you have backups before formatting"},
		{Pattern: `chmod\s+-R\s+777`, Description: "chmod -R 777 makes everything world-writable", Tip: "Use more restrictive permissions"},
		{Pattern: `chown\s+-R\s+.*\s+/`, Description: "chown -R on root changes ownership of system files", Tip: "Be specific about which paths to chown"},

		// System destructive
		{Pattern: `shutdown\s`, Description: "shutdown powers off the system", Tip: "Use with caution - this affects the entire machine"},
		{Pattern: `reboot\s`, Description: "reboot restarts the system", Tip: "Use with caution - this affects the entire machine"},
		{Pattern: `:\(\)\s*\{\s*:\|:&\s*\}\s*;\s*:`, Description: "Fork bomb detected", Tip: "Do not run fork bombs"},

		// Database destructive
		{Pattern: `\bDROP\s+(TABLE|DATABASE|SCHEMA)\b`, Description: "DROP destroys database objects", Tip: "Create a backup before dropping"},
		{Pattern: `\bTRUNCATE\s+(TABLE\s+)?`, Description: "TRUNCATE removes all rows from a table", Tip: "Export data before truncating"},

		// Docker destructive
		{Pattern: `docker\s+system\s+prune`, Description: "docker system prune removes unused data", Tip: "Review what will be removed with 'docker system df'"},
		{Pattern: `docker\s+rm\s+-f\s+.*\$\(`, Description: "docker rm -f forced removal of containers", Tip: "Stop containers gracefully first"},

		// Kubernetes destructive
		{Pattern: `kubectl\s+delete\s+(namespace|ns)\b`, Description: "kubectl delete namespace removes all resources in it", Tip: "Verify you're targeting the correct namespace"},
		{Pattern: `kubectl\s+delete\s+.*--all`, Description: "kubectl delete --all removes all resources of a type", Tip: "Use label selectors to limit scope"},

		// AWS destructive
		{Pattern: `aws\s+ec2\s+terminate-instances`, Description: "Terminates EC2 instances permanently", Tip: "Stop instances instead of terminating"},
		{Pattern: `aws\s+s3\s+rb\s+s3://`, Description: "s3 rb removes an S3 bucket and all contents", Tip: "Empty the bucket manually first to review contents"},
	}
}

// New creates a Guard with the given rules.
func New(rules []Rule) *Guard {
	g := &Guard{}
	g.LoadRules(rules)
	return g
}

// LoadRules compiles and sets the guard rules.
func (g *Guard) LoadRules(rules []Rule) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rules = make([]compiledRule, len(rules))
	for i, r := range rules {
		g.rules[i] = compiledRule{
			re:   regexp.MustCompile("(?i)" + r.Pattern),
			desc: r.Description,
			tip:  r.Tip,
		}
	}
}

// Result describes a guard check outcome.
type Result struct {
	Blocked bool
	Reason  string
	Tip     string
	Command string
	Matched string
}

// Check tests a command against all rules. Returns nil if safe.
func (g *Guard) Check(command string) *Result {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Normalize whitespace for matching
	normalized := strings.Join(strings.Fields(command), " ")

	for _, r := range g.rules {
		if r.re.MatchString(normalized) {
			// Don't block if it looks like we're matching a literal string in data
			if isDataContext(command, r.re) {
				continue
			}
			return &Result{
				Blocked: true,
				Reason:  r.desc,
				Tip:     r.tip,
				Command: command,
				Matched: r.re.String(),
			}
		}
	}
	return nil
}

// isDataContext checks if the matching text appears to be in a data context
// (e.g., grep, echo, cat of a file) rather than an execution context.
func isDataContext(command string, re *regexp.Regexp) bool {
	// If the command starts with echo, cat, grep, or similar read-only commands,
	// the dangerous pattern is likely data, not execution
	lower := strings.ToLower(strings.TrimSpace(command))
	dataPrefixes := []string{"echo ", "cat ", "grep ", "less ", "head ", "tail ", "find ", "ls ", "printf "}
	for _, p := range dataPrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}
