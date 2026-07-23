// Package protocol implements provider-specific API clients.
//
// Each sub-package handles a specific provider API (OpenAI, Anthropic, Google)
// and maps wire-format responses to the standard model.StreamEvent contract.
package protocol

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// --- OpenAI Chat Completions request/response types ---

// ChatCompletionRequest is the request body for OpenAI chat completions.
type ChatCompletionRequest struct {
	Model               string         `json:"model"`
	Messages            []ChatMessage  `json:"messages"`
	Stream              bool           `json:"stream"`
	StreamOptions       *StreamOptions `json:"stream_options,omitempty"`
	MaxCompletionTokens int            `json:"max_completion_tokens,omitempty"`
	Temperature         float64        `json:"temperature,omitempty"`
	Tools               []ChatTool     `json:"tools,omitempty"`
	Store               bool           `json:"store,omitempty"`
}

// StreamOptions configures streaming behavior.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// ChatMessage represents a message in the chat completions conversation.
type ChatMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content,omitempty"`
	ToolCalls  []ChatToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

// ChatToolCall represents a tool call in a message.
type ChatToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ChatFunction `json:"function"`
}

// ChatFunction represents the function part of a tool call.
type ChatFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatTool defines a tool available to the model.
type ChatTool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction defines the function schema.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ChatCompletionChunk is a single chunk from the streaming response.
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
	Usage   *ChunkUsage   `json:"usage,omitempty"`
}

// ChunkChoice represents a single choice in a chunk.
type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChunkDelta `json:"delta"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

// ChunkDelta represents the delta content in a chunk.
type ChunkDelta struct {
	Role             string          `json:"role,omitempty"`
	Content          string          `json:"content,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ToolCalls        []ChunkToolCall `json:"tool_calls,omitempty"`
}

// ChunkToolCall is a partial tool call in a chunk delta.
type ChunkToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function *struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function,omitempty"`
}

// ChunkUsage represents usage information that may appear in the final chunk.
type ChunkUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// --- Streaming adapter ---

