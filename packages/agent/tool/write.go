package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// WriteTool implements file writing with path safety.
type WriteTool struct {
	workspace string
}

func NewWriteTool(workspace string) *WriteTool {
	return &WriteTool{workspace: workspace}
}

func (t *WriteTool) Name() string { return "write" }
func (t *WriteTool) Description() string {
	return "Write content to a file, creating or overwriting it"
}
func (t *WriteTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to write"},
			"content": {"type": "string", "description": "Content to write to the file"}
		},
		"required": ["path", "content"]
	}`)
}

func (t *WriteTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("invalid arguments: " + err.Error())}, IsError: true}, nil
	}

	path := params.Path
	if t.workspace != "" && !filepath.IsAbs(path) {
		path = filepath.Join(t.workspace, path)
	}

	if !isSafePath(t.workspace, path) {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("path outside workspace: " + path)}, IsError: true}, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("cannot create directory: " + err.Error())}, IsError: true}, nil
	}

	if err := os.WriteFile(path, []byte(params.Content), 0644); err != nil {
		return &Result{Content: []model.ContentBlock{model.NewTextContent("write error: " + err.Error())}, IsError: true}, nil
	}

	return &Result{Content: []model.ContentBlock{model.NewTextContent(fmt.Sprintf("wrote %d bytes to %s", len(params.Content), path))}}, nil
}
