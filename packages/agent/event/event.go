// Package event defines the Agent lifecycle events emitted during an agent run.
//
// Events are consumed by UI components (TUI, print mode, RPC) to render
// streaming progress, tool execution status, and session state. The event
// stream is the primary contract between the Agent loop and all consumers.
package event

import (
	"encoding/json"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// Agent event type constants.
const (
	TypeAgentStart          = "agent_start"
	TypeAgentEnd            = "agent_end"
	TypeTurnStart           = "turn_start"
	TypeTurnEnd             = "turn_end"
	TypeMessageStart        = "message_start"
	TypeMessageUpdate       = "message_update"
	TypeMessageEnd          = "message_end"
	TypeToolExecutionStart  = "tool_execution_start"
	TypeToolExecutionUpdate = "tool_execution_update"
	TypeToolExecutionEnd    = "tool_execution_end"
)

// AgentEvent represents a single event emitted during an agent run.
type AgentEvent struct {
	Type string `json:"type"`

	// agent_end
	Messages []Message `json:"messages,omitempty"`

	// turn_end
	Message     *Message                  `json:"message,omitempty"`
	ToolResults []model.ToolResultMessage `json:"toolResults,omitempty"`

	// message_start / message_update / message_end
	Payload *Message `json:"payload,omitempty"`

	// message_update
	AssistantMessageEvent *model.StreamEvent `json:"assistantMessageEvent,omitempty"`

	// tool_execution_*
	ToolCallID string          `json:"toolCallId,omitempty"`
	ToolName   string          `json:"toolName,omitempty"`
	Args       json.RawMessage `json:"args,omitempty"`

	// tool_execution_update / tool_execution_end
	PartialResult json.RawMessage `json:"partialResult,omitempty"`
	Result        json.RawMessage `json:"result,omitempty"`
	IsError       bool            `json:"isError,omitempty"`
}

// Message is a union type that can hold any of the three message types.
type Message struct {
	User       *model.UserMessage
	Assistant  *model.AssistantMessage
	ToolResult *model.ToolResultMessage
}

// Role returns the message role.
func (m *Message) Role() string {
	switch {
	case m.User != nil:
		return "user"
	case m.Assistant != nil:
		return "assistant"
	case m.ToolResult != nil:
		return "toolResult"
	default:
		return ""
	}
}

// MarshalJSON implements json.Marshaler on the value receiver.
func (m Message) MarshalJSON() ([]byte, error) {
	switch {
	case m.User != nil:
		return json.Marshal(m.User)
	case m.Assistant != nil:
		return json.Marshal(m.Assistant)
	case m.ToolResult != nil:
		return json.Marshal(m.ToolResult)
	default:
		return json.Marshal(nil)
	}
}

// UnmarshalJSON implements json.Unmarshaler by reading the role field first.
func (m *Message) UnmarshalJSON(data []byte) error {
	var role struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(data, &role); err != nil {
		return err
	}
	switch role.Role {
	case "user":
		m.User = &model.UserMessage{}
		return json.Unmarshal(data, m.User)
	case "assistant":
		m.Assistant = &model.AssistantMessage{}
		return json.Unmarshal(data, m.Assistant)
	case "toolResult":
		m.ToolResult = &model.ToolResultMessage{}
		return json.Unmarshal(data, m.ToolResult)
	default:
		return model.ErrInvalidRole
	}
}

// NewUserMessage creates a Message wrapping a UserMessage.
func NewUserMessage(msg *model.UserMessage) Message {
	return Message{User: msg}
}

// NewAssistantMessage creates a Message wrapping an AssistantMessage.
func NewAssistantMessage(msg *model.AssistantMessage) Message {
	return Message{Assistant: msg}
}

// NewToolResultMessage creates a Message wrapping a ToolResultMessage.
func NewToolResultMessage(msg *model.ToolResultMessage) Message {
	return Message{ToolResult: msg}
}
