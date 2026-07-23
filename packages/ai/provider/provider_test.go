package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

func collectEvents(ch <-chan model.StreamEvent) []model.StreamEvent {
	var events []model.StreamEvent
	for e := range ch {
		events = append(events, e)
	}
	return events
}

func TestFauxProvider_StreamText(t *testing.T) {
	p := NewFauxProvider()
	p.SetResponses(
		FauxMessage{
			Message: FauxAssistantMessage([]model.ContentBlock{
				FauxText("hello world"),
			}, model.StopReasonStop),
		},
	)

	m := p.GetModel()
	ctx := context.Background()
	c := &Context{
		SystemPrompt: "Be concise.",
		Messages:     []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	events := collectEvents(p.Stream(ctx, m, c, nil))

	if p.CallCount() != 1 {
		t.Errorf("callCount: got %d, want 1", p.CallCount())
	}

	last := events[len(events)-1]
	if last.Type != model.StreamEventDone {
		t.Fatalf("last event type: got %q, want done", last.Type)
	}
	if last.Message == nil {
		t.Fatal("done message is nil")
	}
	if len(last.Message.Content) != 1 || last.Message.Content[0].Text != "hello world" {
		t.Errorf("content mismatch: %+v", last.Message.Content)
	}
}

func TestFauxProvider_ErrorOnExhausted(t *testing.T) {
	p := NewFauxProvider()
	p.SetResponses(
		FauxMessage{Message: FauxAssistantMessage([]model.ContentBlock{FauxText("first")}, model.StopReasonStop)},
	)

	m := p.GetModel()
	ctx := context.Background()
	c := &Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	e1 := collectEvents(p.Stream(ctx, m, c, nil))
	if e1[len(e1)-1].Type != model.StreamEventDone {
		t.Fatal("first call should succeed")
	}

	e2 := collectEvents(p.Stream(ctx, m, c, nil))
	last := e2[len(e2)-1]
	if last.Type != model.StreamEventError {
		t.Fatalf("expected error, got %q", last.Type)
	}
}

func TestFauxProvider_AbortBeforeFirstChunk(t *testing.T) {
	p := NewFauxProvider(WithFauxTokensPerSecond(10))
	p.SetResponses(
		FauxMessage{Message: FauxAssistantMessage([]model.ContentBlock{FauxText("abcdefghij")}, model.StopReasonStop)},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m := p.GetModel()
	c := &Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	events := collectEvents(p.Stream(ctx, m, c, nil))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != model.StreamEventError {
		t.Errorf("expected error, got %q", events[0].Type)
	}
}

func TestFauxProvider_ExactEventOrder(t *testing.T) {
	p := NewFauxProvider(WithFauxTokenSize(1, 1))
	p.SetResponses(
		FauxMessage{
			Message: FauxAssistantMessage([]model.ContentBlock{
				FauxThinking("go"),
				FauxText("ok"),
				FauxToolCall("tool-1", "echo", json.RawMessage(`{}`)),
			}, model.StopReasonToolUse),
		},
	)

	m := p.GetModel()
	c := &Context{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","timestamp":1}`)},
	}

	events := collectEvents(p.Stream(context.Background(), m, c, nil))
	types := make([]string, len(events))
	for i, e := range events {
		types[i] = e.Type
	}

	expected := []string{
		model.StreamEventStart,
		model.StreamEventThinkingStart,
		model.StreamEventThinkingDelta,
		model.StreamEventThinkingEnd,
		model.StreamEventTextStart,
		model.StreamEventTextDelta,
		model.StreamEventTextEnd,
		model.StreamEventToolCallStart,
		model.StreamEventToolCallDelta,
		model.StreamEventToolCallEnd,
		model.StreamEventDone,
	}

	if len(types) != len(expected) {
		t.Fatalf("event count: got %d, want %d\nGot: %v", len(types), len(expected), types)
	}
	for i := range expected {
		if types[i] != expected[i] {
			t.Errorf("event[%d]: got %q, want %q", i, types[i], expected[i])
		}
	}
}
