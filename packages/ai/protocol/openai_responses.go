package protocol

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// --- OpenAI Responses request types ---

type ResponsesRequest struct {
	Model           string               `json:"model"`
	Input           []ResponsesInputItem `json:"input"`
	Instructions    string               `json:"instructions,omitempty"`
	Tools           []ResponsesTool      `json:"tools,omitempty"`
	Stream          bool                 `json:"stream"`
	MaxOutputTokens int                  `json:"max_output_tokens,omitempty"`
	Temperature     float64              `json:"temperature,omitempty"`
}

type ResponsesInputItem struct {
	Type    string          `json:"type"`
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Text    string          `json:"text,omitempty"`
	CallID  string          `json:"call_id,omitempty"`
	Output  string          `json:"output,omitempty"`
}

type ResponsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// --- SSE event types ---

type ResponsesStreamEvent struct {
	Type      string               `json:"type"`
	Item      *ResponsesOutputItem `json:"item,omitempty"`
	ItemID    string               `json:"item_id,omitempty"`
	Delta     string               `json:"delta,omitempty"`
	Arguments string               `json:"arguments,omitempty"`
	Part      *ResponsesPart       `json:"part,omitempty"`
	Response  *ResponsesResponse   `json:"response,omitempty"`
}

type ResponsesOutputItem struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Status  string          `json:"status,omitempty"`
	Role    string          `json:"role,omitempty"`
	Name    string          `json:"name,omitempty"`
	Content []ResponsesPart `json:"content,omitempty"`
}

type ResponsesPart struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Summary   []ResponsesPart `json:"summary,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
}

type ResponsesResponse struct {
	ID     string          `json:"id"`
	Model  string          `json:"model"`
	Status string          `json:"status,omitempty"`
	Usage  *ResponsesUsage `json:"usage,omitempty"`
}

type ResponsesUsage struct {
	InputTokens        int `json:"input_tokens"`
	OutputTokens       int `json:"output_tokens"`
	TotalTokens        int `json:"total_tokens"`
	InputTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`
	OutputTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

// --- Streaming adapter ---

func StreamOpenAIResponses(ctx context.Context, client *HTTPClient, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent {
	ch := make(chan model.StreamEvent, 32)

	go func() {
		defer close(ch)

		req := buildResponsesRequest(m, c, opts)
		resp, err := client.DoStream(ctx, "/v1/responses", req, nil)
		if err != nil {
			ch <- model.NewErrorEvent(model.StopReasonError, errorMessage(m, "http: "+err.Error()))
			return
		}
		defer resp.Body.Close()

		parseResponsesStream(ch, m, resp.Body)
	}()

	return ch
}

func buildResponsesRequest(m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) *ResponsesRequest {
	req := &ResponsesRequest{
		Model:  m.ID,
		Stream: true,
	}

	if c.SystemPrompt != "" {
		req.Instructions = c.SystemPrompt
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req.MaxOutputTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			req.Temperature = opts.Temperature
		}
	}

	for _, raw := range c.Messages {
		item := convertToResponsesItem(raw)
		if item != nil {
			req.Input = append(req.Input, *item)
		}
	}

	for _, t := range c.Tools {
		req.Tools = append(req.Tools, ResponsesTool{
			Type: "function", Name: t.Name,
			Description: t.Description, Parameters: t.Parameters,
		})
	}

	return req
}

func convertToResponsesItem(raw json.RawMessage) *ResponsesInputItem {
	role, content := extractRoleContent(raw)
	switch role {
	case "user":
		var str string
		if json.Unmarshal(content, &str) == nil {
			// User text: {"type":"message","role":"user","content":"text"}
			return &ResponsesInputItem{Type: "message", Role: "user", Content: json.RawMessage(`"` + strings.ReplaceAll(str, `"`, `\"`) + `"`)}
		}
		// User with content blocks - simplify to text for now
		return &ResponsesInputItem{Type: "message", Role: "user", Content: content}

	case "assistant":
		// Assistant messages in input are replayed as-is
		return &ResponsesInputItem{Type: "message", Role: "assistant", Content: content}

	case "toolResult":
		var tr model.ToolResultMessage
		if json.Unmarshal(content, &tr) != nil {
			return nil
		}
		textParts := []string{}
		for _, block := range tr.Content {
			if block.Type == model.ContentTypeText {
				textParts = append(textParts, block.Text)
			}
		}
		return &ResponsesInputItem{
			Type:   "function_call_output",
			CallID: tr.ToolCallID,
			Output: strings.Join(textParts, "\n"),
		}
	}
	return nil
}

