package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/DROWNING2003/pi-go/packages/agent/tool"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

func TestAgentLoop_FauxProvider_ReadTool(t *testing.T) {
	// Create a temp file to read
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world"), 0644)

	// Setup faux provider
	faux := provider.NewFauxProvider(provider.WithFauxTokenSize(1, 1))
	faux.SetResponses(
		provider.FauxResponseFactory(func(ctx context.Context, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions, callCount int) *model.AssistantMessage {
			if callCount == 1 {
				return provider.FauxAssistantMessage(
					[]model.ContentBlock{
						provider.FauxToolCall("call-1", "read", json.RawMessage(`{"path":"`+filepath.Join(dir, "hello.txt")+`"}`)),
					},
					model.StopReasonToolUse,
				)
			}
			return provider.FauxAssistantMessage(
				[]model.ContentBlock{provider.FauxText("I read the file: hello world")},
				model.StopReasonStop,
			)
		}),
		provider.FauxResponseFactory(func(ctx context.Context, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions, callCount int) *model.AssistantMessage {
			return provider.FauxAssistantMessage(
				[]model.ContentBlock{provider.FauxText("I read the file: hello world")},
				model.StopReasonStop,
			)
		}),
	)

	// Setup tool registry
	tools := tool.NewRegistry()
	tools.Register(tool.NewReadTool(dir))

	// Setup config
	config := &Config{
		Model: &provider.ProviderModel{
			ID: "faux-1", API: "faux", Provider: "faux",
		},
		SystemPrompt: "You are a helpful assistant.",
		Tools:        tools,
		MaxTurns:     5,
		StreamFn: func(ctx context.Context, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent {
			return faux.Stream(ctx, m, c, opts)
		},
	}

	prompts := []*model.UserMessage{
		{Role: "user", Content: model.UserContent{model.NewTextContent("read hello.txt")}, Timestamp: 1},
	}

	messages, err := Run(context.Background(), config, prompts)
	if err != nil {
		t.Fatalf("agent loop error: %v", err)
	}

	// Verify the transcript
	if len(messages) < 3 {
		t.Fatalf("expected at least 3 messages (user, assistant+toolCall, toolResult, assistant), got %d", len(messages))
	}

	// User message
	if messages[0].Role() != "user" {
		t.Errorf("message[0] role: %q", messages[0].Role())
	}

	// Assistant message with tool call
	if messages[1].Role() != "assistant" {
		t.Errorf("message[1] role: %q", messages[1].Role())
	}
	hasToolCall := false
	for _, b := range messages[1].Assistant.Content {
		if b.Type == model.ContentTypeToolCall {
			hasToolCall = true
		}
	}
	if !hasToolCall {
		t.Error("assistant message missing tool call")
	}

	// Tool result
	if messages[2].Role() != "toolResult" {
		t.Errorf("message[2] role: %q", messages[2].Role())
	}

	// Final assistant response
	last := messages[len(messages)-1]
	if last.Role() != "assistant" {
		t.Errorf("last message role: %q", last.Role())
	}
}

func TestAgentLoop_FauxProvider_NoTools(t *testing.T) {
	faux := provider.NewFauxProvider()
	faux.SetResponses(
		provider.FauxMessage{
			Message: provider.FauxAssistantMessage(
				[]model.ContentBlock{provider.FauxText("hello world")},
				model.StopReasonStop,
			),
		},
	)

	config := &Config{
		Model: &provider.ProviderModel{ID: "faux-1", API: "faux", Provider: "faux"},
		Tools: tool.NewRegistry(),
		StreamFn: func(ctx context.Context, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent {
			return faux.Stream(ctx, m, c, opts)
		},
	}

	prompts := []*model.UserMessage{
		{Role: "user", Content: model.UserContent{model.NewTextContent("hi")}, Timestamp: 1},
	}

	messages, err := Run(context.Background(), config, prompts)
	if err != nil {
		t.Fatalf("agent loop error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages (user, assistant), got %d", len(messages))
	}
	if messages[0].Role() != "user" || messages[1].Role() != "assistant" {
		t.Errorf("unexpected message roles: %s, %s", messages[0].Role(), messages[1].Role())
	}
}
