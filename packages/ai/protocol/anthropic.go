package protocol

import (
	"context"
	"encoding/json"
	"io"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// --- Anthropic Messages request/response types ---

// AnthropicMessageRequest is the request body for Anthropic Messages API.
type AnthropicMessageRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	System      interface{}        `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Stream      bool               `json:"stream"`
	Tools       []AnthropicTool    `json:"tools,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	Thinking    *AnthropicThinking `json:"thinking,omitempty"`
}

// AnthropicThinking configures extended thinking.
type AnthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// AnthropicMessage is a message in the Anthropic format.
type AnthropicMessage struct {
	Role    string             `json:"role"`
	Content []AnthropicContent `json:"content"`
}

// AnthropicContent is a content block in Anthropic format.
type AnthropicContent struct {
	Type      string           `json:"type"`
	Text      string           `json:"text,omitempty"`
	Thinking  string           `json:"thinking,omitempty"`
	Signature string           `json:"signature,omitempty"`
	Name      string           `json:"name,omitempty"`
	ID        string           `json:"id,omitempty"`
	Input     json.RawMessage  `json:"input,omitempty"`
	ToolUseID string           `json:"tool_use_id,omitempty"`
	Content   json.RawMessage  `json:"content,omitempty"`
	IsError   bool             `json:"is_error,omitempty"`
	Source    *AnthropicSource `json:"source,omitempty"`
}

// AnthropicSource is an image source.
type AnthropicSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// AnthropicTool is a tool definition in Anthropic format.
type AnthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// --- SSE event types ---

// AnthropicSSEEvent is a single SSE event from the Anthropic stream.
type AnthropicSSEEvent struct {
	Type    string `json:"type"`
	Message *struct {
		ID           string             `json:"id"`
		Type         string             `json:"type"`
		Role         string             `json:"role"`
		Model        string             `json:"model"`
		Content      []AnthropicContent `json:"content"`
		StopReason   string             `json:"stop_reason"`
		StopSequence string             `json:"stop_sequence"`
		Usage        *AnthropicUsage    `json:"usage"`
	} `json:"message,omitempty"`

	Index        *int              `json:"index,omitempty"`
	ContentBlock *AnthropicContent `json:"content_block,omitempty"`
	Delta        *AnthropicDelta   `json:"delta,omitempty"`
	Usage        *AnthropicUsage   `json:"usage,omitempty"`
}

// AnthropicDelta represents a content delta.
type AnthropicDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

// AnthropicUsage represents token usage.
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
}

// --- Streaming adapter ---

