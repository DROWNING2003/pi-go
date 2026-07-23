// Package loop implements the agent turn loop: stream assistant response,
// execute tool calls, and continue until the model stops or errors.
package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DROWNING2003/pi-go/packages/agent/event"
	"github.com/DROWNING2003/pi-go/packages/agent/queue"
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
	QueueManager *queue.Manager
	Parallel     bool // execute tool calls in parallel (default: true)
}

// Run executes the agent loop with the given prompt messages.
func Run(ctx context.Context, config *Config, prompts []*model.UserMessage) ([]event.Message, error) {
	messages := make([]event.Message, 0)
	ctxMessages := make([]json.RawMessage, 0)

	qm := config.QueueManager
	if qm == nil {
		qm = queue.NewManager(queue.QueueModeAll)
	}

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

	parallel := config.Parallel
	// Default to parallel

	for turn := 0; turn < maxTurns; turn++ {
		// Check abort
		if qm.IsAborted() {
			break
		}

		// Inject pending steering messages
		steering := qm.DrainSteering()
		for _, s := range steering {
			data, _ := json.Marshal(s)
			ctxMessages = append(ctxMessages, data)
			messages = append(messages, event.NewUserMessage(s))
		}

		// Build context
		c := &provider.Context{
			SystemPrompt: config.SystemPrompt,
			Messages:     ctxMessages,
		}

		// Add tool definitions
		// (tools registered but faux provider handles tool definitions differently)

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

		asstData, _ := json.Marshal(assistantMsg)
		ctxMessages = append(ctxMessages, asstData)
		messages = append(messages, event.NewAssistantMessage(assistantMsg))

		// Check for tool calls
		toolCalls := make([]model.ContentBlock, 0)
		for _, block := range assistantMsg.Content {
			if block.Type == model.ContentTypeToolCall {
				toolCalls = append(toolCalls, block)
			}
		}

		if len(toolCalls) == 0 {
			// Check for follow-up messages
			followUp := qm.DrainFollowUp()
			if len(followUp) > 0 {
				for _, f := range followUp {
					data, _ := json.Marshal(f)
					ctxMessages = append(ctxMessages, data)
					messages = append(messages, event.NewUserMessage(f))
				}
				continue
			}
			break
		}

		// Execute tool calls
		results := executeToolCalls(ctx, config.Tools, toolCalls, parallel)
		for _, result := range results {
			result.Timestamp = time.Now().UnixMilli()
			resultData, _ := json.Marshal(result)
			ctxMessages = append(ctxMessages, resultData)
			messages = append(messages, event.NewToolResultMessage(result))
		}
	}

	return messages, nil
}

func executeToolCalls(ctx context.Context, tools *tool.Registry, toolCalls []model.ContentBlock, parallel bool) []*model.ToolResultMessage {
	if !parallel || len(toolCalls) <= 1 {
		return executeSequential(ctx, tools, toolCalls)
	}
	return executeParallel(ctx, tools, toolCalls)
}

func executeSequential(ctx context.Context, tools *tool.Registry, toolCalls []model.ContentBlock) []*model.ToolResultMessage {
	var results []*model.ToolResultMessage
	for _, tc := range toolCalls {
		result, err := tools.Execute(ctx, tc.ID, tc.Name, tc.Arguments)
		if err != nil {
			result = &model.ToolResultMessage{
				Role:       "toolResult",
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Content:    []model.ContentBlock{model.NewTextContent(err.Error())},
				IsError:    true,
			}
		}
		results = append(results, result)
	}
	return results
}

func executeParallel(ctx context.Context, tools *tool.Registry, toolCalls []model.ContentBlock) []*model.ToolResultMessage {
	type indexedResult struct {
		index  int
		result *model.ToolResultMessage
	}

	ch := make(chan indexedResult, len(toolCalls))
	for i, tc := range toolCalls {
		go func(idx int, call model.ContentBlock) {
			result, err := tools.Execute(ctx, call.ID, call.Name, call.Arguments)
			if err != nil {
				result = &model.ToolResultMessage{
					Role:       "toolResult",
					ToolCallID: call.ID,
					ToolName:   call.Name,
					Content:    []model.ContentBlock{model.NewTextContent(err.Error())},
					IsError:    true,
				}
			}
			ch <- indexedResult{idx, result}
		}(i, tc)
	}

	results := make([]*model.ToolResultMessage, len(toolCalls))
	for range toolCalls {
		r := <-ch
		results[r.index] = r.result
	}
	return results
}
