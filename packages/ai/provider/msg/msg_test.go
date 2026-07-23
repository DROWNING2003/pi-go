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
