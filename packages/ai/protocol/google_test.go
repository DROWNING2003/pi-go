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

func fakeGoogleServer(t *testing.T, responses []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		for _, line := range responses {
			fmt.Fprintln(w, line)
		}
	}))
}

func TestStreamGoogleGenerate_Text(t *testing.T) {
	responses := []string{
		`[{"candidates":[{"content":{"role":"model","parts":[{"text":"Hello"}]}}]}]`,
		`[{"candidates":[{"content":{"role":"model","parts":[{"text":" world"}]}}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2,"totalTokenCount":7}}]`,
	}
	server := fakeGoogleServer(t, responses)
	defer server.Close()

	m := &provider.ProviderModel{ID: "gemini-2.0-flash", API: "google-generative-ai", Provider: "google"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, nil)
	events := collectStreamEvents(StreamGoogleGenerate(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	if len(last.Message.Content) == 0 {
		t.Error("expected text content, got none")
	}
}

func TestStreamGoogleGenerate_Thinking(t *testing.T) {
	responses := []string{
		`[{"candidates":[{"content":{"role":"model","parts":[{"text":"thinking...","thought":true}]}}]}]`,
		`[{"candidates":[{"content":{"role":"model","parts":[{"text":"Answer"}]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":1,"thoughtsTokenCount":2,"totalTokenCount":6}}]`,
	}
	server := fakeGoogleServer(t, responses)
	defer server.Close()

	m := &provider.ProviderModel{ID: "gemini-2.5-pro", API: "google-generative-ai", Provider: "google"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"q","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, nil)
	events := collectStreamEvents(StreamGoogleGenerate(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	if len(last.Message.Content) < 2 {
		t.Fatalf("expected at least 2 blocks, got %d", len(last.Message.Content))
	}
	if last.Message.Content[0].Type != model.ContentTypeThinking {
		t.Errorf("block 0 type: %q", last.Message.Content[0].Type)
	}
	if last.Message.Content[1].Type != model.ContentTypeText {
		t.Errorf("block 1 type: %q", last.Message.Content[1].Type)
	}
}

func TestStreamGoogleGenerate_ToolCall(t *testing.T) {
	responses := []string{
		`[{"candidates":[{"content":{"role":"model","parts":[{"text":"Let me check"}]}}]}]`,
		`[{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"read","args":{"path":"/tmp"}}}]}}],"usageMetadata":{"totalTokenCount":10}}]`,
	}
	server := fakeGoogleServer(t, responses)
	defer server.Close()

	m := &provider.ProviderModel{ID: "gemini-2.0-flash", API: "google-generative-ai", Provider: "google"}
	c := &provider.Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"read /tmp","timestamp":1}`)},
	}

	client := NewHTTPClient(server.URL, nil)
	events := collectStreamEvents(StreamGoogleGenerate(context.Background(), client, m, c, nil))

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event: %q", last.Type)
	}
	// Should mark as toolUse when tool calls present
	if last.Message.StopReason != model.StopReasonToolUse {
		t.Errorf("stopReason: %q, want toolUse", last.Message.StopReason)
	}
	foundTool := false
	for _, block := range last.Message.Content {
		if block.Type == model.ContentTypeToolCall && block.Name == "read" {
			foundTool = true
		}
	}
	if !foundTool {
		t.Error("no tool call found in content")
	}
}
