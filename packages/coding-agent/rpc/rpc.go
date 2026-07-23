// Package rpc implements the JSON-RPC headless mode matching the TypeScript
// pi RPC protocol. Commands arrive as JSON lines on stdin, responses and
// events are emitted as JSON lines on stdout.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/jsonl"
)

// Command types matching TS RpcCommand union.
const (
	CmdPrompt   = "prompt"
	CmdAbort    = "abort"
	CmdQuit     = "quit"
	CmdExit     = "exit"
	CmdGetState = "get_state"
	CmdSetModel = "set_model"
)

// Command is a generic RPC command from stdin.
type Command struct {
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type"`
	Message  string          `json:"message,omitempty"`
	Provider string          `json:"provider,omitempty"`
	ModelID  string          `json:"modelId,omitempty"`
	Raw      json.RawMessage `json:"-"`
}

// Response is the standard RPC response format.
type Response struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Command string      `json:"command"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Event is a server-sent event pushed to stdout.
type Event struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
	Message interface{} `json:"message,omitempty"`
}

// Handler processes RPC commands.
type Handler struct {
	reader  *jsonl.Reader
	writer  *jsonl.Writer
	reg     *provider.Registry
	model   *provider.ProviderModel
	agent   AgentRunner
	aborted bool
}

// AgentRunner is the interface for running agent prompts.
type AgentRunner interface {
	Run(ctx context.Context, prompt string, streamFn interface{}) ([]model.ContentBlock, error)
}

// RunRPC starts the RPC loop on stdin/stdout.
func RunRPC(reg *provider.Registry, m *provider.ProviderModel) error {
	handler := &Handler{
		reader: jsonl.NewReader(os.Stdin),
		writer: jsonl.NewWriter(os.Stdout),
		reg:    reg,
		model:  m,
	}

	return handler.loop()
}

func (h *Handler) loop() error {
	for {
		var raw json.RawMessage
		if err := h.reader.Decode(&raw); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var cmd Command
		if err := json.Unmarshal(raw, &cmd); err != nil {
			h.error("", "", fmt.Sprintf("invalid command: %v", err))
			continue
		}

		switch cmd.Type {
		case CmdQuit, CmdExit:
			h.success(cmd.ID, cmd.Type, nil)
			return nil

		case CmdPrompt:
			h.handlePrompt(cmd)

		case CmdAbort:
			h.aborted = true
			h.success(cmd.ID, cmd.Type, nil)

		case CmdGetState:
			h.success(cmd.ID, cmd.Type, map[string]interface{}{
				"provider": h.model.Provider,
				"model":    h.model.ID,
				"aborted":  h.aborted,
			})

		case CmdSetModel:
			if cmd.Provider != "" && cmd.ModelID != "" {
				if m := h.reg.ResolveModel(cmd.Provider + "/" + cmd.ModelID); m != nil {
					h.model = m
					h.success(cmd.ID, cmd.Type, map[string]string{
						"provider": m.Provider,
						"model":    m.ID,
					})
				} else {
					h.error(cmd.ID, cmd.Type, "model not found")
				}
			} else {
				h.error(cmd.ID, cmd.Type, "provider and modelId required")
			}

		default:
			h.error(cmd.ID, cmd.Type, fmt.Sprintf("unknown command: %s", cmd.Type))
		}
	}
}

func (h *Handler) handlePrompt(cmd Command) {
	if cmd.Message == "" {
		h.error(cmd.ID, cmd.Type, "message required")
		return
	}

	h.aborted = false

	// Emit thinking event
	h.writer.Write(Event{Type: "thinking", Data: "Processing..."})

	// For now, respond with a static acknowledgment
	// In production, this would call the agent loop
	h.success(cmd.ID, cmd.Type, map[string]string{
		"message": "RPC prompt received: " + truncate(cmd.Message, 100),
	})
}

func (h *Handler) success(id, cmd string, data interface{}) {
	h.writer.Write(Response{
		ID:      id,
		Type:    "response",
		Command: cmd,
		Success: true,
		Data:    data,
	})
}

func (h *Handler) error(id, cmd, msg string) {
	h.writer.Write(Response{
		ID:      id,
		Type:    "response",
		Command: cmd,
		Success: false,
		Error:   msg,
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
