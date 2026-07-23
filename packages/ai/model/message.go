// Package model defines the standard AI message types, content blocks, usage,
// stop reasons, and error types shared across all providers and agent components.
//
// All types are designed for JSON round-trip fidelity with the TypeScript
// reference implementation's JSONL format.
package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// CurrentSchemaVersion is the active message envelope schema version.
const CurrentSchemaVersion = 1

// API identifies a provider API protocol.
type API string

// ProviderID uniquely identifies a provider.
type ProviderID string

// ContentBlock represents any content block within a message.
// It supports text, thinking, image, and toolCall block types.
// Use the constructor functions (NewTextContent, NewThinkingContent, etc.)
// or the typed helpers (TextContentBlock, etc.) for safe construction.
type ContentBlock struct {
	Type string `json:"type"`

	// TextContent fields
	Text          string `json:"text,omitempty"`
	TextSignature string `json:"textSignature,omitempty"`

	// ThinkingContent fields
	Thinking          string `json:"thinking,omitempty"`
	ThinkingSignature string `json:"thinkingSignature,omitempty"`
	Redacted          bool   `json:"redacted,omitempty"`

	// ImageContent fields
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`

	// ToolCall fields
	ID               string          `json:"id,omitempty"`
	Name             string          `json:"name,omitempty"`
	Arguments        json.RawMessage `json:"arguments,omitempty"`
	ThoughtSignature string          `json:"thoughtSignature,omitempty"`
}

// ContentBlock type constants.
const (
	ContentTypeText     = "text"
	ContentTypeThinking = "thinking"
	ContentTypeImage    = "image"
	ContentTypeToolCall = "toolCall"
)

// NewTextContent creates a text content block.
func NewTextContent(text string) ContentBlock {
	return ContentBlock{Type: ContentTypeText, Text: text}
}

// NewThinkingContent creates a thinking content block.
func NewThinkingContent(thinking string) ContentBlock {
	return ContentBlock{Type: ContentTypeThinking, Thinking: thinking}
}

// NewImageContent creates an image content block with base64-encoded data.
func NewImageContent(data, mimeType string) ContentBlock {
	return ContentBlock{Type: ContentTypeImage, Data: data, MimeType: mimeType}
}

// NewToolCallContent creates a toolCall content block.
func NewToolCallContent(id, name string, arguments json.RawMessage) ContentBlock {
	return ContentBlock{Type: ContentTypeToolCall, ID: id, Name: name, Arguments: arguments}
}

// TextContentBlock provides typed construction of text content.
type TextContentBlock struct {
	Text          string
	TextSignature string
}

// ToContentBlock converts to a generic ContentBlock.
func (t TextContentBlock) ToContentBlock() ContentBlock {
	return ContentBlock{Type: ContentTypeText, Text: t.Text, TextSignature: t.TextSignature}
}

// ThinkingContentBlock provides typed construction of thinking content.
type ThinkingContentBlock struct {
	Thinking          string
	ThinkingSignature string
	Redacted          bool
}

// ToContentBlock converts to a generic ContentBlock.
func (t ThinkingContentBlock) ToContentBlock() ContentBlock {
	return ContentBlock{
		Type:              ContentTypeThinking,
		Thinking:          t.Thinking,
		ThinkingSignature: t.ThinkingSignature,
		Redacted:          t.Redacted,
	}
}

// ImageContentBlock provides typed construction of image content.
type ImageContentBlock struct {
	Data     string
	MimeType string
}

// ToContentBlock converts to a generic ContentBlock.
func (i ImageContentBlock) ToContentBlock() ContentBlock {
	return ContentBlock{Type: ContentTypeImage, Data: i.Data, MimeType: i.MimeType}
}

// ToolCallContentBlock provides typed construction of toolCall content.
type ToolCallContentBlock struct {
	ID               string
	Name             string
	Arguments        json.RawMessage
	ThoughtSignature string
}

// ToContentBlock converts to a generic ContentBlock.
func (t ToolCallContentBlock) ToContentBlock() ContentBlock {
	return ContentBlock{
		Type:             ContentTypeToolCall,
		ID:               t.ID,
		Name:             t.Name,
		Arguments:        t.Arguments,
		ThoughtSignature: t.ThoughtSignature,
	}
}

// UserContent handles the polymorphic user message content: either a plain
// string or an array of content blocks. It implements json.Marshaler and
// json.Unmarshaler for seamless JSON round-tripping.
type UserContent []ContentBlock

// UnmarshalJSON implements json.Unmarshaler, accepting both a plain string
// (which becomes a single text block) and a JSON array of content blocks.
func (u *UserContent) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*u = []ContentBlock{NewTextContent(s)}
		return nil
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(data, &blocks); err != nil {
		return fmt.Errorf("user content: expected string or array: %w", err)
	}
	*u = blocks
	return nil
}

// MarshalJSON implements json.Marshaler. A single text block without signature
// is serialized as a plain string (matching TypeScript behavior).
func (u UserContent) MarshalJSON() ([]byte, error) {
	if len(u) == 1 && u[0].Type == ContentTypeText && u[0].TextSignature == "" {
		return json.Marshal(u[0].Text)
	}
	return json.Marshal([]ContentBlock(u))
}

// UsageCost represents the cost breakdown in dollars.
type UsageCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

// Usage represents token usage and cost information for a model response.
type Usage struct {
	Input        int       `json:"input"`
	Output       int       `json:"output"`
	CacheRead    int       `json:"cacheRead"`
	CacheWrite   int       `json:"cacheWrite"`
	TotalTokens  int       `json:"totalTokens"`
	CacheWrite1h *int      `json:"cacheWrite1h,omitempty"`
	Reasoning    *int      `json:"reasoning,omitempty"`
	Cost         UsageCost `json:"cost"`
}

// StopReason describes why the model stopped generating.
type StopReason string

const (
	StopReasonStop    StopReason = "stop"
	StopReasonLength  StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonError   StopReason = "error"
	StopReasonAborted StopReason = "aborted"
)

// UserMessage represents a message from the user.
type UserMessage struct {
	Role      string      `json:"role"`
	Content   UserContent `json:"content"`
	Timestamp int64       `json:"timestamp"`
}

// AssistantMessage represents a response from the model.
type AssistantMessage struct {
	Role          string         `json:"role"`
	Content       []ContentBlock `json:"content"`
	API           string         `json:"api"`
	Provider      string         `json:"provider"`
	Model         string         `json:"model"`
	ResponseModel string         `json:"responseModel,omitempty"`
	ResponseID    string         `json:"responseId,omitempty"`
	Usage         Usage          `json:"usage"`
	StopReason    StopReason     `json:"stopReason"`
	ErrorMessage  string         `json:"errorMessage,omitempty"`
	Timestamp     int64          `json:"timestamp"`
}

// ToolResultMessage represents the result of a tool execution.
type ToolResultMessage struct {
	Role           string          `json:"role"`
	ToolCallID     string          `json:"toolCallId"`
	ToolName       string          `json:"toolName"`
	Content        []ContentBlock  `json:"content"`
	Details        json.RawMessage `json:"details,omitempty"`
	Usage          *Usage          `json:"usage,omitempty"`
	AddedToolNames []string        `json:"addedToolNames,omitempty"`
	IsError        bool            `json:"isError"`
	Timestamp      int64           `json:"timestamp"`
}

// Model describes a concrete model available from a provider.
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

// MessageEnvelope wraps a message with a schema version for session storage.
type MessageEnvelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	Message       json.RawMessage `json:"message"`
}

// EncodeMessage serializes a message into a versioned envelope.
func EncodeMessage(msg interface{}) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return json.Marshal(MessageEnvelope{SchemaVersion: CurrentSchemaVersion, Message: payload})
}

// DecodeMessage parses a versioned message envelope and returns the decoded
// message. It supports both envelope-wrapped and raw messages.
func DecodeMessage(data []byte) (interface{}, error) {
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
	return UnmarshalMessage(data)
}

// UnmarshalMessage decodes a raw JSON message into the correct concrete type.
func UnmarshalMessage(data []byte) (interface{}, error) {
	var header struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("decode message role: %w", err)
	}
	switch header.Role {
	case "user":
		var msg UserMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("decode user message: %w", err)
		}
		return msg, nil
	case "assistant":
		var msg AssistantMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("decode assistant message: %w", err)
		}
		return msg, nil
	case "toolResult":
		var msg ToolResultMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("decode tool result message: %w", err)
		}
		return msg, nil
	default:
		return nil, fmt.Errorf("unknown message role %q", header.Role)
	}
}
