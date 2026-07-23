package session

import (
	"encoding/json"
	"fmt"
	"time"
)

// Entry types matching TS SessionTreeEntry union.
const (
	EntryMessage             = "message"
	EntryTree                = "tree"
	EntryLabel               = "label"
	EntryCompaction          = "compaction"
	EntryModelChange         = "model_change"
	EntryThinkingLevelChange = "thinking_level_change"
	EntryActiveToolsChange   = "active_tools_change"
	EntryBranchSummary       = "branch_summary"
	EntryCustomMessage       = "custom_message"
)

// Entry is a parsed session entry with type-based dispatch.
type Entry struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`

	// message entry
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`

	// tree entry
	Action   string `json:"action,omitempty"`
	TargetID string `json:"targetId,omitempty"`

	// label entry
	Label string `json:"label,omitempty"`

	// model_change entry
	Provider string `json:"provider,omitempty"`
	ModelID  string `json:"modelId,omitempty"`

	// thinking_level_change entry
	ThinkingLevel string `json:"thinkingLevel,omitempty"`

	// active_tools_change
	ActiveToolNames []string `json:"activeToolNames,omitempty"`

	// compaction entry
	Summary          string            `json:"summary,omitempty"`
	TokensBefore     int               `json:"tokensBefore,omitempty"`
	FirstKeptEntryID string            `json:"firstKeptEntryId,omitempty"`
	RetainedTail     []json.RawMessage `json:"retainedTail,omitempty"`

	// branch_summary entry
	BranchID string `json:"branchId,omitempty"`

	// custom_message entry
	CustomType string          `json:"customType,omitempty"`
	Display    string          `json:"display,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
}

// Stats holds derived session statistics matching TS SessionStats.
type Stats struct {
	MessageCount int
	UserCount    int
	AsstCount    int
	ToolCount    int
	TotalInput   int64
	TotalOutput  int64
	LastModel    string
	LastProvider string
}

// DeriveState derives the current agent state from a path of tree entries.
func DeriveState(entries []Entry) (thinkingLevel string, modelProvider string, modelID string, activeTools []string) {
	thinkingLevel = "off"
	for _, e := range entries {
		switch e.Type {
		case EntryThinkingLevelChange:
			if e.ThinkingLevel != "" {
				thinkingLevel = e.ThinkingLevel
			}
		case EntryModelChange:
			modelProvider = e.Provider
			modelID = e.ModelID
		case EntryMessage:
			if e.Role == "assistant" {
				// Extract model info from assistant message
				var am struct {
					Provider string `json:"provider"`
					Model    string `json:"model"`
				}
				if json.Unmarshal(e.Content, &am) == nil {
					modelProvider = am.Provider
					modelID = am.Model
				}
			}
		case EntryActiveToolsChange:
			activeTools = append([]string{}, e.ActiveToolNames...)
		}
	}
	return
}

// ComputeStats computes session statistics from entries.
func ComputeStats(entries []Entry) Stats {
	var s Stats
	for _, e := range entries {
		if e.Type != EntryMessage {
			continue
		}
		s.MessageCount++
		switch e.Role {
		case "user":
			s.UserCount++
		case "assistant":
			s.AsstCount++
			s.LastProvider = e.Provider
			s.LastModel = e.ModelID
			// Extract usage
			var am struct {
				Usage struct {
					Input  int64 `json:"input"`
					Output int64 `json:"output"`
				} `json:"usage"`
			}
			json.Unmarshal(e.Content, &am)
			s.TotalInput += am.Usage.Input
			s.TotalOutput += am.Usage.Output
		case "toolResult":
			s.ToolCount++
		}
	}
	return s
}

// CreateCompactionEntry creates a compaction marker entry.
func CreateCompactionEntry(summary string, tokensBefore int, firstKeptEntryID string, retainedTail []json.RawMessage) Entry {
	return Entry{
		Type:             EntryCompaction,
		ID:               fmt.Sprintf("compaction-%d", time.Now().UnixNano()),
		Summary:          summary,
		TokensBefore:     tokensBefore,
		FirstKeptEntryID: firstKeptEntryID,
		RetainedTail:     retainedTail,
		Timestamp:        time.Now().UnixMilli(),
	}
}

// CreateModelChangeEntry creates a model change marker.
func CreateModelChangeEntry(provider, modelID string) Entry {
	return Entry{
		Type:      EntryModelChange,
		ID:        fmt.Sprintf("model-%d", time.Now().UnixNano()),
		Provider:  provider,
		ModelID:   modelID,
		Timestamp: time.Now().UnixMilli(),
	}
}

// CreateLabelEntry creates a label marker for a tree node.
func CreateLabelEntry(targetID, label string) Entry {
	return Entry{
		Type:      EntryLabel,
		ID:        fmt.Sprintf("label-%d", time.Now().UnixNano()),
		TargetID:  targetID,
		Label:     label,
		Timestamp: time.Now().UnixMilli(),
	}
}
