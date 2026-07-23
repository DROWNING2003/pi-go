package guard

import (
	"regexp"
	"strings"
	"sync"
)

// Severity levels matching dcg.
type Severity string

const (
	SevCritical Severity = "critical" // destructive, irreversible
	SevHigh     Severity = "high"     // dangerous, hard to recover
	SevMedium   Severity = "medium"   // potentially problematic
)

// Rule defines a pattern to check against commands.
type Rule struct {
	Name         string   // unique identifier
	Pattern      string   // regex to match
	Description  string   // human-readable
	Tip          string   // safer alternative
	Severity     Severity // critical/high/medium
	SafePatterns []string // patterns that make this command safe (whitelist)
}

// Guard checks shell commands against destructive patterns.
type Guard struct {
	mu    sync.RWMutex
	rules []compiledRule
}

type compiledRule struct {
	rule         Rule
	re           *regexp.Regexp
	safePatterns []*regexp.Regexp
}

// Result describes a guard check outcome.
type Result struct {
	Blocked  bool
	Reason   string
	Tip      string
	Command  string
	Severity Severity
	RuleName string
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
		cr := compiledRule{rule: r, re: regexp.MustCompile("(?i)" + r.Pattern)}
		for _, sp := range r.SafePatterns {
			cr.safePatterns = append(cr.safePatterns, regexp.MustCompile(sp))
		}
		g.rules[i] = cr
	}
}

// Check tests a command against all rules. Returns nil if safe.
func (g *Guard) Check(command string) *Result {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if isDataContext(command) {
		return nil
	}

	normalized := strings.Join(strings.Fields(command), " ")

	for _, cr := range g.rules {
		if !cr.re.MatchString(normalized) {
			continue
		}
		// Check safe patterns - if any match, the command is safe
		for _, sp := range cr.safePatterns {
			if sp.MatchString(normalized) {
				return nil
			}
		}
		return &Result{
			Blocked:  true,
			Reason:   cr.rule.Description,
			Tip:      cr.rule.Tip,
			Command:  command,
			Severity: cr.rule.Severity,
			RuleName: cr.rule.Name,
		}
	}
	return nil
}

func isDataContext(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	prefixes := []string{"echo ", "cat ", "grep ", "less ", "head ", "tail ", "find ", "ls ", "printf ", "awk ", "sed "}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) || strings.Contains(lower, "| "+p) {
			return true
		}
	}
	return false
}

