package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

func fakeAnthropicServer(t *testing.T, chunks []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
		}
	}))
}

func TestStreamAnthropicMessages_Text(t *testing.T) {
	chunks := []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-6","content":[],"stop_reason":null}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"type":"stop_reason","text":"end_turn"},"usage":{"input_tokens":5,"output_tokens":2}}`,
		`{"type":"message_stop"}`,
	}
	server := fakeAnthropicServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "claude-sonnet-4-6", API: "anthropic-messages", Provider: "anthropic"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, map[string]string{"x-api-key": "test"})
	events := collectStreamEvents(StreamAnthropicMessages(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	if last.Message.StopReason != model.StopReasonStop {
		t.Errorf("stopReason: %q", last.Message.StopReason)
	}
	if len(last.Message.Content) != 1 || last.Message.Content[0].Text != "Hello world" {
		t.Errorf("content: %+v", last.Message.Content)
	}
}

func TestStreamAnthropicMessages_Thinking(t *testing.T) {
	chunks := []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[]}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Let me think"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Answer"}}`,
		`{"type":"content_block_stop","index":1}`,
		`{"type":"message_delta","delta":{"type":"stop_reason","text":"end_turn"}}`,
		`{"type":"message_stop"}`,
	}
	server := fakeAnthropicServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "claude", API: "anthropic-messages", Provider: "anthropic"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"q","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, map[string]string{"x-api-key": "test"})
	events := collectStreamEvents(StreamAnthropicMessages(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if len(last.Message.Content) < 2 {
		t.Fatalf("expected 2 blocks, got %d", len(last.Message.Content))
	}
	if last.Message.Content[0].Type != model.ContentTypeThinking {
		t.Errorf("block 0 type: %q", last.Message.Content[0].Type)
	}
	if last.Message.Content[1].Type != model.ContentTypeText {
		t.Errorf("block 1 type: %q", last.Message.Content[1].Type)
	}
}

func TestStreamAnthropicMessages_ToolUse(t *testing.T) {
	chunks := []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[]}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"read"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"/tmp\"}"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"type":"stop_reason","text":"tool_use"}}`,
		`{"type":"message_stop"}`,
	}
	server := fakeAnthropicServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "claude", API: "anthropic-messages", Provider: "anthropic"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"read /tmp","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, map[string]string{"x-api-key": "test"})
	events := collectStreamEvents(StreamAnthropicMessages(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	if last.Message.StopReason != model.StopReasonToolUse {
		t.Errorf("stopReason: %q, want toolUse", last.Message.StopReason)
	}
	if len(last.Message.Content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(last.Message.Content))
	}
	tc := last.Message.Content[0]
	if tc.Type != model.ContentTypeToolCall || tc.Name != "read" {
		t.Errorf("toolCall: %+v", tc)
	}
	if string(tc.Arguments) != `{"path":"/tmp"}` {
		t.Errorf("args: %s", tc.Arguments)
	}
}
