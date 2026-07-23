package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const CurrentSchemaVersion = 1

type API string
type ProviderID string

type StopReason string

const (
	StopReasonStop    StopReason = "stop"
	StopReasonLength  StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonError   StopReason = "error"
	StopReasonAborted StopReason = "aborted"
)

type ContentBlock interface{ contentBlock() }

type TextContent struct {
	Type          string `json:"type"`
	Text          string `json:"text"`
	TextSignature string `json:"textSignature,omitempty"`
}

func (TextContent) contentBlock() {}

type ThinkingContent struct {
	Type              string `json:"type"`
	Thinking          string `json:"thinking"`
	ThinkingSignature string `json:"thinkingSignature,omitempty"`
	Redacted          bool   `json:"redacted,omitempty"`
}

func (ThinkingContent) contentBlock() {}

type ImageContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

func (ImageContent) contentBlock() {}

type ToolCall struct {
	Type             string         `json:"type"`
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Arguments        map[string]any `json:"arguments"`
	ThoughtSignature string         `json:"thoughtSignature,omitempty"`
}

func (ToolCall) contentBlock() {}

type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead,omitempty"`
	CacheWrite float64 `json:"cacheWrite,omitempty"`
	Total      float64 `json:"total"`
}

type Usage struct {
	Input       int64 `json:"input"`
	Output      int64 `json:"output"`
	CacheRead   int64 `json:"cacheRead,omitempty"`
	CacheWrite  int64 `json:"cacheWrite,omitempty"`
	Reasoning   int64 `json:"reasoning,omitempty"`
	TotalTokens int64 `json:"totalTokens"`
	Cost        Cost  `json:"cost"`
}

type Message interface{ message() }

type UserMessage struct {
	Role      string         `json:"role"`
	Content   []ContentBlock `json:"content"`
	Timestamp int64          `json:"timestamp"`
}

func (UserMessage) message() {}

type AssistantMessage struct {
	Role          string         `json:"role"`
	Content       []ContentBlock `json:"content"`
	API           API            `json:"api"`
	Provider      ProviderID     `json:"provider"`
	Model         string         `json:"model"`
	ResponseModel string         `json:"responseModel,omitempty"`
	ResponseID    string         `json:"responseId,omitempty"`
	Usage         Usage          `json:"usage"`
	StopReason    StopReason     `json:"stopReason"`
	ErrorMessage  string         `json:"errorMessage,omitempty"`
	Timestamp     int64          `json:"timestamp"`
}

func (AssistantMessage) message() {}

type ToolResultMessage struct {
	Role           string         `json:"role"`
	ToolCallID     string         `json:"toolCallId"`
	ToolName       string         `json:"toolName"`
	Content        []ContentBlock `json:"content"`
	Details        any            `json:"details,omitempty"`
	Usage          *Usage         `json:"usage,omitempty"`
	AddedToolNames []string       `json:"addedToolNames,omitempty"`
	IsError        bool           `json:"isError"`
	Timestamp      int64          `json:"timestamp"`
}

func (ToolResultMessage) message() {}

type Model struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	API           API        `json:"api"`
	Provider      ProviderID `json:"provider"`
	BaseURL       string     `json:"baseUrl"`
	Reasoning     bool       `json:"reasoning"`
	Input         []string   `json:"input"`
	ContextWindow int64      `json:"contextWindow"`
	MaxTokens     int64      `json:"maxTokens"`
}

type MessageEnvelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	Message       json.RawMessage `json:"message"`
}

func EncodeMessage(message Message) ([]byte, error) {
	payload, err := marshalMessage(message)
	if err != nil {
		return nil, err
	}
	return json.Marshal(MessageEnvelope{SchemaVersion: CurrentSchemaVersion, Message: payload})
}

func DecodeMessage(data []byte) (Message, error) {
	data = bytes.TrimSpace(data)
	var envelope struct {
		SchemaVersion *int            `json:"schemaVersion"`
		Message       json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode message JSON: %w", err)
	}
	if envelope.SchemaVersion != nil {
		if *envelope.SchemaVersion != CurrentSchemaVersion {
			return nil, fmt.Errorf("unsupported schema version %d", *envelope.SchemaVersion)
		}
		if len(envelope.Message) == 0 {
			return nil, fmt.Errorf("message envelope is missing message")
		}
		data = envelope.Message
	}
	return unmarshalMessage(data)
}

