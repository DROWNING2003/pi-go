// Package agentsession provides agent session services and runtime.
package agentsession

import (
	"context"
	"fmt"

	"encoding/json"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/modelresolver"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/modelruntime"
)

// Services holds cwd-bound runtime services for an agent session.
type Services struct {
	CWD      string
	AgentDir string
	Runtime  *modelruntime.Runtime
	Registry *provider.Registry
}

// NewServices creates agent session services for a working directory.
func NewServices(cwd, agentDir string, reg *provider.Registry, modelRef string) (*Services, error) {
	runtime, err := modelruntime.New(modelRef, reg)
	if err != nil {
		return nil, fmt.Errorf("create runtime: %w", err)
	}
	return &Services{
		CWD:      cwd,
		AgentDir: agentDir,
		Runtime:  runtime,
		Registry: reg,
	}, nil
}

// Session holds the runtime state of an agent session.
type Session struct {
	Services      *Services
	Model         *provider.ProviderModel
	ThinkingLevel model.ThinkingLevel
	Messages      []interface{}
	IsStreaming   bool
}

// NewSession creates a new agent session.
func NewSession(svc *Services) *Session {
	return &Session{
		Services:      svc,
		Model:         svc.Runtime.Model,
		ThinkingLevel: model.ThinkingOff,
	}
}

// Prompt sends a message and returns the response stream.
func (s *Session) Prompt(ctx context.Context, message string) (<-chan model.StreamEvent, error) {
	s.IsStreaming = true
	defer func() { s.IsStreaming = false }()

	c := &provider.Context{
		Messages: []json.RawMessage{
			json.RawMessage(fmt.Sprintf(`{"role":"user","content":%q}`, message)),
		},
	}

	return s.Services.Runtime.Stream(ctx, c, nil)
}

// SetModel switches the session's model.
func (s *Session) SetModel(ref string) error {
	runtime, err := modelruntime.New(ref, s.Services.Registry)
	if err != nil {
		return err
	}
	s.Services.Runtime = runtime
	s.Model = runtime.Model
	return nil
}

// CycleModel cycles to the next available model.
func (s *Session) CycleModel() {
	next := modelresolver.CycleModel(s.Model, s.Services.Registry)
	if next != nil {
		s.Model = next
	}
}
