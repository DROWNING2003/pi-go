package model

import "encoding/json"

// StreamEvent types.
const (
	StreamEventStart         = "start"
	StreamEventTextStart     = "text_start"
	StreamEventTextDelta     = "text_delta"
	StreamEventTextEnd       = "text_end"
	StreamEventThinkingStart = "thinking_start"
	StreamEventThinkingDelta = "thinking_delta"
	StreamEventThinkingEnd   = "thinking_end"
	StreamEventToolCallStart = "toolcall_start"
	StreamEventToolCallDelta = "toolcall_delta"
	StreamEventToolCallEnd   = "toolcall_end"
	StreamEventDone          = "done"
	StreamEventError         = "error"
)

// StreamEvent represents a single event in a provider response stream.
type StreamEvent struct {
	Type string `json:"type"`

	ContentIndex int               `json:"contentIndex,omitempty"`
	Partial      *AssistantMessage `json:"partial,omitempty"`

	Delta   string `json:"delta,omitempty"`
	Content string `json:"content,omitempty"`

	ToolCall *ContentBlock `json:"toolCall,omitempty"`

	Reason  string            `json:"reason,omitempty"`
	Message *AssistantMessage `json:"message,omitempty"`
	Error   *AssistantMessage `json:"error,omitempty"`
}

// IsTerminal returns true if the event is "done" or "error".
func (e *StreamEvent) IsTerminal() bool {
	return e.Type == StreamEventDone || e.Type == StreamEventError
}

// NewStartEvent creates a stream start event.
func NewStartEvent(partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventStart, Partial: partial}
}

// NewTextStartEvent creates a text_start event.
func NewTextStartEvent(index int, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventTextStart, ContentIndex: index, Partial: partial}
}

// NewTextDeltaEvent creates a text_delta event.
func NewTextDeltaEvent(index int, delta string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventTextDelta, ContentIndex: index, Delta: delta, Partial: partial}
}

// NewTextEndEvent creates a text_end event.
func NewTextEndEvent(index int, content string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventTextEnd, ContentIndex: index, Content: content, Partial: partial}
}

// NewThinkingStartEvent creates a thinking_start event.
func NewThinkingStartEvent(index int, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventThinkingStart, ContentIndex: index, Partial: partial}
}

// NewThinkingDeltaEvent creates a thinking_delta event.
func NewThinkingDeltaEvent(index int, delta string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventThinkingDelta, ContentIndex: index, Delta: delta, Partial: partial}
}

// NewThinkingEndEvent creates a thinking_end event.
func NewThinkingEndEvent(index int, content string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventThinkingEnd, ContentIndex: index, Content: content, Partial: partial}
}

// NewToolCallStartEvent creates a toolcall_start event.
func NewToolCallStartEvent(index int, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventToolCallStart, ContentIndex: index, Partial: partial}
}

// NewToolCallDeltaEvent creates a toolcall_delta event.
func NewToolCallDeltaEvent(index int, delta string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventToolCallDelta, ContentIndex: index, Delta: delta, Partial: partial}
}

// NewToolCallEndEvent creates a toolcall_end event.
func NewToolCallEndEvent(index int, toolCall *ContentBlock, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventToolCallEnd, ContentIndex: index, ToolCall: toolCall, Partial: partial}
}

// NewDoneEvent creates a terminal done event.
func NewDoneEvent(reason StopReason, message *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventDone, Reason: string(reason), Message: message}
}

// NewErrorEvent creates a terminal error event.
func NewErrorEvent(reason StopReason, err *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventError, Reason: string(reason), Error: err}
}

// Ensure json.RawMessage is used.
var _ json.RawMessage
