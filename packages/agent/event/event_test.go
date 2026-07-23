package event

import (
	"reflect"
	"testing"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

func TestToolExecutionEndCarriesToolCorrelation(t *testing.T) {
	event := ToolExecutionEnd{
		Type:       "tool_execution_end",
		ToolCallID: "call-1",
		ToolName:   "read",
		Result:     map[string]any{"text": "content"},
		IsError:    false,
	}

	if event.Type != "tool_execution_end" || event.ToolCallID != "call-1" || event.ToolName != "read" {
		t.Fatalf("unexpected tool event: %#v", event)
	}
}

func TestAgentEndCarriesTranscript(t *testing.T) {
	message := model.UserMessage{Role: "user", Content: []model.ContentBlock{model.TextContent{Type: "text", Text: "hello"}}, Timestamp: 1}
	event := AgentEnd{Type: "agent_end", Messages: []model.Message{message}}

	if len(event.Messages) != 1 || !reflect.DeepEqual(event.Messages[0], message) {
		t.Fatalf("agent transcript = %#v, want %#v", event.Messages, message)
	}
}
