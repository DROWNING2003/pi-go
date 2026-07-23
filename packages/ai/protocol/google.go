package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

var googleToolCallCounter atomic.Int64

func StreamGoogleGenerate(ctx context.Context, client *HTTPClient, m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) <-chan model.StreamEvent {
	ch := make(chan model.StreamEvent, 32)

	go func() {
		defer close(ch)

		output := &model.AssistantMessage{
			Role: "assistant", Content: []model.ContentBlock{},
			API: m.API, Provider: m.Provider, Model: m.ID,
			Usage: model.Usage{}, StopReason: model.StopReasonStop,
			Timestamp: time.Now().UnixMilli(),
		}

		req, err := buildGoogleRequest(m, c, opts)
		if err != nil {
			output.StopReason = model.StopReasonError
			output.ErrorMessage = "build request: " + err.Error()
			ch <- model.NewErrorEvent(model.StopReasonError, output)
			return
		}

		path := fmt.Sprintf("/v1beta/models/%s:streamGenerateContent", m.ID)
		resp, err := client.DoStream(ctx, path, req, nil)
		if err != nil {
			output.StopReason = model.StopReasonError
			output.ErrorMessage = "http error: " + err.Error()
			ch <- model.NewErrorEvent(model.StopReasonError, output)
			return
		}
		defer resp.Body.Close()

		parseGoogleStream(ch, output, resp.Body)
	}()

	return ch
}

func buildGoogleRequest(m *provider.ProviderModel, c *provider.Context, opts *provider.StreamOptions) (*GoogleGenerateRequest, error) {
	req := &GoogleGenerateRequest{
		GenerationConfig: &GoogleGenConfig{},
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req.GenerationConfig.MaxOutputTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			req.GenerationConfig.Temperature = opts.Temperature
		}
	}

	// System instruction
	if c.SystemPrompt != "" {
		req.SystemInstruction = &GoogleContent{
			Parts: []GooglePart{{Text: c.SystemPrompt}},
		}
	}

	// Convert messages following google-shared.ts convertMessages logic
	for _, raw := range c.Messages {
		role, content := extractRoleContent(raw)
		msg, err := convertToGoogleContent(m, role, content)
		if err != nil || msg == nil {
			continue
		}
		req.Contents = append(req.Contents, *msg)
	}

	// Convert tools
	if len(c.Tools) > 0 {
		var decls []GoogleFuncDecl
		for _, t := range c.Tools {
			decls = append(decls, GoogleFuncDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		req.Tools = []GoogleTool{{FunctionDeclarations: decls}}
	}

	return req, nil
}

func convertToGoogleContent(m *provider.ProviderModel, role string, content json.RawMessage) (*GoogleContent, error) {
	msg := &GoogleContent{Role: googleRole(role)}

	switch role {
	case "user":
		msg.Role = "user"
		var str string
		if json.Unmarshal(content, &str) == nil {
			msg.Parts = []GooglePart{{Text: str}}
		} else {
			var blocks []model.ContentBlock
			if json.Unmarshal(content, &blocks) == nil {
				for _, b := range blocks {
					switch b.Type {
					case model.ContentTypeText:
						msg.Parts = append(msg.Parts, GooglePart{Text: b.Text})
					case model.ContentTypeImage:
						msg.Parts = append(msg.Parts, GooglePart{
							InlineData: &GoogleMedia{MimeType: b.MimeType, Data: b.Data},
						})
					}
				}
			}
		}

	case "assistant":
		msg.Role = "model"
		var blocks []model.ContentBlock
		if json.Unmarshal(content, &blocks) != nil {
			return nil, nil
		}
		for _, b := range blocks {
			switch b.Type {
			case model.ContentTypeText:
				if strings.TrimSpace(b.Text) == "" {
					continue
				}
				part := GooglePart{Text: b.Text}
				msg.Parts = append(msg.Parts, part)
			case model.ContentTypeThinking:
				if strings.TrimSpace(b.Thinking) == "" {
					continue
				}
				msg.Parts = append(msg.Parts, GooglePart{
					Thought: true,
					Text:    b.Thinking,
				})
			case model.ContentTypeToolCall:
				msg.Parts = append(msg.Parts, GooglePart{
					FunctionCall: &GoogleFuncCall{Name: b.Name, Args: b.Arguments},
				})
			}
		}

	case "toolResult":
		msg.Role = "user"
		var tr model.ToolResultMessage
		if json.Unmarshal(content, &tr) != nil {
			return nil, nil
		}
		textParts := []string{}
		for _, block := range tr.Content {
			if block.Type == model.ContentTypeText {
				textParts = append(textParts, block.Text)
			}
		}
		responseValue := strings.Join(textParts, "\n")
		resp := GoogleFuncResp{Name: tr.ToolName}
		if tr.IsError {
			resp.Response = json.RawMessage(fmt.Sprintf(`{"error":%q}`, responseValue))
		} else {
			resp.Response = json.RawMessage(fmt.Sprintf(`{"output":%q}`, responseValue))
		}
		msg.Parts = []GooglePart{{FunctionResponse: &resp}}
	}

	if len(msg.Parts) == 0 {
		return nil, nil
	}
	return msg, nil
}

func googleRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return "user"
}

