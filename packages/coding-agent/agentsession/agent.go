// Package agentsession provides the main agent session orchestrator
// matching TS agent-session.ts core logic.
package agentsession

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/DROWNING2003/pi-go/packages/agent/loop"
	"github.com/DROWNING2003/pi-go/packages/agent/tool"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/modelruntime"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/sessionmgr"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/settingsmgr"
	storagesession "github.com/DROWNING2003/pi-go/packages/storage/session"
)

// AgentSession orchestrates agent lifecycle, prompt processing, and session persistence.
type AgentSession struct {
	mu       sync.Mutex
	runtime  *modelruntime.Runtime
	session  *sessionmgr.Manager
	settings *settingsmgr.Manager
	tools    *tool.Registry
	registry *provider.Registry
	cwd      string

	// State
	model         *provider.ProviderModel
	thinkingLevel model.ThinkingLevel
	isStreaming   bool
	messages      []interface{}
	subs          map[string][]chan Event
}

// Event is an agent lifecycle event.
type Event struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data,omitempty"`
	Messages interface{} `json:"messages,omitempty"`
}

// New creates a new agent session.
func New(runtime *modelruntime.Runtime, sess *sessionmgr.Manager, settings *settingsmgr.Manager, tools *tool.Registry, reg *provider.Registry, cwd string) *AgentSession {
	return &AgentSession{
		runtime:  runtime,
		session:  sess,
		settings: settings,
		tools:    tools,
		registry: reg,
		cwd:      cwd,
		model:    runtime.Model,
		subs:     make(map[string][]chan Event),
	}
}

// Subscribe registers an event listener. Returns an unsubscribe function.
func (a *AgentSession) Subscribe(eventType string) (<-chan Event, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ch := make(chan Event, 64)
	a.subs[eventType] = append(a.subs[eventType], ch)
	return ch, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		for i, c := range a.subs[eventType] {
			if c == ch {
				a.subs[eventType] = append(a.subs[eventType][:i], a.subs[eventType][i+1:]...)
				return
			}
		}
	}
}

// emit sends an event to all subscribers.
func (a *AgentSession) emit(event Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ch := range a.subs[event.Type] {
		select {
		case ch <- event:
		default:
		}
	}
}

// Prompt sends a user message and streams the response.
func (a *AgentSession) Prompt(ctx context.Context, message string) error {
	a.mu.Lock()
	a.isStreaming = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isStreaming = false
		a.mu.Unlock()
	}()

	// Emit agent_start
	a.emit(Event{Type: "agent_start"})

	// Create user message
	userData, _ := json.Marshal(model.UserMessage{
		Role:      "user",
		Content:   model.UserContent{model.NewTextContent(message)},
		Timestamp: time.Now().UnixMilli(),
	})

	// Save user message to session
	a.session.AppendEntry(storagesession.Entry{
		Type: "message", Role: "user",
		Content:   userData,
		Timestamp: time.Now().UnixMilli(),
	})

	// Build stream function
	streamFn := func(ctx context.Context, pm *provider.ProviderModel, c *provider.Context, so *provider.StreamOptions) <-chan model.StreamEvent {
		ch, err := a.runtime.Stream(ctx, c, so)
		if err != nil {
			ech := make(chan model.StreamEvent, 1)
			ech <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{ErrorMessage: err.Error()})
			close(ech)
			return ech
		}
		return ch
	}

	// Run agent loop
	config := &loop.Config{
		Model:    a.model,
		Tools:    a.tools,
		MaxTurns: 10,
		StreamFn: streamFn,
	}

	userMsg := &model.UserMessage{
		Role:      "user",
		Content:   model.UserContent{model.NewTextContent(message)},
		Timestamp: time.Now().UnixMilli(),
	}

	messages, err := loop.Run(ctx, config, []*model.UserMessage{userMsg})
	if err != nil {
		a.emit(Event{Type: "agent_end", Messages: []interface{}{}})
		return fmt.Errorf("agent loop: %w", err)
	}

	// Save all messages to session
	for _, msg := range messages {
		data, _ := json.Marshal(msg)
		a.session.AppendEntry(storagesession.Entry{
			Type: "message", Role: msg.Role(),
			Content: data, Timestamp: time.Now().UnixMilli(),
		})
	}

	a.session.Save()

	// Emit agent_end with messages
	a.emit(Event{Type: "agent_end", Messages: messages})
	return nil
}

// SetModel switches the active model.
func (a *AgentSession) SetModel(ref string) error {
	runtime, err := modelruntime.New(ref, a.registry)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.runtime = runtime
	a.model = runtime.Model
	a.mu.Unlock()

	// Save model change entry
	a.session.AppendEntry(storagesession.CreateModelChangeEntry(a.model.Provider, a.model.ID))
	return nil
}

// SetThinkingLevel sets the thinking level.
func (a *AgentSession) SetThinkingLevel(level model.ThinkingLevel) {
	a.mu.Lock()
	a.thinkingLevel = level
	a.mu.Unlock()
}

// State returns the current agent state.
func (a *AgentSession) State() map[string]interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]interface{}{
		"model":         map[string]string{"provider": a.model.Provider, "model": a.model.ID},
		"thinkingLevel": string(a.thinkingLevel),
		"isStreaming":   a.isStreaming,
		"cwd":           a.cwd,
	}
}

// Close cleans up session resources.
func (a *AgentSession) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, chs := range a.subs {
		for _, ch := range chs {
			close(ch)
		}
	}
	a.subs = make(map[string][]chan Event)
}

// Ensure imports used
var _ = fmt.Sprintf
var _ = storagesession.CreateModelChangeEntry
