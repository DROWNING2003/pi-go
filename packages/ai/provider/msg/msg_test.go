package msg

import (
	"encoding/json"
	"testing"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

func TestTransformMessages_ImageDowngrade(t *testing.T) {
	m := &provider.ProviderModel{Input: []string{"text"}} // no image support
	userMsg := model.UserMessage{
		Role: "user",
		Content: model.UserContent{
			model.NewTextContent("look"),
			model.NewImageContent("abc", "image/png"),
		},
		Timestamp: 1,
	}
	data, _ := json.Marshal(userMsg)

	result := TransformMessages([]json.RawMessage{data}, m)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	var transformed model.UserMessage
	json.Unmarshal(result[0], &transformed)
	if len(transformed.Content) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(transformed.Content))
	}
	if transformed.Content[1].Type != model.ContentTypeText {
		t.Error("image should be replaced with text placeholder")
	}
}

func TestTransformMessages_KeepImageForVisionModel(t *testing.T) {
	m := &provider.ProviderModel{Input: []string{"text", "image"}}
	userMsg := model.UserMessage{
		Role: "user",
		Content: model.UserContent{
			model.NewTextContent("look"),
			model.NewImageContent("abc", "image/png"),
		},
		Timestamp: 1,
	}
	data, _ := json.Marshal(userMsg)

	result := TransformMessages([]json.RawMessage{data}, m)
	var transformed model.UserMessage
	json.Unmarshal(result[0], &transformed)
	if len(transformed.Content) != 2 {
		t.Errorf("vision model should keep image, got %d blocks", len(transformed.Content))
	}
}

func TestNormalizeToolCallID(t *testing.T) {
	result := NormalizeToolCallID("call_123|openai|abc")
	if result != "call_123_openai_abc" {
		t.Errorf("normalized: %q", result)
	}
	// Long IDs truncated
	long := "a" + string(make([]byte, 100))
	result = NormalizeToolCallID(long)
	if len(result) > 64 {
		t.Errorf("too long: %d", len(result))
	}
}

func TestTransformMessages_OrphanedToolCalls(t *testing.T) {
	// Tool call without tool result should get synthetic empty result inserted
	asstWithTool := model.AssistantMessage{
		Role: "assistant", API: "faux", Provider: "faux",
		Content: []model.ContentBlock{
			model.NewToolCallContent("call-1", "read", json.RawMessage(`{"path":"/tmp"}`)),
		},
		StopReason: model.StopReasonToolUse,
		Usage:      model.Usage{},
	}
	userMsg := model.UserMessage{Role: "user", Content: model.UserContent{model.NewTextContent("hi")}, Timestamp: 1}
	asstData, _ := json.Marshal(asstWithTool)
	userData, _ := json.Marshal(userMsg)

	m := &provider.ProviderModel{API: "faux", Provider: "faux", Input: []string{"text"}}

	result := TransformMessages([]json.RawMessage{userData, asstData, userData}, m)
	// Should have: user, assistant(toolCall), synthetic toolResult, user
	if len(result) < 4 {
		t.Fatalf("expected >=4 messages, got %d", len(result))
	}
	// Third message should be the synthetic toolResult
	var tr model.ToolResultMessage
	if json.Unmarshal(result[2], &tr) == nil {
		if tr.IsError != true || tr.ToolCallID != "call-1" {
			t.Errorf("synthetic tool result: %+v", tr)
		}
	} else {
		t.Error("third message should be toolResult")
	}
}

func TestTransformMessages_SkipAborted(t *testing.T) {
	// Aborted assistant messages should be skipped
	asstAborted := model.AssistantMessage{
		Role: "assistant", API: "faux", Provider: "faux",
		StopReason:   model.StopReasonAborted,
		ErrorMessage: "aborted",
		Usage:        model.Usage{},
	}
	userMsg := model.UserMessage{Role: "user", Content: model.UserContent{model.NewTextContent("hi")}, Timestamp: 1}
	asstData, _ := json.Marshal(asstAborted)
	userData, _ := json.Marshal(userMsg)

	m := &provider.ProviderModel{Input: []string{"text"}}
	result := TransformMessages([]json.RawMessage{userData, asstData, userData}, m)
	// Aborted should be removed, only user-user remains
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
}