func parseGoogleStream(ch chan<- model.StreamEvent, output *model.AssistantMessage, body io.Reader) {
	parser := NewJSONLinesParser(body)

	ch <- model.NewStartEvent(cpAssistant(output))

	var currentBlock *model.ContentBlock
	blocks := &output.Content

	startBlock := func(isThinking bool) {
		if currentBlock != nil {
			endBlock(currentBlock, blocks, output, ch)
		}
		if isThinking {
			currentBlock = &model.ContentBlock{Type: model.ContentTypeThinking}
			*blocks = append(*blocks, *currentBlock)
			idx := len(*blocks) - 1
			ch <- model.NewThinkingStartEvent(idx, cpAssistant(output))
		} else {
			currentBlock = &model.ContentBlock{Type: model.ContentTypeText}
			*blocks = append(*blocks, *currentBlock)
			idx := len(*blocks) - 1
			ch <- model.NewTextStartEvent(idx, cpAssistant(output))
		}
	}

	for {
		line, err := parser.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			output.StopReason = model.StopReasonError
			output.ErrorMessage = "jsonl parse: " + err.Error()
			ch <- model.NewErrorEvent(model.StopReasonError, output)
			return
		}

		// Google streams as JSON array or individual objects per line
		var responses []GoogleStreamResponse
		if len(line) > 0 && line[0] == '[' {
			if json.Unmarshal(line, &responses) != nil {
				continue
			}
		} else {
			var single GoogleStreamResponse
			if json.Unmarshal(line, &single) != nil {
				continue
			}
			responses = []GoogleStreamResponse{single}
		}

		for _, resp := range responses {
			// Usage metadata
			if resp.UsageMetadata != nil {
				meta := resp.UsageMetadata
				output.Usage = model.Usage{
					Input:       meta.PromptTokenCount - meta.CachedContentTokenCount,
					Output:      meta.CandidatesTokenCount + meta.ThoughtsTokenCount,
					CacheRead:   meta.CachedContentTokenCount,
					Reasoning:   &meta.ThoughtsTokenCount,
					TotalTokens: meta.TotalTokenCount,
				}
			}

			for _, candidate := range resp.Candidates {
				// Finish reason
				if candidate.FinishReason != "" {
					output.StopReason = mapGoogleStopReason(candidate.FinishReason)
				}

				if candidate.Content == nil {
					continue
				}

				for _, part := range candidate.Content.Parts {
					// Thinking vs text: both come as part.text, distinguished by thought flag
					if part.Text != "" || part.Thought {
						isThinking := part.Thought
						if currentBlock == nil ||
							(isThinking && currentBlock.Type != model.ContentTypeThinking) ||
							(!isThinking && currentBlock.Type != model.ContentTypeText) {
							startBlock(isThinking)
						}
						if isThinking {
							currentBlock.Thinking += part.Text
							idx := len(*blocks) - 1
							ch <- model.NewThinkingDeltaEvent(idx, part.Text, cpAssistant(output))
						} else {
							currentBlock.Text += part.Text
							idx := len(*blocks) - 1
							ch <- model.NewTextDeltaEvent(idx, part.Text, cpAssistant(output))
						}
						continue
					}

					// Function call (tool call)
					if part.FunctionCall != nil {
						if currentBlock != nil {
							endBlock(currentBlock, blocks, output, ch)
							currentBlock = nil
						}

						fc := part.FunctionCall
						toolCallID := fc.Name + "_" + strconv.FormatInt(time.Now().UnixMilli(), 10) + "_" + strconv.FormatInt(googleToolCallCounter.Add(1), 10)
						tc := &model.ContentBlock{
							Type:      model.ContentTypeToolCall,
							ID:        toolCallID,
							Name:      fc.Name,
							Arguments: fc.Args,
						}
						*blocks = append(*blocks, *tc)
						idx := len(*blocks) - 1
						ch <- model.NewToolCallStartEvent(idx, cpAssistant(output))
						argsJSON, _ := json.Marshal(fc.Args)
						ch <- model.NewToolCallDeltaEvent(idx, string(argsJSON), cpAssistant(output))
						ch <- model.NewToolCallEndEvent(idx, cpContent(tc), cpAssistant(output))
					}
				}
			}
		}
	}

	// Commit remaining block
	if currentBlock != nil {
		endBlock(currentBlock, blocks, output, ch)
	}

	// If content has tool calls, force toolUse stop reason
	for _, b := range output.Content {
		if b.Type == model.ContentTypeToolCall {
			output.StopReason = model.StopReasonToolUse
			break
		}
	}

	ch <- model.NewDoneEvent(output.StopReason, output)
}

