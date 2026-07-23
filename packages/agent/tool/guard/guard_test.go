package guard

import (
	"testing"
)

func TestAllRules_BlocksDangerous(t *testing.T) {
	g := New(AllRules())

	tests := []struct {
		command string
		blocked bool
		name    string
	}{
		// Git - should block
		{"git reset --hard HEAD~5", true, "reset-hard"},
		{"git push --force origin main", true, "push-force"},
		{"git clean -fd", true, "clean-force"},
		{"git branch -D feature", true, "branch-delete"},
		{"git stash clear", true, "stash-clear"},
		{"git checkout -- file.txt", true, "checkout-discard"},
		{"git restore file.txt", true, "restore-worktree"},

		// Git - safe patterns should pass
		{"git reset --hard HEAD", false, "reset-to-head"},
		{"git clean -n", false, "clean-dry-run"},
		{"git push --force-with-lease origin main", false, "push-with-lease"},
		{"git restore --staged file.txt", false, "restore-staged"},

		// Filesystem
		{"rm -rf /", true, "rm-root"},
		{"shred secret.txt", true, "shred"},
		{"mkfs.ext4 /dev/sda", true, "mkfs"},

		// Database
		{"DROP DATABASE production", true, "drop-db"},
		{"DROP TABLE users", true, "drop-table"},
		{"TRUNCATE TABLE logs", true, "truncate"},

		// Docker
		{"docker system prune -f", true, "system-prune"},
		{"docker rm -f container", true, "rm-force"},

		// K8s
		{"kubectl delete namespace prod", true, "delete-namespace"},
		{"kubectl delete pods --all", true, "delete-all"},

		// Cloud
		{"aws ec2 terminate-instances --instance-ids i-123", true, "terminate"},
		{"aws s3 rb s3://my-bucket", true, "s3-rb"},
		{"gcloud projects delete my-project", true, "gcloud-delete"},
		{"az group delete -n mygroup", true, "az-delete"},

		// Terraform
		{"terraform destroy", true, "tf-destroy"},
		{"terraform destroy -target=module.app", false, "tf-destroy-targeted"},

		// NPM
		{"npm unpublish my-package", true, "npm-unpublish"},

		// Safe commands
		{"echo hello", false, "echo"},
		{"ls -la", false, "ls"},
		{"git status", false, "status"},
		{"grep 'DROP TABLE' schema.sql", false, "grep-data"},
	}

	for _, tt := range tests {
		result := g.Check(tt.command)
		blocked := result != nil
		if blocked != tt.blocked {
			if tt.blocked {
				t.Errorf("BLOCK expected but allowed: %s (%s)", tt.command, tt.name)
			} else {
				t.Errorf("ALLOW expected but blocked: %s (%s) - reason: %s", tt.command, tt.name, result.Reason)
			}
		}
	}
}

func TestAllRules_DataContextSafe(t *testing.T) {
	g := New(AllRules())
	safe := []string{
		"echo 'DROP TABLE users'",
		"grep 'rm -rf' README.md",
		"cat 'git reset --hard' instructions.txt",
	}
	for _, cmd := range safe {
		if result := g.Check(cmd); result != nil {
			t.Errorf("data context blocked: %s - %s", cmd, result.Reason)
		}
	}
}

func TestSeverity(t *testing.T) {
	g := New(AllRules())
	r := g.Check("rm -rf /")
	if r == nil || r.Severity != SevCritical {
		t.Errorf("rm -rf / should be critical, got %v", r)
	}
	r = g.Check("git stash drop")
	if r == nil || r.Severity != SevHigh {
		t.Errorf("stash drop should be high, got %v", r)
	}
}
