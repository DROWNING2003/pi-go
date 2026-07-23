package options

import (
	"encoding/json"
	"testing"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

func TestBuildStreamOptions(t *testing.T) {
	m := &provider.ProviderModel{ContextWindow: 128000, MaxTokens: 4096}
	c := &provider.Context{
		SystemPrompt: "Be concise.",
		Messages: []json.RawMessage{
			json.RawMessage(`{"role":"user","content":"hi"}`),
		},
	}

	opts := BuildStreamOptions(m, c, 1024, 0.7)
	if opts.MaxTokens <= 0 {
		t.Errorf("maxTokens: %d", opts.MaxTokens)
	}
}

func TestClampReasoning(t *testing.T) {
	if r := ClampReasoning(model.ThinkingXHigh); r != model.ThinkingHigh {
		t.Errorf("xhigh → high: got %s", r)
	}
	if r := ClampReasoning(model.ThinkingHigh); r != model.ThinkingHigh {
		t.Errorf("high → high: got %s", r)
	}
}

func TestAdjustMaxTokensForThinking(t *testing.T) {
	maxT, budget := AdjustMaxTokensForThinking(0, 16384, model.ThinkingMedium)
	if maxT <= 0 || budget <= 0 {
		t.Errorf("max=%d budget=%d", maxT, budget)
	}
}