func endBlock(b *model.ContentBlock, blocks *[]model.ContentBlock, output *model.AssistantMessage, ch chan<- model.StreamEvent) {
	idx := findBlockIndex(blocks, b)
	if b.Type == model.ContentTypeText {
		ch <- model.NewTextEndEvent(idx, b.Text, cpAssistant(output))
	} else if b.Type == model.ContentTypeThinking {
		ch <- model.NewThinkingEndEvent(idx, b.Thinking, cpAssistant(output))
	}
}

func findBlockIndex(blocks *[]model.ContentBlock, b *model.ContentBlock) int {
	for i := range *blocks {
		if &(*blocks)[i] == b {
			return i
		}
	}
	return len(*blocks) - 1
}

func mapGoogleStopReason(reason string) model.StopReason {
	switch reason {
	case "STOP":
		return model.StopReasonStop
	case "MAX_TOKENS":
		return model.StopReasonLength
	default:
		return model.StopReasonError
	}
}

// Reuse types from previous google.go (just keep stream adapter above)
type GoogleGenerateRequest struct {
	Contents          []GoogleContent  `json:"contents"`
	SystemInstruction *GoogleContent   `json:"system_instruction,omitempty"`
	Tools             []GoogleTool     `json:"tools,omitempty"`
	GenerationConfig  *GoogleGenConfig `json:"generationConfig,omitempty"`
}
type GoogleGenConfig struct {
	Temperature     float64               `json:"temperature,omitempty"`
	MaxOutputTokens int                   `json:"maxOutputTokens,omitempty"`
	ThinkingConfig  *GoogleThinkingConfig `json:"thinkingConfig,omitempty"`
}
type GoogleThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget,omitempty"`
}
type GoogleContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GooglePart `json:"parts"`
}
type GooglePart struct {
	Text             string          `json:"text,omitempty"`
	InlineData       *GoogleMedia    `json:"inlineData,omitempty"`
	FunctionCall     *GoogleFuncCall `json:"functionCall,omitempty"`
	FunctionResponse *GoogleFuncResp `json:"functionResponse,omitempty"`
	Thought          bool            `json:"thought,omitempty"`
}
type GoogleMedia struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}
type GoogleFuncCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}
type GoogleFuncResp struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}
type GoogleTool struct {
	FunctionDeclarations []GoogleFuncDecl `json:"functionDeclarations,omitempty"`
}
type GoogleFuncDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}
type GoogleStreamResponse struct {
	Candidates    []GoogleCandidate `json:"candidates,omitempty"`
	UsageMetadata *GoogleUsageMeta  `json:"usageMetadata,omitempty"`
}
type GoogleCandidate struct {
	Content      *GoogleContent `json:"content,omitempty"`
	FinishReason string         `json:"finishReason,omitempty"`
}
type GoogleUsageMeta struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	TotalTokenCount         int `json:"totalTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount"`
	ThoughtsTokenCount      int `json:"thoughtsTokenCount"`
}
