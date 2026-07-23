package model

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestMessageRoundTripPreservesAssistantContent(t *testing.T) {
	message := AssistantMessage{
		Role:       "assistant",
		Content:    []ContentBlock{TextContent{Type: "text", Text: "answer"}, ThinkingContent{Type: "thinking", Thinking: "reason"}, ToolCall{Type: "toolCall", ID: "call-1", Name: "read", Arguments: map[string]any{"path": "README.md"}}},
		API:        "faux",
		Provider:   "faux",
		Model:      "test-model",
		Usage:      Usage{Input: 10, Output: 4, TotalTokens: 14, Cost: Cost{Input: 0.1, Output: 0.2, Total: 0.3}},
		StopReason: "toolUse",
		Timestamp:  1234,
	}

	encoded, err := EncodeMessage(message)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	if !reflect.DeepEqual(message, decoded) {
		t.Fatalf("decoded message differs:\nwant %#v\n got %#v", message, decoded)
	}
}

func TestMessageFixturesRoundTrip(t *testing.T) {
	file, err := os.Open("../../../testdata/model/messages.jsonl")
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		message, err := DecodeMessage(scanner.Bytes())
		if err != nil {
			t.Fatalf("DecodeMessage(line %d) error = %v", line, err)
		}
		encoded, err := EncodeMessage(message)
		if err != nil {
			t.Fatalf("EncodeMessage(line %d) error = %v", line, err)
		}
		decoded, err := DecodeMessage(encoded)
		if err != nil {
			t.Fatalf("DecodeMessage(round trip line %d) error = %v", line, err)
		}
		if !reflect.DeepEqual(message, decoded) {
			t.Fatalf("fixture line %d changed after round trip:\nwant %#v\n got %#v", line, message, decoded)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan fixture: %v", err)
	}
	if line != 3 {
		t.Fatalf("fixture lines = %d, want 3", line)
	}
}

func TestDecodeMessageRejectsUnknownContentType(t *testing.T) {
	_, err := DecodeMessage([]byte(`{"role":"assistant","content":[{"type":"audio","data":"..."}],"api":"faux","provider":"faux","model":"test","usage":{"input":0,"output":0,"totalTokens":0,"cost":{"input":0,"output":0,"total":0}},"stopReason":"stop","timestamp":1}`))
	if err == nil {
		t.Fatal("DecodeMessage() error = nil, want unknown content type error")
	}
	if !strings.Contains(err.Error(), "unknown content type") {
		t.Fatalf("DecodeMessage() error = %q, want unknown content type", err)
	}
}

func TestDecodeMessageRejectsUnsupportedSchemaVersion(t *testing.T) {
	payload := map[string]any{
		"schemaVersion": 2,
		"message": map[string]any{
			"role":      "user",
			"content":   []any{map[string]any{"type": "text", "text": "hello"}},
			"timestamp": 1,
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	_, err = DecodeMessage(encoded)
	if err == nil {
		t.Fatal("DecodeMessage() error = nil, want unsupported schema version error")
	}
	if !strings.Contains(err.Error(), "unsupported schema version") {
		t.Fatalf("DecodeMessage() error = %q, want unsupported schema version", err)
	}
}
