package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

func TestFauxProviderStreamsScriptedEventsAndResult(t *testing.T) {
	assistant := model.AssistantMessage{Role: "assistant", Content: []model.ContentBlock{model.TextContent{Type: "text", Text: "hello"}}, API: "faux", Provider: "faux", Model: "test", StopReason: model.StopReasonStop, Timestamp: 2}
	provider := NewFauxProvider([]FauxResponse{{
		Events: []model.StreamEvent{model.StartEvent{Type: "start", Partial: assistant}},
		Result: assistant,
	}})

	stream := provider.Stream(context.Background(), model.Model{ID: "test", Provider: "faux"}, Context{})
	var events []model.StreamEvent
	for event := range stream.Events() {
		events = append(events, event)
	}
	result, err := stream.Result()
	if err != nil {
		t.Fatalf("Stream.Result() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("received %d events, want 1", len(events))
	}
	if result.Model != "test" || result.StopReason != model.StopReasonStop {
		t.Fatalf("result = %#v, want scripted assistant", result)
	}
}

func TestFauxProviderEmitsStandardErrorEvent(t *testing.T) {
	provider := NewFauxProvider([]FauxResponse{{
		Result: model.AssistantMessage{Role: "assistant", API: "faux", Provider: "faux", Model: "test"},
		Err:    errors.New("scripted failure"),
	}})
	stream := provider.Stream(context.Background(), model.Model{ID: "test", Provider: "faux"}, Context{})

	var events []model.StreamEvent
	for event := range stream.Events() {
		events = append(events, event)
	}
	result, err := stream.Result()
	if err == nil || err.Error() != "scripted failure" {
		t.Fatalf("Stream.Result() error = %v, want scripted failure", err)
	}
	if result.StopReason != model.StopReasonError || result.ErrorMessage != "scripted failure" {
		t.Fatalf("error result = %#v, want standard error message", result)
	}
	if len(events) != 1 {
		t.Fatalf("received %d events, want one error event", len(events))
	}
	errorEvent, ok := events[0].(model.ErrorEvent)
	if !ok || errorEvent.Reason != model.StopReasonError || errorEvent.Error.ErrorMessage != "scripted failure" {
		t.Fatalf("event = %#v, want standard error event", events[0])
	}
}

func TestFauxProviderStopsAfterContextCancellation(t *testing.T) {
	provider := NewFauxProvider([]FauxResponse{{
		Delay:  time.Second,
		Result: model.AssistantMessage{Role: "assistant", StopReason: model.StopReasonStop},
	}})
	ctx, cancel := context.WithCancel(context.Background())
	stream := provider.Stream(ctx, model.Model{ID: "test", Provider: "faux"}, Context{})
	cancel()

	select {
	case <-stream.Done():
	case <-time.After(time.Second):
		t.Fatal("faux stream did not stop after context cancellation")
	}
}
