package model

import "encoding/json"

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

type StreamEvent struct {
	Type         string            `json:"type"`
	ContentIndex int               `json:"contentIndex,omitempty"`
	Partial      *AssistantMessage `json:"partial,omitempty"`
	Delta        string            `json:"delta,omitempty"`
	Content      string            `json:"content,omitempty"`
	ToolCall     *ContentBlock     `json:"toolCall,omitempty"`
	Reason       string            `json:"reason,omitempty"`
	Message      *AssistantMessage `json:"message,omitempty"`
	Error        *AssistantMessage `json:"error,omitempty"`
}

func (e *StreamEvent) IsTerminal() bool {
	return e.Type == StreamEventDone || e.Type == StreamEventError
}

func NewStartEvent(partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventStart, Partial: partial}
}
func NewTextStartEvent(index int, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventTextStart, ContentIndex: index, Partial: partial}
}
func NewTextDeltaEvent(index int, delta string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventTextDelta, ContentIndex: index, Delta: delta, Partial: partial}
}
func NewTextEndEvent(index int, content string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventTextEnd, ContentIndex: index, Content: content, Partial: partial}
}
func NewThinkingStartEvent(index int, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventThinkingStart, ContentIndex: index, Partial: partial}
}
func NewThinkingDeltaEvent(index int, delta string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventThinkingDelta, ContentIndex: index, Delta: delta, Partial: partial}
}
func NewThinkingEndEvent(index int, content string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventThinkingEnd, ContentIndex: index, Content: content, Partial: partial}
}
func NewToolCallStartEvent(index int, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventToolCallStart, ContentIndex: index, Partial: partial}
}
func NewToolCallDeltaEvent(index int, delta string, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventToolCallDelta, ContentIndex: index, Delta: delta, Partial: partial}
}
func NewToolCallEndEvent(index int, toolCall *ContentBlock, partial *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventToolCallEnd, ContentIndex: index, ToolCall: toolCall, Partial: partial}
}
func NewDoneEvent(reason StopReason, message *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventDone, Reason: string(reason), Message: message}
}
func NewErrorEvent(reason StopReason, err *AssistantMessage) StreamEvent {
	return StreamEvent{Type: StreamEventError, Reason: string(reason), Error: err}
}

var _ json.RawMessage