// AllRules returns the complete rule set inspired by destructive_command_guard packs.
func AllRules() []Rule {
	rules := []Rule{
		// ===== core.git =====
		{
			Name: "git-reset-hard", Pattern: `git\s+reset\s+--hard`,
			Description:  "git reset --hard destroys uncommitted changes. Use 'git stash' first.",
			Tip:          "Run 'git stash' first to save your changes, or use 'git diff' to review what will be lost.",
			Severity:     SevCritical,
			SafePatterns: []string{`git\s+reset\s+--hard\s+HEAD\s*$`}, // reset to HEAD is safe
		},
		{
			Name: "git-reset-merge", Pattern: `git\s+reset\s+--merge`,
			Description: "git reset --merge can lose uncommitted changes.",
			Tip:         "Review changes with 'git diff' before resetting.",
			Severity:    SevHigh,
		},
		{
			Name: "git-push-force", Pattern: `git\s+push\s+.*(--force|-f)\b`,
			Description:  "Force push can destroy remote history.",
			Tip:          "Use 'git push --force-with-lease' to safely force-push.",
			Severity:     SevCritical,
			SafePatterns: []string{`git\s+push\s+.*--force-with-lease`},
		},
		{
			Name: "git-clean-force", Pattern: `git\s+clean\s+.*-[a-z]*[f][a-z]*`,
			Description:  "git clean removes untracked files permanently.",
			Tip:          "Run 'git clean -n' first to preview what will be deleted.",
			Severity:     SevCritical,
			SafePatterns: []string{`git\s+clean\s+.*-[a-z]*n[a-z]*`, `git\s+clean\s+--dry-run`},
		},
		{
			Name: "git-branch-delete-force", Pattern: `git\s+branch\s+-D`,
			Description: "git branch -D force-deletes a branch, losing unmerged commits.",
			Tip:         "Use 'git branch -d' to safely delete only fully merged branches.",
			Severity:    SevHigh,
		},
		{
			Name: "git-stash-clear", Pattern: `git\s+stash\s+clear`,
			Description: "git stash clear permanently deletes ALL stashed changes.",
			Tip:         "Use 'git stash drop' to remove individual stashes, or 'git stash list' to review.",
			Severity:    SevCritical,
		},
		{
			Name: "git-stash-drop", Pattern: `git\s+stash\s+drop`,
			Description: "git stash drop deletes a stash entry.",
			Tip:         "Review with 'git stash list' first. Dropped stashes can be recovered via 'git fsck'.",
			Severity:    SevHigh,
		},
		{
			Name: "git-checkout-discard", Pattern: `git\s+checkout\s+--\s`,
			Description: "git checkout -- discards uncommitted changes to a file.",
			Tip:         "Use 'git diff' to review changes first, or 'git stash' to save them.",
			Severity:    SevHigh,
		},
		{
			Name: "git-restore-worktree", Pattern: `git\s+restore\s+`,
			Description:  "git restore discards uncommitted working tree changes.",
			Tip:          "Use 'git stash' or review with 'git diff' first.",
			Severity:     SevHigh,
			SafePatterns: []string{`git\s+restore\s+.*(--staged|-S)`},
		},
		{
			Name: "git-rebase-abort", Pattern: `git\s+rebase\s+--abort`,
			Description: "git rebase --abort discards all rebase progress.",
			Tip:         "Confirm you want to lose all rebase progress before aborting.",
			Severity:    SevMedium,
		},

		// ===== core.filesystem =====
		{
			Name: "rm-rf-root", Pattern: `rm\s+-rf?\s+/`,
			Description: "rm -rf / destroys the entire filesystem.",
			Tip:         "Never run this command. It will destroy your system.",
			Severity:    SevCritical,
		},
		{
			Name: "rm-rf-home", Pattern: `rm\s+-rf?\s+~`,
			Description: "rm -rf ~ destroys your home directory.",
			Tip:         "Never run this command. It will destroy all your personal files.",
			Severity:    SevCritical,
		},
		{
			Name: "rm-rf-recursive", Pattern: `rm\s+-rf?\s`,
			Description: "rm -rf removes files recursively and forcefully. No recovery possible.",
			Tip:         "Run 'ls' on the target first, or use 'rm -ri' for interactive confirmation.",
			Severity:    SevHigh,
		},
		{
			Name: "dd-disk-write", Pattern: `dd\s+.*of=/dev/`,
			Description: "dd can overwrite disk partitions, destroying all data.",
			Tip:         "Double-check the output device. Use 'lsblk' to list available devices.",
			Severity:    SevCritical,
		},
		{
			Name: "mkfs", Pattern: `\bmkfs\.`,
			Description: "mkfs formats a filesystem, permanently destroying all data on it.",
			Tip:         "Verify the target device with 'lsblk' and ensure you have backups.",
			Severity:    SevCritical,
		},
		{
			Name: "chmod-777", Pattern: `chmod\s+.*777`,
			Description: "chmod 777 makes files world-writable, a security risk.",
			Tip:         "Use more restrictive permissions. Most files only need 644 or 755.",
			Severity:    SevHigh,
		},
		{
			Name: "chown-root", Pattern: `chown\s+(-R\s+)?root`,
			Description: "Changing ownership to root can break system functionality.",
			Tip:         "Only change ownership when absolutely necessary, and specify exact paths.",
			Severity:    SevHigh,
		},
		{
			Name: "shred", Pattern: `\bshred\b`,
			Description: "shred securely deletes files by overwriting them multiple times. Irreversible.",
			Tip:         "Ensure you have backups before shredding. This cannot be undone.",
			Severity:    SevCritical,
		},
		{
			Name: "write-device", Pattern: `>\s*/dev/sd[a-z]`,
			Description: "Writing directly to block devices can corrupt partitions and data.",
			Tip:         "Use proper disk utilities and verify the target device.",
			Severity:    SevCritical,
		},
		{
			Name: "mount-bind", Pattern: `mount\s+--bind`,
			Description: "Bind mounts can expose restricted directories.",
			Tip:         "Ensure you understand the security implications of bind mounts.",
			Severity:    SevHigh,
		},

		// ===== system =====
		{
			Name: "fork-bomb", Pattern: `:\(\)\s*\{\s*:\|:&\s*\}\s*;\s*:`,
			Description: "Fork bomb detected. This will crash the system.",
			Tip:         "Fork bombs consume all system resources. Do not run this.",
			Severity:    SevCritical,
		},
		{
			Name: "shutdown", Pattern: `\b(shutdown|halt|poweroff)\b`,
			Description: "Shutdown powers off the system immediately.",
			Tip:         "Use with extreme caution. Schedule with '+N' for a delay.",
			Severity:    SevCritical,
		},
		{
			Name: "reboot", Pattern: `\breboot\b`,
			Description: "Reboot restarts the machine immediately.",
			Tip:         "Ensure all work is saved before rebooting.",
			Severity:    SevHigh,
		},
		{
			Name: "killall", Pattern: `\bkillall\b`,
			Description: "killall terminates all processes matching a name.",
			Tip:         "Use 'pgrep' first to see which processes would be killed.",
			Severity:    SevHigh,
		},
		{
			Name: "kill-minus9", Pattern: `kill\s+-9`,
			Description: "kill -9 force-kills a process, preventing cleanup.",
			Tip:         "Try 'kill' (SIGTERM) first, which allows graceful shutdown.",
			Severity:    SevHigh,
		},

		// ===== database.postgresql / database.mysql =====
		{
			Name: "drop-database", Pattern: `\bDROP\s+DATABASE\b`,
			Description: "DROP DATABASE permanently deletes all data, tables, and schemas.",
			Tip:         "Create a full backup with 'pg_dump' or 'mysqldump' before dropping.",
			Severity:    SevCritical,
		},
		{
			Name: "drop-table", Pattern: `\bDROP\s+TABLE\b`,
			Description: "DROP TABLE permanently deletes the table and all its data.",
			Tip:         "Export the table data first, or rename it as a backup.",
			Severity:    SevCritical,
		},
		{
			Name: "truncate-table", Pattern: `\bTRUNCATE\s+(TABLE\s+)?`,
			Description: "TRUNCATE removes all rows from a table instantly. No rollback in most databases.",
			Tip:         "Export data first with SELECT INTO or pg_dump.",
			Severity:    SevCritical,
		},
		{
			Name: "delete-without-where", Pattern: `\bDELETE\s+FROM\s+\w+\s*;`,
			Description: "DELETE FROM without WHERE deletes all rows.",
			Tip:         "Always add a WHERE clause, or use a transaction with ROLLBACK safety net.",
			Severity:    SevCritical,
		},

		// ===== containers.docker =====
		{
			Name: "docker-system-prune", Pattern: `docker\s+system\s+prune`,
			Description: "docker system prune removes all unused containers, networks, images, and volumes.",
			Tip:         "Review with 'docker system df' first. Use specific prune commands instead.",
			Severity:    SevHigh,
		},
		{
			Name: "docker-volume-prune", Pattern: `docker\s+volume\s+prune`,
			Description: "docker volume prune deletes all unused volumes and their data.",
			Tip:         "List volumes with 'docker volume ls' and back up important data.",
			Severity:    SevHigh,
		},
		{
			Name: "docker-rm-force", Pattern: `docker\s+rm\s+-f`,
			Description: "docker rm -f force-removes a container without graceful shutdown.",
			Tip:         "Stop the container first with 'docker stop' for graceful shutdown.",
			Severity:    SevHigh,
		},
		{
			Name: "docker-rmi-force", Pattern: `docker\s+rmi\s+-f`,
			Description: "docker rmi -f force-removes images, potentially breaking dependent containers.",
			Tip:         "Check dependent containers with 'docker ps -a --filter ancestor=<image>'.",
			Severity:    SevHigh,
		},

		// ===== kubernetes.kubectl =====
		{
			Name: "kubectl-delete-namespace", Pattern: `kubectl\s+delete\s+(namespace|ns)\b`,
			Description: "Deleting a namespace removes ALL resources within it.",
			Tip:         "Backup resources first with 'kubectl get all -n <ns> -o yaml > backup.yaml'.",
			Severity:    SevCritical,
		},
		{
			Name: "kubectl-delete-all", Pattern: `kubectl\s+delete\s+.*--all`,
			Description: "kubectl delete --all removes all resources of a type.",
			Tip:         "Use label selectors to limit scope, or backup first.",
			Severity:    SevCritical,
		},
		{
			Name: "kubectl-delete-pvc", Pattern: `kubectl\s+delete\s+pvc`,
			Description: "Deleting a PVC may delete the underlying storage volume and all data.",
			Tip:         "Check reclaim policy first with 'kubectl describe pvc'.",
			Severity:    SevCritical,
		},

		// ===== cloud.aws / cloud.gcp / cloud.azure =====
		{
			Name: "aws-terminate-instances", Pattern: `aws\s+ec2\s+terminate-instances`,
			Description: "Terminates EC2 instances permanently. Cannot be recovered.",
			Tip:         "Stop instances instead, or create an AMI backup before terminating.",
			Severity:    SevCritical,
		},
		{
			Name: "aws-delete-s3-bucket", Pattern: `aws\s+s3\s+rb\s+s3://`,
			Description: "Removes an S3 bucket and ALL its contents permanently.",
			Tip:         "Empty the bucket manually first to review contents, or enable versioning.",
			Severity:    SevCritical,
		},
		{
			Name: "aws-delete-rds", Pattern: `aws\s+rds\s+delete-db`,
			Description: "Deletes an RDS database instance. Data may be unrecoverable.",
			Tip:         "Create a final snapshot with --final-db-snapshot-identifier before deleting.",
			Severity:    SevCritical,
		},
		{
			Name: "gcloud-delete-project", Pattern: `gcloud\s+projects\s+delete`,
			Description: "Deletes an entire GCP project and all resources within it.",
			Tip:         "Export all resources and data before deleting a project.",
			Severity:    SevCritical,
		},
		{
			Name: "az-delete-resource-group", Pattern: `az\s+group\s+delete`,
			Description: "Deletes an Azure resource group and ALL resources in it.",
			Tip:         "Export resource list with 'az resource list' before deleting.",
			Severity:    SevCritical,
		},

		// ===== terraform =====
		{
			Name: "terraform-destroy", Pattern: `terraform\s+destroy`,
			Description:  "terraform destroy deletes ALL resources in the state file.",
			Tip:          "Run 'terraform plan -destroy' first to preview. Consider using 'terraform apply -target' for selective deletion.",
			Severity:     SevCritical,
			SafePatterns: []string{`terraform\s+destroy\s+.*-target=`},
		},
		{
			Name: "terraform-force-unlock", Pattern: `terraform\s+force-unlock`,
			Description: "Force-unlocking a Terraform state can cause state corruption.",
			Tip:         "Ensure no other process is holding the lock before force-unlocking.",
			Severity:    SevHigh,
		},

		// ===== npm / package managers =====
		{
			Name: "npm-unpublish", Pattern: `npm\s+unpublish`,
			Description: "npm unpublish removes a package from the registry permanently.",
			Tip:         "Consider 'npm deprecate' instead, which marks the package without removing it.",
			Severity:    SevCritical,
		},
		{
			Name: "pip-uninstall-all", Pattern: `pip\s+(freeze\s+.*\|\s*xargs\s+pip\s+uninstall|uninstall\s+-r\s+.*-y)`,
			Description: "Bulk pip uninstall removes all packages, potentially breaking the Python environment.",
			Tip:         "Use a virtual environment and review the package list before bulk removal.",
			Severity:    SevHigh,
		},
	}
	return rules
}
