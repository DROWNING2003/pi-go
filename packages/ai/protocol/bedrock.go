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

// Bedrock ConverseStream request types
type BedrockConverseRequest struct {
	ModelID         string                  `json:"-"`
	System          []BedrockSystemContent  `json:"system,omitempty"`
	Messages        []BedrockMessage        `json:"messages"`
	InferenceConfig *BedrockInferenceConfig `json:"inferenceConfig,omitempty"`
	ToolConfig      *BedrockToolConfig      `json:"toolConfig,omitempty"`
}

type BedrockSystemContent struct {
	Text string `json:"text"`
}

type BedrockMessage struct {
	Role    string                `json:"role"`
	Content []BedrockContentBlock `json:"content"`
}

type BedrockContentBlock struct {
	Text             string                `json:"text,omitempty"`
	Image            *BedrockImage         `json:"image,omitempty"`
	ToolUse          *BedrockToolUse       `json:"toolUse,omitempty"`
	ToolResult       *BedrockToolResult    `json:"toolResult,omitempty"`
	ReasoningContent *BedrockReasoningText `json:"reasoningContent,omitempty"`
}

type BedrockImage struct {
	Format string `json:"format"`
	Source struct {
		Bytes string `json:"bytes"`
	} `json:"source"`
}

type BedrockToolUse struct {
	ToolUseID string          `json:"toolUseId"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
}

type BedrockToolResult struct {
	ToolUseID string                `json:"toolUseId"`
	Content   []BedrockContentBlock `json:"content"`
	Status    string                `json:"status,omitempty"`
}

type BedrockReasoningText struct {
	Text      string `json:"text"`
	Signature string `json:"signature,omitempty"`
}

type BedrockInferenceConfig struct {
	MaxTokens   int     `json:"maxTokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

type BedrockToolConfig struct {
	Tools []BedrockTool `json:"tools"`
}

type BedrockTool struct {
	ToolSpec BedrockToolSpec `json:"toolSpec"`
}

type BedrockToolSpec struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	InputSchema BedrockSchema `json:"inputSchema"`
}

type BedrockSchema struct {
	JSON json.RawMessage `json:"json"`
}

// Bedrock streaming response events
type BedrockStreamEvent struct {
	ContentBlockStart *struct {
		ContentBlockIndex int                  `json:"contentBlockIndex"`
		Start             *BedrockContentBlock `json:"start,omitempty"`
	} `json:"contentBlockStart,omitempty"`
	ContentBlockDelta *struct {
		ContentBlockIndex int           `json:"contentBlockIndex"`
		Delta             *BedrockDelta `json:"delta,omitempty"`
	} `json:"contentBlockDelta,omitempty"`
	ContentBlockStop *struct {
		ContentBlockIndex int `json:"contentBlockIndex"`
	} `json:"contentBlockStop,omitempty"`
	MessageStart *struct {
		Role string `json:"role"`
	} `json:"messageStart,omitempty"`
	MessageStop *struct {
		StopReason string `json:"stopReason"`
	} `json:"messageStop,omitempty"`
	Metadata *struct {
		Usage *BedrockUsage `json:"usage,omitempty"`
	} `json:"metadata,omitempty"`
}

type BedrockDelta struct {
	Text             string `json:"text,omitempty"`
	ReasoningContent *struct {
		Text string `json:"text"`
	} `json:"reasoningContent,omitempty"`
	ToolUse *struct {
		Input string `json:"input"`
	} `json:"toolUse,omitempty"`
}

type BedrockUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

