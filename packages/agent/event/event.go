// Package event defines the Agent lifecycle events emitted during an agent run.
package event

import (
	"encoding/json"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

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

type AgentEvent struct {
	Type string `json:"type"`

	Messages    []Message                 `json:"messages,omitempty"`
	Message     *Message                  `json:"message,omitempty"`
	ToolResults []model.ToolResultMessage `json:"toolResults,omitempty"`
	Payload     *Message                  `json:"payload,omitempty"`

	AssistantMessageEvent *model.StreamEvent `json:"assistantMessageEvent,omitempty"`

	ToolCallID    string          `json:"toolCallId,omitempty"`
	ToolName      string          `json:"toolName,omitempty"`
	Args          json.RawMessage `json:"args,omitempty"`
	PartialResult json.RawMessage `json:"partialResult,omitempty"`
	Result        json.RawMessage `json:"result,omitempty"`
	IsError       bool            `json:"isError,omitempty"`
}

type Message struct {
	User       *model.UserMessage
	Assistant  *model.AssistantMessage
	ToolResult *model.ToolResultMessage
}

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

func NewUserMessage(msg *model.UserMessage) Message             { return Message{User: msg} }
func NewAssistantMessage(msg *model.AssistantMessage) Message   { return Message{Assistant: msg} }
func NewToolResultMessage(msg *model.ToolResultMessage) Message { return Message{ToolResult: msg} }