// StreamAnthropicMessages sends an Anthropic Messages request and returns
// standard StreamEvent channel.
func StreamAnthropicMessages(ctx context.Context, client *HTTPClient, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent {
	ch := make(chan model.StreamEvent, 32)

	go func() {
		defer close(ch)

		req := buildAnthropicRequest(m, c, opts)
		resp, err := client.DoStream(ctx, "/v1/messages", req, nil)
		if err != nil {
			errMsg := errorMessage(m, "http error: "+err.Error())
			ch <- model.NewErrorEvent(model.StopReasonError, errMsg)
			return
		}
		defer resp.Body.Close()

		parseAnthropicStream(ctx, ch, m, resp.Body)
	}()

	return ch
}

func buildAnthropicRequest(m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) *AnthropicMessageRequest {
	req := &AnthropicMessageRequest{
		Model:     m.ID,
		MaxTokens: 4096,
		Stream:    true,
	}

	if opts != nil && opts.MaxTokens > 0 {
		req.MaxTokens = opts.MaxTokens
	}

	// System prompt
	if c.SystemPrompt != "" {
		req.System = c.SystemPrompt
	}

	// Convert messages
	for _, raw := range c.Messages {
		msg := AnthropicMessage{Role: "user"}
		role, content := extractRoleContent(raw)
		switch role {
		case "user":
			msg.Role = "user"
		case "assistant":
			msg.Role = "assistant"
		case "toolResult":
			msg.Role = "user"
		}

		// Try to parse content as array or string
		var blocks []AnthropicContent
		var str string
		if json.Unmarshal(content, &str) == nil {
			blocks = []AnthropicContent{{Type: "text", Text: str}}
		} else if json.Unmarshal(content, &blocks) == nil {
			// Convert model.ContentBlock format to Anthropic format
			var rawBlocks []model.ContentBlock
			if json.Unmarshal(content, &rawBlocks) == nil {
				blocks = convertContentBlocks(rawBlocks)
			}
		}
		msg.Content = blocks
		req.Messages = append(req.Messages, msg)
	}

	// Convert tools
	for _, t := range c.Tools {
		req.Tools = append(req.Tools, AnthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	return req
}

func convertContentBlocks(blocks []model.ContentBlock) []AnthropicContent {
	var out []AnthropicContent
	for _, b := range blocks {
		switch b.Type {
		case model.ContentTypeText:
			out = append(out, AnthropicContent{Type: "text", Text: b.Text})
		case model.ContentTypeThinking:
			out = append(out, AnthropicContent{Type: "thinking", Thinking: b.Thinking, Signature: b.ThinkingSignature})
		case model.ContentTypeImage:
			out = append(out, AnthropicContent{
				Type:   "image",
				Source: &AnthropicSource{Type: "base64", MediaType: b.MimeType, Data: b.Data},
			})
		case model.ContentTypeToolCall:
			out = append(out, AnthropicContent{Type: "tool_use", ID: b.ID, Name: b.Name, Input: b.Arguments})
		}
	}
	return out
}

func parseAnthropicStream(ctx context.Context, ch chan<- model.StreamEvent, m *provider.ProviderModel, body io.Reader) {
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

	var responseID string
	contentBlocks := map[int]*model.ContentBlock{}
	toolInputs := map[int]*stringBuilder{}
	var stopReason string
	var usage model.Usage

	for {
		event, err := parser.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ch <- model.NewErrorEvent(model.StopReasonError, errorMessage(m, "sse error: "+err.Error()))
			return
		}

		var sse AnthropicSSEEvent
		if err := json.Unmarshal([]byte(event.Data), &sse); err != nil {
			continue
		}

		switch sse.Type {
		case "message_start":
			if sse.Message != nil {
				responseID = sse.Message.ID
			}

		case "content_block_start":
			if sse.ContentBlock == nil || sse.Index == nil {
				continue
			}
			idx := *sse.Index
			cb := sse.ContentBlock
			switch cb.Type {
			case "text":
				contentBlocks[idx] = &model.ContentBlock{Type: model.ContentTypeText}
				ch <- model.NewTextStartEvent(idx, cpAssistant(partial))
			case "thinking":
				contentBlocks[idx] = &model.ContentBlock{Type: model.ContentTypeThinking}
				ch <- model.NewThinkingStartEvent(idx, cpAssistant(partial))
			case "tool_use":
				contentBlocks[idx] = &model.ContentBlock{
					Type: model.ContentTypeToolCall,
					ID:   cb.ID,
					Name: cb.Name,
				}
				toolInputs[idx] = &stringBuilder{}
				ch <- model.NewToolCallStartEvent(idx, cpAssistant(partial))
			}

		case "content_block_delta":
			if sse.Delta == nil || sse.Index == nil {
				continue
			}
			idx := *sse.Index
			block, ok := contentBlocks[idx]
			if !ok {
				continue
			}
			switch sse.Delta.Type {
			case "text_delta":
				block.Text += sse.Delta.Text
				ch <- model.NewTextDeltaEvent(idx, sse.Delta.Text, cpAssistant(partial))
			case "thinking_delta":
				block.Thinking += sse.Delta.Thinking
				ch <- model.NewThinkingDeltaEvent(idx, sse.Delta.Thinking, cpAssistant(partial))
			case "input_json_delta":
				if sb, ok := toolInputs[idx]; ok {
					sb.WriteString(sse.Delta.PartialJSON)
					ch <- model.NewToolCallDeltaEvent(idx, sse.Delta.PartialJSON, cpAssistant(partial))
				}
			}

		case "content_block_stop":
			if sse.Index == nil {
				continue
			}
			idx := *sse.Index
			block, ok := contentBlocks[idx]
			if !ok {
				continue
			}
			// Finalize arguments for tool calls
			if block.Type == model.ContentTypeToolCall {
				if sb, ok := toolInputs[idx]; ok {
					block.Arguments = json.RawMessage(sb.String())
				}
				ch <- model.NewToolCallEndEvent(idx, cpContent(block), cpAssistant(partial))
			} else if block.Type == model.ContentTypeText {
				ch <- model.NewTextEndEvent(idx, block.Text, cpAssistant(partial))
			} else if block.Type == model.ContentTypeThinking {
				ch <- model.NewThinkingEndEvent(idx, block.Thinking, cpAssistant(partial))
			}

		case "message_delta":
			if sse.Delta != nil && sse.Delta.Type == "stop_reason" {
				stopReason = sse.Delta.Text
			}
			if sse.Usage != nil {
				usage = convertAnthropicUsage(sse.Usage)
			}

		case "message_stop":
			// Finalize
		}
	}

	// Build final content array from collected blocks
	for idx := 0; ; idx++ {
		block, ok := contentBlocks[idx]
		if !ok {
			break
		}
		partial.Content = append(partial.Content, *block)
	}

	partial.StopReason = anthropicStopReason(stopReason)
	partial.Usage = usage
	partial.ResponseID = responseID

	if partial.StopReason == model.StopReasonError {
		ch <- model.NewErrorEvent(partial.StopReason, partial)
	} else {
		ch <- model.NewDoneEvent(partial.StopReason, partial)
	}
}

func anthropicStopReason(sr string) model.StopReason {
	switch sr {
	case "end_turn":
		return model.StopReasonStop
	case "max_tokens":
		return model.StopReasonLength
	case "tool_use":
		return model.StopReasonToolUse
	default:
		if sr != "" {
			return model.StopReasonStop
		}
		return model.StopReasonStop
	}
}

func convertAnthropicUsage(u *AnthropicUsage) model.Usage {
	return model.Usage{
		Input:       u.InputTokens,
		Output:      u.OutputTokens,
		CacheRead:   u.CacheReadInputTokens,
		CacheWrite:  u.CacheCreationInputTokens,
		TotalTokens: u.InputTokens + u.OutputTokens,
	}
}

// stringBuilder is a simple strings.Builder wrapper that satisfies io.Writer.
type stringBuilder struct {
	builder []byte
}

func (s *stringBuilder) WriteString(str string) {
	s.builder = append(s.builder, str...)
}

func (s *stringBuilder) String() string {
	return string(s.builder)
}
