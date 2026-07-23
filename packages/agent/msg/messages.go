// Package msg provides message construction utilities matching TS harness/messages.ts.
package msg

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

const (
	CompactionSummaryPrefix = "The conversation history before this point was compacted into the following summary:\n\n<summary>\n"
	CompactionSummarySuffix = "\n</summary>"
	BranchSummaryPrefix     = "The following is a summary of a branch that this conversation came back from:\n\n<summary>\n"
	BranchSummarySuffix     = "</summary>"
)

// BashExecutionMessage represents a bash command execution.
type BashExecutionMessage struct {
	Role               string `json:"role"`
	Command            string `json:"command"`
	Output             string `json:"output"`
	ExitCode           int    `json:"exitCode"`
	Cancelled          bool   `json:"cancelled"`
	Truncated          bool   `json:"truncated"`
	FullOutputPath     string `json:"fullOutputPath,omitempty"`
	ExcludeFromContext bool   `json:"excludeFromContext,omitempty"`
	Timestamp          int64  `json:"timestamp"`
}

// BranchSummaryMessage represents a branch navigation summary.
type BranchSummaryMessage struct {
	Role      string `json:"role"`
	Summary   string `json:"summary"`
	FromID    string `json:"fromId"`
	Timestamp int64  `json:"timestamp"`
}

// CompactionSummaryMessage represents a context compaction summary.
type CompactionSummaryMessage struct {
	Role         string `json:"role"`
	Summary      string `json:"summary"`
	TokensBefore int    `json:"tokensBefore"`
	Timestamp    int64  `json:"timestamp"`
}

// CreateBranchSummaryMessage creates a branch summary message.
func CreateBranchSummaryMessage(summary, fromID string) BranchSummaryMessage {
	return BranchSummaryMessage{
		Role: "branchSummary", Summary: summary, FromID: fromID,
		Timestamp: time.Now().UnixMilli(),
	}
}

// CreateCompactionSummaryMessage creates a compaction summary message.
func CreateCompactionSummaryMessage(summary string, tokensBefore int) CompactionSummaryMessage {
	return CompactionSummaryMessage{
		Role: "compactionSummary", Summary: summary, TokensBefore: tokensBefore,
		Timestamp: time.Now().UnixMilli(),
	}
}

// CompactionSummaryToUser converts a compaction summary to a user message for LLM context.
func CompactionSummaryToUser(msg CompactionSummaryMessage) model.UserMessage {
	return model.UserMessage{
		Role: "user",
		Content: model.UserContent{
			model.NewTextContent(CompactionSummaryPrefix + msg.Summary + CompactionSummarySuffix),
		},
		Timestamp: msg.Timestamp,
	}
}

// BranchSummaryToUser converts a branch summary to a user message for LLM context.
func BranchSummaryToUser(msg BranchSummaryMessage) model.UserMessage {
	return model.UserMessage{
		Role: "user",
		Content: model.UserContent{
			model.NewTextContent(BranchSummaryPrefix + msg.Summary + BranchSummarySuffix),
		},
		Timestamp: msg.Timestamp,
	}
}

// BashExecutionToText formats a bash execution for display.
func BashExecutionToText(msg BashExecutionMessage) string {
	text := fmt.Sprintf("Ran `%s`\n", msg.Command)
	if msg.Output != "" {
		text += "```\n" + msg.Output + "\n```"
	} else {
		text += "(no output)"
	}
	if msg.Cancelled {
		text += "\n\n(command cancelled)"
	} else if msg.ExitCode != 0 {
		text += fmt.Sprintf("\n\nCommand exited with code %d", msg.ExitCode)
	}
	if msg.Truncated && msg.FullOutputPath != "" {
		text += fmt.Sprintf("\n\n[Output truncated. Full output: %s]", msg.FullOutputPath)
	}
	return text
}

// ConvertToLLM converts agent messages to LLM-compatible messages.
func ConvertToLLM(messages []json.RawMessage) []json.RawMessage {
	var llmMsgs []json.RawMessage
	for _, raw := range messages {
		var header struct {
			Role string `json:"role"`
		}
		if json.Unmarshal(raw, &header) != nil {
			continue
		}
		switch header.Role {
		case "bashExecution":
			var msg BashExecutionMessage
			if json.Unmarshal(raw, &msg) == nil && !msg.ExcludeFromContext {
				um := model.UserMessage{
					Role: "user", Content: model.UserContent{model.NewTextContent(BashExecutionToText(msg))},
					Timestamp: msg.Timestamp,
				}
				data, _ := json.Marshal(um)
				llmMsgs = append(llmMsgs, data)
			}
		case "branchSummary":
			var msg BranchSummaryMessage
			if json.Unmarshal(raw, &msg) == nil {
				um := BranchSummaryToUser(msg)
				data, _ := json.Marshal(um)
				llmMsgs = append(llmMsgs, data)
			}
		case "compactionSummary":
			var msg CompactionSummaryMessage
			if json.Unmarshal(raw, &msg) == nil {
				um := CompactionSummaryToUser(msg)
				data, _ := json.Marshal(um)
				llmMsgs = append(llmMsgs, data)
			}
		default:
			llmMsgs = append(llmMsgs, raw)
		}
	}
	return llmMsgs
}