// StreamChatCompletion sends a chat completion request and returns a channel
// of standard StreamEvent. It handles SSE parsing, delta accumulation,
// and maps tool calls, thinking, and text to the standard event protocol.
func StreamChatCompletion(ctx context.Context, client *HTTPClient, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent {
	ch := make(chan model.StreamEvent, 32)

	go func() {
		defer close(ch)

		req := buildChatRequest(m, c, opts)
		resp, err := client.DoStream(ctx, "/v1/chat/completions", req, nil)
		if err != nil {
			errMsg := errorMessage(m, "http error: "+err.Error())
			ch <- model.NewErrorEvent(model.StopReasonError, errMsg)
			return
		}
		defer resp.Body.Close()

		parseChatStream(ctx, ch, m, resp.Body)
	}()

	return ch
}

func buildChatRequest(m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) *ChatCompletionRequest {
	req := &ChatCompletionRequest{
		Model:         m.ID,
		Stream:        true,
		Store:         false,
		StreamOptions: &StreamOptions{IncludeUsage: true},
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req.MaxCompletionTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			req.Temperature = opts.Temperature
		}
	}

	// Convert messages
	for _, raw := range c.Messages {
		role, content := extractRoleContent(raw)
		msg := ChatMessage{Role: role, Content: content}
		req.Messages = append(req.Messages, msg)
	}

	// Convert tools
	for _, t := range c.Tools {
		req.Tools = append(req.Tools, ChatTool{
			Type: "function",
			Function: ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	return req
}

func extractRoleContent(raw json.RawMessage) (role string, content json.RawMessage) {
	var msg struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	json.Unmarshal(raw, &msg)
	return msg.Role, msg.Content
}

func parseChatStream(ctx context.Context, ch chan<- model.StreamEvent, m *provider.ProviderModel, body io.Reader) {
	parser := NewSSEParser(body)

	partial := &model.AssistantMessage{
		Role:       "assistant",
		Content:    []model.ContentBlock{},
		API:        m.API,
		Provider:   m.Provider,
		Model:      m.ID,
		StopReason: model.StopReasonStop,
	}

	ch <- model.NewStartEvent(partial)

	var (
		currentText     *model.ContentBlock
		currentThinking *model.ContentBlock
		toolCallsByIdx  = map[int]*model.ContentBlock{}
		toolCallArgs    = map[int]*strings.Builder{}
		finishReason    string
		usage           model.Usage
	)

	commitText := func() {
		if currentText != nil && currentText.Text != "" {
			idx := len(partial.Content)
			partial.Content = append(partial.Content, *currentText)
			ch <- model.NewTextEndEvent(idx, currentText.Text, cpAssistant(partial))
			currentText = nil
		}
	}
	commitThinking := func() {
		if currentThinking != nil && currentThinking.Thinking != "" {
			idx := len(partial.Content)
			partial.Content = append(partial.Content, *currentThinking)
			ch <- model.NewThinkingEndEvent(idx, currentThinking.Thinking, cpAssistant(partial))
			currentThinking = nil
		}
	}
	commitToolCalls := func() {
		for idx := 0; ; idx++ {
			tc, ok := toolCallsByIdx[idx]
			if !ok {
				break
			}
			if sb, ok := toolCallArgs[idx]; ok && sb.Len() > 0 {
				tc.Arguments = json.RawMessage(sb.String())
			}
			pos := len(partial.Content)
			partial.Content = append(partial.Content, *tc)
			ch <- model.NewToolCallEndEvent(pos, cpContent(tc), cpAssistant(partial))
		}
		toolCallsByIdx = map[int]*model.ContentBlock{}
		toolCallArgs = map[int]*strings.Builder{}
	}

	for {
		select {
		case <-ctx.Done():
			partial.StopReason = model.StopReasonAborted
			partial.ErrorMessage = "aborted"
			ch <- model.NewErrorEvent(model.StopReasonAborted, partial)
			return
		default:
		}

		event, err := parser.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			errMsg := errorMessage(m, "sse parse error: "+err.Error())
			ch <- model.NewErrorEvent(model.StopReasonError, errMsg)
			return
		}

		if event.Data == "[DONE]" {
			break
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(event.Data), &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		if chunk.Usage != nil {
			usage = model.Usage{
				Input:       chunk.Usage.PromptTokens,
				Output:      chunk.Usage.CompletionTokens,
				TotalTokens: chunk.Usage.TotalTokens,
			}
		}

		for _, choice := range chunk.Choices {
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}

			delta := choice.Delta

			// Handle thinking/reasoning content
			if delta.ReasoningContent != "" {
				commitText()
				commitToolCalls()
				if currentThinking == nil {
					currentThinking = &model.ContentBlock{Type: model.ContentTypeThinking}
					ch <- model.NewThinkingStartEvent(len(partial.Content), cpAssistant(partial))
				}
				currentThinking.Thinking += delta.ReasoningContent
				ch <- model.NewThinkingDeltaEvent(len(partial.Content), delta.ReasoningContent, cpAssistant(partial))
			}

			// Handle text content
			if delta.Content != "" {
				commitThinking()
				commitToolCalls()
				if currentText == nil {
					currentText = &model.ContentBlock{Type: model.ContentTypeText}
					ch <- model.NewTextStartEvent(len(partial.Content), cpAssistant(partial))
				}
				currentText.Text += delta.Content
				ch <- model.NewTextDeltaEvent(len(partial.Content), delta.Content, cpAssistant(partial))
			}

			// Handle tool calls
			for _, tc := range delta.ToolCalls {
				commitText()
				commitThinking()

				idx := tc.Index
				if _, ok := toolCallsByIdx[idx]; !ok {
					toolCallsByIdx[idx] = &model.ContentBlock{
						Type: model.ContentTypeToolCall,
						ID:   tc.ID,
					}
					toolCallArgs[idx] = &strings.Builder{}
					ch <- model.NewToolCallStartEvent(len(partial.Content)+idx, cpAssistant(partial))
				}

				if tc.ID != "" {
					toolCallsByIdx[idx].ID = tc.ID
				}
				if tc.Function != nil {
					if tc.Function.Name != "" {
						toolCallsByIdx[idx].Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						toolCallArgs[idx].WriteString(tc.Function.Arguments)
						ch <- model.NewToolCallDeltaEvent(len(partial.Content)+idx, tc.Function.Arguments, cpAssistant(partial))
					}
				}
			}
		}
	}

	// Commit any pending blocks
	commitText()
	commitThinking()
	commitToolCalls()

	// Finalize
	stopReason := finishReasonToStopReason(finishReason)
	partial.StopReason = stopReason
	partial.Usage = usage

	if stopReason == model.StopReasonError {
		ch <- model.NewErrorEvent(stopReason, partial)
	} else {
		ch <- model.NewDoneEvent(stopReason, partial)
	}
}

func finishReasonToStopReason(fr string) model.StopReason {
	switch fr {
	case "stop":
		return model.StopReasonStop
	case "length":
		return model.StopReasonLength
	case "tool_calls":
		return model.StopReasonToolUse
	default:
		return model.StopReasonStop
	}
}

func errorMessage(m *provider.ProviderModel, text string) *model.AssistantMessage {
	return &model.AssistantMessage{
		Role:         "assistant",
		Content:      []model.ContentBlock{},
		API:          m.API,
		Provider:     m.Provider,
		Model:        m.ID,
		StopReason:   model.StopReasonError,
		ErrorMessage: text,
	}
}

func cpAssistant(msg *model.AssistantMessage) *model.AssistantMessage {
	c := *msg
	c.Content = append([]model.ContentBlock{}, msg.Content...)
	return &c
}

func cpContent(b *model.ContentBlock) *model.ContentBlock {
	c := *b
	return &c
}