func marshalMessage(message Message) ([]byte, error) {
	switch value := message.(type) {
	case UserMessage:
		return json.Marshal(value)
	case AssistantMessage:
		return json.Marshal(value)
	case ToolResultMessage:
		return json.Marshal(value)
	default:
		return nil, fmt.Errorf("unsupported message type %T", message)
	}
}

func unmarshalMessage(data []byte) (Message, error) {
	var header struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("decode message role: %w", err)
	}
	switch header.Role {
	case "user":
		var raw struct {
			Role      string          `json:"role"`
			Content   json.RawMessage `json:"content"`
			Timestamp int64           `json:"timestamp"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode user message: %w", err)
		}
		content, err := decodeContent(raw.Content, false)
		if err != nil {
			return nil, err
		}
		return UserMessage{Role: raw.Role, Content: content, Timestamp: raw.Timestamp}, nil
	case "assistant":
		var raw struct {
			Role          string          `json:"role"`
			Content       json.RawMessage `json:"content"`
			API           API             `json:"api"`
			Provider      ProviderID      `json:"provider"`
			Model         string          `json:"model"`
			ResponseModel string          `json:"responseModel"`
			ResponseID    string          `json:"responseId"`
			Usage         Usage           `json:"usage"`
			StopReason    StopReason      `json:"stopReason"`
			ErrorMessage  string          `json:"errorMessage"`
			Timestamp     int64           `json:"timestamp"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode assistant message: %w", err)
		}
		content, err := decodeContent(raw.Content, true)
		if err != nil {
			return nil, err
		}
		return AssistantMessage{Role: raw.Role, Content: content, API: raw.API, Provider: raw.Provider, Model: raw.Model, ResponseModel: raw.ResponseModel, ResponseID: raw.ResponseID, Usage: raw.Usage, StopReason: raw.StopReason, ErrorMessage: raw.ErrorMessage, Timestamp: raw.Timestamp}, nil
	case "toolResult":
		var raw struct {
			Role           string          `json:"role"`
			ToolCallID     string          `json:"toolCallId"`
			ToolName       string          `json:"toolName"`
			Content        json.RawMessage `json:"content"`
			Details        any             `json:"details"`
			Usage          *Usage          `json:"usage"`
			AddedToolNames []string        `json:"addedToolNames"`
			IsError        bool            `json:"isError"`
			Timestamp      int64           `json:"timestamp"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode tool result message: %w", err)
		}
		content, err := decodeContent(raw.Content, false)
		if err != nil {
			return nil, err
		}
		return ToolResultMessage{Role: raw.Role, ToolCallID: raw.ToolCallID, ToolName: raw.ToolName, Content: content, Details: raw.Details, Usage: raw.Usage, AddedToolNames: raw.AddedToolNames, IsError: raw.IsError, Timestamp: raw.Timestamp}, nil
	default:
		return nil, fmt.Errorf("unknown message role %q", header.Role)
	}
}

func decodeContent(data json.RawMessage, allowToolCall bool) ([]ContentBlock, error) {
	var blocks []json.RawMessage
	if err := json.Unmarshal(data, &blocks); err != nil {
		var text string
		if textErr := json.Unmarshal(data, &text); textErr == nil {
			return []ContentBlock{TextContent{Type: "text", Text: text}}, nil
		}
		return nil, fmt.Errorf("decode content blocks: %w", err)
	}
	result := make([]ContentBlock, 0, len(blocks))
	for _, raw := range blocks {
		var header struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return nil, fmt.Errorf("decode content type: %w", err)
		}
		switch header.Type {
		case "text":
			var value TextContent
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("decode text content: %w", err)
			}
			result = append(result, value)
		case "thinking":
			var value ThinkingContent
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("decode thinking content: %w", err)
			}
			result = append(result, value)
		case "image":
			var value ImageContent
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("decode image content: %w", err)
			}
			result = append(result, value)
		case "toolCall":
			if !allowToolCall {
				return nil, fmt.Errorf("tool call content is not allowed in this message")
			}
			var value ToolCall
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("decode tool call: %w", err)
			}
			result = append(result, value)
		default:
			return nil, fmt.Errorf("unknown content type %q", header.Type)
		}
	}
	return result, nil
}
