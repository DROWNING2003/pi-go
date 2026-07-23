package msg

import (
	"encoding/json"
	"testing"
)

func TestBranchSummaryMessage(t *testing.T) {
	msg := CreateBranchSummaryMessage("merged feature-x", "branch-123")
	if msg.Role != "branchSummary" {
		t.Errorf("role: %q", msg.Role)
	}
	if msg.Summary != "merged feature-x" {
		t.Errorf("summary: %q", msg.Summary)
	}

	um := BranchSummaryToUser(msg)
	if um.Role != "user" {
		t.Errorf("user role: %q", um.Role)
	}
}

func TestCompactionSummaryMessage(t *testing.T) {
	msg := CreateCompactionSummaryMessage("conversation compacted", 5000)
	if msg.Role != "compactionSummary" {
		t.Errorf("role: %q", msg.Role)
	}
	if msg.TokensBefore != 5000 {
		t.Errorf("tokens: %d", msg.TokensBefore)
	}

	um := CompactionSummaryToUser(msg)
	if um.Role != "user" {
		t.Error("should convert to user message")
	}
}

func TestBashExecutionMessage(t *testing.T) {
	msg := BashExecutionMessage{
		Role: "bashExecution", Command: "ls -la",
		Output: "file1\nfile2", ExitCode: 0,
	}
	text := BashExecutionToText(msg)
	if text == "" {
		t.Error("empty text")
	}
}

func TestConvertToLLM(t *testing.T) {
	compactionJSON, _ := json.Marshal(CreateCompactionSummaryMessage("compact", 100))
	branchJSON, _ := json.Marshal(CreateBranchSummaryMessage("branch", "id-1"))
	userJSON := json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)

	messages := []json.RawMessage{compactionJSON, branchJSON, userJSON}
	result := ConvertToLLM(messages)

	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
}
