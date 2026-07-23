// Package agent provides a high-level Agent type with state management,
// event subscriptions, and conversation orchestration.
package agent

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/DROWNING2003/pi-go/packages/agent/loop"
	"github.com/DROWNING2003/pi-go/packages/agent/tool"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// State holds the agent's mutable runtime state.
type State struct {
	SystemPrompt  string
	Model         *provider.ProviderModel
	ThinkingLevel model.ThinkingLevel
	Tools         *tool.Registry
	IsStreaming   bool
}

// Agent orchestrates the conversation with state management.
type Agent struct {
	mu       sync.Mutex
	state    State
	messages []json.RawMessage
	loopCfg  *loop.Config
	registry *provider.Registry
}

// New creates a new agent.
func New(state State, reg *provider.Registry) *Agent {
	return &Agent{
		state:    state,
		registry: reg,
		messages: make([]json.RawMessage, 0),
	}
}

// Prompt sends a user message and returns the resulting messages.
func (a *Agent) Prompt(ctx context.Context, prompt string) ([]model.ContentBlock, error) {
	a.mu.Lock()
	a.state.IsStreaming = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.IsStreaming = false
		a.mu.Unlock()
	}()

	userMsg := model.UserMessage{
		Role:    "user",
		Content: model.UserContent{model.NewTextContent(prompt)},
	}
	data, _ := json.Marshal(userMsg)
	a.messages = append(a.messages, data)

	streamFn := a.resolveStreamFn()

	cfg := &loop.Config{
		Model:        a.state.Model,
		SystemPrompt: a.state.SystemPrompt,
		Tools:        a.state.Tools,
		MaxTurns:     10,
		StreamFn:     streamFn,
	}

	messages, err := loop.Run(ctx, cfg, []*model.UserMessage{&userMsg})
	if err != nil {
		return nil, err
	}

	// Append all messages
	for _, m := range messages[1:] { // skip the user message
		data, _ := json.Marshal(m)
		a.messages = append(a.messages, data)
	}

	// Return last assistant content
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Assistant != nil {
			return messages[i].Assistant.Content, nil
		}
	}
	return nil, nil
}

// Messages returns the full conversation transcript.
func (a *Agent) Messages() []json.RawMessage {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]json.RawMessage, len(a.messages))
	copy(out, a.messages)
	return out
}

func (a *Agent) resolveStreamFn() loop.StreamFunc {
	return func(ctx context.Context, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent {
		ch, err := a.registry.Stream(ctx, m.ID, c, opts)
		if err != nil {
			ech := make(chan model.StreamEvent, 1)
			ech <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{
				ErrorMessage: err.Error(),
			})
			close(ech)
			return ech
		}
		return ch
	}
}
