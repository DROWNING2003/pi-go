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

func fakeOpenAIServer(t *testing.T, chunks []string) *httptest.Server {
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
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

func collectStreamEvents(ch <-chan model.StreamEvent) []model.StreamEvent {
	var events []model.StreamEvent
	for e := range ch {
		events = append(events, e)
	}
	return events
}

func TestStreamChatCompletion_Text(t *testing.T) {
	chunks := []string{
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hello"}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
	}
	server := fakeOpenAIServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "gpt-4", API: "openai-completions", Provider: "openai"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, map[string]string{"Authorization": "Bearer test"})
	events := collectStreamEvents(StreamChatCompletion(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	if last.Message.StopReason != model.StopReasonStop {
		t.Errorf("stopReason: %q", last.Message.StopReason)
	}
	if len(last.Message.Content) != 1 || last.Message.Content[0].Text != "hello world" {
		t.Errorf("content: %+v", last.Message.Content)
	}
}

func TestStreamChatCompletion_ToolCall(t *testing.T) {
	chunks := []string{
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Let me check."}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"read","arguments":"{\"pa"}}]}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"/tmp\"}"}}]}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	server := fakeOpenAIServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "gpt-4", API: "openai-completions", Provider: "openai"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"check path","timestamp":1}`)},
		Tools: []provider.ToolDef{
			{Name: "read", Description: "Read a file", Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)},
		},
	}

	client := NewHTTPClient(server.URL, map[string]string{"Authorization": "Bearer test"})
	events := collectStreamEvents(StreamChatCompletion(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	if last.Message.StopReason != model.StopReasonToolUse {
		t.Errorf("stopReason: %q, want toolUse", last.Message.StopReason)
	}

	// Should have text + toolCall
	if len(last.Message.Content) < 2 {
		t.Fatalf("expected at least 2 content blocks, got %d: %+v", len(last.Message.Content), last.Message.Content)
	}
	foundTool := false
	for _, block := range last.Message.Content {
		if block.Type == model.ContentTypeToolCall {
			foundTool = true
			if block.Name != "read" {
				t.Errorf("tool name: %q", block.Name)
			}
			if string(block.Arguments) != `{"path":"/tmp"}` {
				t.Errorf("tool args: %s", block.Arguments)
			}
		}
	}
	if !foundTool {
		t.Error("no tool call found in content")
	}
}

func TestStreamChatCompletion_Thinking(t *testing.T) {
	chunks := []string{
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"reasoning_content":"Let me think"}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"reasoning_content":" about this"}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Answer here"}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
	}
	server := fakeOpenAIServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "o1", API: "openai-completions", Provider: "openai"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"question","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, map[string]string{"Authorization": "Bearer test"})
	events := collectStreamEvents(StreamChatCompletion(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	if len(last.Message.Content) < 2 {
		t.Fatalf("expected thinking + text, got %d blocks", len(last.Message.Content))
	}

	hasThinking := false
	hasText := false
	for _, block := range last.Message.Content {
		if block.Type == model.ContentTypeThinking {
			hasThinking = true
			if block.Thinking != "Let me think about this" {
				t.Errorf("thinking: %q", block.Thinking)
			}
		}
		if block.Type == model.ContentTypeText {
			hasText = true
		}
	}
	if !hasThinking || !hasText {
		t.Errorf("thinking=%v text=%v", hasThinking, hasText)
	}
}

func TestStreamChatCompletion_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized"}`)
	}))
	defer server.Close()

	m := &provider.ProviderModel{ID: "gpt-4", API: "openai-completions", Provider: "openai"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, nil)
	events := collectStreamEvents(StreamChatCompletion(context.Background(), client, m, c, nil))

	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}
	if events[0].Type != model.StreamEventError {
		t.Errorf("expected error, got %q", events[0].Type)
	}
	if events[0].Error == nil || events[0].Error.ErrorMessage == "" {
		t.Error("missing error message")
	}
}

func TestStreamChatCompletion_UsageInFinalChunk(t *testing.T) {
	chunks := []string{
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"done"}}]}`,
		`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}`,
	}
	server := fakeOpenAIServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "gpt-4", API: "openai-completions", Provider: "openai"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, nil)
	events := collectStreamEvents(StreamChatCompletion(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Message.Usage.Input != 10 || last.Message.Usage.Output != 3 {
		t.Errorf("usage: %+v", last.Message.Usage)
	}
}