func parseResponsesStream(ch chan<- model.StreamEvent, m *provider.ProviderModel, body io.Reader) {
	parser := NewSSEParser(body)

	output := &model.AssistantMessage{
		Role: "assistant", Content: []model.ContentBlock{},
		API: m.API, Provider: m.Provider, Model: m.ID,
		StopReason: model.StopReasonStop, Timestamp: time.Now().UnixMilli(),
	}

	ch <- model.NewStartEvent(cpAssistant(output))

	var (
		currentText     *model.ContentBlock
		currentThinking *model.ContentBlock
		toolCallsById   = map[string]*model.ContentBlock{}
		responseID      string
		usage           model.Usage
	)

	commitText := func() {
		if currentText != nil && currentText.Text != "" {
			idx := len(output.Content)
			output.Content = append(output.Content, *currentText)
			ch <- model.NewTextEndEvent(idx, currentText.Text, cpAssistant(output))
			currentText = nil
		}
	}
	commitThinking := func() {
		if currentThinking != nil && currentThinking.Thinking != "" {
			idx := len(output.Content)
			output.Content = append(output.Content, *currentThinking)
			ch <- model.NewThinkingEndEvent(idx, currentThinking.Thinking, cpAssistant(output))
			currentThinking = nil
		}
	}
	commitToolCalls := func() {
		for _, tc := range toolCallsById {
			idx := len(output.Content)
			output.Content = append(output.Content, *tc)
			ch <- model.NewToolCallEndEvent(idx, cpContent(tc), cpAssistant(output))
		}
		toolCallsById = map[string]*model.ContentBlock{}
	}

	for {
		event, err := parser.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			output.StopReason = model.StopReasonError
			output.ErrorMessage = "sse: " + err.Error()
			ch <- model.NewErrorEvent(model.StopReasonError, output)
			return
		}

		var sse ResponsesStreamEvent
		if json.Unmarshal([]byte(event.Data), &sse) != nil {
			continue
		}

		switch sse.Type {
		case "response.created":
			if sse.Response != nil {
				responseID = sse.Response.ID
			}

		case "response.output_text.delta":
			commitThinking()
			commitToolCalls()
			if currentText == nil {
				currentText = &model.ContentBlock{Type: model.ContentTypeText}
				ch <- model.NewTextStartEvent(len(output.Content), cpAssistant(output))
			}
			currentText.Text += sse.Delta
			ch <- model.NewTextDeltaEvent(len(output.Content), sse.Delta, cpAssistant(output))

		case "response.reasoning_text.delta":
			commitText()
			commitToolCalls()
			if currentThinking == nil {
				currentThinking = &model.ContentBlock{Type: model.ContentTypeThinking}
				ch <- model.NewThinkingStartEvent(len(output.Content), cpAssistant(output))
			}
			currentThinking.Thinking += sse.Delta
			ch <- model.NewThinkingDeltaEvent(len(output.Content), sse.Delta, cpAssistant(output))

		case "response.output_item.added":
			commitText()
			commitThinking()
			if sse.Item != nil && sse.Item.Type == "function_call" {
				tc := &model.ContentBlock{
					Type: model.ContentTypeToolCall,
					ID:   sse.Item.ID,
					Name: sse.Item.Name,
				}
				toolCallsById[sse.Item.ID] = tc
				ch <- model.NewToolCallStartEvent(len(output.Content)+len(toolCallsById)-1, cpAssistant(output))
			}

		case "response.function_call_arguments.delta":
			if _, ok := toolCallsById[sse.ItemID]; ok {
				ch <- model.NewToolCallDeltaEvent(len(output.Content), sse.Delta, cpAssistant(output))
			}

		case "response.function_call_arguments.done":
			if tc, ok := toolCallsById[sse.ItemID]; ok {
				tc.Arguments = json.RawMessage(sse.Arguments)
			}

		case "response.output_item.done":
			// Item finalized

		case "response.completed":
			if sse.Response != nil && sse.Response.Usage != nil {
				u := sse.Response.Usage
				usage = model.Usage{
					Input:       u.InputTokens,
					Output:      u.OutputTokens,
					TotalTokens: u.TotalTokens,
					CacheRead:   u.InputTokensDetails.CachedTokens,
					Reasoning:   &u.OutputTokensDetails.ReasoningTokens,
				}
			}
		}
	}

	commitText()
	commitThinking()
	commitToolCalls()

	// Force toolUse if content has tool calls
	for _, b := range output.Content {
		if b.Type == model.ContentTypeToolCall {
			output.StopReason = model.StopReasonToolUse
			break
		}
	}

	output.ResponseID = responseID
	output.Usage = usage

	ch <- model.NewDoneEvent(output.StopReason, output)
}
