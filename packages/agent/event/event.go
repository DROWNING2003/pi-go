package event

import "github.com/DROWNING2003/pi-go/packages/ai/model"

type Event interface{ agentEvent() }

type AgentStart struct {
	Type string `json:"type"`
}

func (AgentStart) agentEvent() {}

type AgentEnd struct {
	Type     string          `json:"type"`
	Messages []model.Message `json:"messages"`
}

func (AgentEnd) agentEvent() {}

type TurnStart struct {
	Type string `json:"type"`
}

func (TurnStart) agentEvent() {}

type TurnEnd struct {
	Type        string                    `json:"type"`
	Message     model.Message             `json:"message"`
	ToolResults []model.ToolResultMessage `json:"toolResults"`
}

func (TurnEnd) agentEvent() {}

type MessageStart struct {
	Type    string        `json:"type"`
	Message model.Message `json:"message"`
}

func (MessageStart) agentEvent() {}

type MessageUpdate struct {
	Type                  string            `json:"type"`
	Message               model.Message     `json:"message"`
	AssistantMessageEvent model.StreamEvent `json:"assistantMessageEvent"`
}

func (MessageUpdate) agentEvent() {}

type MessageEnd struct {
	Type    string        `json:"type"`
	Message model.Message `json:"message"`
}

func (MessageEnd) agentEvent() {}

type ToolExecutionStart struct {
	Type       string `json:"type"`
	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	Args       any    `json:"args"`
}

func (ToolExecutionStart) agentEvent() {}

type ToolExecutionUpdate struct {
	Type          string `json:"type"`
	ToolCallID    string `json:"toolCallId"`
	ToolName      string `json:"toolName"`
	Args          any    `json:"args"`
	PartialResult any    `json:"partialResult"`
}

func (ToolExecutionUpdate) agentEvent() {}

type ToolExecutionEnd struct {
	Type       string `json:"type"`
	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	Result     any    `json:"result"`
	IsError    bool   `json:"isError"`
}

func (ToolExecutionEnd) agentEvent() {}
