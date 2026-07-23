package guard

import (
	"testing"
)

func TestBlocksDangerousCommands(t *testing.T) {
	g := New(DefaultRules())

	tests := []struct {
		command string
		blocked bool
	}{
		{"rm -rf /", true},
		{"rm -rf /tmp/test", true},
		{"rm -r /tmp/test", true},
		{"git reset --hard HEAD~5", true},
		{"git push --force origin main", true},
		{"git clean -fd", true},
		{"git branch -D feature", true},
		{"mkfs.ext4 /dev/sda", true},
		{"chmod -R 777 /var/www", true},
		{"DROP TABLE users", true},
		{"docker system prune -f", true},
		{"kubectl delete namespace production", true},
		{":(){ :|:& };:", true}, // fork bomb
		{"aws ec2 terminate-instances --instance-ids i-123", true},

		// Safe commands
		{"echo hello", false},
		{"ls -la", false},
		{"go test ./...", false},
		{"git status", false},
		{"git log --oneline", false},
		{"cat README.md", false},
		{"grep 'DROP TABLE' schema.sql", false}, // data context
		{"echo 'rm -rf /'", false},              // data context
	}

	for _, tt := range tests {
		result := g.Check(tt.command)
		blocked := result != nil
		if blocked != tt.blocked {
			if tt.blocked {
				t.Errorf("should BLOCK: %s", tt.command)
			} else {
				t.Errorf("should ALLOW: %s (blocked: %v)", tt.command, result)
			}
		}
	}
}

func TestDataContextNotBlocked(t *testing.T) {
	g := New(DefaultRules())

	// These look dangerous but are in data contexts
	safe := []string{
		"echo 'DROP TABLE users'",
		"cat 'rm -rf' instructions.txt",
		"grep 'git reset --hard' file.txt",
		"echo \"git push --force origin main\"",
	}
	for _, cmd := range safe {
		if result := g.Check(cmd); result != nil {
			t.Errorf("should not block data context: %s (blocked: %v)", cmd, result.Reason)
		}
	}
}

func TestCustomRules(t *testing.T) {
	g := New([]Rule{
		{Pattern: `dangerous_command`, Description: "Custom block", Tip: "Don't do it"},
	})
	if g.Check("run dangerous_command now") == nil {
		t.Error("should block custom rule")
	}
	if g.Check("safe command") != nil {
		t.Error("should allow safe command")
	}
}
