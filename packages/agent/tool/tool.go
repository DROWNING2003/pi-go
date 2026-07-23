// Package tool defines the tool contract, parameter validation, and the
// four built-in tools: read, write, edit, bash.
package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// Result is the output of a tool execution.
type Result struct {
	Content []model.ContentBlock `json:"content"`
	IsError bool                 `json:"isError"`
}

// Tool is the interface that all tools must implement.
type Tool interface {
	// Name returns the tool name used in tool call requests.
	Name() string
	// Description returns a human-readable description for the model.
	Description() string
	// Parameters returns the JSON Schema for the tool's arguments.
	Parameters() json.RawMessage
	// Execute runs the tool with the given arguments.
	// It must respect context cancellation.
	Execute(ctx context.Context, args json.RawMessage) (*Result, error)
}

// Def is a static tool definition for registration.
type Def struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Registry holds available tools and looks them up by name.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) Tool {
	return r.tools[name]
}

// Execute runs a tool call and returns a ToolResultMessage.
func (r *Registry) Execute(ctx context.Context, toolCallID, toolName string, args json.RawMessage) (*model.ToolResultMessage, error) {
	t := r.Get(toolName)
	if t == nil {
		return &model.ToolResultMessage{
			Role:       "toolResult",
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Content:    []model.ContentBlock{model.NewTextContent(fmt.Sprintf("unknown tool: %s", toolName))},
			IsError:    true,
		}, nil
	}

	result, err := t.Execute(ctx, args)
	if err != nil {
		return &model.ToolResultMessage{
			Role:       "toolResult",
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Content:    []model.ContentBlock{model.NewTextContent(err.Error())},
			IsError:    true,
		}, nil
	}

	return &model.ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Content:    result.Content,
		IsError:    result.IsError,
	}, nil
}

// ValidateJSONSchema performs basic validation of JSON arguments against a
// JSON Schema. This is a simplified validation; for production use, a full
// JSON Schema validator library should be used.
func ValidateJSONSchema(schema json.RawMessage, args json.RawMessage) error {
	// For now, just ensure args is valid JSON
	if !json.Valid(args) {
		return fmt.Errorf("invalid JSON arguments")
	}
	return nil
}
