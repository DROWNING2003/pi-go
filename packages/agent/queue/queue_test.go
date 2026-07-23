package queue

import (
	"testing"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

func TestAbort(t *testing.T) {
	m := NewManager(QueueModeAll)
	if m.IsAborted() {
		t.Error("should not be aborted initially")
	}
	m.Abort("user cancelled")
	if !m.IsAborted() {
		t.Error("should be aborted")
	}
	if m.AbortReason() != "user cancelled" {
		t.Errorf("reason: %q", m.AbortReason())
	}
	m.Reset()
	if m.IsAborted() {
		t.Error("should not be aborted after reset")
	}
}

func TestSteering(t *testing.T) {
	m := NewManager(QueueModeAll)
	msg := &model.UserMessage{Role: "user", Content: model.UserContent{model.NewTextContent("steer")}, Timestamp: time.Now().UnixMilli()}
	m.PushSteering(msg)

	if !m.HasPending() {
		t.Error("should have pending")
	}
	drained := m.DrainSteering()
	if len(drained) != 1 || drained[0].Content[0].Text != "steer" {
		t.Errorf("drained: %+v", drained)
	}
	if m.HasPending() {
		t.Error("should not have pending after drain")
	}
}

func TestFollowUp_AllMode(t *testing.T) {
	m := NewManager(QueueModeAll)
	m.PushFollowUp(&model.UserMessage{Role: "user", Content: model.UserContent{model.NewTextContent("first")}, Timestamp: 1})
	m.PushFollowUp(&model.UserMessage{Role: "user", Content: model.UserContent{model.NewTextContent("second")}, Timestamp: 2})

	drained := m.DrainFollowUp()
	if len(drained) != 2 {
		t.Errorf("expected 2, got %d", len(drained))
	}
}

func TestFollowUp_OneAtATime(t *testing.T) {
	m := NewManager(QueueModeOneAtATime)
	m.PushFollowUp(&model.UserMessage{Role: "user", Content: model.UserContent{model.NewTextContent("first")}, Timestamp: 1})
	m.PushFollowUp(&model.UserMessage{Role: "user", Content: model.UserContent{model.NewTextContent("second")}, Timestamp: 2})

	first := m.DrainFollowUp()
	if len(first) != 1 || first[0].Content[0].Text != "first" {
		t.Errorf("first drain: %+v", first)
	}
	second := m.DrainFollowUp()
	if len(second) != 1 || second[0].Content[0].Text != "second" {
		t.Errorf("second drain: %+v", second)
	}
	if m.HasPending() {
		t.Error("should be empty")
	}
}
