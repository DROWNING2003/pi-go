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

func fakeResponsesServer(t *testing.T, chunks []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
		}
	}))
}

func TestStreamOpenAIResponses_Text(t *testing.T) {
	chunks := []string{
		`{"type":"response.created","response":{"id":"resp_1","model":"gpt-5"}}`,
		`{"type":"response.output_text.delta","delta":"Hello"}`,
		`{"type":"response.output_text.delta","delta":" world"}`,
		`{"type":"response.completed","response":{"id":"resp_1","status":"completed","usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7,"input_tokens_details":{"cached_tokens":0},"output_tokens_details":{"reasoning_tokens":0}}}}`,
	}
	server := fakeResponsesServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "gpt-5", API: "openai-responses", Provider: "openai"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, map[string]string{"Authorization": "Bearer test"})
	events := collectStreamEvents(StreamOpenAIResponses(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	if len(last.Message.Content) != 1 || last.Message.Content[0].Text != "Hello world" {
		t.Errorf("content: %+v", last.Message.Content)
	}
	if last.Message.Usage.Input != 5 {
		t.Errorf("usage: %+v", last.Message.Usage)
	}
}

func TestStreamOpenAIResponses_ToolCall(t *testing.T) {
	chunks := []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.output_item.added","item":{"id":"fc_1","type":"function_call","name":"read"}}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"{\"pa"}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"th\":\"/tmp\"}"}`,
		`{"type":"response.function_call_arguments.done","item_id":"fc_1","arguments":"{\"path\":\"/tmp\"}"}`,
		`{"type":"response.output_item.done","item":{"id":"fc_1","type":"function_call","status":"completed"}}`,
		`{"type":"response.completed","response":{"id":"resp_1"}}`,
	}
	server := fakeResponsesServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "gpt-5", API: "openai-responses", Provider: "openai"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"read /tmp","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, map[string]string{"Authorization": "Bearer test"})
	events := collectStreamEvents(StreamOpenAIResponses(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	found := false
	for _, b := range last.Message.Content {
		if b.Type == model.ContentTypeToolCall && b.Name == "read" {
			found = true
		}
	}
	if !found {
		t.Error("no tool call found")
	}
}

func TestStreamOpenAIResponses_Reasoning(t *testing.T) {
	chunks := []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.reasoning_text.delta","delta":"Let me think"}`,
		`{"type":"response.reasoning_text.delta","delta":" about this"}`,
		`{"type":"response.output_text.delta","delta":"Answer"}`,
		`{"type":"response.completed","response":{"id":"resp_1"}}`,
	}
	server := fakeResponsesServer(t, chunks)
	defer server.Close()

	m := &provider.ProviderModel{ID: "o1", API: "openai-responses", Provider: "openai"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"q","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, nil)
	events := collectStreamEvents(StreamOpenAIResponses(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if len(last.Message.Content) < 2 {
		t.Fatalf("expected thinking + text, got %d blocks", len(last.Message.Content))
	}
	if last.Message.Content[0].Type != model.ContentTypeThinking {
		t.Errorf("block 0: %q", last.Message.Content[0].Type)
	}
}
