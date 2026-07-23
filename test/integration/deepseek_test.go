// Package integration contains smoke tests for real provider APIs.
// These tests require API keys in environment variables.
// Run with: go test -tags=integration -count=1 ./test/integration/
package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/protocol"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// TestDeepSeekChat requires DEEPSEEK_API_KEY env var.
// Usage: DEEPSEEK_API_KEY=sk-xxx go test -tags=integration -run TestDeepSeek -v ./test/integration/
func TestDeepSeekChat(t *testing.T) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}

	client := protocol.NewHTTPClient("https://api.deepseek.com", map[string]string{
		"Authorization": "Bearer " + apiKey,
	})

	m := &provider.ProviderModel{
		ID:       "deepseek-chat",
		API:      "openai-completions",
		Provider: "deepseek",
	}

	c := &provider.Context{
		Messages: []json.RawMessage{
			json.RawMessage(`{"role":"user","content":"Say hello in one sentence.","timestamp":1}`),
		},
	}

	ctx := context.Background()
	ch := protocol.StreamChatCompletion(ctx, client, m, c, nil)

	t.Log("streaming...")
	for event := range ch {
		switch event.Type {
		case model.StreamEventTextDelta:
			t.Logf("  delta: %q", event.Delta)
		case model.StreamEventTextEnd:
			t.Logf("  text: %q", event.Content)
		case model.StreamEventDone:
			t.Logf("done: stopReason=%s usage=%+v", event.Message.StopReason, event.Message.Usage)
			for _, block := range event.Message.Content {
				t.Logf("  content[%s]: %q", block.Type, block.Text)
			}
		case model.StreamEventError:
			t.Fatalf("error: %s", event.Error.ErrorMessage)
		}
	}
}
