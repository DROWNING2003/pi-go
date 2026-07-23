// Package loop implements the agent turn loop: stream assistant response,
// execute tool calls, and continue until the model stops or errors.
package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DROWNING2003/pi-go/packages/agent/event"
	"github.com/DROWNING2003/pi-go/packages/agent/tool"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// StreamFunc is the function signature for streaming model responses.
type StreamFunc func(ctx context.Context, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent

// Config holds agent loop configuration.
type Config struct {
	Model        *provider.ProviderModel
	SystemPrompt string
	Tools        *tool.Registry
	MaxTurns     int
	StreamFn     StreamFunc
}

// Run executes the agent loop with the given prompt messages.
// It emits agent lifecycle events and returns the full message transcript.
func Run(ctx context.Context, config *Config, prompts []*model.UserMessage) ([]event.Message, error) {
	messages := make([]event.Message, 0)
	ctxMessages := make([]json.RawMessage, 0)

	// Add prompts
	for _, p := range prompts {
		data, _ := json.Marshal(p)
		ctxMessages = append(ctxMessages, data)
		messages = append(messages, event.NewUserMessage(p))
	}

	maxTurns := config.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 10
	}

	for turn := 0; turn < maxTurns; turn++ {
		// Build context
		c := &provider.Context{
			SystemPrompt: config.SystemPrompt,
			Messages:     ctxMessages,
		}

		// Add tool definitions
		// (tools are registered but the faux provider handles tool calls differently)

		// Stream response
		ch := config.StreamFn(ctx, config.Model, c, nil)

		var assistantMsg *model.AssistantMessage
		for evt := range ch {
			switch evt.Type {
			case model.StreamEventDone:
				assistantMsg = evt.Message
			case model.StreamEventError:
				return messages, fmt.Errorf("stream error: %s", evt.Error.ErrorMessage)
			}
		}

		if assistantMsg == nil {
			return messages, fmt.Errorf("no assistant response")
		}

		// Add assistant message to transcript
		asstData, _ := json.Marshal(assistantMsg)
		ctxMessages = append(ctxMessages, asstData)
		messages = append(messages, event.NewAssistantMessage(assistantMsg))

		// Check for tool calls
		hasToolCalls := false
		for _, block := range assistantMsg.Content {
			if block.Type == model.ContentTypeToolCall {
				hasToolCalls = true
				break
			}
		}

		if !hasToolCalls {
			// No tool calls, conversation complete
			break
		}

		// Execute tool calls
		for _, block := range assistantMsg.Content {
			if block.Type != model.ContentTypeToolCall {
				continue
			}

			result, err := config.Tools.Execute(ctx, block.ID, block.Name, block.Arguments)
			if err != nil {
				return messages, fmt.Errorf("tool %s: %w", block.Name, err)
			}

			result.Timestamp = time.Now().UnixMilli()
			resultData, _ := json.Marshal(result)
			ctxMessages = append(ctxMessages, resultData)
			messages = append(messages, event.NewToolResultMessage(result))
		}
	}

	return messages, nil
}
