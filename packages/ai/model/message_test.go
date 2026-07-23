package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestContentBlock_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		block ContentBlock
		json  string
	}{
		{
			name:  "text block",
			block: NewTextContent("hello world"),
			json:  `{"type":"text","text":"hello world"}`,
		},
		{
			name:  "text block with signature",
			block: TextContentBlock{Text: "hello", TextSignature: "sig1"}.ToContentBlock(),
			json:  `{"type":"text","text":"hello","textSignature":"sig1"}`,
		},
		{
			name:  "thinking block",
			block: NewThinkingContent("thinking..."),
			json:  `{"type":"thinking","thinking":"thinking..."}`,
		},
		{
			name:  "thinking block with signature and redacted",
			block: ThinkingContentBlock{Thinking: "think", ThinkingSignature: "sig2", Redacted: true}.ToContentBlock(),
			json:  `{"type":"thinking","thinking":"think","thinkingSignature":"sig2","redacted":true}`,
		},
		{
			name:  "image block",
			block: NewImageContent("base64data", "image/png"),
			json:  `{"type":"image","data":"base64data","mimeType":"image/png"}`,
		},
		{
			name:  "toolCall block",
			block: NewToolCallContent("tool-1", "read", json.RawMessage(`{"path":"/tmp"}`)),
			json:  `{"type":"toolCall","id":"tool-1","name":"read","arguments":{"path":"/tmp"}}`,
		},
		{
			name: "toolCall block with thought signature",
			block: ToolCallContentBlock{
				ID: "tool-2", Name: "write", Arguments: json.RawMessage(`{}`),
				ThoughtSignature: "thought-sig",
			}.ToContentBlock(),
			json: `{"type":"toolCall","id":"tool-2","name":"write","arguments":{},"thoughtSignature":"thought-sig"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			got, err := json.Marshal(tt.block)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			if string(got) != tt.json {
				t.Errorf("marshal mismatch:\n  got:  %s\n  want: %s", got, tt.json)
			}

			// Unmarshal
			var decoded ContentBlock
			if err := json.Unmarshal([]byte(tt.json), &decoded); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if decoded.Type != tt.block.Type {
				t.Errorf("type mismatch: got %q, want %q", decoded.Type, tt.block.Type)
			}
		})
	}
}

func TestContentBlock_UnknownType(t *testing.T) {
	data := `{"type":"unknown_field","value":42}`
	var block ContentBlock
	err := json.Unmarshal([]byte(data), &block)
	if err != nil {
		t.Fatalf("unknown types should not error: %v", err)
	}
	if block.Type != "unknown_field" {
		t.Errorf("expected type 'unknown_field', got %q", block.Type)
	}
	// Unknown fields should still round-trip
	remarshaled, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("remarshal error: %v", err)
	}
	var back ContentBlock
	if err := json.Unmarshal(remarshaled, &back); err != nil {
		t.Fatalf("second unmarshal error: %v", err)
	}
	if back.Type != "unknown_field" {
		t.Errorf("round-trip lost type")
	}
}

func TestUsage_RoundTrip(t *testing.T) {
	usage := Usage{
		Input:       100,
		Output:      50,
		CacheRead:   20,
		CacheWrite:  10,
		TotalTokens: 180,
		Cost: UsageCost{
			Input:      0.001,
			Output:     0.002,
			CacheRead:  0.0002,
			CacheWrite: 0.001,
			Total:      0.0032,
		},
	}
	got, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Usage
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Input != 100 || decoded.Output != 50 || decoded.CacheRead != 20 || decoded.CacheWrite != 10 {
		t.Errorf("token counts mismatch")
	}
	if decoded.Cost.Input != 0.001 || decoded.Cost.Output != 0.002 {
		t.Errorf("cost mismatch")
	}
}

func TestUsage_WithCacheWrite1h(t *testing.T) {
	v := 5
	usage := Usage{
		Input:        100,
		Output:       50,
		CacheRead:    20,
		CacheWrite:   10,
		CacheWrite1h: &v,
		TotalTokens:  180,
		Cost:         UsageCost{},
	}
	got, _ := json.Marshal(usage)
	var decoded Usage
	json.Unmarshal(got, &decoded)
	if decoded.CacheWrite1h == nil || *decoded.CacheWrite1h != 5 {
		t.Errorf("CacheWrite1h mismatch: %v", decoded.CacheWrite1h)
	}
}

func TestUserMessage_RoundTrip(t *testing.T) {
	now := time.Now().UnixMilli()
	msg := UserMessage{
		Role:      "user",
		Content:   UserContent{NewTextContent("hello world")},
		Timestamp: now,
	}

	got, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded UserMessage
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Role != "user" {
		t.Errorf("role mismatch: %q", decoded.Role)
	}
	if decoded.Timestamp != now {
		t.Errorf("timestamp mismatch")
	}
	if len(decoded.Content) != 1 || decoded.Content[0].Type != "text" || decoded.Content[0].Text != "hello world" {
		t.Errorf("content mismatch: %+v", decoded.Content)
	}
}

func TestUserMessage_PlainStringContent(t *testing.T) {
	// User messages can have content as a plain string or array
	jsonStr := `{"role":"user","content":"hello plain string","timestamp":1}`
	var msg UserMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if msg.Role != "user" {
		t.Errorf("role mismatch: %q", msg.Role)
	}
	if len(msg.Content) != 1 || msg.Content[0].Type != "text" || msg.Content[0].Text != "hello plain string" {
		t.Errorf("plain string content not parsed: %+v", msg.Content)
	}

	// Round-trip: single text block should marshal back to string
	remarshaled, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(remarshaled) != jsonStr {
		t.Errorf("round-trip mismatch:\n  got:  %s\n  want: %s", remarshaled, jsonStr)
	}
}

func TestUserMessage_ArrayContent(t *testing.T) {
	jsonStr := `{"role":"user","content":[{"type":"text","text":"hello"},{"type":"image","data":"abc","mimeType":"image/png"}],"timestamp":1}`
	var msg UserMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(msg.Content))
	}
	if msg.Content[0].Type != "text" || msg.Content[0].Text != "hello" {
		t.Errorf("first block mismatch: %+v", msg.Content[0])
	}
	if msg.Content[1].Type != "image" || msg.Content[1].Data != "abc" {
		t.Errorf("second block mismatch: %+v", msg.Content[1])
	}
}

func TestAssistantMessage_RoundTrip(t *testing.T) {
	now := time.Now().UnixMilli()
	msg := AssistantMessage{
		Role: "assistant",
		Content: []ContentBlock{
			NewThinkingContent("think"),
			NewTextContent("answer"),
			NewToolCallContent("tool-1", "read", json.RawMessage(`{"path":"/tmp"}`)),
		},
		API:        "anthropic-messages",
		Provider:   "anthropic",
		Model:      "claude-sonnet-4-6",
		ResponseID: "resp-123",
		Usage: Usage{
			Input:       100,
			Output:      50,
			CacheRead:   0,
			CacheWrite:  20,
			TotalTokens: 170,
			Cost: UsageCost{
				Input:      0.001,
				Output:     0.002,
				CacheWrite: 0.0005,
				Total:      0.0035,
			},
		},
		StopReason: "toolUse",
		Timestamp:  now,
	}

	got, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded AssistantMessage
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Role != "assistant" || decoded.API != "anthropic-messages" || decoded.Provider != "anthropic" {
		t.Errorf("metadata mismatch")
	}
	if len(decoded.Content) != 3 {
		t.Errorf("content length mismatch: %d", len(decoded.Content))
	}
	if decoded.Content[0].Type != "thinking" || decoded.Content[0].Thinking != "think" {
		t.Errorf("thinking block mismatch")
	}
	if decoded.Content[1].Type != "text" || decoded.Content[1].Text != "answer" {
		t.Errorf("text block mismatch")
	}
	if decoded.Content[2].Type != "toolCall" || decoded.Content[2].Name != "read" {
		t.Errorf("toolCall block mismatch")
	}
	if decoded.StopReason != "toolUse" {
		t.Errorf("stopReason mismatch: %q", decoded.StopReason)
	}
	if decoded.Usage.Input != 100 || decoded.Usage.Output != 50 || decoded.Usage.Cost.Input != 0.001 {
		t.Errorf("usage mismatch")
	}
	if decoded.Timestamp != now {
		t.Errorf("timestamp mismatch")
	}
}

func TestToolResultMessage_RoundTrip(t *testing.T) {
	now := time.Now().UnixMilli()
	msg := ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: "tool-1",
		ToolName:   "read",
		Content: []ContentBlock{
			NewTextContent("file contents here"),
		},
		IsError:   false,
		Timestamp: now,
	}

	got, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ToolResultMessage
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Role != "toolResult" || decoded.ToolCallID != "tool-1" || decoded.ToolName != "read" {
		t.Errorf("metadata mismatch")
	}
	if decoded.IsError {
		t.Errorf("isError should be false")
	}
}

func TestToolResultMessage_Error(t *testing.T) {
	jsonStr := `{"role":"toolResult","toolCallId":"tool-1","toolName":"bash","content":[{"type":"text","text":"permission denied"}],"isError":true,"timestamp":1}`
	var msg ToolResultMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !msg.IsError {
		t.Errorf("isError should be true")
	}
	if msg.Content[0].Text != "permission denied" {
		t.Errorf("content mismatch")
	}
}

func TestStopReason_Values(t *testing.T) {
	tests := []struct {
		reason StopReason
		str    string
	}{
		{StopReasonStop, "stop"},
		{StopReasonLength, "length"},
		{StopReasonToolUse, "toolUse"},
		{StopReasonError, "error"},
		{StopReasonAborted, "aborted"},
	}
	for _, tt := range tests {
		if string(tt.reason) != tt.str {
			t.Errorf("StopReason %q: got %q", tt.str, string(tt.reason))
		}
	}
}

func TestMessage_DiscriminatedRoundTrip(t *testing.T) {
	// Test that each message type can be read and re-serialized from JSONL-compatible JSON
	tests := []struct {
		name string
		json string
	}{
		{
			name: "user text message",
			json: `{"role":"user","content":"hi","timestamp":1}`,
		},
		{
			name: "user multimodal message",
			json: `{"role":"user","content":[{"type":"text","text":"look"},{"type":"image","data":"abc","mimeType":"image/png"}],"timestamp":1}`,
		},
		{
			name: "assistant message",
			json: `{"role":"assistant","content":[{"type":"text","text":"hello"}],"api":"openai-completions","provider":"openai","model":"gpt-5","usage":{"input":10,"output":5,"cacheRead":0,"cacheWrite":0,"totalTokens":15,"cost":{"input":0.001,"output":0.002,"cacheRead":0,"cacheWrite":0,"total":0.003}},"stopReason":"stop","timestamp":1}`,
		},
		{
			name: "tool result message",
			json: `{"role":"toolResult","toolCallId":"tool-1","toolName":"read","content":[{"type":"text","text":"ok"}],"isError":false,"timestamp":1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First parse as a generic message to get the role
			var raw struct {
				Role string `json:"role"`
			}
			if err := json.Unmarshal([]byte(tt.json), &raw); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			// Then re-marshal and compare (to ensure round-trip)
			switch raw.Role {
			case "user":
				var msg UserMessage
				if err := json.Unmarshal([]byte(tt.json), &msg); err != nil {
					t.Fatalf("unmarshal UserMessage: %v", err)
				}
				remarshaled, err := json.Marshal(msg)
				if err != nil {
					t.Fatalf("remarshal error: %v", err)
				}
				// Parse again to compare structurally
				var decoded UserMessage
				json.Unmarshal(remarshaled, &decoded)
				if decoded.Role != "user" {
					t.Error("role lost")
				}
			case "assistant":
				var msg AssistantMessage
				if err := json.Unmarshal([]byte(tt.json), &msg); err != nil {
					t.Fatalf("unmarshal AssistantMessage: %v", err)
				}
				remarshaled, err := json.Marshal(msg)
				if err != nil {
					t.Fatalf("remarshal error: %v", err)
				}
				var decoded AssistantMessage
				json.Unmarshal(remarshaled, &decoded)
				if decoded.Role != "assistant" {
					t.Error("role lost")
				}
			case "toolResult":
				var msg ToolResultMessage
				if err := json.Unmarshal([]byte(tt.json), &msg); err != nil {
					t.Fatalf("unmarshal ToolResultMessage: %v", err)
				}
				remarshaled, err := json.Marshal(msg)
				if err != nil {
					t.Fatalf("remarshal error: %v", err)
				}
				var decoded ToolResultMessage
				json.Unmarshal(remarshaled, &decoded)
				if decoded.Role != "toolResult" || decoded.ToolCallID != "tool-1" {
					t.Error("data lost")
				}
			default:
				t.Fatalf("unknown role: %q", raw.Role)
			}
		})
	}
}
