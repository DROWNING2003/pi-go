package session

import (
	"encoding/json"
	"testing"
)

func TestDeriveState(t *testing.T) {
	entries := []Entry{
		{Type: EntryThinkingLevelChange, ThinkingLevel: "high"},
		{Type: EntryModelChange, Provider: "deepseek", ModelID: "deepseek-chat"},
		{Type: EntryActiveToolsChange, ActiveToolNames: []string{"read", "write"}},
	}

	level, prov, model, tools := DeriveState(entries)
	if level != "high" {
		t.Errorf("level: %s", level)
	}
	if prov != "deepseek" || model != "deepseek-chat" {
		t.Errorf("model: %s/%s", prov, model)
	}
	if len(tools) != 2 {
		t.Errorf("tools: %v", tools)
	}
}

func TestComputeStats(t *testing.T) {
	asstJSON := json.RawMessage(`{"role":"assistant","provider":"deepseek","model":"deepseek-chat","usage":{"input":10,"output":5}}`)
	entries := []Entry{
		{Type: EntryMessage, Role: "user", Content: json.RawMessage(`"hi"`)},
		{Type: EntryMessage, Role: "assistant", Provider: "deepseek", ModelID: "deepseek-chat", Content: asstJSON},
		{Type: EntryMessage, Role: "toolResult"},
	}

	stats := ComputeStats(entries)
	if stats.MessageCount != 3 {
		t.Errorf("messages: %d", stats.MessageCount)
	}
	if stats.UserCount != 1 || stats.AsstCount != 1 || stats.ToolCount != 1 {
		t.Errorf("counts: u=%d a=%d t=%d", stats.UserCount, stats.AsstCount, stats.ToolCount)
	}
	if stats.LastProvider != "deepseek" {
		t.Errorf("last provider: %s", stats.LastProvider)
	}
}

func TestCreateCompactionEntry(t *testing.T) {
	e := CreateCompactionEntry("summary text", 1000, "entry-123", nil)
	if e.Type != EntryCompaction || e.Summary != "summary text" || e.TokensBefore != 1000 {
		t.Errorf("compaction entry: %+v", e)
	}
}

func TestCreateModelChangeEntry(t *testing.T) {
	e := CreateModelChangeEntry("openai", "gpt-4o")
	if e.Type != EntryModelChange || e.Provider != "openai" {
		t.Errorf("model change: %+v", e)
	}
}

func TestCreateLabelEntry(t *testing.T) {
	e := CreateLabelEntry("sess-1", "my session")
	if e.Type != EntryLabel || e.Label != "my session" || e.TargetID != "sess-1" {
		t.Errorf("label: %+v", e)
	}
}
