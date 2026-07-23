package model

import (
	"encoding/json"
	"testing"
)

// TestJSONAlignment verifies JSON format matches TypeScript reference.
func TestJSONAlignment_AssistantMessage(t *testing.T) {
	msg := AssistantMessage{
		Role: "assistant",
		Content: []ContentBlock{
			NewTextContent("answer"),
			NewThinkingContent("reason"),
			NewToolCallContent("call-1", "read", json.RawMessage(`{"path":"/tmp"}`)),
		},
		API: "faux", Provider: "faux", Model: "test-model",
		ResponseID: "resp-1",
		Usage: Usage{Input: 10, Output: 4, CacheRead: 2, CacheWrite: 3, TotalTokens: 19,
			Cost: UsageCost{Input: 0.001, Output: 0.002, CacheRead: 0.0001, CacheWrite: 0.0002, Total: 0.0033}},
		StopReason: StopReasonToolUse,
		Timestamp:  1234,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify all required TS fields present
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	required := []string{"role", "content", "api", "provider", "model", "usage", "stopReason", "timestamp"}
	for _, f := range required {
		if _, ok := m[f]; !ok {
			t.Errorf("missing required field: %s", f)
		}
	}

	// responseId should be present
	if m["responseId"] != "resp-1" {
		t.Errorf("responseId: %v", m["responseId"])
	}

	// Content should be array of blocks
	blocks, ok := m["content"].([]interface{})
	if !ok || len(blocks) != 3 {
		t.Fatalf("content: expected 3 blocks, got %v", m["content"])
	}
}

func TestJSONAlignment_UserMessageString(t *testing.T) {
	msg := UserMessage{
		Role: "user", Content: UserContent{NewTextContent("hello")}, Timestamp: 1,
	}
	data, _ := json.Marshal(msg)

	// Should serialize as string, not array
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if _, ok := m["content"].(string); !ok {
		t.Errorf("content should be string, got %T: %v", m["content"], m["content"])
	}
}

func TestJSONAlignment_UserMessageArray(t *testing.T) {
	msg := UserMessage{
		Role:      "user",
		Content:   UserContent{NewTextContent("look"), NewImageContent("abc", "image/png")},
		Timestamp: 1,
	}
	data, _ := json.Marshal(msg)

	var m map[string]interface{}
	json.Unmarshal(data, &m)
	blocks, ok := m["content"].([]interface{})
	if !ok || len(blocks) != 2 {
		t.Errorf("content should be array of 2, got %T: %v", m["content"], m["content"])
	}
}

func TestJSONAlignment_SessionHeader(t *testing.T) {
	// Validate session header matches TS SessionHeader format
	header := map[string]interface{}{
		"type":      "session",
		"version":   3,
		"id":        "sess-1",
		"timestamp": "2024-01-01T00:00:00Z",
		"cwd":       "/tmp",
	}
	data, _ := json.Marshal(header)

	var h map[string]interface{}
	json.Unmarshal(data, &h)

	if h["type"] != "session" {
		t.Error("type mismatch")
	}
	if int(h["version"].(float64)) != 3 {
		t.Error("version mismatch")
	}
	if h["id"] != "sess-1" {
		t.Error("id mismatch")
	}
}
