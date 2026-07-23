package event

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

func makeUserMsg() *model.UserMessage {
	return &model.UserMessage{
		Role: "user", Content: model.UserContent{model.NewTextContent("hello")},
		Timestamp: time.Now().UnixMilli(),
	}
}
func makeAssistantMsg() *model.AssistantMessage {
	return &model.AssistantMessage{
		Role: "assistant", Content: []model.ContentBlock{model.NewTextContent("hi")},
		API: "test", Provider: "test", Model: "test",
		Usage: model.Usage{TotalTokens: 10}, StopReason: "stop",
		Timestamp: time.Now().UnixMilli(),
	}
}
func makeToolResultMsg() *model.ToolResultMessage {
	return &model.ToolResultMessage{
		Role: "toolResult", ToolCallID: "tool-1", ToolName: "read",
		Content: []model.ContentBlock{model.NewTextContent("ok")},
		IsError: false, Timestamp: time.Now().UnixMilli(),
	}
}

func TestMessage_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{"user", NewUserMessage(makeUserMsg())},
		{"assistant", NewAssistantMessage(makeAssistantMsg())},
		{"toolResult", NewToolResultMessage(makeToolResultMsg())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var decoded Message
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if decoded.Role() != tt.msg.Role() {
				t.Errorf("role: got %q, want %q", decoded.Role(), tt.msg.Role())
			}
		})
	}
}

func TestMessage_InvalidRole(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"role":"unknown"}`), &msg); err == nil {
		t.Error("expected error for unknown role")
	}
}

func TestAgentEvent_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		event AgentEvent
	}{
		{"agent_start", AgentEvent{Type: TypeAgentStart}},
		{"agent_end", AgentEvent{Type: TypeAgentEnd, Messages: []Message{NewUserMessage(makeUserMsg())}}},
		{"turn_start", AgentEvent{Type: TypeTurnStart}},
		{"message_start", AgentEvent{Type: TypeMessageStart, Payload: &Message{User: makeUserMsg()}}},
		{"tool_execution_start", AgentEvent{
			Type: TypeToolExecutionStart, ToolCallID: "tool-1", ToolName: "read",
			Args: json.RawMessage(`{"path":"/tmp"}`),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var decoded AgentEvent
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if decoded.Type != tt.event.Type {
				t.Errorf("type: got %q, want %q", decoded.Type, tt.event.Type)
			}
		})
	}
}
