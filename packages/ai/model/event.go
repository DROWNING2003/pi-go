package model

import (
	"encoding/json"
	"fmt"
)

type StreamEvent interface{ streamEvent() }

type StartEvent struct {
	Type    string           `json:"type"`
	Partial AssistantMessage `json:"partial"`
}

func (StartEvent) streamEvent() {}

type TextStartEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	Partial      AssistantMessage `json:"partial"`
}

func (TextStartEvent) streamEvent() {}

type TextDeltaEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	Delta        string           `json:"delta"`
	Partial      AssistantMessage `json:"partial"`
}

func (TextDeltaEvent) streamEvent() {}

type TextEndEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	Content      string           `json:"content"`
	Partial      AssistantMessage `json:"partial"`
}

func (TextEndEvent) streamEvent() {}

type ThinkingStartEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	Partial      AssistantMessage `json:"partial"`
}

func (ThinkingStartEvent) streamEvent() {}

type ThinkingDeltaEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	Delta        string           `json:"delta"`
	Partial      AssistantMessage `json:"partial"`
}

func (ThinkingDeltaEvent) streamEvent() {}

type ThinkingEndEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	Content      string           `json:"content"`
	Partial      AssistantMessage `json:"partial"`
}

func (ThinkingEndEvent) streamEvent() {}

type ToolCallStartEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	Partial      AssistantMessage `json:"partial"`
}

func (ToolCallStartEvent) streamEvent() {}

type ToolCallDeltaEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	Delta        string           `json:"delta"`
	Partial      AssistantMessage `json:"partial"`
}

func (ToolCallDeltaEvent) streamEvent() {}

type ToolCallEndEvent struct {
	Type         string           `json:"type"`
	ContentIndex int              `json:"contentIndex"`
	ToolCall     ToolCall         `json:"toolCall"`
	Partial      AssistantMessage `json:"partial"`
}

func (ToolCallEndEvent) streamEvent() {}

type DoneEvent struct {
	Type    string           `json:"type"`
	Reason  StopReason       `json:"reason"`
	Message AssistantMessage `json:"message"`
}

func (DoneEvent) streamEvent() {}

type ErrorEvent struct {
	Type   string           `json:"type"`
	Reason StopReason       `json:"reason"`
	Error  AssistantMessage `json:"error"`
}

func (ErrorEvent) streamEvent() {}

func EncodeStreamEvent(event StreamEvent) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("stream event is nil")
	}
	return json.Marshal(event)
}

func DecodeStreamEvent(data []byte) (StreamEvent, error) {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("decode stream event type: %w", err)
	}
	switch header.Type {
	case "start":
		var raw struct {
			Type    string          `json:"type"`
			Partial json.RawMessage `json:"partial"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode start event: %w", err)
		}
		partial, err := decodeAssistant(raw.Partial)
		if err != nil {
			return nil, err
		}
		return StartEvent{Type: raw.Type, Partial: partial}, nil
	case "text_start":
		return decodePartialEvent[TextStartEvent](data, header.Type)
	case "text_delta":
		return decodePartialEvent[TextDeltaEvent](data, header.Type)
	case "text_end":
		return decodePartialEvent[TextEndEvent](data, header.Type)
	case "thinking_start":
		return decodePartialEvent[ThinkingStartEvent](data, header.Type)
	case "thinking_delta":
		return decodePartialEvent[ThinkingDeltaEvent](data, header.Type)
	case "thinking_end":
		return decodePartialEvent[ThinkingEndEvent](data, header.Type)
	case "toolcall_start":
		return decodePartialEvent[ToolCallStartEvent](data, header.Type)
	case "toolcall_delta":
		return decodePartialEvent[ToolCallDeltaEvent](data, header.Type)
	case "toolcall_end":
		var raw struct {
			Type         string          `json:"type"`
			ContentIndex int             `json:"contentIndex"`
			ToolCall     ToolCall        `json:"toolCall"`
			Partial      json.RawMessage `json:"partial"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode tool call end event: %w", err)
		}
		partial, err := decodeAssistant(raw.Partial)
		if err != nil {
			return nil, err
		}
		return ToolCallEndEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, ToolCall: raw.ToolCall, Partial: partial}, nil
	case "done":
		var raw struct {
			Type    string          `json:"type"`
			Reason  StopReason      `json:"reason"`
			Message json.RawMessage `json:"message"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode done event: %w", err)
		}
		message, err := decodeAssistant(raw.Message)
		if err != nil {
			return nil, err
		}
		return DoneEvent{Type: raw.Type, Reason: raw.Reason, Message: message}, nil
	case "error":
		var raw struct {
			Type   string          `json:"type"`
			Reason StopReason      `json:"reason"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode error event: %w", err)
		}
		message, err := decodeAssistant(raw.Error)
		if err != nil {
			return nil, err
		}
		return ErrorEvent{Type: raw.Type, Reason: raw.Reason, Error: message}, nil
	default:
		return nil, fmt.Errorf("unknown stream event type %q", header.Type)
	}
}

func decodeAssistant(data json.RawMessage) (AssistantMessage, error) {
	message, err := unmarshalMessage(data)
	if err != nil {
		return AssistantMessage{}, err
	}
	assistant, ok := message.(AssistantMessage)
	if !ok {
		return AssistantMessage{}, fmt.Errorf("stream event partial message is not assistant")
	}
	return assistant, nil
}

func decodePartialEvent[T StreamEvent](data []byte, eventType string) (T, error) {
	var raw struct {
		Type         string          `json:"type"`
		ContentIndex int             `json:"contentIndex"`
		Delta        string          `json:"delta"`
		Content      string          `json:"content"`
		Partial      json.RawMessage `json:"partial"`
	}
	var result T
	if err := json.Unmarshal(data, &raw); err != nil {
		return result, fmt.Errorf("decode %s event: %w", eventType, err)
	}
	partial, err := decodeAssistant(raw.Partial)
	if err != nil {
		return result, err
	}
	switch any(result).(type) {
	case TextStartEvent:
		result = any(TextStartEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, Partial: partial}).(T)
	case TextDeltaEvent:
		result = any(TextDeltaEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, Delta: raw.Delta, Partial: partial}).(T)
	case TextEndEvent:
		result = any(TextEndEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, Content: raw.Content, Partial: partial}).(T)
	case ThinkingStartEvent:
		result = any(ThinkingStartEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, Partial: partial}).(T)
	case ThinkingDeltaEvent:
		result = any(ThinkingDeltaEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, Delta: raw.Delta, Partial: partial}).(T)
	case ThinkingEndEvent:
		result = any(ThinkingEndEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, Content: raw.Content, Partial: partial}).(T)
	case ToolCallStartEvent:
		result = any(ToolCallStartEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, Partial: partial}).(T)
	case ToolCallDeltaEvent:
		result = any(ToolCallDeltaEvent{Type: raw.Type, ContentIndex: raw.ContentIndex, Delta: raw.Delta, Partial: partial}).(T)
	default:
		return result, fmt.Errorf("unsupported partial event type %q", eventType)
	}
	return result, nil
}
