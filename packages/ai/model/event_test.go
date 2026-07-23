package model

import (
	"encoding/json"
	"testing"
	"time"
)

func makePartial() *AssistantMessage {
	return &AssistantMessage{
		Role: "assistant", Content: []ContentBlock{},
		API: "test", Provider: "test", Model: "test",
		Usage: Usage{}, StopReason: "stop",
		Timestamp: time.Now().UnixMilli(),
	}
}

func TestStreamEvent_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		event StreamEvent
		want  string
	}{
		{"start", NewStartEvent(makePartial()), StreamEventStart},
		{"text_start", NewTextStartEvent(0, makePartial()), StreamEventTextStart},
		{"text_delta", NewTextDeltaEvent(0, "hi", makePartial()), StreamEventTextDelta},
		{"text_end", NewTextEndEvent(0, "hello", makePartial()), StreamEventTextEnd},
		{"thinking_start", NewThinkingStartEvent(0, makePartial()), StreamEventThinkingStart},
		{"thinking_delta", NewThinkingDeltaEvent(0, "...", makePartial()), StreamEventThinkingDelta},
		{"thinking_end", NewThinkingEndEvent(0, "think", makePartial()), StreamEventThinkingEnd},
		{"toolcall_start", NewToolCallStartEvent(0, makePartial()), StreamEventToolCallStart},
		{"toolcall_delta", NewToolCallDeltaEvent(0, "x", makePartial()), StreamEventToolCallDelta},
		{"done", NewDoneEvent("stop", makePartial()), StreamEventDone},
		{"error", NewErrorEvent("error", makePartial()), StreamEventError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var decoded StreamEvent
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if decoded.Type != tt.want {
				t.Errorf("type: got %q, want %q", decoded.Type, tt.want)
			}
		})
	}
}

func TestStreamEvent_DoneEvent(t *testing.T) {
	partial := makePartial()
	partial.StopReason = "toolUse"
	event := NewDoneEvent("toolUse", partial)
	data, _ := json.Marshal(event)
	var decoded StreamEvent
	json.Unmarshal(data, &decoded)
	if !decoded.IsTerminal() || decoded.Message == nil {
		t.Error("done event round-trip failed")
	}
}

func TestStreamEvent_ErrorEvent(t *testing.T) {
	partial := makePartial()
	partial.ErrorMessage = "boom"
	event := NewErrorEvent("error", partial)
	data, _ := json.Marshal(event)
	var decoded StreamEvent
	json.Unmarshal(data, &decoded)
	if !decoded.IsTerminal() || decoded.Error == nil || decoded.Error.ErrorMessage != "boom" {
		t.Error("error event round-trip failed")
	}
}
