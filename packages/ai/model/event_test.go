package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestStreamEventRoundTripPreservesToolCallEnd(t *testing.T) {
	event := ToolCallEndEvent{
		Type:         "toolcall_end",
		ContentIndex: 2,
		ToolCall: ToolCall{
			Type:      "toolCall",
			ID:        "call-1",
			Name:      "read",
			Arguments: map[string]any{"path": "go.mod"},
		},
		Partial: AssistantMessage{Role: "assistant", Content: []ContentBlock{ToolCall{Type: "toolCall", ID: "call-1", Name: "read", Arguments: map[string]any{"path": "go.mod"}}}, API: "faux", Provider: "faux", Model: "test", StopReason: StopReasonToolUse, Timestamp: 2},
	}

	encoded, err := EncodeStreamEvent(event)
	if err != nil {
		t.Fatalf("EncodeStreamEvent() error = %v", err)
	}
	decoded, err := DecodeStreamEvent(encoded)
	if err != nil {
		t.Fatalf("DecodeStreamEvent() error = %v", err)
	}
	if !reflect.DeepEqual(event, decoded) {
		t.Fatalf("decoded event differs:\nwant %#v\n got %#v", event, decoded)
	}
}

func TestDecodeStreamEventRejectsUnknownType(t *testing.T) {
	_, err := DecodeStreamEvent([]byte(`{"type":"audio_delta","delta":"..."}`))
	if err == nil {
		t.Fatal("DecodeStreamEvent() error = nil, want unknown event type error")
	}
	if !strings.Contains(err.Error(), "unknown stream event type") {
		t.Fatalf("DecodeStreamEvent() error = %q, want unknown stream event type", err)
	}
}