// StreamBedrockConverse sends a Bedrock Converse Stream request.
func StreamBedrockConverse(ctx context.Context, client *HTTPClient, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent {
	ch := make(chan model.StreamEvent, 32)

	go func() {
		defer close(ch)

		req := buildBedrockRequest(m, c, opts)
		resp, err := client.DoStream(ctx, "/model/"+m.ID+"/converse-stream", req, nil)
		if err != nil {
			ch <- model.NewErrorEvent(model.StopReasonError, errorMessage(m, "bedrock: "+err.Error()))
			return
		}
		defer resp.Body.Close()

		parseBedrockStream(ch, m, resp.Body)
	}()

	return ch
}

func buildBedrockRequest(m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) *BedrockConverseRequest {
	req := &BedrockConverseRequest{ModelID: m.ID}

	if c.SystemPrompt != "" {
		req.System = []BedrockSystemContent{{Text: c.SystemPrompt}}
	}

	if opts != nil && opts.MaxTokens > 0 {
		req.InferenceConfig = &BedrockInferenceConfig{MaxTokens: opts.MaxTokens}
	}

	for _, raw := range c.Messages {
		role, content := extractRoleContent(raw)
		msg := BedrockMessage{Role: bedrockRole(role)}
		msg.Content = convertBedrockContent(content)
		if len(msg.Content) > 0 {
			req.Messages = append(req.Messages, msg)
		}
	}

	if len(c.Tools) > 0 {
		req.ToolConfig = &BedrockToolConfig{}
		for _, t := range c.Tools {
			req.ToolConfig.Tools = append(req.ToolConfig.Tools, BedrockTool{
				ToolSpec: BedrockToolSpec{
					Name: t.Name, Description: t.Description,
					InputSchema: BedrockSchema{JSON: t.Parameters},
				},
			})
		}
	}

	return req
}

func bedrockRole(role string) string {
	switch role {
	case "assistant":
		return "assistant"
	case "toolResult":
		return "user"
	default:
		return "user"
	}
}

func convertBedrockContent(content json.RawMessage) []BedrockContentBlock {
	var blocks []BedrockContentBlock
	var str string
	if json.Unmarshal(content, &str) == nil {
		return []BedrockContentBlock{{Text: str}}
	}
	var modelBlocks []model.ContentBlock
	if json.Unmarshal(content, &modelBlocks) != nil {
		return nil
	}
	for _, b := range modelBlocks {
		switch b.Type {
		case model.ContentTypeText:
			blocks = append(blocks, BedrockContentBlock{Text: b.Text})
		case model.ContentTypeImage:
			blocks = append(blocks, BedrockContentBlock{Image: &BedrockImage{
				Format: strings.TrimPrefix(b.MimeType, "image/"),
				Source: struct {
					Bytes string `json:"bytes"`
				}{Bytes: b.Data},
			}})
		case model.ContentTypeToolCall:
			blocks = append(blocks, BedrockContentBlock{ToolUse: &BedrockToolUse{
				ToolUseID: b.ID, Name: b.Name, Input: b.Arguments,
			}})
		}
	}
	return blocks
}

func parseBedrockStream(ch chan<- model.StreamEvent, m *provider.ProviderModel, body io.Reader) {
	parser := NewSSEParser(body)

	output := &model.AssistantMessage{
		Role: "assistant", Content: []model.ContentBlock{},
		API: m.API, Provider: m.Provider, Model: m.ID,
		StopReason: model.StopReasonStop,
		Timestamp:  time.Now().UnixMilli(),
	}

	ch <- model.NewStartEvent(cpAssistant(output))

	var (
		currentText     *model.ContentBlock
		currentThinking *model.ContentBlock
		toolCalls       = map[int]*model.ContentBlock{}
		toolInputs      = map[int]*stringBuilder{}
		stopReason      = "stop"
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
		for idx := 0; ; idx++ {
			tc, ok := toolCalls[idx]
			if !ok {
				break
			}
			if sb, ok := toolInputs[idx]; ok {
				tc.Arguments = json.RawMessage(sb.String())
			}
			pos := len(output.Content)
			output.Content = append(output.Content, *tc)
			ch <- model.NewToolCallEndEvent(pos, cpContent(tc), cpAssistant(output))
		}
		toolCalls = map[int]*model.ContentBlock{}
		toolInputs = map[int]*stringBuilder{}
	}

	for {
		event, err := parser.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ch <- model.NewErrorEvent(model.StopReasonError, errorMessage(m, "bedrock sse: "+err.Error()))
			return
		}

		var be BedrockStreamEvent
		if json.Unmarshal([]byte(event.Data), &be) != nil {
			continue
		}

		if be.MessageStart != nil {
			output.Role = be.MessageStart.Role
		}

		if be.ContentBlockStart != nil && be.ContentBlockStart.Start != nil {
			idx := be.ContentBlockStart.ContentBlockIndex
			cb := be.ContentBlockStart.Start
			switch {
			case cb.Text != "" || (cb.ReasoningContent == nil && cb.ToolUse == nil && cb.ToolResult == nil):
				commitThinking()
				commitToolCalls()
				currentText = &model.ContentBlock{Type: model.ContentTypeText}
				ch <- model.NewTextStartEvent(idx, cpAssistant(output))
			case cb.ReasoningContent != nil:
				commitText()
				commitToolCalls()
				currentThinking = &model.ContentBlock{Type: model.ContentTypeThinking}
				ch <- model.NewThinkingStartEvent(idx, cpAssistant(output))
			case cb.ToolUse != nil:
				commitText()
				commitThinking()
				toolCalls[idx] = &model.ContentBlock{
					Type: model.ContentTypeToolCall,
					ID:   cb.ToolUse.ToolUseID,
					Name: cb.ToolUse.Name,
				}
				toolInputs[idx] = &stringBuilder{}
				ch <- model.NewToolCallStartEvent(idx, cpAssistant(output))
			}
		}

		if be.ContentBlockDelta != nil && be.ContentBlockDelta.Delta != nil {
			idx := be.ContentBlockDelta.ContentBlockIndex
			delta := be.ContentBlockDelta.Delta

			switch {
			case delta.Text != "":
				if currentText != nil {
					currentText.Text += delta.Text
					ch <- model.NewTextDeltaEvent(idx, delta.Text, cpAssistant(output))
				}
			case delta.ReasoningContent != nil:
				if currentThinking != nil {
					currentThinking.Thinking += delta.ReasoningContent.Text
					ch <- model.NewThinkingDeltaEvent(idx, delta.ReasoningContent.Text, cpAssistant(output))
				}
			case delta.ToolUse != nil:
				if sb, ok := toolInputs[idx]; ok {
					sb.WriteString(delta.ToolUse.Input)
					ch <- model.NewToolCallDeltaEvent(idx, delta.ToolUse.Input, cpAssistant(output))
				}
			}
		}

		if be.ContentBlockStop != nil {
			idx := be.ContentBlockStop.ContentBlockIndex
			if currentText != nil {
				commitText()
			} else if currentThinking != nil {
				commitThinking()
			} else if tc, ok := toolCalls[idx]; ok {
				if sb, ok := toolInputs[idx]; ok {
					tc.Arguments = json.RawMessage(sb.String())
				}
				ch <- model.NewToolCallEndEvent(idx, cpContent(tc), cpAssistant(output))
			}
		}

		if be.MessageStop != nil {
			stopReason = be.MessageStop.StopReason
		}

		if be.Metadata != nil && be.Metadata.Usage != nil {
			usage = model.Usage{
				Input:       be.Metadata.Usage.InputTokens,
				Output:      be.Metadata.Usage.OutputTokens,
				TotalTokens: be.Metadata.Usage.TotalTokens,
			}
		}
	}

	commitText()
	commitThinking()
	commitToolCalls()

	output.StopReason = bedrockMapStopReason(stopReason)
	output.Usage = usage

	ch <- model.NewDoneEvent(output.StopReason, output)
}

func bedrockMapStopReason(sr string) model.StopReason {
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
