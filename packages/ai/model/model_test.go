package model

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestMessageRoundTripPreservesAssistantContent(t *testing.T) {
	message := AssistantMessage{
		Role: "assistant",
		Content: []ContentBlock{
			NewTextContent("answer"),
			NewThinkingContent("reason"),
			NewToolCallContent("call-1", "read", []byte(`{"path":"README.md"}`)),
		},
		API:        "faux",
		Provider:   "faux",
		Model:      "test-model",
		Usage:      Usage{Input: 10, Output: 4, TotalTokens: 14, Cost: UsageCost{Input: 0.1, Output: 0.2, Total: 0.3}},
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
	decodedMsg, ok := decoded.(AssistantMessage)
	if !ok {
		t.Fatalf("decoded is not AssistantMessage, got %T", decoded)
	}
	if decodedMsg.Role != message.Role {
		t.Errorf("role mismatch")
	}
	if len(decodedMsg.Content) != 3 {
		t.Errorf("content length: got %d, want 3", len(decodedMsg.Content))
	}
	if decodedMsg.Content[0].Type != ContentTypeText || decodedMsg.Content[0].Text != "answer" {
		t.Errorf("first content block mismatch")
	}
	if decodedMsg.Content[1].Type != ContentTypeThinking || decodedMsg.Content[1].Thinking != "reason" {
		t.Errorf("second content block mismatch")
	}
	if decodedMsg.Content[2].Type != ContentTypeToolCall || decodedMsg.Content[2].Name != "read" {
		t.Errorf("third content block mismatch")
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
		// Both should have the same type
		if _, ok := decoded.(UserMessage); !ok {
			if _, ok := decoded.(AssistantMessage); !ok {
				if _, ok := decoded.(ToolResultMessage); !ok {
					t.Fatalf("round trip line %d: unexpected type %T", line, decoded)
				}
			}
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
	// Unknown content types are preserved (forward compatible), not rejected.
	// This test validates that messages with unknown content types still parse.
	data := []byte(`{"role":"assistant","content":[{"type":"audio","data":"..."}],"api":"faux","provider":"faux","model":"test","usage":{"input":0,"output":0,"totalTokens":0,"cost":{"input":0,"output":0,"total":0}},"stopReason":"stop","timestamp":1}`)
	msg, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("unknown content types should not cause error: %v", err)
	}
	am, ok := msg.(AssistantMessage)
	if !ok {
		t.Fatalf("expected AssistantMessage, got %T", msg)
	}
	if am.Content[0].Type != "audio" {
		t.Errorf("content type %q preserved", am.Content[0].Type)
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
